package handlers

import (
	"context"
	"cyphera-api/internal/client"
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionHandler manages subscription-related HTTP endpoints
type SubscriptionHandler struct {
	common               *CommonServices
	delegationClient     *client.DelegationClient
	lastRedemptionTxHash string // Stores the transaction hash from the last successful redemption
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
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get pagination parameters
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

	params := db.ListSubscriptionDetailsWithPaginationParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	}

	subscriptions, err := h.common.db.ListSubscriptionDetailsWithPagination(c.Request.Context(), params)
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

	sendPaginatedSuccess(c, http.StatusOK, subscriptions, int(page), int(limit), int(totalCount))
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
	case string(db.SubscriptionStatusActive), string(db.SubscriptionStatusCanceled), string(db.SubscriptionStatusExpired), string(db.SubscriptionStatusSuspended), string(db.SubscriptionStatusFailed):
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
		case string(db.SubscriptionStatusActive), string(db.SubscriptionStatusCanceled), string(db.SubscriptionStatusExpired), string(db.SubscriptionStatusSuspended), string(db.SubscriptionStatusFailed):
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
	case string(db.SubscriptionStatusActive), string(db.SubscriptionStatusCanceled), string(db.SubscriptionStatusExpired), string(db.SubscriptionStatusSuspended), string(db.SubscriptionStatusFailed):
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

// CalculateNextRedemption computes the next scheduled redemption date based on interval type.
// This is the central function for all interval calculations in the system.
// For testing and development purposes, 1min and 5mins intervals are supported.
func CalculateNextRedemption(intervalType db.IntervalType, currentTime time.Time) time.Time {
	switch intervalType {
	case db.IntervalType1min:
		return currentTime.Add(1 * time.Minute)
	case db.IntervalType5mins:
		return currentTime.Add(5 * time.Minute)
	case db.IntervalTypeDaily:
		return currentTime.AddDate(0, 0, 1) // Next day
	case db.IntervalTypeWeek:
		return currentTime.AddDate(0, 0, 7) // Next week
	case db.IntervalTypeMonth:
		return currentTime.AddDate(0, 1, 0) // Next month
	case db.IntervalTypeYear:
		return currentTime.AddDate(1, 0, 0) // Next year
	default:
		return currentTime.AddDate(0, 1, 0) // Default to monthly
	}
}

// CalculatePeriodEnd determines the end date of a subscription period based on interval type and term length.
// This function is used when creating or updating subscription periods.
func CalculatePeriodEnd(start time.Time, intervalType db.IntervalType, termLength int32) time.Time {
	switch intervalType {
	case db.IntervalType1min:
		return start.Add(time.Duration(termLength) * time.Minute)
	case db.IntervalType5mins:
		return start.Add(time.Duration(termLength*5) * time.Minute)
	case db.IntervalTypeDaily:
		return start.AddDate(0, 0, int(termLength))
	case db.IntervalTypeWeek:
		return start.AddDate(0, 0, int(termLength*7))
	case db.IntervalTypeMonth:
		return start.AddDate(0, int(termLength), 0)
	case db.IntervalTypeYear:
		return start.AddDate(int(termLength), 0, 0)
	default:
		return start.AddDate(0, int(termLength), 0) // Default to monthly
	}
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

// ProcessDueSubscriptionsResult contains statistics about the processing job
type ProcessDueSubscriptionsResult struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Completed int `json:"completed"`
}

// processSubscriptionParams contains all parameters needed to process a subscription
type processSubscriptionParams struct {
	ctx            context.Context
	subscription   db.Subscription
	product        db.Product
	productToken   db.GetProductTokenRow
	delegationData db.DelegationDatum
	merchantWallet db.Wallet
	token          db.Token
	isFinalPayment bool
	now            time.Time
	queries        db.Querier                     // Database queries interface (could be transaction or regular)
	tx             pgx.Tx                         // Optional transaction for atomic operations
	results        *ProcessDueSubscriptionsResult // Optional results tracker for ProcessDueSubscriptions
}

// processSubscriptionResult contains the result of processing a subscription
type processSubscriptionResult struct {
	isProcessed bool   // Successfully processed (payment redeemed)
	isCompleted bool   // Subscription was completed (final payment processed)
	txHash      string // Transaction hash from successful redemption
}

// processSubscription handles the core logic of processing a subscription
// It is used by both ProcessDueSubscriptions and RedeemDueSubscriptions methods
func (h *SubscriptionHandler) processSubscription(params processSubscriptionParams) (processSubscriptionResult, error) {
	// Initialize result
	result := processSubscriptionResult{}
	subscription := params.subscription

	// Marshal delegation to JSON bytes for redemption
	delegationBytes, err := json.Marshal(params.delegationData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal delegation data for subscription %s: %v",
			subscription.ID, err)
		log.Println(errMsg)

		// Create appropriate event based on transaction mode
		if params.tx != nil {
			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID: subscription.ID,
				EventType:      db.SubscriptionEventTypeFailedRedemption,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				AmountInCents:  params.product.PriceInPennies,
				OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					subscription.ID, eventErr)
			}
			// Update counters if tracking results
			if params.results != nil {
				params.results.Failed++
			}
		} else {
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: subscription.ID,
				AmountInCents:  params.product.PriceInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					subscription.ID, eventErr)
			}
		}
		result.isProcessed = false // Set isProcessed to false since increment failed
		return result, fmt.Errorf("json marshal error: %w", err)
	}

	// Redeem the delegation with retries
	redemptionSuccess := false
	var redemptionError error

	// Retry configuration
	const (
		maxRetries     = 3
		initialBackoff = 1 * time.Second
		maxBackoff     = 10 * time.Second
		backoffFactor  = 2
	)

	// Attempt to redeem with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Calculate backoff duration (exponential with jitter)
		backoff := initialBackoff * time.Duration(math.Pow(float64(backoffFactor), float64(attempt)))
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		// Add jitter (±20%)
		jitter := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))

		if attempt > 0 {
			log.Printf("Retrying delegation redemption for subscription %s (attempt %d/%d) after %v",
				subscription.ID, attempt+1, maxRetries, jitter)
			// Wait before retry
			time.Sleep(jitter)
		}

		// Attempt redemption
		// Get token info for redemption
		merchantAddress := params.merchantWallet.WalletAddress
		tokenAddress := params.token.ContractAddress
		price := fmt.Sprintf("%.2f", float64(params.product.PriceInPennies)/100.0)

		// Call the delegation client to redeem
		txHash, redemptionErr := h.delegationClient.RedeemDelegationDirectly(
			params.ctx,
			delegationBytes,
			merchantAddress,
			tokenAddress,
			price,
		)

		if redemptionErr == nil {
			// Success! Store the transaction hash
			h.lastRedemptionTxHash = txHash
			redemptionSuccess = true
			result.txHash = txHash
			break
		}

		// Check if it's a permanent error that shouldn't be retried
		if isPermanentRedemptionError(redemptionErr) {
			redemptionError = redemptionErr
			log.Printf("Permanent error redeeming delegation for subscription %s, won't retry: %v",
				subscription.ID, redemptionErr)
			break
		}

		// Temporary error, we'll retry if we have attempts left
		redemptionError = redemptionErr
		log.Printf("Temporary error redeeming delegation for subscription %s (attempt %d/%d): %v",
			subscription.ID, attempt+1, maxRetries, redemptionErr)
	}

	// Check if redemption was successful
	if redemptionSuccess {
		// Successfully redeemed delegation, but we still need to increment
		// result.isProcessed will be set to true after increment succeeds

		// Update next redemption date based on product interval
		var nextRedemptionDate pgtype.Timestamptz

		// Calculate next redemption date using the product interval type
		nextDate := CalculateNextRedemption(params.product.IntervalType, params.now)
		nextRedemptionDate = pgtype.Timestamptz{
			Time:  nextDate,
			Valid: true,
		}

		// Prepare update parameters for incrementing subscription
		incrementParams := db.IncrementSubscriptionRedemptionParams{
			ID:                 subscription.ID,
			TotalAmountInCents: params.product.PriceInPennies,
			NextRedemptionDate: nextRedemptionDate,
		}

		// Update the subscription with new redemption data
		_, err := params.queries.IncrementSubscriptionRedemption(params.ctx, incrementParams)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to update subscription %s after successful redemption: %v",
				subscription.ID, err)
			log.Println(errMsg)

			// Create failure event
			if params.tx != nil {
				_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
					SubscriptionID: subscription.ID,
					EventType:      db.SubscriptionEventTypeFailedRedemption,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					AmountInCents:  params.product.PriceInPennies,
					OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					log.Printf("Failed to record failure event for subscription %s: %v",
						subscription.ID, eventErr)
				}
				if params.results != nil {
					params.results.Failed++
				}
			} else {
				_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
					SubscriptionID: subscription.ID,
					AmountInCents:  params.product.PriceInPennies,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					log.Printf("Failed to record failure event for subscription %s: %v",
						subscription.ID, eventErr)
				}
			}
			result.isProcessed = false
			return result, fmt.Errorf("failed to update subscription: %w", err)
		}

		// Both redemption and increment succeeded
		result.isProcessed = true

		// If this was the final payment and it was successful, mark the subscription as completed
		if params.isFinalPayment {
			updateParams := db.UpdateSubscriptionStatusParams{
				ID:     subscription.ID,
				Status: db.SubscriptionStatusCompleted,
			}
			if _, updateErr := params.queries.UpdateSubscriptionStatus(params.ctx, updateParams); updateErr != nil {
				log.Printf("Warning: Failed to mark subscription %s as completed: %v",
					subscription.ID, updateErr)
			} else {
				result.isCompleted = true
				log.Printf("Marked subscription %s as completed after successful final payment", subscription.ID)
			}
		}

		// Record successful event with appropriate event type (based on whether it's final payment)
		eventType := db.SubscriptionEventTypeRedeemed
		if result.isCompleted {
			eventType = db.SubscriptionEventTypeCompleted
		}

		// Create success event
		if params.tx != nil {
			metadataBytes, _ := json.Marshal(map[string]interface{}{
				"next_redemption": nextRedemptionDate.Time,
				"is_final":        result.isCompleted,
			})

			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID:  subscription.ID,
				EventType:       eventType,
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.product.PriceInPennies,
				OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:        metadataBytes,
			})
			if eventErr != nil {
				log.Printf("Warning: Failed to record success event for subscription %s: %v",
					subscription.ID, eventErr)
			}
			if params.results != nil {
				if result.isCompleted {
					params.results.Completed++
				} else {
					params.results.Succeeded++
				}
			}
		} else {
			metadataBytes, _ := json.Marshal(map[string]interface{}{
				"next_redemption": nextRedemptionDate.Time,
				"is_final":        result.isCompleted,
			})

			_, eventErr := params.queries.CreateRedemptionEvent(params.ctx, db.CreateRedemptionEventParams{
				SubscriptionID:  subscription.ID,
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.product.PriceInPennies,
				Metadata:        metadataBytes,
			})
			if eventErr != nil {
				log.Printf("Warning: Failed to record success event for subscription %s: %v",
					subscription.ID, eventErr)
			}
		}

		log.Printf("Successfully processed subscription %s, next redemption at %s",
			subscription.ID, nextRedemptionDate.Time)
	} else {
		// Redemption failed
		errMsg := fmt.Sprintf("Failed to redeem delegation for subscription %s: %v",
			subscription.ID, redemptionError)
		log.Println(errMsg)

		// If this was the final payment and redemption failed, keep as active but mark as overdue
		var updateErr error
		if params.isFinalPayment {
			// Update subscription to failed status
			updateParams := db.UpdateSubscriptionStatusParams{
				ID:     subscription.ID,
				Status: db.SubscriptionStatusFailed,
			}
			if _, updateErr = params.queries.UpdateSubscriptionStatus(params.ctx, updateParams); updateErr != nil {
				log.Printf("Failed to update subscription %s to failed status: %v",
					subscription.ID, updateErr)
			} else {
				log.Printf("Marked subscription %s as failed due to failed final payment", subscription.ID)
			}
		}
		// Create failure event
		if params.tx != nil {
			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID: subscription.ID,
				EventType:      db.SubscriptionEventTypeFailedRedemption,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				AmountInCents:  params.product.PriceInPennies,
				OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					subscription.ID, eventErr)
			}
			if params.results != nil {
				params.results.Failed++
			}
		} else {
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: subscription.ID,
				AmountInCents:  params.product.PriceInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					subscription.ID, eventErr)
			}
		}

		result.isProcessed = false // Set isProcessed to false since increment failed
		return result, fmt.Errorf("redemption failed: %w", redemptionError)
	}

	return result, nil
}

// isPermanentRedemptionError determines if a redemption error should not be retried
func isPermanentRedemptionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for signatures of permanent errors
	errMsg := err.Error()
	permanentErrorSigns := []string{
		"invalid signature",
		"delegation expired",
		"invalid delegation format",
		"invalid token",
		"unauthorized",
		"insufficient funds",
	}

	for _, sign := range permanentErrorSigns {
		if strings.Contains(strings.ToLower(errMsg), sign) {
			return true
		}
	}

	return false
}

// RedeemDueSubscriptions processes all subscriptions with the specified IDs
// This is used by the RedeemDueSubscriptionsHTTP endpoint
func (h *SubscriptionHandler) RedeemDueSubscriptions(ctx context.Context, subscriptionIDs []uuid.UUID) (ProcessDueSubscriptionsResult, error) {
	results := ProcessDueSubscriptionsResult{}
	results.Total = len(subscriptionIDs)

	// Process each subscription
	for _, subscriptionID := range subscriptionIDs {
		// Get subscription details
		subscription, err := h.common.db.GetSubscription(ctx, subscriptionID)
		if err != nil {
			log.Printf("Failed to get subscription %s: %v", subscriptionID, err)
			results.Failed++
			continue
		}

		// Skip subscriptions that are not active
		if subscription.Status != db.SubscriptionStatusActive {
			log.Printf("Skipping non-active subscription %s with status %s", subscriptionID, subscription.Status)
			continue
		}

		// Get required data for processing
		product, err := h.common.db.GetProduct(ctx, subscription.ProductID)
		if err != nil {
			log.Printf("Failed to get product for subscription %s: %v", subscriptionID, err)
			results.Failed++
			continue
		}

		productToken, err := h.common.db.GetProductToken(ctx, subscription.ProductTokenID)
		if err != nil {
			log.Printf("Failed to get product token for subscription %s: %v", subscriptionID, err)
			results.Failed++
			continue
		}

		token, err := h.common.db.GetToken(ctx, productToken.TokenID)
		if err != nil {
			log.Printf("Failed to get token for subscription %s: %v", subscriptionID, err)
			results.Failed++
			continue
		}

		merchantWallet, err := h.common.db.GetWalletByID(ctx, product.WalletID)
		if err != nil {
			log.Printf("Failed to get merchant wallet for subscription %s: %v", subscriptionID, err)
			results.Failed++
			continue
		}

		delegationData, err := h.common.db.GetDelegationData(ctx, subscription.DelegationID)
		if err != nil {
			log.Printf("Failed to get delegation data for subscription %s: %v", subscriptionID, err)
			results.Failed++
			continue
		}

		// Check if current time is past the current period end
		now := time.Now()
		isFinalPayment := subscription.CurrentPeriodEnd.Time.Before(now)

		// Process the subscription
		params := processSubscriptionParams{
			ctx:            ctx,
			subscription:   subscription,
			product:        product,
			productToken:   productToken,
			delegationData: delegationData,
			merchantWallet: merchantWallet,
			token:          token,
			isFinalPayment: isFinalPayment,
			now:            now,
			queries:        h.common.db,
			tx:             nil, // No transaction for batch redemption
			results:        &results,
		}

		_, err = h.processSubscription(params)
		if err != nil {
			// Error handling is done in processSubscription, just continue
			continue
		}
	}

	return results, nil
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
func (h *SubscriptionHandler) RedeemDueSubscriptionsHTTP(c *gin.Context) {
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

	// Extract IDs from subscriptions for processing
	subscriptionIDs := make([]uuid.UUID, len(subscriptions))
	for i, sub := range subscriptions {
		subscriptionIDs[i] = sub.ID
	}

	// Process the subscriptions using our consolidated helper
	results, err := h.RedeemDueSubscriptions(ctx, subscriptionIDs)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Error processing due subscriptions", err)
		return
	}

	// Return stats about the processing
	sendSuccess(c, http.StatusOK, results)
}

// ProcessDueSubscriptions finds and processes all subscriptions that are due for redemption
// It uses a transaction for atomicity and updates subscription status based on the result
func (h *SubscriptionHandler) ProcessDueSubscriptions(ctx context.Context) (ProcessDueSubscriptionsResult, error) {
	results := ProcessDueSubscriptionsResult{}
	now := time.Now()

	// Start a transaction using the BeginTx helper
	tx, qtx, err := h.common.BeginTx(ctx)
	if err != nil {
		return results, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if tx != nil {
			if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
				log.Printf("Failed to rollback transaction: %v", err)
			}
		}
	}()

	// Query for subscriptions due for redemption and lock them for processing
	nowPgType := pgtype.Timestamptz{Time: now, Valid: true}
	subscriptions, err := qtx.ListSubscriptionsDueForRenewal(ctx, nowPgType)
	if err != nil {
		return results, fmt.Errorf("failed to fetch subscriptions due for redemption: %w", err)
	}

	// Update result count
	results.Total = len(subscriptions)
	if results.Total == 0 {
		// No subscriptions to process, commit empty transaction
		if err := tx.Commit(ctx); err != nil {
			return results, fmt.Errorf("failed to commit empty transaction: %w", err)
		}
		tx = nil // Set to nil to avoid double rollback
		return results, nil
	}

	log.Printf("Found %d subscriptions due for redemption", results.Total)

	// Process each subscription within the transaction
	for _, subscription := range subscriptions {
		// Skip subscriptions that are not active
		if subscription.Status != db.SubscriptionStatusActive {
			log.Printf("Skipping non-active subscription %s with status %s", subscription.ID, subscription.Status)
			continue
		}

		// Get required data for processing
		product, err := qtx.GetProduct(ctx, subscription.ProductID)
		if err != nil {
			log.Printf("Failed to get product for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}

		productToken, err := qtx.GetProductToken(ctx, subscription.ProductTokenID)
		if err != nil {
			log.Printf("Failed to get product token for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}

		token, err := qtx.GetToken(ctx, productToken.TokenID)
		if err != nil {
			log.Printf("Failed to get token for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}

		merchantWallet, err := qtx.GetWalletByID(ctx, product.WalletID)
		if err != nil {
			log.Printf("Failed to get merchant wallet for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}

		delegationData, err := qtx.GetDelegationData(ctx, subscription.DelegationID)
		if err != nil {
			log.Printf("Failed to get delegation data for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}

		// Check if this is the final payment
		isFinalPayment := subscription.CurrentPeriodEnd.Time.Before(now)

		// Process the subscription
		params := processSubscriptionParams{
			ctx:            ctx,
			subscription:   subscription,
			product:        product,
			productToken:   productToken,
			delegationData: delegationData,
			merchantWallet: merchantWallet,
			token:          token,
			isFinalPayment: isFinalPayment,
			now:            now,
			queries:        qtx,
			tx:             tx,
			results:        &results,
		}

		_, err = h.processSubscription(params)
		if err != nil {
			// Error handling is done in processSubscription, just continue
			continue
		}
	}

	// Commit the transaction if we got this far
	if err := tx.Commit(ctx); err != nil {
		return results, fmt.Errorf("failed to commit transaction: %w", err)
	}
	tx = nil // Set to nil to avoid double rollback

	return results, nil
}

// ProcessDueSubscriptionsHTTP godoc
// @Summary Process all subscriptions due for redemption
// @Description Find and process all subscriptions that are due for redemption
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {object} ProcessDueSubscriptionsResult
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/process-due [post]
func (h *SubscriptionHandler) ProcessDueSubscriptionsHTTP(c *gin.Context) {
	ctx := c.Request.Context()

	// Process all due subscriptions using transaction
	results, err := h.ProcessDueSubscriptions(ctx)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Error processing due subscriptions", err)
		return
	}

	// Return stats about the processing
	sendSuccess(c, http.StatusOK, results)
}
