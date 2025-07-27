package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// SwaggerMetadata is used to represent JSON metadata in Swagger docs
type SwaggerMetadata map[string]interface{}

// ProductHandler handles product-related operations
type ProductHandler struct {
	common           *CommonServices
	delegationClient *dsClient.DelegationClient
	productService   interfaces.ProductService
	logger           *zap.Logger
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
		logger:           logger,
	}
}

// PriceResponse represents a price object in API responses
type PriceResponse struct {
	ID                  string          `json:"id"`
	Object              string          `json:"object"`
	ProductID           string          `json:"product_id"`
	Active              bool            `json:"active"`
	Type                string          `json:"type"`
	Nickname            string          `json:"nickname,omitempty"`
	Currency            string          `json:"currency"`
	UnitAmountInPennies int32           `json:"unit_amount_in_pennies"`
	IntervalType        string          `json:"interval_type,omitempty"`
	IntervalCount       int32           `json:"interval_count,omitempty"`
	TermLength          int32           `json:"term_length,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt           int64           `json:"created_at"`
	UpdatedAt           int64           `json:"updated_at"`
}

// CreatePriceRequest represents the request body for creating a new price
type CreatePriceRequest struct {
	Active              bool            `json:"active"`
	Type                string          `json:"type" binding:"required"`
	Nickname            string          `json:"nickname"`
	Currency            string          `json:"currency" binding:"required"`
	UnitAmountInPennies int32           `json:"unit_amount_in_pennies" binding:"required"`
	IntervalType        string          `json:"interval_type"`
	IntervalCount       int32           `json:"interval_count"`
	TermLength          int32           `json:"term_length"`
	Metadata            json.RawMessage `json:"metadata" swaggertype:"object"`
}

// ProductResponse represents the standardized API response for product operations
type ProductResponse struct {
	ID            string                         `json:"id"`
	Object        string                         `json:"object"`
	WorkspaceID   string                         `json:"workspace_id"`
	WalletID      string                         `json:"wallet_id"`
	Name          string                         `json:"name"`
	Description   string                         `json:"description,omitempty"`
	ImageURL      string                         `json:"image_url,omitempty"`
	URL           string                         `json:"url,omitempty"`
	Active        bool                           `json:"active"`
	Metadata      json.RawMessage                `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt     int64                          `json:"created_at"`
	UpdatedAt     int64                          `json:"updated_at"`
	Prices        []PriceResponse                `json:"prices,omitempty"`
	ProductTokens []helpers.ProductTokenResponse `json:"product_tokens,omitempty"`
}

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name          string                              `json:"name" binding:"required"`
	WalletID      string                              `json:"wallet_id" binding:"required"`
	Description   string                              `json:"description"`
	ImageURL      string                              `json:"image_url"`
	URL           string                              `json:"url"`
	Active        bool                                `json:"active"`
	Metadata      json.RawMessage                     `json:"metadata" swaggertype:"object"`
	Prices        []CreatePriceRequest                `json:"prices" binding:"required,dive"`
	ProductTokens []helpers.CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name          string                              `json:"name,omitempty"`
	WalletID      string                              `json:"wallet_id,omitempty"`
	Description   string                              `json:"description,omitempty"`
	ImageURL      string                              `json:"image_url,omitempty"`
	URL           string                              `json:"url,omitempty"`
	Active        *bool                               `json:"active,omitempty"`
	Metadata      json.RawMessage                     `json:"metadata,omitempty" swaggertype:"object"`
	ProductTokens []helpers.CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

type PublicProductResponse struct {
	ID            string                       `json:"id"`
	AccountID     string                       `json:"account_id"`
	WorkspaceID   string                       `json:"workspace_id"`
	WalletAddress string                       `json:"wallet_address"`
	Name          string                       `json:"name"`
	Description   string                       `json:"description"`
	ImageURL      string                       `json:"image_url,omitempty"`
	URL           string                       `json:"url,omitempty"`
	ProductTokens []PublicProductTokenResponse `json:"product_tokens,omitempty"`
	Price         PriceResponse                `json:"price"`
}

type PublicProductTokenResponse struct {
	ProductTokenID string `json:"product_token_id"`
	NetworkID      string `json:"network_id"`
	NetworkName    string `json:"network_name"`
	NetworkChainID string `json:"network_chain_id"`
	TokenID        string `json:"token_id"`
	TokenAddress   string `json:"token_address"`
	TokenName      string `json:"token_name"`
	TokenSymbol    string `json:"token_symbol"`
	TokenImageURL  string `json:"token_image_url"`
	TokenDecimals  int32  `json:"token_decimals"`
}

// ListProductsResponse represents the paginated response for product list operations
type ListProductsResponse struct {
	Object  string            `json:"object"`
	Data    []ProductResponse `json:"data"`
	HasMore bool              `json:"has_more"`
	Total   int64             `json:"total"`
}

// CaveatStruct represents a single caveat in a delegation
type CaveatStruct struct {
	// TODO: add caveat fields
	// Define the fields for CaveatStruct based on your needs
}

// DelegationStruct represents the delegation data structure
type DelegationStruct struct {
	Delegate  string         `json:"delegate"`
	Delegator string         `json:"delegator"`
	Authority string         `json:"authority"`
	Caveats   []CaveatStruct `json:"caveats"`
	Salt      string         `json:"salt"`
	Signature string         `json:"signature"`
}

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
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Parse product ID
	productId := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	// Get product using service
	product, prices, err := h.productService.GetProduct(c.Request.Context(), services.GetProductParams{
		ProductID:   parsedProductID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(c, http.StatusNotFound, "Product not found", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve product", err)
		}
		return
	}

	// Convert to response format
	response := helpers.ToProductDetailResponse(*product, prices)
	sendSuccess(c, http.StatusOK, response)
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
		sendError(c, http.StatusBadRequest, "Invalid price ID format", err)
		return
	}

	// Get public product using service
	response, err := h.productService.GetPublicProductByPriceID(c.Request.Context(), parsedPriceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not active") {
			sendError(c, http.StatusNotFound, err.Error(), err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve product", err)
		}
		return
	}

	sendSuccess(c, http.StatusOK, *response)
}

// validateSubscriptionRequest validates the basic request parameters
func (h *ProductHandler) validateSubscriptionRequest(request SubscribeRequest, productID uuid.UUID) error {
	if request.SubscriberAddress == "" {
		return fmt.Errorf("subscriber address is required")
	}

	if _, err := uuid.Parse(request.PriceID); err != nil {
		return fmt.Errorf("invalid price ID format")
	}

	if _, err := uuid.Parse(request.ProductTokenID); err != nil {
		return fmt.Errorf("invalid product token ID format")
	}

	tokenAmount, err := strconv.ParseInt(request.TokenAmount, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token amount format")
	}

	if tokenAmount <= 0 {
		return fmt.Errorf("token amount must be greater than 0")
	}

	if request.Delegation.Delegate != h.common.GetCypheraSmartWalletAddress() {
		return fmt.Errorf("delegate address does not match cyphera smart wallet address, %s != %s", request.Delegation.Delegate, h.common.GetCypheraSmartWalletAddress())
	}

	if request.Delegation.Delegate == "" || request.Delegation.Delegator == "" ||
		request.Delegation.Authority == "" || request.Delegation.Salt == "" ||
		request.Delegation.Signature == "" {
		return fmt.Errorf("incomplete delegation data")
	}

	return nil
}

// normalizeWalletAddress ensures consistent wallet address format based on network type
func normalizeWalletAddress(address, networkType string) string {
	if networkType == string(db.NetworkTypeEvm) {
		return strings.ToLower(address)
	}
	return address
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

	networkType := determineNetworkType(productToken.NetworkType)

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
	delegation DelegationStruct,
) (db.DelegationDatum, error) {
	delegationParams := db.CreateDelegationDataParams{
		Delegate:  delegation.Delegate,
		Delegator: delegation.Delegator,
		Authority: delegation.Authority,
		Caveats:   marshalCaveats(delegation.Caveats),
		Salt:      delegation.Salt,
		Signature: delegation.Signature,
	}

	return tx.CreateDelegationData(ctx, delegationParams)
}

// calculateSubscriptionPeriods determines the start, end, and next redemption dates based on a price
func (h *ProductHandler) calculateSubscriptionPeriods(price db.Price) (time.Time, time.Time, time.Time) {
	now := time.Now()
	periodStart := now
	var periodEnd time.Time
	var nextRedemption time.Time

	termLength := 1
	if price.TermLength != 0 {
		termLength = int(price.TermLength)
	}

	if price.Type == db.PriceTypeRecurring {
		periodEnd = CalculatePeriodEnd(now, string(price.IntervalType), int32(termLength))
		nextRedemption = CalculateNextRedemption(string(price.IntervalType), now)
	} else {
		periodEnd = now
		nextRedemption = now
	}

	return periodStart, periodEnd, nextRedemption
}

// createSubscription creates the subscription record in the database
func (h *ProductHandler) createSubscription(
	ctx context.Context,
	tx *db.Queries,
	params CreateSubscriptionParams,
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
	delegation DelegationStruct,
	executionObject dsClient.ExecutionObject,
) (db.Subscription, error) {
	logger.Info("Performing initial redemption",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("customer_id", customer.ID.String()),
		zap.String("price_id", price.ID.String()))

	rawCaveats := marshalCaveats(delegation.Caveats)
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

	metadataBytes, _ := json.Marshal(metadata)

	successEventParams := db.CreateRedemptionEventParams{
		SubscriptionID: subscription.ID,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		AmountInCents: price.UnitAmountInPennies,
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
		nextDate := CalculateNextRedemption(string(price.IntervalType), time.Now())
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
		TotalAmountInCents: price.UnitAmountInPennies,
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

func toPublicProductResponse(workspace db.Workspace, product db.Product, price db.Price, productTokens []db.GetActiveProductTokensByProductRow, wallet db.Wallet) PublicProductResponse {
	publicProductTokens := make([]PublicProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		publicProductTokens[i] = PublicProductTokenResponse{
			ProductTokenID: pt.ID.String(),
			NetworkID:      pt.NetworkID.String(),
			NetworkName:    pt.NetworkName,
			NetworkChainID: strconv.Itoa(int(pt.ChainID)),
			TokenID:        pt.TokenID.String(),
			TokenName:      pt.TokenName,
			TokenSymbol:    pt.TokenSymbol,
			TokenDecimals:  int32(pt.Decimals),
		}
	}

	return PublicProductResponse{
		ID:            product.ID.String(),
		AccountID:     workspace.AccountID.String(),
		WorkspaceID:   workspace.ID.String(),
		WalletAddress: wallet.WalletAddress,
		Name:          product.Name,
		Description:   product.Description.String,
		ImageURL:      product.ImageUrl.String,
		URL:           product.Url.String,
		ProductTokens: publicProductTokens,
		Price:         toPriceResponse(price),
	}
}

// validatePaginationParams is defined in common.go

// validateProductUpdate validates core product update parameters
func (h *ProductHandler) validateProductUpdate(c *gin.Context, req UpdateProductRequest, existingProduct db.Product) error {
	if req.Name != "" {
		if len(req.Name) > 255 {
			return fmt.Errorf("name must be less than 255 characters")
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
			return fmt.Errorf("description must be less than 1000 characters")
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
			return fmt.Errorf("invalid metadata JSON format")
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
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	limit, page, err := validatePaginationParams(c)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Use service to list products
	result, err := h.productService.ListProducts(c.Request.Context(), services.ListProductsParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve products", err)
		return
	}

	responseList := make([]ProductResponse, len(result.Products))
	for i, product := range result.Products {
		dbPrices, err := h.common.db.ListPricesByProduct(c.Request.Context(), product.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve prices for product %s", product.ID), err)
			return
		}
		productResponse := toProductResponse(product, dbPrices)

		productTokenList, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), product.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve product tokens", err)
			return
		}
		productTokenListResponse := make([]helpers.ProductTokenResponse, len(productTokenList))
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

	sendSuccess(c, http.StatusOK, listProductsResponse)
}

// validatePriceTermLength validates the term length for recurring prices
func validatePriceTermLength(priceType db.PriceType, termLength int32, intervalType db.IntervalType, intervalCount int32) error {
	if priceType == db.PriceTypeRecurring {
		if intervalType == "" || intervalCount <= 0 {
			return fmt.Errorf("interval_type and interval_count are required for recurring prices")
		}
		if termLength <= 0 {
			return fmt.Errorf("term length must be greater than 0 for recurring prices")
		}
	} else if priceType == db.PriceTypeOneOff {
		if intervalType != "" || intervalCount != 0 || termLength != 0 {
			return fmt.Errorf("interval_type, interval_count, and term_length must not be set for one_off prices")
		}
	}
	return nil
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
		return fmt.Errorf("wallet does not belong to workspace")
	}

	return nil
}

// createProductTokens creates the associated product tokens for a product
func (h *ProductHandler) createProductTokens(c *gin.Context, productID uuid.UUID, tokens []helpers.CreateProductTokenRequest) error {
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

// validatePriceInPennies validates that the price value is non-negative
func validatePriceInPennies(price int32) error {
	if price < 0 {
		return fmt.Errorf("unit_amount_in_pennies cannot be negative")
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
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	parsedWalletID, err := helpers.ValidateWalletID(req.WalletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Convert request prices to service params
	servicePrices := make([]services.CreatePriceParams, len(req.Prices))
	for i, price := range req.Prices {
		servicePrices[i] = services.CreatePriceParams{
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

	// Use service to create product
	product, prices, err := h.productService.CreateProduct(c.Request.Context(), services.CreateProductParams{
		WorkspaceID:   parsedWorkspaceID,
		WalletID:      parsedWalletID,
		Name:          req.Name,
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		URL:           req.URL,
		Active:        req.Active,
		Metadata:      req.Metadata,
		Prices:        servicePrices,
		ProductTokens: req.ProductTokens,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create product", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toProductResponse(*product, prices))
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
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert request to service params
	updateParams := services.UpdateProductParams{
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
			sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
			return
		}
		updateParams.WalletID = &parsedWalletID
	}

	// Use service to update product
	product, err := h.productService.UpdateProduct(c.Request.Context(), updateParams)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(c, http.StatusNotFound, "Product not found", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to update product", err)
		}
		return
	}

	sendSuccess(c, http.StatusOK, toProductResponse(*product, []db.Price{}))
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
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	// Use service to delete product
	err = h.productService.DeleteProduct(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(c, http.StatusNotFound, "Product not found", err)
		} else if strings.Contains(err.Error(), "active subscriptions") {
			sendError(c, http.StatusConflict, "Cannot delete product with active subscriptions", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to delete product", err)
		}
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// Helper function to convert db.Price to PriceResponse
func toPriceResponse(p db.Price) PriceResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling price metadata: %v", err)
	}

	return PriceResponse{
		ID:                  p.ID.String(),
		Object:              "price",
		ProductID:           p.ProductID.String(),
		Active:              p.Active,
		Type:                string(p.Type),
		Nickname:            p.Nickname.String,
		Currency:            string(p.Currency),
		UnitAmountInPennies: p.UnitAmountInPennies,
		IntervalType:        string(p.IntervalType),
		TermLength:          p.TermLength,
		Metadata:            p.Metadata,
		CreatedAt:           p.CreatedAt.Time.Unix(),
		UpdatedAt:           p.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert database model to API response
func toProductResponse(p db.Product, dbPrices []db.Price) ProductResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling product metadata: %v", err)
	}

	apiPrices := make([]PriceResponse, len(dbPrices))
	for i, dbPrice := range dbPrices {
		apiPrices[i] = toPriceResponse(dbPrice)
	}

	return ProductResponse{
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

// marshalCaveats converts the caveats array to JSON for storage
func marshalCaveats(caveats []CaveatStruct) json.RawMessage {
	bytes, err := json.Marshal(caveats)
	if err != nil {
		return json.RawMessage("{}")
	}
	return bytes
}

// determineNetworkType maps network names to their network types
func determineNetworkType(networkTypeStr string) string {
	networkType := strings.ToLower(networkTypeStr)
	switch networkType {
	case "ethereum", "sepolia", "goerli", "arbitrum", "optimism", "polygon", "base", "linea":
		return string(db.NetworkTypeEvm)
	case "solana":
		return string(db.NetworkTypeSolana)
	case "cosmos":
		return string(db.NetworkTypeCosmos)
	case "bitcoin":
		return string(db.NetworkTypeBitcoin)
	case "polkadot":
		return string(db.NetworkTypePolkadot)
	default:
		return string(db.NetworkTypeEvm)
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

	errorType := h.determineErrorType(err)
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

// determineErrorType determines the error type based on the error message
func (h *ProductHandler) determineErrorType(err error) db.SubscriptionEventType {
	errorMsg := err.Error()
	if strings.Contains(errorMsg, "validation") {
		return db.SubscriptionEventTypeFailedValidation
	} else if strings.Contains(errorMsg, "customer") && strings.Contains(errorMsg, "create") {
		return db.SubscriptionEventTypeFailedCustomerCreation
	} else if strings.Contains(errorMsg, "wallet") && strings.Contains(errorMsg, "create") {
		return db.SubscriptionEventTypeFailedWalletCreation
	} else if strings.Contains(errorMsg, "delegation") {
		return db.SubscriptionEventTypeFailedDelegationStorage
	} else if strings.Contains(errorMsg, "subscription already exists") {
		return db.SubscriptionEventTypeFailedDuplicate
	} else if strings.Contains(errorMsg, "database") || strings.Contains(errorMsg, "db") {
		return db.SubscriptionEventTypeFailedSubscriptionDb
	} else {
		return db.SubscriptionEventTypeFailed
	}
}

// toComprehensiveSubscriptionResponse converts a db.Subscription to a comprehensive SubscriptionResponse
// that includes all subscription fields plus the initial transaction hash from subscription events
func (h *ProductHandler) toComprehensiveSubscriptionResponse(ctx context.Context, subscription db.Subscription) (*SubscriptionResponse, error) {
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
			log.Printf("Error unmarshaling subscription metadata: %v", err)
			metadata = make(map[string]interface{})
		}
	}

	// Convert to response
	response := &SubscriptionResponse{
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
		Price: PriceResponse{
			ID:                  subscriptionDetails.PriceID.String(),
			Object:              "price",
			ProductID:           subscriptionDetails.ProductID.String(),
			Active:              true, // Assuming active for subscriptions
			Type:                string(subscriptionDetails.PriceType),
			Currency:            string(subscriptionDetails.PriceCurrency),
			UnitAmountInPennies: subscriptionDetails.PriceUnitAmountInPennies,
			IntervalType:        string(subscriptionDetails.PriceIntervalType),
			TermLength:          subscriptionDetails.PriceTermLength,
			CreatedAt:           time.Now().Unix(), // Default timestamp
			UpdatedAt:           time.Now().Unix(), // Default timestamp
		},
		Product: ProductResponse{
			ID:     subscriptionDetails.ProductID.String(),
			Name:   subscriptionDetails.ProductName,
			Active: true, // Assuming active for subscriptions
			Object: "product",
		},
		ProductToken: helpers.ProductTokenResponse{
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
