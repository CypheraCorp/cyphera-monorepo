package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"strconv"
	"strings"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	// Error messages for product handlers
	errMsgInvalidProductIDFormat   = "Invalid product ID format"
	errMsgInvalidPriceIDFormat     = "Invalid price ID format"
	errMsgInvalidTokenIDFormat     = "Invalid product token ID format"
	errMsgInvalidTokenAmountFormat = "Invalid token amount format"
	errMsgInvalidRequestFormat     = "Invalid request format"
	errMsgInvalidWalletIDFormat    = "Invalid wallet ID format"
)

// SwaggerMetadata is used to represent JSON metadata in Swagger docs
type SwaggerMetadata map[string]interface{}

// SubscriptionExistsError represents an error when a subscription already exists
type SubscriptionExistsError struct {
	Subscription *db.Subscription
}

func (e *SubscriptionExistsError) Error() string {
	return "subscription already exists"
}

// ProductHandler handles product-related operations
type ProductHandler struct {
	common              *CommonServices
	delegationClient    *dsClient.DelegationClient
	productService      interfaces.ProductService
	subscriptionService interfaces.SubscriptionService
	customerService     interfaces.CustomerService
}

// NewProductHandler creates a handler with interface dependencies
func NewProductHandler(
	common *CommonServices,
	delegationClient *dsClient.DelegationClient,
	productService interfaces.ProductService,
	subscriptionService interfaces.SubscriptionService,
	customerService interfaces.CustomerService,
	logger *zap.Logger,
) *ProductHandler {
	// Ensure logger is initialized even if not currently used
	_ = logger // Suppress ineffassign warning - logger parameter kept for consistency
	return &ProductHandler{
		common:              common,
		delegationClient:    delegationClient,
		productService:      productService,
		subscriptionService: subscriptionService,
		customerService:     customerService,
	}
}

// Use types from the centralized packages
type (
	PublicProductResponse = responses.PublicProductResponse
	ProductTokenResponse  = responses.ProductTokenResponse
	ListProductsResponse  = responses.ListProductsResponse

	CreatePriceRequest   = requests.CreatePriceRequest
	CreateProductRequest = requests.CreateProductRequest
	UpdateProductRequest = requests.UpdateProductRequest
)

// GetProduct godoc
// @Summary Get product by ID
// @Description Get product details by product ID
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	// Parse workspace ID
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Parse product ID
	productId := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productId)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidProductIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Get product using service
	product, err := h.productService.GetProduct(c.Request.Context(), params.GetProductParams{
		ProductID:   parsedProductID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.common.HandleError(c, err, "Product not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to retrieve product", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	// Convert to response format
	response := helpers.ToProductDetailResponse(*product)
	c.JSON(http.StatusOK, response)
}

// GetPublicProductByID godoc
// @Summary Get public product by product ID
// @Description Get public product details by product ID
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} PublicProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Tags exclude
func (h *ProductHandler) GetPublicProductByID(c *gin.Context) {
	// Parse product ID
	productIDStr := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidProductIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Get public product using service
	response, err := h.productService.GetPublicProductByID(c.Request.Context(), parsedProductID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not active") {
			h.common.HandleError(c, err, err.Error(), http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to retrieve product", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	c.JSON(http.StatusOK, *response)
}

// validatePaginationParams is defined in common.go

// ListProducts godoc
// @Summary List products
// @Description Retrieves paginated products for a workspace
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Number of products to return (default 10, max 100)"
// @Param offset query int false "Number of products to skip (default 0)"
// @Success 200 {object} ListProductsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	pageParams, err := helpers.ParsePaginationParams(c)
	if err != nil {
		h.common.HandleError(c, err, err.Error(), http.StatusBadRequest, h.common.GetLogger())
		return
	}
	limit, page := pageParams.Limit, pageParams.Page

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Use service to list products
	result, err := h.productService.ListProducts(c.Request.Context(), params.ListProductsParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		h.common.HandleError(c, err, "Failed to retrieve products", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	responseList := make([]responses.ProductResponse, len(result.Products))
	for i, productDetail := range result.Products {
		// Convert product detail to db.Product for compatibility with all embedded price fields
		product := db.Product{
			ID:           uuid.MustParse(productDetail.ID),
			WorkspaceID:  uuid.MustParse(productDetail.WorkspaceID),
			WalletID:     uuid.MustParse(productDetail.WalletID),
			Name:         productDetail.Name,
			Description:  pgtype.Text{String: productDetail.Description, Valid: productDetail.Description != ""},
			ImageUrl:     pgtype.Text{String: productDetail.ImageURL, Valid: productDetail.ImageURL != ""},
			Url:          pgtype.Text{String: productDetail.URL, Valid: productDetail.URL != ""},
			Active:       productDetail.Active,
			ProductType:  pgtype.Text{String: productDetail.ProductType, Valid: productDetail.ProductType != ""},
			ProductGroup: pgtype.Text{String: productDetail.ProductGroup, Valid: productDetail.ProductGroup != ""},
			// Embedded price fields
			PriceType:           db.PriceType(productDetail.PriceType),
			Currency:            productDetail.Currency,
			UnitAmountInPennies: int32(productDetail.UnitAmountInPennies),
			IntervalType: func() db.NullIntervalType {
				if productDetail.IntervalType != "" {
					return db.NullIntervalType{IntervalType: db.IntervalType(productDetail.IntervalType), Valid: true}
				}
				return db.NullIntervalType{Valid: false}
			}(),
			TermLength:      pgtype.Int4{Int32: productDetail.TermLength, Valid: productDetail.TermLength > 0},
			PriceNickname:   pgtype.Text{String: productDetail.PriceNickname, Valid: productDetail.PriceNickname != ""},
			PriceExternalID: pgtype.Text{String: productDetail.PriceExternalID, Valid: productDetail.PriceExternalID != ""},
			Metadata:        productDetail.Metadata,
			CreatedAt:       pgtype.Timestamptz{Time: time.Unix(productDetail.CreatedAt, 0), Valid: true},
			UpdatedAt:       pgtype.Timestamptz{Time: time.Unix(productDetail.UpdatedAt, 0), Valid: true},
		}

		productResponse := toProductResponse(product)

		productTokenList, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), product.ID)
		if err != nil {
			h.common.HandleError(c, err, "Failed to retrieve product tokens", http.StatusInternalServerError, h.common.GetLogger())
			return
		}
		// TODO: Fix type conversion from helpers.ProductTokenResponse to responses.ProductTokenResponse
		productTokenListResponse := make([]responses.ProductTokenResponse, len(productTokenList))
		for j, productToken := range productTokenList {
			productTokenListResponse[j] = helpers.ToActiveProductTokenByProductResponse(productToken)
		}
		productResponse.ProductTokens = productTokenListResponse
		responseList[i] = productResponse
	}

	listProductsResponse := ListProductsResponse{
		Object:  "list",
		Data:    responseList,
		HasMore: result.HasMore,
		Total:   result.Total,
	}

	c.JSON(http.StatusOK, listProductsResponse)
}


// CreateProduct godoc
// @Summary Create product
// @Description Creates a new product with associated prices
// @Tags products
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "Product and prices creation data"
// @Success 201 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	parsedWalletID, err := helpers.ValidateWalletID(req.WalletID)
	if err != nil {
		h.common.HandleError(c, nil, err.Error(), http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// No need to convert prices array - pricing is now embedded in the product

	// Convert request product tokens to service params
	serviceTokens := make([]params.CreateProductTokenParams, len(req.ProductTokens))
	for i, token := range req.ProductTokens {
		networkID, err := uuid.Parse(token.NetworkID)
		if err != nil {
			h.common.HandleError(c, err, "Invalid network ID format", http.StatusBadRequest, h.common.GetLogger())
			return
		}
		tokenID, err := uuid.Parse(token.TokenID)
		if err != nil {
			h.common.HandleError(c, err, "Invalid token ID format", http.StatusBadRequest, h.common.GetLogger())
			return
		}
		serviceTokens[i] = params.CreateProductTokenParams{
			NetworkID: networkID,
			TokenID:   tokenID,
			Active:    token.Active,
		}
	}

	serviceParams := params.CreateProductParams{
		WorkspaceID:   parsedWorkspaceID,
		WalletID:      parsedWalletID,
		Name:          req.Name,
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		URL:           req.URL,
		Active:        req.Active,
		Metadata:      req.Metadata,
		ProductTokens: serviceTokens,
		// Embedded price fields
		PriceType:           req.PriceType,
		Currency:            req.Currency,
		UnitAmountInPennies: int32(req.UnitAmountInPennies),
		IntervalType:        req.IntervalType,
		TermLength:          req.TermLength,
		PriceNickname:       req.PriceNickname,
	}

	// Use service to create product
	product, err := h.productService.CreateProduct(c.Request.Context(), serviceParams)
	if err != nil {
		h.common.HandleError(c, err, "Failed to create product", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusCreated, toProductResponse(*product))
}

// UpdateProduct godoc
// @Summary Update product
// @Description Updates an existing product. Price updates should be done via price-specific endpoints.
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param product body UpdateProductRequest true "Product update data (prices are not updated here)"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidProductIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Convert request to service params
	updateParams := params.UpdateProductParams{
		ProductID:   parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	}

	if req.Name != "" {
		updateParams.Name = &req.Name
	}
	if req.Description != "" {
		updateParams.Description = &req.Description
	}
	if req.ImageURL != "" {
		updateParams.ImageURL = &req.ImageURL
	}
	if req.URL != "" {
		updateParams.URL = &req.URL
	}
	if req.Active != nil {
		updateParams.Active = req.Active
	}
	if req.Metadata != nil {
		updateParams.Metadata = req.Metadata
	}
	if req.WalletID != "" {
		parsedWalletID, err := uuid.Parse(req.WalletID)
		if err != nil {
			h.common.HandleError(c, err, errMsgInvalidWalletIDFormat, http.StatusBadRequest, h.common.GetLogger())
			return
		}
		updateParams.WalletID = &parsedWalletID
	}

	// Use service to update product
	product, err := h.productService.UpdateProduct(c.Request.Context(), updateParams)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.common.HandleError(c, err, "Product not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to update product", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	c.JSON(http.StatusOK, toProductResponse(*product))
}

// DeleteProduct godoc
// @Summary Delete product
// @Description Deletes a product and its associated prices (soft delete)
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidProductIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Use service to delete product
	err = h.productService.DeleteProduct(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.common.HandleError(c, err, "Product not found", http.StatusNotFound, h.common.GetLogger())
		} else if strings.Contains(err.Error(), "active subscriptions") {
			h.common.HandleError(c, err, "Cannot delete product with active subscriptions", http.StatusConflict, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to delete product", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Helper function to convert database model to API response
func toProductResponse(p db.Product) responses.ProductResponse {
	return responses.ProductResponse{
		ID:           p.ID.String(),
		Object:       "product",
		WorkspaceID:  p.WorkspaceID.String(),
		WalletID:     p.WalletID.String(),
		Name:         p.Name,
		Description:  p.Description.String,
		ImageURL:     p.ImageUrl.String,
		URL:          p.Url.String,
		Active:       p.Active,
		Metadata:     p.Metadata,
		ProductType:  p.ProductType.String,
		ProductGroup: p.ProductGroup.String,
		// Embedded price fields
		PriceType:           string(p.PriceType),
		Currency:            string(p.Currency),
		UnitAmountInPennies: int64(p.UnitAmountInPennies),
		IntervalType: func() string {
			if p.IntervalType.Valid {
				return string(p.IntervalType.IntervalType)
			}
			return ""
		}(),
		TermLength:      p.TermLength.Int32,
		PriceNickname:   p.PriceNickname.String,
		PriceExternalID: p.PriceExternalID.String,
		CreatedAt:       p.CreatedAt.Time.Unix(),
		UpdatedAt:       p.UpdatedAt.Time.Unix(),
	}
}

// logFailedSubscriptionCreation records information about failed subscription creation attempts
func (h *ProductHandler) logFailedSubscriptionCreation(
	ctx context.Context,
	customerId *uuid.UUID,
	product db.Product,
	productToken db.GetProductTokenRow,
	walletAddress string,
	delegationSignature string,
	err error,
) {
	logger.Error("Failed to create subscription",
		zap.Any("customer_id", customerId),
		zap.String("workspace_id", product.WorkspaceID.String()),
		zap.String("product_id", product.ID.String()),
		zap.String("product_id", product.ID.String()),
		zap.String("product_token_id", productToken.ID.String()),
		zap.String("wallet_address", walletAddress),
		zap.String("delegation_signature", delegationSignature),
		zap.Error(err),
	)

	errorType := helpers.DetermineErrorType(err)
	var customerIDPgType pgtype.UUID
	if customerId != nil {
		customerIDPgType = pgtype.UUID{Bytes: *customerId, Valid: true}
	} else {
		customerIDPgType = pgtype.UUID{Valid: false}
	}

	customerWalletIDPgType := pgtype.UUID{Valid: false}
	var delegationSignaturePgType pgtype.Text
	if delegationSignature != "" {
		delegationSignaturePgType = pgtype.Text{String: delegationSignature, Valid: true}
	} else {
		delegationSignaturePgType = pgtype.Text{Valid: false}
	}

	_, dbErr := h.common.db.CreateFailedSubscriptionAttempt(ctx, db.CreateFailedSubscriptionAttemptParams{
		CustomerID:          customerIDPgType,
		ProductID:           product.ID,
		ProductTokenID:      productToken.ID,
		CustomerWalletID:    customerWalletIDPgType,
		WalletAddress:       walletAddress,
		ErrorType:           errorType,
		ErrorMessage:        err.Error(),
		ErrorDetails:        []byte(`{"product_id":"` + product.ID.String() + `"}`),
		DelegationSignature: delegationSignaturePgType,
		Metadata:            []byte("{}"),
	})

	if dbErr != nil {
		logger.Error("Failed to create failed subscription attempt record", zap.Error(dbErr))
	}
}

// SubscribeToProductByID godoc
// @Summary Subscribe to a product by product ID
// @Description Subscribe to a product by specifying the product ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Tags exclude
func (h *ProductHandler) SubscribeToProductByID(c *gin.Context) {
	ctx := c.Request.Context()
	productIDStr := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidProductIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	var request requests.SubscribeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.common.HandleError(c, err, errMsgInvalidRequestFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}
	request.ProductID = productIDStr

	// Convert caveats to JSON for validation
	caveatsJSON, err := json.Marshal(request.Delegation.Caveats)
	if err != nil {
		h.common.HandleError(c, err, "Failed to marshal caveats", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	// The authority field is already a hex string in MetaMask delegation format
	authorityHex := request.Delegation.Authority

	// Use product service to validate subscription request
	if err := h.productService.ValidateSubscriptionRequest(ctx, params.ValidateSubscriptionParams{
		SubscriberAddress: request.SubscriberAddress,
		ProductID:         request.ProductID,
		ProductTokenID:    request.ProductTokenID,
		TokenAmount:       request.TokenAmount,
		Delegation: params.DelegationParams{
			Delegate:  request.Delegation.Delegate,
			Delegator: request.Delegation.Delegator,
			Authority: authorityHex,
			Salt:      request.Delegation.Salt,
			Signature: request.Delegation.Signature,
			Caveats:   caveatsJSON,
		},
		CypheraSmartWalletAddress: h.common.GetCypheraSmartWalletAddress(),
	}); err != nil {
		h.common.HandleError(c, err, err.Error(), http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Get product and validate it's active
	product, err := h.common.db.GetProductWithoutWorkspaceId(ctx, parsedProductID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get product", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	if !product.Active {
		h.common.HandleError(c, errors.New("Cannot subscribe to inactive product"), "Product is not active", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Parse and get required entities
	parsedProductTokenID, err := uuid.Parse(request.ProductTokenID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidTokenIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	merchantWallet, err := h.common.db.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		h.common.HandleError(c, err, "Failed to get merchant wallet", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	productToken, err := h.common.db.GetProductToken(ctx, parsedProductTokenID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get product token", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	if productToken.ProductID != product.ID {
		h.common.HandleError(c, errors.New("Product token does not belong to the specified product"), "Invalid product token", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	token, err := h.common.db.GetToken(ctx, productToken.TokenID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get token", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	network, err := h.common.db.GetNetwork(ctx, token.NetworkID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get network details", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	tokenAmount, err := strconv.ParseInt(request.TokenAmount, 10, 64)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidTokenAmountFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	normalizedAddress := helpers.NormalizeWalletAddress(request.SubscriberAddress, helpers.DetermineNetworkType(productToken.NetworkType))

	// Create delegation params
	delegationParams := params.StoreDelegationDataParams{
		Delegate:  request.Delegation.Delegate,
		Delegator: request.Delegation.Delegator,
		Authority: authorityHex,
		Caveats:   caveatsJSON,
		Salt:      request.Delegation.Salt,
		Signature: request.Delegation.Signature,
	}

	// Convert addons from request to params
	var addons []params.SubscriptionAddonParams
	if len(request.Addons) > 0 {
		for _, addon := range request.Addons {
			addonProductID, err := uuid.Parse(addon.ProductID)
			if err != nil {
				h.common.HandleError(c, err, "Invalid addon product ID format", http.StatusBadRequest, h.common.GetLogger())
				return
			}
			addons = append(addons, params.SubscriptionAddonParams{
				ProductID: addonProductID,
				Quantity:  addon.Quantity,
			})
		}
	}

	// Get database pool for transaction handling
	pool, err := h.common.GetDBPool()
	if err != nil {
		h.common.HandleError(c, err, "Failed to get database pool", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	var subscriptionResult *params.SubscriptionCreationResult

	// Execute the new flow - blockchain transaction first, then DB records
	err = helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		// Use the service method to create subscription with delegation
		// This now executes the blockchain transaction FIRST
		result, err := h.subscriptionService.CreateSubscriptionWithDelegation(ctx, tx, params.CreateSubscriptionWithDelegationParams{
			Product:           product,
			ProductToken:      productToken,
			MerchantWallet:    merchantWallet,
			Token:             token,
			Network:           network,
			DelegationData:    delegationParams,
			SubscriberAddress: normalizedAddress,
			ProductTokenID:    parsedProductTokenID,
			TokenAmount:       tokenAmount,
			Addons:            addons, // Pass addons to create line items
		})
		if err != nil {
			// The service will have already logged to failed_subscription_attempts
			return err
		}

		subscriptionResult = result
		return nil
	})

	// Transaction completed successfully, now create invoice
	if err == nil && subscriptionResult != nil && subscriptionResult.Subscription != nil {
		// Calculate period dates for invoice
		periodStart, periodEnd, _ := helpers.CalculateSubscriptionPeriods(product)
		
		// Get payment ID from the subscription events
		events, evtErr := h.common.db.ListSubscriptionEventsBySubscription(ctx, subscriptionResult.Subscription.ID)
		
		if evtErr == nil && len(events) > 0 {
			// Find the redeemed event to get payment ID
			for _, event := range events {
				if event.EventType == db.SubscriptionEventTypeRedeemed && event.TransactionHash.Valid {
					// Get payment by transaction hash
					payments, payErr := h.common.db.GetPaymentsByTransactionHash(ctx, event.TransactionHash)
					if payErr == nil && len(payments) > 0 {
						// Create invoice after transaction is committed
						invErr := h.subscriptionService.CreateInvoiceForSubscriptionPayment(
							ctx,
							subscriptionResult.Subscription.ID,
							payments[0].ID,
							periodStart,
							periodEnd,
							event.TransactionHash.String,
						)
						if invErr != nil {
							logger.Error("Failed to create invoice after subscription creation",
								zap.Error(invErr),
								zap.String("subscription_id", subscriptionResult.Subscription.ID.String()),
								zap.String("payment_id", payments[0].ID.String()))
							// Don't fail the whole operation - subscription was created successfully
						} else {
							// For recurring subscriptions, generate invoice for next period
							if product.PriceType == db.PriceTypeRecurring {
								// Get the updated subscription to check if it's still active
								updatedSub, subErr := h.common.db.GetSubscription(ctx, subscriptionResult.Subscription.ID)
								if subErr == nil && updatedSub.Status == db.SubscriptionStatusActive && 
									updatedSub.NextRedemptionDate.Valid {
									// Calculate next period dates
									nextPeriodStart := periodEnd
									nextPeriodEnd := updatedSub.NextRedemptionDate.Time
									
									// Generate open invoice for next period
									if h.subscriptionService != nil {
										nextInvoice, nextErr := h.subscriptionService.GenerateNextPeriodInvoice(
											ctx,
											subscriptionResult.Subscription.ID,
											nextPeriodStart,
											nextPeriodEnd,
										)
										if nextErr != nil {
											logger.Error("Failed to generate invoice for next period",
												zap.Error(nextErr),
												zap.String("subscription_id", subscriptionResult.Subscription.ID.String()),
												zap.Time("next_period_start", nextPeriodStart),
												zap.Time("next_period_end", nextPeriodEnd))
										} else {
											logger.Info("Generated invoice for next subscription period",
												zap.String("subscription_id", subscriptionResult.Subscription.ID.String()),
												zap.String("invoice_id", nextInvoice.ID.String()),
												zap.String("invoice_number", nextInvoice.InvoiceNumber),
												zap.Time("period_start", nextPeriodStart),
												zap.Time("period_end", nextPeriodEnd))
										}
									}
								}
							}
						}
						break
					}
				}
			}
		}
	}
	
	// Handle errors
	if err != nil {
		// Check for specific error types
		var subExistsErr *SubscriptionExistsError
		if errors.As(err, &subExistsErr) {
			c.JSON(http.StatusConflict, gin.H{
				"message":      "Subscription already exists for this product",
				"subscription": subExistsErr.Subscription,
			})
			return
		}
		
		// For other errors, check the message to provide appropriate status codes
		errMsg := err.Error()
		if strings.Contains(errMsg, "delegation redemption failed") {
			h.common.HandleError(c, err, "Payment transaction failed", http.StatusPaymentRequired, h.common.GetLogger())
		} else if strings.Contains(errMsg, "failed to create subscription records after successful payment") {
			// This is critical - payment went through but DB operations failed
			h.common.HandleError(c, err, "Payment successful but subscription creation failed - please contact support", http.StatusInternalServerError, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to create subscription", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	if subscriptionResult == nil {
		h.common.HandleError(c, errors.New("subscription creation failed"), "Subscription creation failed", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	// Create comprehensive response using the helper
	h.common.GetLogger().Info("Creating comprehensive subscription response",
		zap.String("subscription_id", subscriptionResult.Subscription.ID.String()),
		zap.String("transaction_hash", subscriptionResult.TransactionHash))

	// Create a context with timeout to avoid indefinite hangs
	responseCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	comprehensiveResponse, err := helpers.ToComprehensiveSubscriptionResponse(responseCtx, h.common.db, *subscriptionResult.Subscription)
	if err != nil {
		logger.Error("Failed to create comprehensive subscription response",
			zap.Error(err),
			zap.String("subscription_id", subscriptionResult.Subscription.ID.String()))
		h.common.HandleError(c, err, "Failed to create subscription response", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	h.common.GetLogger().Info("Successfully created comprehensive subscription response",
		zap.String("subscription_id", subscriptionResult.Subscription.ID.String()))

	c.JSON(http.StatusCreated, comprehensiveResponse)
}
