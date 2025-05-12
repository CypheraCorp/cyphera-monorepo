package handlers

import (
	"bytes"
	"context"
	dsClient "cyphera-api/internal/client/delegation_server"
	"cyphera-api/internal/constants"
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// SwaggerMetadata is used to represent JSON metadata in Swagger docs
type SwaggerMetadata map[string]interface{}

// ProductHandler handles product-related operations
type ProductHandler struct {
	common           *CommonServices
	delegationClient *dsClient.DelegationClient
}

// NewProductHandler creates a new ProductHandler instance
func NewProductHandler(common *CommonServices, delegationClient *dsClient.DelegationClient) *ProductHandler {
	return &ProductHandler{
		common:           common,
		delegationClient: delegationClient,
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
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	WorkspaceID   string                 `json:"workspace_id"`
	WalletID      string                 `json:"wallet_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	ImageURL      string                 `json:"image_url,omitempty"`
	URL           string                 `json:"url,omitempty"`
	Active        bool                   `json:"active"`
	Metadata      json.RawMessage        `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
	Prices        []PriceResponse        `json:"prices,omitempty"`
	ProductTokens []ProductTokenResponse `json:"product_tokens,omitempty"`
}

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name          string                      `json:"name" binding:"required"`
	WalletID      string                      `json:"wallet_id" binding:"required"`
	Description   string                      `json:"description"`
	ImageURL      string                      `json:"image_url"`
	URL           string                      `json:"url"`
	Active        bool                        `json:"active"`
	Metadata      json.RawMessage             `json:"metadata" swaggertype:"object"`
	Prices        []CreatePriceRequest        `json:"prices" binding:"required,dive"`
	ProductTokens []CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name          string                      `json:"name,omitempty"`
	WalletID      string                      `json:"wallet_id,omitempty"`
	Description   string                      `json:"description,omitempty"`
	ImageURL      string                      `json:"image_url,omitempty"`
	URL           string                      `json:"url,omitempty"`
	Active        *bool                       `json:"active,omitempty"`
	Metadata      json.RawMessage             `json:"metadata,omitempty" swaggertype:"object"`
	ProductTokens []CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

type PublicProductResponse struct {
	ID                      string                       `json:"id"`
	AccountID               string                       `json:"account_id"`
	WorkspaceID             string                       `json:"workspace_id"`
	WalletAddress           string                       `json:"wallet_address"`
	Name                    string                       `json:"name"`
	Description             string                       `json:"description"`
	ImageURL                string                       `json:"image_url,omitempty"`
	URL                     string                       `json:"url,omitempty"`
	ProductTokens           []PublicProductTokenResponse `json:"product_tokens,omitempty"`
	Price                   PriceResponse                `json:"price"`
	SmartAccountAddress     string                       `json:"smart_account_address,omitempty"`
	SmartAccountExplorerURL string                       `json:"smart_account_explorer_url,omitempty"`
	SmartAccountNetwork     string                       `json:"smart_account_network,omitempty"`
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

const (
	CurrencyUSD = "USD"
	CurrencyEUR = "EUR"
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

	product, err := h.common.db.GetProduct(c.Request.Context(), db.GetProductParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	dbPrices, err := h.common.db.ListPricesByProduct(c.Request.Context(), product.ID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve prices for product", err)
		return
	}

	sendSuccess(c, http.StatusOK, toProductResponse(product, dbPrices))
}

// GetPublicProductByID godoc
// @Summary Get public product and price details by Price ID
// @Description Get public product and specific price details by Price ID
// @Tags products
// @Accept json
// @Produce json
// @Param price_id path string true "Price ID"
// @Success 200 {object} PublicProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /public/prices/{price_id} [get]
// @exclude
func (h *ProductHandler) GetPublicProductByPriceID(c *gin.Context) {
	priceIDStr := c.Param("price_id")
	parsedPriceID, err := uuid.Parse(priceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid price ID format", err)
		return
	}

	price, err := h.common.db.GetPrice(c.Request.Context(), parsedPriceID)
	if err != nil {
		handleDBError(c, err, "Price not found")
		return
	}

	if !price.Active {
		sendError(c, http.StatusNotFound, "Price is not active", nil)
		return
	}

	product, err := h.common.db.GetProductWithoutWorkspaceId(c.Request.Context(), price.ProductID)
	if err != nil {
		handleDBError(c, err, "Product not found for the given price")
		return
	}

	if !product.Active {
		sendError(c, http.StatusNotFound, "Product associated with this price is not active", nil)
		return
	}

	wallet, err := h.common.db.GetWalletByID(c.Request.Context(), db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Wallet not found for the product")
		return
	}

	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), product.WorkspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found for the product")
		return
	}

	productTokens, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), product.ID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve product tokens", err)
		return
	}

	response := toPublicProductResponse(workspace, product, price, productTokens, wallet)

	for i, pt := range response.ProductTokens {
		token, err := h.common.db.GetToken(c.Request.Context(), uuid.MustParse(pt.TokenID))
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve token", err)
			return
		}
		response.ProductTokens[i].TokenAddress = token.ContractAddress
	}

	sendSuccess(c, http.StatusOK, response)
}

// SubscribeToProductByPriceID godoc
// @Summary Subscribe to a product's price
// @Description Creates a subscription for a product's specific price with the given delegation
// @Tags products
// @Accept json
// @Produce json
// @Param price_id path string true "Price ID to subscribe to"
// @Param subscription body SubscribeRequest true "Subscription details"
// @Success 201 {object} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /prices/{price_id}/subscribe [post]
// @exclude
func (h *ProductHandler) SubscribeToProductByPriceID(c *gin.Context) {
	ctx := c.Request.Context()
	priceIDStr := c.Param("price_id")
	parsedPriceID, err := uuid.Parse(priceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid price ID format", err)
		return
	}

	var request SubscribeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}
	request.PriceID = priceIDStr

	price, err := h.common.db.GetPrice(ctx, parsedPriceID)
	if err != nil {
		handleDBError(c, err, "Price not found")
		return
	}

	if !price.Active {
		sendError(c, http.StatusBadRequest, "Cannot subscribe to inactive price", nil)
		return
	}

	product, err := h.common.db.GetProductWithoutWorkspaceId(ctx, price.ProductID)
	if err != nil {
		handleDBError(c, err, "Product associated with the price not found")
		return
	}

	if err := h.validateSubscriptionRequest(request, product.ID); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	if !product.Active {
		sendError(c, http.StatusBadRequest, "Cannot subscribe to a price of an inactive product", nil)
		return
	}

	parsedProductTokenID, err := uuid.Parse(request.ProductTokenID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product token ID format", err)
		return
	}

	merchantWallet, err := h.common.db.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Merchant wallet not found")
		return
	}

	productToken, err := h.common.db.GetProductToken(ctx, parsedProductTokenID)
	if err != nil {
		handleDBError(c, err, "Product token not found")
		return
	}

	token, err := h.common.db.GetToken(ctx, productToken.TokenID)
	if err != nil {
		handleDBError(c, err, "Token not found")
		return
	}

	network, err := h.common.db.GetNetwork(ctx, token.NetworkID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get network details", err)
		return
	}

	tokenAmount, err := strconv.ParseInt(request.TokenAmount, 10, 64)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid token amount format", err)
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
		sendError(c, http.StatusBadRequest, "Product token does not belong to the specified product", nil)
		return
	}

	normalizedAddress := normalizeWalletAddress(request.SubscriberAddress, determineNetworkType(productToken.NetworkType))

	tx, qtx, err := h.common.BeginTx(ctx)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to start transaction", err)
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			logger.Error("Failed to rollback transaction", zap.Error(err))
		}
	}()

	customer, customerWallet, err := h.processCustomerAndWallet(ctx, qtx, normalizedAddress, product, productToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to process customer or wallet", err)
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
		CustomerID:  customer.ID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to check for existing subscription", err)
		return
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

		if tx != nil {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				logger.Error("Failed to commit transaction", zap.Error(commitErr))
			}
		}

		c.JSON(http.StatusConflict, gin.H{
			"message":      "Subscription already exists for this price",
			"subscription": existingSubscription,
		})
		return
	}

	delegationData, err := h.storeDelegationData(ctx, qtx, request.Delegation)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to store delegation data", err)
		return
	}

	periodStart, periodEnd, nextRedemption := h.calculateSubscriptionPeriods(price)

	subscription, err := h.createSubscription(ctx, qtx, CreateSubscriptionParams{
		Customer:       customer,
		CustomerWallet: customerWallet,
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
		sendError(c, http.StatusInternalServerError, "Failed to create subscription", err)
		return
	}

	eventMetadata, _ := json.Marshal(map[string]interface{}{
		"product_name":   product.Name,
		"price_type":     price.Type,
		"wallet_address": customerWallet.WalletAddress,
		"network_type":   customerWallet.NetworkType,
	})

	_, err = qtx.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
		SubscriptionID: subscription.ID,
		EventType:      db.SubscriptionEventTypeCreated,
		OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		AmountInCents:  price.UnitAmountInPennies,
		Metadata:       eventMetadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create subscription creation event", err)
		return
	}

	logger.Info("Created subscription event",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("event_type", string(db.SubscriptionEventTypeCreated)))

	if err := tx.Commit(ctx); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to commit transaction", err)
		return
	}

	updatedSubscription, err := h.performInitialRedemption(ctx, customer, customerWallet, subscription, product, price, productToken, request.Delegation, executionObject)
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

		sendError(c, http.StatusInternalServerError, "Initial redemption failed, subscription marked as failed and soft-deleted", err)
		return
	}

	sendSuccess(c, http.StatusCreated, updatedSubscription)
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
		WorkspaceID: product.WorkspaceID,
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
	if price.TermLength.Valid {
		termLength = int(price.TermLength.Int32)
	}

	if price.Type == db.PriceTypeRecurring {
		periodEnd = CalculatePeriodEnd(now, price.IntervalType.IntervalType, int32(termLength))
		nextRedemption = CalculateNextRedemption(price.IntervalType.IntervalType, now)
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
		PriceID:        params.Price.ID,
		ProductTokenID: params.ProductTokenID,
		TokenAmount: pgtype.Numeric{
			Int:   big.NewInt(int64(params.TokenAmount)),
			Valid: true,
		},
		DelegationID: params.DelegationData.ID,
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
		zap.String("customer_id", params.Customer.ID.String()),
		zap.String("product_id", params.ProductID.String()),
		zap.String("price_id", params.Price.ID.String()),
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
		nextDate := CalculateNextRedemption(price.IntervalType.IntervalType, time.Now())
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
		ID:                      product.ID.String(),
		AccountID:               workspace.AccountID.String(),
		WorkspaceID:             workspace.ID.String(),
		WalletAddress:           wallet.WalletAddress,
		Name:                    product.Name,
		Description:             product.Description.String,
		ImageURL:                product.ImageUrl.String,
		URL:                     product.Url.String,
		ProductTokens:           publicProductTokens,
		Price:                   toPriceResponse(price),
		SmartAccountAddress:     "",
		SmartAccountExplorerURL: "",
		SmartAccountNetwork:     "",
	}
}

// validatePriceType validates the price type and returns a db.PriceType if valid
func validatePriceType(priceTypeStr string) (db.PriceType, error) {
	if priceTypeStr == "" {
		return "", fmt.Errorf("price type is required")
	}
	if priceTypeStr != string(db.PriceTypeRecurring) && priceTypeStr != string(db.PriceTypeOneOff) {
		return "", fmt.Errorf("invalid price type. Must be '%s' or '%s'", db.PriceTypeRecurring, db.PriceTypeOneOff)
	}
	return db.PriceType(priceTypeStr), nil
}

// validateCurrency validates the currency and returns a db.Currency if valid
func validateCurrency(currency string) (db.Currency, error) {
	if currency == "" {
		return "", fmt.Errorf("currency is required")
	}

	validCurrencies := map[string]bool{
		CurrencyUSD: true,
		CurrencyEUR: true,
	}

	if !validCurrencies[currency] {
		return "", fmt.Errorf("invalid currency. Must be '%s' or '%s'", CurrencyUSD, CurrencyEUR)
	}

	return db.Currency(currency), nil
}

// validateIntervalType validates the interval type and returns a db.IntervalType if valid
func validateIntervalType(intervalType string) (db.IntervalType, error) {
	if intervalType == "" {
		return "", nil
	}

	validIntervalTypes := map[string]bool{
		constants.IntervalType1Minute:  true,
		constants.IntervalType5Minutes: true,
		constants.IntervalTypeDaily:    true,
		constants.IntervalTypeWeekly:   true,
		constants.IntervalTypeMonthly:  true,
		constants.IntervalTypeYearly:   true,
	}

	if !validIntervalTypes[intervalType] {
		return "", fmt.Errorf("invalid interval type")
	}

	return db.IntervalType(intervalType), nil
}

// validateWalletID validates and parses the wallet ID
func validateWalletID(walletID string) (uuid.UUID, error) {
	if walletID == "" {
		return uuid.Nil, fmt.Errorf("wallet ID is required")
	}
	parsed, err := uuid.Parse(walletID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid wallet ID format: %w", err)
	}
	return parsed, nil
}

// updateProductParams creates the update parameters for a product
func (h *ProductHandler) updateProductParams(id uuid.UUID, req UpdateProductRequest, existingProduct db.Product) (db.UpdateProductParams, error) {
	params := db.UpdateProductParams{
		ID:          id,
		Name:        existingProduct.Name,
		Description: existingProduct.Description,
		ImageUrl:    existingProduct.ImageUrl,
		Url:         existingProduct.Url,
		Active:      existingProduct.Active,
		Metadata:    existingProduct.Metadata,
		WalletID:    existingProduct.WalletID,
	}

	if req.Name != "" && req.Name != existingProduct.Name {
		params.Name = req.Name
	}
	if req.Description != "" && req.Description != existingProduct.Description.String {
		params.Description = pgtype.Text{String: req.Description, Valid: true}
	}
	if req.ImageURL != "" && req.ImageURL != existingProduct.ImageUrl.String {
		params.ImageUrl = pgtype.Text{String: req.ImageURL, Valid: true}
	}
	if req.URL != "" && req.URL != existingProduct.Url.String {
		params.Url = pgtype.Text{String: req.URL, Valid: true}
	}
	if req.Active != nil && *req.Active != existingProduct.Active {
		params.Active = *req.Active
	}
	if req.Metadata != nil && !bytes.Equal(req.Metadata, existingProduct.Metadata) {
		params.Metadata = req.Metadata
	}
	if req.WalletID != "" && req.WalletID != existingProduct.WalletID.String() {
		parsedWalletID, err := uuid.Parse(req.WalletID)
		if err != nil {
			return params, fmt.Errorf("invalid wallet ID format: %w", err)
		}
		params.WalletID = parsedWalletID
	}

	return params, nil
}

// validatePaginationParams validates and returns pagination parameters
func validatePaginationParams(c *gin.Context) (limit int32, page int32, err error) {
	const maxLimit int32 = 100
	limit = 10

	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit parameter")
		}
		if parsedLimit > int64(maxLimit) {
			limit = maxLimit
		} else if parsedLimit > 0 {
			limit = int32(parsedLimit)
		}
	}

	if pageStr := c.Query("page"); pageStr != "" {
		parsedPage, err := strconv.ParseInt(pageStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid page parameter")
		}
		if parsedPage > 0 {
			page = int32(parsedPage)
		}
	}

	return limit, page, nil
}

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

	total, err := h.common.db.CountProducts(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count products", err)
		return
	}

	products, err := h.common.db.ListProductsWithPagination(c.Request.Context(), db.ListProductsWithPaginationParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve products", err)
		return
	}

	responseList := make([]ProductResponse, len(products))
	for i, product := range products {
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
		productTokenListResponse := make([]ProductTokenResponse, len(productTokenList))
		for j, productToken := range productTokenList {
			productTokenListResponse[j] = toActiveProductTokenByProductResponse(productToken)
		}
		productResponse.ProductTokens = productTokenListResponse
		responseList[i] = productResponse
	}

	var hasMore bool
	if total > 0 {
		hasMore = (int64(offset) + int64(limit)) < total
	}

	listProductsResponse := ListProductsResponse{
		Object:  "list",
		Data:    responseList,
		HasMore: hasMore,
		Total:   total,
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
func (h *ProductHandler) createProductTokens(c *gin.Context, productID uuid.UUID, tokens []CreateProductTokenRequest) error {
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

	parsedWalletID, err := validateWalletID(req.WalletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := h.validateWallet(c, parsedWalletID, parsedWorkspaceID); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	product, err := h.common.db.CreateProduct(c.Request.Context(), db.CreateProductParams{
		WorkspaceID: parsedWorkspaceID,
		WalletID:    parsedWalletID,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ImageUrl:    pgtype.Text{String: req.ImageURL, Valid: req.ImageURL != ""},
		Url:         pgtype.Text{String: req.URL, Valid: req.URL != ""},
		Active:      req.Active,
		Metadata:    req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create product", err)
		return
	}

	prices := make([]db.Price, len(req.Prices))

	if len(req.Prices) > 0 {
		for i, price := range req.Prices {
			dbPrice, err := h.common.db.CreatePrice(c, db.CreatePriceParams{
				ProductID:           product.ID,
				Active:              price.Active,
				Type:                db.PriceType(price.Type),
				Nickname:            pgtype.Text{String: price.Nickname, Valid: true},
				Currency:            db.Currency(price.Currency),
				UnitAmountInPennies: price.UnitAmountInPennies,
				IntervalType:        db.NullIntervalType{IntervalType: db.IntervalType(price.IntervalType), Valid: true},
				TermLength:          pgtype.Int4{Int32: price.TermLength, Valid: true},
				Metadata:            price.Metadata,
			})
			if err != nil {
				sendError(c, http.StatusInternalServerError, "Failed to create prices", err)
				return
			}
			prices[i] = dbPrice
		}
	}

	if len(req.ProductTokens) > 0 {
		if err := h.createProductTokens(c, product.ID, req.ProductTokens); err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create product tokens", err)
			return
		}
	}

	sendSuccess(c, http.StatusCreated, toProductResponse(product, prices))
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

	existingProduct, err := h.common.db.GetProduct(c.Request.Context(), db.GetProductParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	if err := h.validateProductUpdate(c, req, existingProduct); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	params, err := h.updateProductParams(parsedUUID, req, existingProduct)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid update parameters", err)
		return
	}
	params.WorkspaceID = parsedWorkspaceID

	product, err := h.common.db.UpdateProduct(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update product")
		return
	}

	sendSuccess(c, http.StatusOK, toProductResponse(product, []db.Price{}))
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

	err = h.common.db.DeleteProduct(c.Request.Context(), db.DeleteProductParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Failed to delete product")
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
		IntervalType:        string(p.IntervalType.IntervalType),
		TermLength:          p.TermLength.Int32,
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
