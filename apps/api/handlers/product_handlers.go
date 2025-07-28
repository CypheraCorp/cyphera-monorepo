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
	"github.com/cyphera/cyphera-api/libs/go/types/business"

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
	logger *zap.Logger,
) *ProductHandler {
	if logger == nil {
		logger = zap.L()
	}
	return &ProductHandler{
		common:           common,
		delegationClient: delegationClient,
		productService:   productService,
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

// validateSubscriptionRequest validates the basic request parameters
func (h *ProductHandler) validateSubscriptionRequest(request requests.SubscribeRequest, productID uuid.UUID) error {
	if request.SubscriberAddress == "" {
		return errors.New("subscriber address is required")
	}

	if _, err := uuid.Parse(request.PriceID); err != nil {
		return errors.New("invalid price ID format")
	}

	if _, err := uuid.Parse(request.ProductTokenID); err != nil {
		return errors.New("invalid product token ID format")
	}

	tokenAmount, err := strconv.ParseInt(request.TokenAmount, 10, 64)
	if err != nil {
		return errors.New("invalid token amount format")
	}

	if tokenAmount <= 0 {
		return errors.New("token amount must be greater than 0")
	}

	cypheraAddress := h.common.GetCypheraSmartWalletAddress()
	if request.Delegation.Delegate != cypheraAddress {
		return fmt.Errorf("delegate address does not match cyphera smart wallet address, %s != %s", request.Delegation.Delegate, cypheraAddress)
	}

	if request.Delegation.Delegate == "" || request.Delegation.Delegator == "" ||
		request.Delegation.Authority == "" || request.Delegation.Salt == "" ||
		request.Delegation.Signature == "" {
		return errors.New("incomplete delegation data")
	}

	return nil
}

// processCustomerAndWallet handles customer lookup/creation and wallet association
func (h *ProductHandler) processCustomerAndWallet(
	ctx context.Context,
	tx *db.Queries,
	walletAddress string,
	product db.Product,
	productToken db.GetProductTokenRow,
) (db.Customer, db.CustomerWallet, error) {
	customers, err := tx.GetCustomersByWalletAddress(ctx, walletAddress)
	if err != nil {
		logger.Error("Failed to check for existing customers",
			zap.Error(err),
			zap.String("wallet_address", walletAddress))
		return db.Customer{}, db.CustomerWallet{}, err
	}

	networkType := helpers.DetermineNetworkType(productToken.NetworkType)

	if len(customers) == 0 {
		return h.createNewCustomerWithWallet(ctx, tx, walletAddress, product, networkType)
	}

	customer := customers[0]

	// Ensure customer is associated with the current workspace
	isAssociated, err := tx.IsCustomerInWorkspace(ctx, db.IsCustomerInWorkspaceParams{
		WorkspaceID: product.WorkspaceID,
		CustomerID:  customer.ID,
	})
	if err != nil {
		logger.Error("Failed to check customer workspace association",
			zap.Error(err),
			zap.String("customer_id", customer.ID.String()),
			zap.String("workspace_id", product.WorkspaceID.String()))
		return db.Customer{}, db.CustomerWallet{}, err
	}

	// If customer is not associated with this workspace, create the association
	if !isAssociated {
		_, err = tx.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
			WorkspaceID: product.WorkspaceID,
			CustomerID:  customer.ID,
		})
		if err != nil {
			logger.Error("Failed to associate customer with workspace",
				zap.Error(err),
				zap.String("customer_id", customer.ID.String()),
				zap.String("workspace_id", product.WorkspaceID.String()))
			return db.Customer{}, db.CustomerWallet{}, err
		}

		logger.Info("Associated existing customer with workspace",
			zap.String("customer_id", customer.ID.String()),
			zap.String("workspace_id", product.WorkspaceID.String()),
			zap.String("wallet_address", walletAddress))
	}

	customerWallet, err := h.findOrCreateCustomerWallet(ctx, tx, customer, walletAddress, networkType, product.ID.String())

	return customer, customerWallet, err
}

// createNewCustomerWithWallet creates a new customer and associated wallet
func (h *ProductHandler) createNewCustomerWithWallet(
	ctx context.Context,
	tx *db.Queries,
	walletAddress string,
	product db.Product,
	networkType string,
) (db.Customer, db.CustomerWallet, error) {
	logger.Info("Creating new customer for wallet address",
		zap.String("wallet_address", walletAddress),
		zap.String("product_id", product.ID.String()))

	metadata := map[string]interface{}{
		"source":                  "product_subscription",
		"created_from_product_id": product.ID.String(),
		"wallet_address":          walletAddress,
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return db.Customer{}, db.CustomerWallet{}, err
	}

	createCustomerParams := db.CreateCustomerParams{
		Email: pgtype.Text{
			String: "",
			Valid:  false,
		},
		Name: pgtype.Text{
			String: "Wallet Customer: " + walletAddress,
			Valid:  true,
		},
		Description: pgtype.Text{
			String: "Customer created from product subscription",
			Valid:  true,
		},
		Metadata: metadataBytes,
	}

	customer, err := tx.CreateCustomer(ctx, createCustomerParams)
	if err != nil {
		return db.Customer{}, db.CustomerWallet{}, err
	}

	// Associate customer with workspace using the new association table
	_, err = tx.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
		WorkspaceID: product.WorkspaceID,
		CustomerID:  customer.ID,
	})
	if err != nil {
		return db.Customer{}, db.CustomerWallet{}, err
	}

	walletMetadata := map[string]interface{}{
		"source":     "product_subscription",
		"product_id": product.ID.String(),
		"created_at": time.Now().Format(time.RFC3339),
	}
	walletMetadataBytes, err := json.Marshal(walletMetadata)
	if err != nil {
		return db.Customer{}, db.CustomerWallet{}, err
	}

	createWalletParams := db.CreateCustomerWalletParams{
		CustomerID:    customer.ID,
		WalletAddress: walletAddress,
		NetworkType:   db.NetworkType(networkType),
		Nickname: pgtype.Text{
			String: "Subscription Wallet",
			Valid:  true,
		},
		IsPrimary: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
		Verified: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
		Metadata: walletMetadataBytes,
	}

	customerWallet, err := tx.CreateCustomerWallet(ctx, createWalletParams)
	return customer, customerWallet, err
}

// findOrCreateCustomerWallet finds an existing wallet or creates a new one
func (h *ProductHandler) findOrCreateCustomerWallet(
	ctx context.Context,
	tx *db.Queries,
	customer db.Customer,
	walletAddress string,
	networkType string,
	productID string,
) (db.CustomerWallet, error) {
	wallets, err := tx.ListCustomerWallets(ctx, customer.ID)
	if err != nil {
		return db.CustomerWallet{}, err
	}

	for _, wallet := range wallets {
		if strings.EqualFold(wallet.WalletAddress, walletAddress) {
			updatedWallet, err := tx.UpdateCustomerWalletUsageTime(ctx, wallet.ID)
			if err != nil {
				logger.Warn("Failed to update wallet usage time",
					zap.Error(err),
					zap.String("wallet_id", wallet.ID.String()))
				return wallet, nil
			}
			return updatedWallet, nil
		}
	}

	walletMetadata := map[string]interface{}{
		"source":     "product_subscription",
		"product_id": productID,
		"created_at": time.Now().Format(time.RFC3339),
	}
	walletMetadataBytes, err := json.Marshal(walletMetadata)
	if err != nil {
		return db.CustomerWallet{}, err
	}

	createWalletParams := db.CreateCustomerWalletParams{
		CustomerID:    customer.ID,
		WalletAddress: walletAddress,
		NetworkType:   db.NetworkType(networkType),
		Nickname: pgtype.Text{
			String: "Subscription Wallet",
			Valid:  true,
		},
		IsPrimary: pgtype.Bool{
			Bool:  len(wallets) == 0,
			Valid: true,
		},
		Verified: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
		Metadata: walletMetadataBytes,
	}

	return tx.CreateCustomerWallet(ctx, createWalletParams)
}

// storeDelegationData creates a record of the delegation information
func (h *ProductHandler) storeDelegationData(
	ctx context.Context,
	tx *db.Queries,
	delegation business.DelegationStruct,
) (db.DelegationDatum, error) {
	delegationParams := db.CreateDelegationDataParams{
		Delegate:  delegation.Delegate,
		Delegator: delegation.Delegator,
		Authority: delegation.Authority,
		Caveats:   helpers.MarshalCaveats(delegation.Caveats),
		Salt:      delegation.Salt,
		Signature: delegation.Signature,
	}

	return tx.CreateDelegationData(ctx, delegationParams)
}

// createSubscription creates the subscription record in the database
func (h *ProductHandler) createSubscription(
	ctx context.Context,
	tx *db.Queries,
	params params.CreateSubscriptionParams,
) (db.Subscription, error) {
	metadata := map[string]interface{}{
		"created_at":     time.Now().Format(time.RFC3339),
		"wallet_address": params.CustomerWallet.WalletAddress,
		"network_type":   params.CustomerWallet.NetworkType,
		"price_id":       params.Price.ID.String(),
	}
	subscriptionMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return db.Subscription{}, err
	}

	subscriptionParams := db.CreateSubscriptionParams{
		CustomerID:     params.Customer.ID,
		ProductID:      params.ProductID,
		WorkspaceID:    params.WorkspaceID,
		PriceID:        params.Price.ID,
		ProductTokenID: params.ProductTokenID,
		TokenAmount:    int32(params.TokenAmount),
		DelegationID:   params.DelegationData.ID,
		CustomerWalletID: pgtype.UUID{
			Bytes: params.CustomerWallet.ID,
			Valid: true,
		},
		Status: db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{
			Time:  params.PeriodStart,
			Valid: true,
		},
		CurrentPeriodEnd: pgtype.Timestamptz{
			Time:  params.PeriodEnd,
			Valid: true,
		},
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  params.NextRedemption,
			Valid: true,
		},
		TotalRedemptions:   0,
		TotalAmountInCents: 0,
		Metadata:           subscriptionMetadataBytes,
	}

	logger.Info("Creating new subscription",
		zap.String("product_id", params.ProductID.String()),
		zap.String("customer_id", params.Customer.ID.String()),
		zap.String("workspace_id", params.WorkspaceID.String()),
		zap.String("price_id", params.Price.ID.String()),
		zap.String("product_token_id", params.ProductTokenID.String()),
		zap.String("delegation_id", params.DelegationData.ID.String()),
		zap.String("customer_id", params.Customer.ID.String()),
		zap.String("customer_wallet_id", params.CustomerWallet.ID.String()))

	return tx.CreateSubscription(ctx, subscriptionParams)
}

// performInitialRedemption executes the initial token redemption for a new subscription
func (h *ProductHandler) performInitialRedemption(
	ctx context.Context,
	customer db.Customer,
	customerWallet db.CustomerWallet,
	subscription db.Subscription,
	product db.Product,
	price db.Price,
	productToken db.GetProductTokenRow,
	delegation business.DelegationStruct,
	executionObject dsClient.ExecutionObject,
) (db.Subscription, error) {
	logger.Info("Performing initial redemption",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("customer_id", customer.ID.String()),
		zap.String("price_id", price.ID.String()))

	rawCaveats := helpers.MarshalCaveats(delegation.Caveats)
	delegationData := dsClient.DelegationData{
		Delegate:  delegation.Delegate,
		Delegator: delegation.Delegator,
		Authority: delegation.Authority,
		Caveats:   rawCaveats,
		Salt:      delegation.Salt,
		Signature: delegation.Signature,
	}

	delegationBytes, err := json.Marshal(delegationData)
	if err != nil {
		return subscription, fmt.Errorf("failed to marshal delegation data: %w", err)
	}

	txHash, err := h.delegationClient.RedeemDelegation(ctx, delegationBytes, executionObject)
	if err != nil {
		return subscription, fmt.Errorf("delegation redemption failed: %w", err)
	}

	metadata := map[string]interface{}{
		"product_id":        product.ID.String(),
		"product_name":      product.Name,
		"price_id":          price.ID.String(),
		"price_type":        price.Type,
		"product_token_id":  productToken.ID.String(),
		"token_symbol":      productToken.TokenSymbol,
		"network_name":      productToken.NetworkName,
		"wallet_address":    customerWallet.WalletAddress,
		"customer_id":       customer.ID.String(),
		"customer_name":     customer.Name.String,
		"customer_email":    customer.Email.String,
		"redemption_time":   time.Now().Unix(),
		"subscription_type": string(price.Type),
		"tx_hash":           txHash,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.Error("Failed to marshal metadata", zap.Error(err))
		metadataBytes = []byte("{}")
	}

	successEventParams := db.CreateRedemptionEventParams{
		SubscriptionID: subscription.ID,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		AmountInCents: int32(price.UnitAmountInPennies),
		Metadata:      metadataBytes,
	}

	_, eventErr := h.common.db.CreateRedemptionEvent(ctx, successEventParams)
	if eventErr != nil {
		logger.Error("Failed to record successful redemption event",
			zap.Error(eventErr),
			zap.String("subscription_id", subscription.ID.String()))
	}

	var nextRedemptionDate pgtype.Timestamptz
	if price.Type == db.PriceTypeRecurring {
		nextDate := helpers.CalculateNextRedemption(string(price.IntervalType), time.Now())
		nextRedemptionDate = pgtype.Timestamptz{
			Time:  nextDate,
			Valid: true,
		}
	} else {
		nextRedemptionDate = pgtype.Timestamptz{
			Valid: false,
		}
	}

	updateParams := db.IncrementSubscriptionRedemptionParams{
		ID:                 subscription.ID,
		TotalAmountInCents: int32(price.UnitAmountInPennies),
		NextRedemptionDate: nextRedemptionDate,
	}

	updatedSubscription, err := h.common.db.IncrementSubscriptionRedemption(ctx, updateParams)
	if err != nil {
		logger.Error("Failed to update subscription redemption details",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))
		return subscription, err
	}

	_, walletErr := h.common.db.UpdateCustomerWalletUsageTime(ctx, customerWallet.ID)
	if walletErr != nil {
		logger.Warn("Failed to update wallet last used timestamp",
			zap.Error(walletErr),
			zap.String("wallet_id", customerWallet.ID.String()))
	}

	logger.Info("Initial redemption successful",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("transaction_hash", txHash))

	return updatedSubscription, nil
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

	responseList := make([]responses.ProductResponse, 0, len(result.Products))
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
		productTokenListResponse := make([]responses.ProductTokenResponse, 0, len(productTokenList))
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

// createProductTokens creates the associated product tokens for a product
func (h *ProductHandler) createProductTokens(c *gin.Context, productID uuid.UUID, tokens []requests.CreateProductTokenRequest) error {
	for _, pt := range tokens {
		networkID, err := uuid.Parse(pt.NetworkID)
		if err != nil {
			return fmt.Errorf("invalid network ID format: %w", err)
		}

		tokenID, err := uuid.Parse(pt.TokenID)
		if err != nil {
			return fmt.Errorf("invalid token ID format: %w", err)
		}

		_, err = h.common.db.CreateProductToken(c.Request.Context(), db.CreateProductTokenParams{
			ProductID: productID,
			NetworkID: networkID,
			TokenID:   tokenID,
			Active:    pt.Active,
		})
		if err != nil {
			return fmt.Errorf("failed to create product token: %w", err)
		}
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
	servicePrices := make([]params.CreatePriceParams, 0, len(req.Prices))
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
	serviceTokens := make([]params.CreateProductTokenParams, 0, len(req.ProductTokens))
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
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		logger.Error("Error unmarshaling price metadata", zap.Error(err))
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
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		logger.Error("Error unmarshaling product metadata", zap.Error(err))
	}

	apiPrices := make([]responses.PriceResponse, 0, len(dbPrices))
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

// toComprehensiveSubscriptionResponse converts a db.Subscription to a comprehensive SubscriptionResponse
// that includes all subscription fields plus the initial transaction hash from subscription events
func (h *ProductHandler) toComprehensiveSubscriptionResponse(ctx context.Context, subscription db.Subscription) (*responses.SubscriptionResponse, error) {
	// Get the subscription details with related data
	subscriptionDetails, err := h.common.db.GetSubscriptionWithDetails(ctx, db.GetSubscriptionWithDetailsParams{
		ID:          subscription.ID,
		WorkspaceID: subscription.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription details: %w", err)
	}

	// Get the initial transaction hash from the first redemption event
	initialTxHash := ""
	events, err := h.common.db.ListSubscriptionEventsBySubscription(ctx, subscription.ID)
	if err == nil {
		for _, event := range events {
			if event.EventType == db.SubscriptionEventTypeRedeemed && event.TransactionHash.Valid {
				initialTxHash = event.TransactionHash.String
				break
			}
		}
	}

	// Parse metadata
	var metadata map[string]interface{}
	if len(subscription.Metadata) > 0 {
		if err := json.Unmarshal(subscription.Metadata, &metadata); err != nil {
			logger.Error("Error unmarshaling subscription metadata", zap.Error(err))
			metadata = make(map[string]interface{})
		}
	}

	// Convert to response
	response := &responses.SubscriptionResponse{
		ID:                     subscription.ID,
		WorkspaceID:            subscription.WorkspaceID,
		CustomerID:             subscription.CustomerID,
		Status:                 string(subscription.Status),
		CurrentPeriodStart:     subscription.CurrentPeriodStart.Time,
		CurrentPeriodEnd:       subscription.CurrentPeriodEnd.Time,
		TotalRedemptions:       subscription.TotalRedemptions,
		TotalAmountInCents:     subscription.TotalAmountInCents,
		TokenAmount:            subscription.TokenAmount,
		DelegationID:           subscription.DelegationID,
		InitialTransactionHash: initialTxHash,
		Metadata:               metadata,
		CreatedAt:              subscription.CreatedAt.Time,
		UpdatedAt:              subscription.UpdatedAt.Time,
		CustomerName:           subscriptionDetails.CustomerName.String,
		CustomerEmail:          subscriptionDetails.CustomerEmail.String,
		Price: responses.PriceResponse{
			ID:                  subscriptionDetails.PriceID.String(),
			Object:              "price",
			ProductID:           subscriptionDetails.ProductID.String(),
			Active:              true, // Assuming active for subscriptions
			Type:                string(subscriptionDetails.PriceType),
			Currency:            string(subscriptionDetails.PriceCurrency),
			UnitAmountInPennies: int64(subscriptionDetails.PriceUnitAmountInPennies), // Convert int32 to int64
			IntervalType:        string(subscriptionDetails.PriceIntervalType),
			TermLength:          subscriptionDetails.PriceTermLength,
			CreatedAt:           time.Now().Unix(), // Default timestamp
			UpdatedAt:           time.Now().Unix(), // Default timestamp
		},
		Product: responses.ProductResponse{
			ID:     subscriptionDetails.ProductID.String(),
			Name:   subscriptionDetails.ProductName,
			Active: true, // Assuming active for subscriptions
			Object: "product",
		},
		ProductToken: responses.ProductTokenResponse{
			ID:          subscriptionDetails.ProductTokenID.String(),
			TokenSymbol: subscriptionDetails.TokenSymbol,
			NetworkID:   subscriptionDetails.ProductTokenID.String(),
			CreatedAt:   time.Now().Unix(), // Default timestamp
			UpdatedAt:   time.Now().Unix(), // Default timestamp
		},
	}

	// Handle nullable fields
	if subscription.NextRedemptionDate.Valid {
		response.NextRedemptionDate = &subscription.NextRedemptionDate.Time
	}

	if subscription.CustomerWalletID.Valid {
		walletID := uuid.UUID(subscription.CustomerWalletID.Bytes)
		response.CustomerWalletID = &walletID
	}

	if subscription.ExternalID.Valid {
		response.ExternalID = subscription.ExternalID.String
	}

	if subscription.PaymentSyncStatus.Valid {
		response.PaymentSyncStatus = subscription.PaymentSyncStatus.String
	}

	if subscription.PaymentSyncedAt.Valid {
		response.PaymentSyncedAt = &subscription.PaymentSyncedAt.Time
	}

	if subscription.PaymentSyncVersion.Valid {
		response.PaymentSyncVersion = subscription.PaymentSyncVersion.Int32
	}

	if subscription.PaymentProvider.Valid {
		response.PaymentProvider = subscription.PaymentProvider.String
	}

	return response, nil
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

	price, err := h.common.db.GetPrice(ctx, parsedPriceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Price not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get price", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	if !price.Active {
		h.common.HandleError(c, nil, "Cannot subscribe to inactive price", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	product, err := h.common.db.GetProductWithoutWorkspaceId(ctx, price.ProductID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Product associated with the price not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get product", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	if err := h.validateSubscriptionRequest(request, product.ID); err != nil {
		h.common.HandleError(c, err, err.Error(), http.StatusBadRequest, h.common.GetLogger())
		return
	}

	if !product.Active {
		h.common.HandleError(c, nil, "Cannot subscribe to a price of an inactive product", http.StatusBadRequest, h.common.GetLogger())
		return
	}

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
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Merchant wallet not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get merchant wallet", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	productToken, err := h.common.db.GetProductToken(ctx, parsedProductTokenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Product token not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get product token", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	token, err := h.common.db.GetToken(ctx, productToken.TokenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Token not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get token", http.StatusInternalServerError, h.common.GetLogger())
		}
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

	executionObject := dsClient.ExecutionObject{
		MerchantAddress:      merchantWallet.WalletAddress,
		TokenContractAddress: token.ContractAddress,
		TokenDecimals:        token.Decimals,
		TokenAmount:          tokenAmount,
		ChainID:              uint32(network.ChainID),
		NetworkName:          network.Name,
	}

	if productToken.ProductID != product.ID {
		h.common.HandleError(c, nil, "Product token does not belong to the specified product", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	normalizedAddress := helpers.NormalizeWalletAddress(request.SubscriberAddress, helpers.DetermineNetworkType(productToken.NetworkType))

	// Get database pool
	pool, err := h.common.GetDBPool()
	if err != nil {
		h.common.HandleError(c, err, "Failed to get database pool", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	var subscription db.Subscription
	var updatedSubscription db.Subscription
	var customer db.Customer
	var customerWallet db.CustomerWallet

	// Execute within transaction
	err = helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		qtx := h.common.WithTx(tx)

		var err error
		customer, customerWallet, err = h.processCustomerAndWallet(ctx, qtx, normalizedAddress, product, productToken)
		if err != nil {
			return fmt.Errorf("failed to process customer or wallet: %w", err)
		}

		subscriptions, err := h.common.db.ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
			CustomerID:  customer.ID,
			WorkspaceID: product.WorkspaceID,
		})
		if err != nil {
			return fmt.Errorf("failed to check for existing subscription: %w", err)
		}

		var existingSubscription *db.Subscription
		for i, sub := range subscriptions {
			if sub.PriceID == price.ID && sub.Status == db.SubscriptionStatusActive {
				existingSubscription = &subscriptions[i]
				break
			}
		}

		if existingSubscription != nil {
			logger.Info("Subscription already exists for this price",
				zap.String("subscription_id", existingSubscription.ID.String()),
				zap.String("customer_id", customer.ID.String()),
				zap.String("price_id", price.ID.String()))

			duplicateErr := fmt.Errorf("subscription already exists for customer %s and price %s", customer.ID, price.ID)
			h.logFailedSubscriptionCreation(ctx, &customer.ID, product, price, productToken, normalizedAddress, request.Delegation.Signature, duplicateErr)

			// Return a custom error that we'll handle after the transaction
			return &SubscriptionExistsError{Subscription: existingSubscription}
		}

		delegationData, err := h.storeDelegationData(ctx, qtx, request.Delegation)
		if err != nil {
			return fmt.Errorf("failed to store delegation data: %w", err)
		}

		periodStart, periodEnd, nextRedemption := helpers.CalculateSubscriptionPeriods(price)

		subscription, err = h.createSubscription(ctx, qtx, params.CreateSubscriptionParams{
			Customer:       customer,
			CustomerWallet: customerWallet,
			WorkspaceID:    product.WorkspaceID,
			ProductID:      product.ID,
			Price:          price,
			ProductTokenID: parsedProductTokenID,
			TokenAmount:    tokenAmount,
			DelegationData: delegationData,
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
			NextRedemption: nextRedemption,
		})
		if err != nil {
			h.logFailedSubscriptionCreation(ctx, &customer.ID, product, price, productToken, normalizedAddress, request.Delegation.Signature, err)
			return fmt.Errorf("failed to create subscription: %w", err)
		}

		eventMetadata, err := json.Marshal(map[string]interface{}{
			"product_name":   product.Name,
			"price_type":     price.Type,
			"wallet_address": customerWallet.WalletAddress,
			"network_type":   customerWallet.NetworkType,
		})
		if err != nil {
			logger.Error("Failed to marshal event metadata", zap.Error(err))
			eventMetadata = []byte("{}")
		}

		_, err = qtx.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
			SubscriptionID: subscription.ID,
			EventType:      db.SubscriptionEventTypeCreated,
			OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			AmountInCents:  price.UnitAmountInPennies,
			Metadata:       eventMetadata,
		})
		if err != nil {
			return fmt.Errorf("failed to create subscription creation event: %w", err)
		}

		logger.Info("Created subscription event",
			zap.String("subscription_id", subscription.ID.String()),
			zap.String("event_type", string(db.SubscriptionEventTypeCreated)))

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

	updatedSubscription, err = h.performInitialRedemption(ctx, customer, customerWallet, subscription, product, price, productToken, request.Delegation, executionObject)
	if err != nil {
		logger.Error("Initial redemption failed, subscription marked as failed and soft-deleted",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()),
			zap.String("customer_id", customer.ID.String()),
			zap.String("price_id", price.ID.String()))

		_, updateErr := h.common.db.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     subscription.ID,
			Status: db.SubscriptionStatusFailed,
		})

		if updateErr != nil {
			logger.Error("Failed to update subscription status after redemption failure",
				zap.Error(updateErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		deleteErr := h.common.db.DeleteSubscription(ctx, subscription.ID)
		if deleteErr != nil {
			logger.Error("Failed to soft-delete subscription after redemption failure",
				zap.Error(deleteErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		errorMsg := fmt.Sprintf("Initial redemption failed: %v", err)
		_, eventErr := h.common.db.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
			SubscriptionID:  subscription.ID,
			EventType:       db.SubscriptionEventTypeFailedRedemption,
			TransactionHash: pgtype.Text{String: "", Valid: false},
			AmountInCents:   price.UnitAmountInPennies,
			ErrorMessage:    pgtype.Text{String: errorMsg, Valid: true},
			OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			Metadata:        json.RawMessage(`{}`),
		})

		if eventErr != nil {
			logger.Error("Failed to create subscription event after redemption failure",
				zap.Error(eventErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		h.common.HandleError(c, err, "Initial redemption failed, subscription marked as failed and soft-deleted", http.StatusInternalServerError, h.common.GetLogger())
		return
	}
	// Create comprehensive response with all subscription fields and initial transaction hash
	comprehensiveResponse, err := h.toComprehensiveSubscriptionResponse(ctx, updatedSubscription)
	if err != nil {
		logger.Error("Failed to create comprehensive subscription response",
			zap.Error(err),
			zap.String("subscription_id", updatedSubscription.ID.String()))
		h.common.HandleError(c, err, "Failed to create subscription response", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusCreated, comprehensiveResponse)
}
