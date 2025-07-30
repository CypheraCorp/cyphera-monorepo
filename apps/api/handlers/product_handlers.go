package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	common                *CommonServices
	delegationClient      *dsClient.DelegationClient
	productService        interfaces.ProductService
	subscriptionService   interfaces.SubscriptionService
	customerService       interfaces.CustomerService
	workspaceService      interfaces.WorkspaceService
	walletService         interfaces.WalletService
	tokenService          interfaces.TokenService
	networkService        interfaces.NetworkService
	gasSponsorshipService interfaces.GasSponsorshipService
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
	if logger == nil {
		logger = zap.L()
	}
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
	product, prices, err := h.productService.GetProduct(c.Request.Context(), params.GetProductParams{
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
	response := helpers.ToProductDetailResponse(*product, prices)
	c.JSON(http.StatusOK, response)
}

// GetPublicProductByPriceID godoc
// @Summary Get public product by price ID
// @Description Get public product details by price ID
// @Tags products
// @Accept json
// @Produce json
// @Param price_id path string true "Price ID"
// @Success 200 {object} PublicProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Tags exclude
func (h *ProductHandler) GetPublicProductByPriceID(c *gin.Context) {
	// Parse price ID
	priceIDStr := c.Param("price_id")
	parsedPriceID, err := uuid.Parse(priceIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidPriceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Get public product using service
	response, err := h.productService.GetPublicProductByPriceID(c.Request.Context(), parsedPriceID)
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

// validateProductUpdate validates core product update parameters
func (h *ProductHandler) validateProductUpdate(c *gin.Context, req UpdateProductRequest, existingProduct db.Product) error {
	if req.Name != "" {
		if len(req.Name) > 255 {
			return errors.New("name must be less than 255 characters")
		}
	}
	if req.WalletID != "" {
		parsedWalletID, err := uuid.Parse(req.WalletID)
		if err != nil {
			return fmt.Errorf("invalid wallet ID format: %w", err)
		}
		if err := h.validateWallet(c, parsedWalletID, existingProduct.WorkspaceID); err != nil {
			return err
		}
	}
	if req.Description != "" {
		if len(req.Description) > 1000 {
			return errors.New("description must be less than 1000 characters")
		}
	}
	if req.ImageURL != "" {
		if _, err := url.ParseRequestURI(req.ImageURL); err != nil {
			return fmt.Errorf("invalid image URL format: %w", err)
		}
	}
	if req.URL != "" {
		if _, err := url.ParseRequestURI(req.URL); err != nil {
			return fmt.Errorf("invalid URL format: %w", err)
		}
	}
	if req.Metadata != nil {
		if !json.Valid(req.Metadata) {
			return errors.New("invalid metadata JSON format")
		}
	}
	return nil
}

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
		// Convert product detail to db.Product for compatibility
		product := db.Product{
			ID:          uuid.MustParse(productDetail.ID),
			WorkspaceID: uuid.MustParse(productDetail.WorkspaceID),
			WalletID:    uuid.MustParse(productDetail.WalletID),
			Name:        productDetail.Name,
			Active:      productDetail.Active,
			CreatedAt:   pgtype.Timestamptz{Time: time.Unix(productDetail.CreatedAt, 0), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Unix(productDetail.UpdatedAt, 0), Valid: true},
		}

		dbPrices, err := h.common.db.ListPricesByProduct(c.Request.Context(), product.ID)
		if err != nil {
			h.common.HandleError(c, err, fmt.Sprintf("Failed to retrieve prices for product %s", product.ID), http.StatusInternalServerError, h.common.GetLogger())
			return
		}
		productResponse := toProductResponse(product, dbPrices)

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

// validateWallet validates the wallet exists and belongs to the workspace's account
func (h *ProductHandler) validateWallet(ctx *gin.Context, walletID uuid.UUID, workspaceID uuid.UUID) error {
	wallet, err := h.common.db.GetWalletByID(ctx.Request.Context(), db.GetWalletByIDParams{
		ID:          walletID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	workspace, err := h.common.db.GetWorkspace(ctx.Request.Context(), workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if wallet.WorkspaceID != workspace.ID {
		return errors.New("wallet does not belong to workspace")
	}

	return nil
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

	// Convert request prices to service params
	servicePrices := make([]params.CreatePriceParams, len(req.Prices))
	for i, price := range req.Prices {
		servicePrices[i] = params.CreatePriceParams{
			Active:              price.Active,
			Type:                price.Type,
			Nickname:            price.Nickname,
			Currency:            price.Currency,
			UnitAmountInPennies: price.UnitAmountInPennies,
			IntervalType:        price.IntervalType,
			IntervalCount:       price.IntervalCount,
			TermLength:          price.TermLength,
			Metadata:            price.Metadata,
		}
	}

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
		Prices:        servicePrices,
		ProductTokens: serviceTokens,
	}

	// Use service to create product
	product, prices, err := h.productService.CreateProduct(c.Request.Context(), serviceParams)
	if err != nil {
		h.common.HandleError(c, err, "Failed to create product", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusCreated, toProductResponse(*product, prices))
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

	c.JSON(http.StatusOK, toProductResponse(*product, []db.Price{}))
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

// Helper function to convert db.Price to PriceResponse
func toPriceResponse(p db.Price) responses.PriceResponse {
	var metadata map[string]interface{}
	if len(p.Metadata) > 0 && string(p.Metadata) != "null" {
		if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
			logger.Error("Error unmarshaling price metadata", zap.Error(err))
			// Set empty metadata on error
			metadata = make(map[string]interface{})
		}
	} else {
		// Set empty metadata for null or empty JSON
		metadata = make(map[string]interface{})
	}

	return responses.PriceResponse{
		ID:                  p.ID.String(),
		Object:              "price",
		ProductID:           p.ProductID.String(),
		Active:              p.Active,
		Type:                string(p.Type),
		Nickname:            p.Nickname.String,
		Currency:            string(p.Currency),
		UnitAmountInPennies: int64(p.UnitAmountInPennies), // Convert int32 to int64
		IntervalType:        string(p.IntervalType),
		TermLength:          p.TermLength,
		Metadata:            p.Metadata,
		CreatedAt:           p.CreatedAt.Time.Unix(),
		UpdatedAt:           p.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert database model to API response
func toProductResponse(p db.Product, dbPrices []db.Price) responses.ProductResponse {
	var metadata map[string]interface{}
	if len(p.Metadata) > 0 && string(p.Metadata) != "null" {
		if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
			logger.Error("Error unmarshaling product metadata", zap.Error(err))
			// Set empty metadata on error
			metadata = make(map[string]interface{})
		}
	} else {
		// Set empty metadata for null or empty JSON
		metadata = make(map[string]interface{})
	}

	apiPrices := make([]responses.PriceResponse, len(dbPrices))
	for i, dbPrice := range dbPrices {
		apiPrices[i] = toPriceResponse(dbPrice)
	}

	return responses.ProductResponse{
		ID:          p.ID.String(),
		Object:      "product",
		WorkspaceID: p.WorkspaceID.String(),
		WalletID:    p.WalletID.String(),
		Name:        p.Name,
		Description: p.Description.String,
		ImageURL:    p.ImageUrl.String,
		URL:         p.Url.String,
		Active:      p.Active,
		Metadata:    p.Metadata,
		Prices:      apiPrices,
		CreatedAt:   p.CreatedAt.Time.Unix(),
		UpdatedAt:   p.UpdatedAt.Time.Unix(),
	}
}

// logFailedSubscriptionCreation records information about failed subscription creation attempts
func (h *ProductHandler) logFailedSubscriptionCreation(
	ctx context.Context,
	customerId *uuid.UUID,
	product db.Product,
	price db.Price,
	productToken db.GetProductTokenRow,
	walletAddress string,
	delegationSignature string,
	err error,
) {
	logger.Error("Failed to create subscription",
		zap.Any("customer_id", customerId),
		zap.String("workspace_id", product.WorkspaceID.String()),
		zap.String("product_id", product.ID.String()),
		zap.String("price_id", price.ID.String()),
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
		ErrorDetails:        []byte(`{"price_id":"` + price.ID.String() + `"}`),
		DelegationSignature: delegationSignaturePgType,
		Metadata:            []byte("{}"),
	})

	if dbErr != nil {
		logger.Error("Failed to create failed subscription attempt record", zap.Error(dbErr))
	}
}

// SubscribeToProductByPriceID godoc
// @Summary Subscribe to a product by price ID
// @Description Subscribe to a product by specifying the price ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Tags exclude
func (h *ProductHandler) SubscribeToProductByPriceID(c *gin.Context) {
	ctx := c.Request.Context()
	priceIDStr := c.Param("price_id")
	parsedPriceID, err := uuid.Parse(priceIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidPriceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	var request requests.SubscribeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.common.HandleError(c, err, errMsgInvalidRequestFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}
	request.PriceID = priceIDStr

	// Convert caveats to JSON for validation
	caveatsJSON, err := json.Marshal(request.Delegation.Caveats)
	if err != nil {
		h.common.HandleError(c, err, "Failed to marshal caveats", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	// Extract the signer field from authority for delegation server compatibility
	// The delegation server expects authority as a hex string (the signer address)
	authorityHex := request.Delegation.Authority.Signer

	// Use product service to validate subscription request
	if err := h.productService.ValidateSubscriptionRequest(ctx, params.ValidateSubscriptionParams{
		SubscriberAddress: request.SubscriberAddress,
		PriceID:           request.PriceID,
		ProductTokenID:    request.ProductTokenID,
		TokenAmount:       request.TokenAmount,
		ProductID:         uuid.Nil, // Will be validated by the service
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

	// Get database pool for transaction handling
	pool, err := h.common.GetDBPool()
	if err != nil {
		h.common.HandleError(c, err, "Failed to get database pool", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	var subscriptionResult *params.SubscriptionCreationResult

	// Capture authorityHex for use inside transaction
	capturedAuthorityHex := authorityHex

	// Execute within transaction
	err = helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		// Get price and validate it's active
		price, err := h.common.db.GetPrice(ctx, parsedPriceID)
		if err != nil {
			return fmt.Errorf("failed to get price: %w", err)
		}

		if !price.Active {
			return errors.New("Cannot subscribe to inactive price")
		}

		// Get product and validate it's active
		product, err := h.common.db.GetProductWithoutWorkspaceId(ctx, price.ProductID)
		if err != nil {
			return fmt.Errorf("failed to get product: %w", err)
		}

		if !product.Active {
			return errors.New("Cannot subscribe to a price of an inactive product")
		}

		// Parse and get required entities
		parsedProductTokenID, err := uuid.Parse(request.ProductTokenID)
		if err != nil {
			return fmt.Errorf("invalid product token ID format: %w", err)
		}

		merchantWallet, err := h.common.db.GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          product.WalletID,
			WorkspaceID: product.WorkspaceID,
		})
		if err != nil {
			return fmt.Errorf("failed to get merchant wallet: %w", err)
		}

		productToken, err := h.common.db.GetProductToken(ctx, parsedProductTokenID)
		if err != nil {
			return fmt.Errorf("failed to get product token: %w", err)
		}

		if productToken.ProductID != product.ID {
			return errors.New("Product token does not belong to the specified product")
		}

		token, err := h.common.db.GetToken(ctx, productToken.TokenID)
		if err != nil {
			return fmt.Errorf("failed to get token: %w", err)
		}

		network, err := h.common.db.GetNetwork(ctx, token.NetworkID)
		if err != nil {
			return fmt.Errorf("failed to get network details: %w", err)
		}

		tokenAmount, err := strconv.ParseInt(request.TokenAmount, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid token amount format: %w", err)
		}

		normalizedAddress := helpers.NormalizeWalletAddress(request.SubscriberAddress, helpers.DetermineNetworkType(productToken.NetworkType))

		// Create delegation params
		delegationParams := params.StoreDelegationDataParams{
			Delegate:  request.Delegation.Delegate,
			Delegator: request.Delegation.Delegator,
			Authority: capturedAuthorityHex, // Use the extracted hex string
			Caveats:   caveatsJSON,
			Salt:      request.Delegation.Salt,
			Signature: request.Delegation.Signature,
		}

		// Use the service method to create subscription with delegation
		result, err := h.subscriptionService.CreateSubscriptionWithDelegation(ctx, tx, params.CreateSubscriptionWithDelegationParams{
			Price:             price,
			Product:           product,
			ProductToken:      productToken,
			MerchantWallet:    merchantWallet,
			Token:             token,
			Network:           network,
			DelegationData:    delegationParams,
			SubscriberAddress: normalizedAddress,
			ProductTokenID:    parsedProductTokenID,
			TokenAmount:       tokenAmount,
		})
		if err != nil {
			var subExistsErr *SubscriptionExistsError
			if errors.As(err, &subExistsErr) {
				return err // Pass through the subscription exists error
			}
			h.logFailedSubscriptionCreation(ctx, nil, product, price, productToken, normalizedAddress, request.Delegation.Signature, err)
			return fmt.Errorf("failed to create subscription: %w", err)
		}

		subscriptionResult = result
		return nil
	})

	// Handle transaction errors
	if err != nil {
		var subExistsErr *SubscriptionExistsError
		if errors.As(err, &subExistsErr) {
			c.JSON(http.StatusConflict, gin.H{
				"message":      "Subscription already exists for this price",
				"subscription": subExistsErr.Subscription,
			})
			return
		}
		h.common.HandleError(c, err, "Transaction failed", http.StatusInternalServerError, h.common.GetLogger())
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
