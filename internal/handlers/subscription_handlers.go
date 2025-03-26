package handlers

import (
	"context"
	"cyphera-api/internal/client"
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionHandler manages subscription-related HTTP endpoints
type SubscriptionHandler struct {
	common           *CommonServices
	delegationClient *client.DelegationClient
}

// NewSubscriptionHandler creates a new subscription handler with the required dependencies
func NewSubscriptionHandler(common *CommonServices, delegationClient *client.DelegationClient) *SubscriptionHandler {
	return &SubscriptionHandler{
		common:           common,
		delegationClient: delegationClient,
	}
}

// CreateSubscriptionRequest represents the request body for creating a subscription
type CreateSubscriptionRequest struct {
	CustomerID       string          `json:"customer_id" binding:"required"`
	ProductID        string          `json:"product_id" binding:"required"`
	ProductTokenID   string          `json:"product_token_id" binding:"required"`
	DelegationID     string          `json:"delegation_id" binding:"required"`
	CustomerWalletID string          `json:"customer_wallet_id"`
	Status           string          `json:"status" binding:"required"`
	StartDate        int64           `json:"start_date" binding:"required"`
	EndDate          int64           `json:"end_date" binding:"required"`
	NextRedemption   int64           `json:"next_redemption" binding:"required"`
	Metadata         json.RawMessage `json:"metadata"`
}

// RedeemSubscriptionResponse represents the response for a subscription redemption
type RedeemSubscriptionResponse struct {
	SubscriptionID     string `json:"subscription_id"`
	TransactionHash    string `json:"transaction_hash,omitempty"`
	Status             string `json:"status"`
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	NextRedemptionDate int64  `json:"next_redemption_date,omitempty"`
}

// GetSubscription godoc
// @Summary Get a subscription by ID
// @Description Retrieves a subscription by its ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id} [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	subscription, err := h.common.db.GetSubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	sendSuccess(c, http.StatusOK, subscription)
}

// GetSubscriptionWithDetails godoc
// @Summary Get a subscription with related details
// @Description Retrieves a subscription with product, customer, token and network details
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} db.GetSubscriptionWithDetailsRow
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/details [get]
func (h *SubscriptionHandler) GetSubscriptionWithDetails(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	details, err := h.common.db.GetSubscriptionWithDetails(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Subscription details not found")
		return
	}

	sendSuccess(c, http.StatusOK, details)
}

// ListSubscriptions godoc
// @Summary List all subscriptions
// @Description Get a list of all non-deleted subscriptions
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {array} db.Subscription
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions [get]
func (h *SubscriptionHandler) ListSubscriptions(c *gin.Context) {
	// Check for pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// If pagination is requested, use it
	if c.Query("page") != "" || c.Query("limit") != "" {
		offset := (page - 1) * limit

		params := db.ListSubscriptionsWithPaginationParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		}

		subscriptions, err := h.common.db.ListSubscriptionsWithPagination(c.Request.Context(), params)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve subscriptions", err)
			return
		}

		// Get the total count for pagination metadata
		totalCount, err := h.common.db.CountSubscriptions(c.Request.Context())
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to count subscriptions", err)
			return
		}

		sendPaginatedSuccess(c, http.StatusOK, subscriptions, page, limit, int(totalCount))
		return
	}

	// Otherwise get all subscriptions
	subscriptions, err := h.common.db.ListSubscriptions(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscriptions", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscriptions)
}

// ListActiveSubscriptions godoc
// @Summary List active subscriptions
// @Description Get a list of all active subscriptions
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {array} db.Subscription
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/active [get]
func (h *SubscriptionHandler) ListActiveSubscriptions(c *gin.Context) {
	subscriptions, err := h.common.db.ListActiveSubscriptions(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve active subscriptions", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscriptions)
}

// ListSubscriptionsByCustomer godoc
// @Summary List subscriptions by customer
// @Description Get a list of all subscriptions for a specific customer
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Success 200 {array} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{customer_id}/subscriptions [get]
func (h *SubscriptionHandler) ListSubscriptionsByCustomer(c *gin.Context) {
	customerID := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve customer subscriptions", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscriptions)
}

// ListSubscriptionsByProduct godoc
// @Summary List subscriptions by product
// @Description Get a list of all subscriptions for a specific product
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {array} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/subscriptions [get]
func (h *SubscriptionHandler) ListSubscriptionsByProduct(c *gin.Context) {
	productID := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve product subscriptions", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscriptions)
}

// GetSubscriptionsByDelegation godoc
// @Summary List subscriptions by delegation
// @Description Get a list of all subscriptions using a specific delegation
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param delegation_id path string true "Delegation ID"
// @Success 200 {array} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /delegations/{delegation_id}/subscriptions [get]
func (h *SubscriptionHandler) GetSubscriptionsByDelegation(c *gin.Context) {
	delegationID := c.Param("delegation_id")
	parsedUUID, err := uuid.Parse(delegationID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid delegation ID format", err)
		return
	}

	subscriptions, err := h.common.db.GetSubscriptionsByDelegation(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscriptions by delegation", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscriptions)
}

// CreateSubscription godoc
// @Summary Create a new subscription
// @Description Creates a new subscription with the provided details
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body CreateSubscriptionRequest true "Subscription details"
// @Success 201 {object} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var request CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Parse UUIDs
	customerID, err := uuid.Parse(request.CustomerID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID", err)
		return
	}

	productID, err := uuid.Parse(request.ProductID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	productTokenID, err := uuid.Parse(request.ProductTokenID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product token ID", err)
		return
	}

	delegationID, err := uuid.Parse(request.DelegationID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid delegation ID", err)
		return
	}

	// Parse customer wallet ID if provided
	var customerWalletID pgtype.UUID
	if request.CustomerWalletID != "" {
		parsedCustomerWalletID, err := uuid.Parse(request.CustomerWalletID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid customer wallet ID", err)
			return
		}
		customerWalletID = pgtype.UUID{
			Bytes: parsedCustomerWalletID,
			Valid: true,
		}
	} else {
		customerWalletID = pgtype.UUID{
			Valid: false,
		}
	}

	// Parse status
	var status db.SubscriptionStatus
	switch request.Status {
	case "active", "canceled", "expired", "suspended", "failed":
		status = db.SubscriptionStatus(request.Status)
	default:
		sendError(c, http.StatusBadRequest, "Invalid status value", nil)
		return
	}

	// Create database params
	params := db.CreateSubscriptionParams{
		CustomerID:       customerID,
		ProductID:        productID,
		ProductTokenID:   productTokenID,
		DelegationID:     delegationID,
		CustomerWalletID: customerWalletID,
		Status:           status,
		CurrentPeriodStart: pgtype.Timestamptz{
			Time:  time.Unix(request.StartDate, 0),
			Valid: request.StartDate > 0,
		},
		CurrentPeriodEnd: pgtype.Timestamptz{
			Time:  time.Unix(request.EndDate, 0),
			Valid: request.EndDate > 0,
		},
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Unix(request.NextRedemption, 0),
			Valid: request.NextRedemption > 0,
		},
		TotalRedemptions:   0, // Start with 0 redemptions
		TotalAmountInCents: 0, // Start with 0 amount
		Metadata:           request.Metadata,
	}

	subscription, err := h.common.db.CreateSubscription(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create subscription", err)
		return
	}

	sendSuccess(c, http.StatusCreated, subscription)
}

// UpdateSubscriptionRequest represents the request body for updating a subscription
type UpdateSubscriptionRequest struct {
	CustomerID       string          `json:"customer_id"`
	ProductID        string          `json:"product_id"`
	ProductTokenID   string          `json:"product_token_id"`
	DelegationID     string          `json:"delegation_id"`
	CustomerWalletID string          `json:"customer_wallet_id"`
	Status           string          `json:"status"`
	StartDate        int64           `json:"start_date"`
	EndDate          int64           `json:"end_date"`
	NextRedemption   int64           `json:"next_redemption"`
	Metadata         json.RawMessage `json:"metadata"`
}

// UpdateSubscription godoc
// @Summary Update an existing subscription
// @Description Updates a subscription with the provided details
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Param subscription body UpdateSubscriptionRequest true "Updated subscription details"
// @Success 200 {object} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id} [put]
func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	ctx := c.Request.Context()
	subscriptionID := c.Param("subscription_id")
	parsedSubscriptionID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	// Check if subscription exists
	existingSubscription, err := h.common.db.GetSubscription(ctx, parsedSubscriptionID)
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	var request UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Initialize update params with existing values
	params := db.UpdateSubscriptionParams{
		ID:                 parsedSubscriptionID,
		CustomerID:         existingSubscription.CustomerID,
		ProductID:          existingSubscription.ProductID,
		ProductTokenID:     existingSubscription.ProductTokenID,
		DelegationID:       existingSubscription.DelegationID,
		CustomerWalletID:   existingSubscription.CustomerWalletID,
		Status:             existingSubscription.Status,
		CurrentPeriodStart: existingSubscription.CurrentPeriodStart,
		CurrentPeriodEnd:   existingSubscription.CurrentPeriodEnd,
		NextRedemptionDate: existingSubscription.NextRedemptionDate,
		TotalRedemptions:   existingSubscription.TotalRedemptions,
		TotalAmountInCents: existingSubscription.TotalAmountInCents,
		Metadata:           existingSubscription.Metadata,
	}

	// Update with provided values
	if request.CustomerID != "" {
		parsedCustomerID, err := uuid.Parse(request.CustomerID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
			return
		}
		params.CustomerID = parsedCustomerID
	}

	if request.ProductID != "" {
		parsedProductID, err := uuid.Parse(request.ProductID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
			return
		}
		params.ProductID = parsedProductID
	}

	if request.ProductTokenID != "" {
		parsedProductTokenID, err := uuid.Parse(request.ProductTokenID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid product token ID format", err)
			return
		}
		params.ProductTokenID = parsedProductTokenID
	}

	if request.DelegationID != "" {
		parsedDelegationID, err := uuid.Parse(request.DelegationID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid delegation ID format", err)
			return
		}
		params.DelegationID = parsedDelegationID
	}

	if request.CustomerWalletID != "" {
		parsedCustomerWalletID, err := uuid.Parse(request.CustomerWalletID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid customer wallet ID format", err)
			return
		}
		params.CustomerWalletID = pgtype.UUID{
			Bytes: parsedCustomerWalletID,
			Valid: true,
		}
	}

	if request.Status != "" {
		switch request.Status {
		case "active", "canceled", "expired", "suspended", "failed":
			params.Status = db.SubscriptionStatus(request.Status)
		default:
			sendError(c, http.StatusBadRequest, "Invalid status value", nil)
			return
		}
	}

	if request.StartDate > 0 {
		params.CurrentPeriodStart = pgtype.Timestamptz{
			Time:  time.Unix(request.StartDate, 0),
			Valid: true,
		}
	}

	if request.EndDate > 0 {
		params.CurrentPeriodEnd = pgtype.Timestamptz{
			Time:  time.Unix(request.EndDate, 0),
			Valid: true,
		}
	}

	if request.NextRedemption > 0 {
		params.NextRedemptionDate = pgtype.Timestamptz{
			Time:  time.Unix(request.NextRedemption, 0),
			Valid: true,
		}
	}

	if request.Metadata != nil {
		params.Metadata = request.Metadata
	}

	// Update subscription
	subscription, err := h.common.db.UpdateSubscription(ctx, params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update subscription", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscription)
}

// UpdateSubscriptionStatus godoc
// @Summary Update a subscription's status
// @Description Updates just the status of a subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Param status body struct { Status string `json:"status"` } true "New status"
// @Success 200 {object} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/status [patch]
func (h *SubscriptionHandler) UpdateSubscriptionStatus(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	// Parse request body
	var request struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Validate status
	var status db.SubscriptionStatus
	switch request.Status {
	case "active", "canceled", "expired", "suspended", "failed":
		status = db.SubscriptionStatus(request.Status)
	default:
		sendError(c, http.StatusBadRequest, "Invalid status value", nil)
		return
	}

	// Update status
	params := db.UpdateSubscriptionStatusParams{
		ID:     parsedUUID,
		Status: status,
	}

	updatedSubscription, err := h.common.db.UpdateSubscriptionStatus(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update subscription status")
		return
	}

	sendSuccess(c, http.StatusOK, updatedSubscription)
}

// CancelSubscription godoc
// @Summary Cancel a subscription
// @Description Sets a subscription status to canceled
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} db.Subscription
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/cancel [post]
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	canceledSubscription, err := h.common.db.CancelSubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to cancel subscription")
		return
	}

	sendSuccess(c, http.StatusOK, canceledSubscription)
}

// DeleteSubscription godoc
// @Summary Delete a subscription
// @Description Soft-deletes a subscription (marks as deleted)
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id} [delete]
func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	err = h.common.db.DeleteSubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete subscription")
		return
	}

	sendSuccessMessage(c, http.StatusOK, "Subscription successfully deleted")
}

// GetExpiredSubscriptions godoc
// @Summary Get all expired subscriptions
// @Description Retrieves all subscriptions that have expired but haven't been marked as expired
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {array} db.Subscription
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/expired [get]
func (h *SubscriptionHandler) GetExpiredSubscriptions(c *gin.Context) {
	subscriptions, err := h.common.db.GetExpiredSubscriptions(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve expired subscriptions", err)
		return
	}

	sendSuccess(c, http.StatusOK, subscriptions)
}

// RedeemSubscription godoc
// @Summary Redeem a subscription
// @Description Process a redemption for an active subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} RedeemSubscriptionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/redeem [post]
func (h *SubscriptionHandler) RedeemSubscription(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	// Get the subscription details
	subscription, err := h.common.db.GetSubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	// Verify subscription is active
	if subscription.Status != db.SubscriptionStatusActive {
		sendError(c, http.StatusBadRequest, "Cannot redeem inactive subscription", nil)
		return
	}

	// Get the product and delegation data
	product, err := h.common.db.GetProduct(c.Request.Context(), subscription.ProductID)
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	// Get the product token to determine price
	productToken, err := h.common.db.GetProductToken(c.Request.Context(), subscription.ProductTokenID)
	if err != nil {
		handleDBError(c, err, "Product token not found")
		return
	}

	// Get the delegation data
	delegationData, err := h.common.db.GetDelegationData(c.Request.Context(), subscription.DelegationID)
	if err != nil {
		handleDBError(c, err, "Delegation data not found")
		return
	}

	// Process the redemption
	result, err := h.processRedemption(c.Request.Context(), subscription, product, productToken, delegationData)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to redeem subscription", err)
		return
	}

	sendSuccess(c, http.StatusOK, result)
}

// processRedemption handles the core redemption logic
// It redeems the delegation, creates events, and updates subscription state
func (h *SubscriptionHandler) processRedemption(ctx context.Context, subscription db.Subscription, product db.Product, productToken db.GetProductTokenRow, delegationData db.DelegationDatum) (*RedeemSubscriptionResponse, error) {
	response := &RedeemSubscriptionResponse{
		SubscriptionID: subscription.ID.String(),
		Status:         string(subscription.Status),
		Success:        false,
	}

	// Use the product price in pennies for the amount
	amountInCents := product.PriceInPennies

	// Create metadata with relevant information
	metadata := map[string]interface{}{
		"product_id":      product.ID.String(),
		"product_name":    product.Name,
		"token_id":        productToken.TokenID.String(),
		"price_in_cents":  amountInCents,
		"redemption_time": time.Now().Unix(),
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return response, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert signature to bytes if it's not already
	var signatureBytes []byte
	if len(delegationData.Signature) > 0 {
		signatureBytes = []byte(delegationData.Signature)
	}

	// Get merchant wallet information
	merchantWallet, err := h.common.db.GetWalletByID(ctx, product.WalletID)
	if err != nil {
		return response, fmt.Errorf("failed to get merchant wallet: %w", err)
	}

	// Get token contract address
	token, err := h.common.db.GetToken(ctx, productToken.TokenID)
	if err != nil {
		return response, fmt.Errorf("failed to get token details: %w", err)
	}

	// Format price in dollars (convert pennies to dollars)
	price := fmt.Sprintf("%.2f", float64(amountInCents)/100.0)

	// Attempt to redeem the delegation
	txHash, err := h.delegationClient.RedeemDelegationDirectly(
		ctx,
		signatureBytes,
		merchantWallet.WalletAddress,
		token.ContractAddress,
		price,
	)
	if err != nil {
		// Record failure event
		errorMessage := pgtype.Text{
			String: err.Error(),
			Valid:  true,
		}

		failureParams := db.CreateFailedRedemptionEventParams{
			SubscriptionID: subscription.ID,
			AmountInCents:  amountInCents,
			ErrorMessage:   errorMessage,
			Metadata:       metadataBytes,
		}

		_, eventErr := h.common.db.CreateFailedRedemptionEvent(ctx, failureParams)
		if eventErr != nil {
			log.Printf("Failed to record redemption failure: %v", eventErr)
		}

		response.Message = fmt.Sprintf("Delegation redemption failed: %v", err)
		return response, err
	}

	// Record successful redemption event
	txHashPg := pgtype.Text{
		String: txHash,
		Valid:  true,
	}

	eventParams := db.CreateRedemptionEventParams{
		SubscriptionID:  subscription.ID,
		TransactionHash: txHashPg,
		AmountInCents:   amountInCents,
		Metadata:        metadataBytes,
	}

	_, eventErr := h.common.db.CreateRedemptionEvent(ctx, eventParams)
	if eventErr != nil {
		// Log error but continue with subscription update
		log.Printf("Error recording redemption event: %v", eventErr)
	}

	// Calculate next redemption date
	nextRedemptionDate := CalculateNextRedemption(product, time.Now())

	// Update subscription with new redemption count and date
	incrementParams := db.IncrementSubscriptionRedemptionParams{
		ID:                 subscription.ID,
		TotalAmountInCents: amountInCents,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  nextRedemptionDate,
			Valid: !nextRedemptionDate.IsZero(),
		},
	}

	updatedSubscription, err := h.common.db.IncrementSubscriptionRedemption(ctx, incrementParams)
	if err != nil {
		return response, fmt.Errorf("transaction recorded but failed to update subscription: %w", err)
	}

	// Populate response with success details
	response.Success = true
	response.TransactionHash = txHash
	response.Message = "Subscription successfully redeemed"

	if updatedSubscription.NextRedemptionDate.Valid {
		response.NextRedemptionDate = updatedSubscription.NextRedemptionDate.Time.Unix()
	}

	return response, nil
}

// RedeemDueSubscriptions godoc
// @Summary Process all subscriptions due for redemption
// @Description Find and process all subscriptions that are due for redemption
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/redeem-due [post]
func (h *SubscriptionHandler) RedeemDueSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()
	now := pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}

	// Get all subscriptions due for renewal
	subscriptions, err := h.common.db.ListSubscriptionsDueForRenewal(ctx, now)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscriptions due for renewal", err)
		return
	}

	if len(subscriptions) == 0 {
		sendSuccessMessage(c, http.StatusOK, "No subscriptions due for redemption")
		return
	}

	// Track processing statistics
	stats := struct {
		Total     int `json:"total"`
		Succeeded int `json:"succeeded"`
		Failed    int `json:"failed"`
	}{
		Total: len(subscriptions),
	}

	// Process each subscription
	for _, subscription := range subscriptions {
		// Get required data for processing
		product, err := h.common.db.GetProduct(ctx, subscription.ProductID)
		if err != nil {
			log.Printf("Error fetching product for subscription %s: %v", subscription.ID, err)
			stats.Failed++
			continue
		}

		productToken, err := h.common.db.GetProductToken(ctx, subscription.ProductTokenID)
		if err != nil {
			log.Printf("Error fetching product token for subscription %s: %v", subscription.ID, err)
			stats.Failed++
			continue
		}

		delegationData, err := h.common.db.GetDelegationData(ctx, subscription.DelegationID)
		if err != nil {
			log.Printf("Error fetching delegation data for subscription %s: %v", subscription.ID, err)
			stats.Failed++
			continue
		}

		// Get merchant wallet
		merchantWallet, err := h.common.db.GetWalletByID(ctx, product.WalletID)
		if err != nil {
			log.Printf("Error fetching merchant wallet for subscription %s: %v", subscription.ID, err)
			stats.Failed++
			continue
		}

		// Get token details
		token, err := h.common.db.GetToken(ctx, productToken.TokenID)
		if err != nil {
			log.Printf("Error fetching token for subscription %s: %v", subscription.ID, err)
			stats.Failed++
			continue
		}

		// Format price
		price := fmt.Sprintf("%.2f", float64(product.PriceInPennies)/100.0)

		// Convert signature to bytes
		var signatureBytes []byte
		if len(delegationData.Signature) > 0 {
			signatureBytes = []byte(delegationData.Signature)
		} else {
			log.Printf("Empty signature for subscription %s", subscription.ID)
			stats.Failed++
			continue
		}

		// Attempt to redeem the delegation
		txHash, err := h.delegationClient.RedeemDelegationDirectly(
			ctx,
			signatureBytes,
			merchantWallet.WalletAddress,
			token.ContractAddress,
			price,
		)
		if err != nil {
			log.Printf("Failed to redeem subscription %s: %v", subscription.ID, err)

			// Record failure event
			errorMessage := pgtype.Text{
				String: err.Error(),
				Valid:  true,
			}

			failureParams := db.CreateFailedRedemptionEventParams{
				SubscriptionID: subscription.ID,
				AmountInCents:  product.PriceInPennies,
				ErrorMessage:   errorMessage,
			}

			_, eventErr := h.common.db.CreateFailedRedemptionEvent(ctx, failureParams)
			if eventErr != nil {
				log.Printf("Failed to record redemption failure for subscription %s: %v", subscription.ID, eventErr)
			}

			stats.Failed++
			continue
		}

		// Record successful redemption event
		txHashPg := pgtype.Text{
			String: txHash,
			Valid:  true,
		}

		eventParams := db.CreateRedemptionEventParams{
			SubscriptionID:  subscription.ID,
			TransactionHash: txHashPg,
			AmountInCents:   product.PriceInPennies,
		}

		_, eventErr := h.common.db.CreateRedemptionEvent(ctx, eventParams)
		if eventErr != nil {
			log.Printf("Error recording redemption event for subscription %s: %v", subscription.ID, eventErr)
		}

		// Calculate next redemption date
		nextRedemptionDate := CalculateNextRedemption(product, time.Now())

		// Update subscription with new redemption count and date
		incrementParams := db.IncrementSubscriptionRedemptionParams{
			ID:                 subscription.ID,
			TotalAmountInCents: product.PriceInPennies,
			NextRedemptionDate: pgtype.Timestamptz{
				Time:  nextRedemptionDate,
				Valid: !nextRedemptionDate.IsZero(),
			},
		}

		_, err = h.common.db.IncrementSubscriptionRedemption(ctx, incrementParams)
		if err != nil {
			log.Printf("Failed to update subscription %s: %v", subscription.ID, err)
			stats.Failed++
			continue
		}

		log.Printf("Successfully redeemed subscription %s with transaction %s", subscription.ID, txHash)
		stats.Succeeded++
	}

	// Return stats about the processing
	sendSuccess(c, http.StatusOK, stats)
}

// CalculateNextRedemption calculates the next redemption date based on product interval
func CalculateNextRedemption(product db.Product, now time.Time) time.Time {
	// Return same time for one-off products
	if product.ProductType == db.ProductTypeOneOff {
		return time.Time{}
	}

	// For recurring products, calculate based on interval
	switch product.IntervalType {
	case db.IntervalType5mins:
		return now.Add(5 * time.Minute)
	case db.IntervalTypeDaily:
		return now.AddDate(0, 0, 1)
	case db.IntervalTypeWeek:
		return now.AddDate(0, 0, 7)
	case db.IntervalTypeMonth:
		return now.AddDate(0, 1, 0)
	case db.IntervalTypeYear:
		return now.AddDate(1, 0, 0)
	default:
		return now.AddDate(0, 1, 0) // Default to monthly
	}
}

// GetRedemptionStatus godoc
// @Summary Get the redemption status for a subscription
// @Description Retrieves the current redemption status for a given subscription ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} GetRedemptionStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /subscriptions/{subscription_id}/redemption-status [get]
func (h *SubscriptionHandler) GetRedemptionStatus(c *gin.Context) {
	ctx := c.Request.Context()
	subscriptionID := c.Param("subscription_id")

	// Validate subscription ID
	if subscriptionID == "" {
		sendError(c, http.StatusBadRequest, "Subscription ID is required", nil)
		return
	}

	subID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	// Check if subscription exists
	subscription, err := h.common.db.GetSubscription(ctx, subID)
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	// Get latest events for the subscription
	events, err := h.common.db.ListSubscriptionEventsBySubscription(ctx, subID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscription events", err)
		return
	}

	// Initialize response with default values
	response := GetRedemptionStatusResponse{
		SubscriptionID:   subscriptionID,
		LastRedemptionAt: nil,
		Status:           "pending",
		Message:          "No redemption attempts found",
		TotalRedemptions: subscription.TotalRedemptions,
		NextRedemptionAt: subscription.NextRedemptionDate.Time,
		TransactionHash:  "",
		FailureReason:    "",
		LastAttemptedAt:  nil,
	}

	// No events found
	if len(events) == 0 {
		c.JSON(http.StatusOK, response)
		return
	}

	// Check for redemption events
	var lastRedemptionEvent *db.SubscriptionEvent
	var lastFailedRedemptionEvent *db.SubscriptionEvent

	for i := range events {
		event := events[i]
		if event.EventType == db.SubscriptionEventTypeRedeemed {
			if lastRedemptionEvent == nil || event.CreatedAt.Time.After(lastRedemptionEvent.CreatedAt.Time) {
				lastRedemptionEvent = &event
			}
		} else if event.EventType == db.SubscriptionEventTypeFailedRedemption {
			if lastFailedRedemptionEvent == nil || event.CreatedAt.Time.After(lastFailedRedemptionEvent.CreatedAt.Time) {
				lastFailedRedemptionEvent = &event
			}
		}
	}

	// Determine status based on the events found
	if lastRedemptionEvent != nil {
		// If we have a successful redemption, and it's more recent than any failed attempts
		if lastFailedRedemptionEvent == nil || lastRedemptionEvent.CreatedAt.Time.After(lastFailedRedemptionEvent.CreatedAt.Time) {
			response.Status = "success"
			response.Message = "Subscription successfully redeemed"
			lastRedemptionTime := lastRedemptionEvent.CreatedAt.Time
			response.LastRedemptionAt = &lastRedemptionTime
			response.LastAttemptedAt = &lastRedemptionTime
			response.TransactionHash = lastRedemptionEvent.TransactionHash.String
		}
	}

	// If we have a failed redemption, and it's more recent than any successful attempts
	if lastFailedRedemptionEvent != nil {
		if lastRedemptionEvent == nil || lastFailedRedemptionEvent.CreatedAt.Time.After(lastRedemptionEvent.CreatedAt.Time) {
			response.Status = "failed"
			response.Message = "Redemption attempt failed"
			lastFailedTime := lastFailedRedemptionEvent.CreatedAt.Time
			response.LastAttemptedAt = &lastFailedTime
			response.FailureReason = lastFailedRedemptionEvent.ErrorMessage.String
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetRedemptionStatusResponse represents the response for the redemption status endpoint
type GetRedemptionStatusResponse struct {
	SubscriptionID   string     `json:"subscription_id"`
	Status           string     `json:"status"` // pending, success, failed
	Message          string     `json:"message"`
	LastRedemptionAt *time.Time `json:"last_redemption_at,omitempty"`
	LastAttemptedAt  *time.Time `json:"last_attempted_at,omitempty"`
	TotalRedemptions int32      `json:"total_redemptions"`
	NextRedemptionAt time.Time  `json:"next_redemption_at"`
	TransactionHash  string     `json:"transaction_hash,omitempty"`
	FailureReason    string     `json:"failure_reason,omitempty"`
}
