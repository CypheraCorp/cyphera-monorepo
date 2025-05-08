package handlers

import (
	"context"
	dsClient "cyphera-api/internal/client/delegation_server"
	"cyphera-api/internal/db"
	"encoding/json"
	"errors"
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
	delegationClient     *dsClient.DelegationClient
	lastRedemptionTxHash string // Stores the transaction hash from the last successful redemption
}

// NewSubscriptionHandler creates a new subscription handler with the required dependencies
func NewSubscriptionHandler(common *CommonServices, delegationClient *dsClient.DelegationClient) *SubscriptionHandler {
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
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}
	productID := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByProduct(c.Request.Context(), db.ListSubscriptionsByProductParams{
		ProductID:   parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
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

// GetOverdueSubscriptions godoc
// @Summary Get all overdue subscriptions
// @Description Retrieves all subscriptions that have overdue but haven't been marked as overdue
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {array} db.Subscription
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/overdue [get]
func (h *SubscriptionHandler) GetOverdueSubscriptions(c *gin.Context) {
	subscriptions, err := h.common.db.GetOverdueSubscriptions(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve overdue subscriptions", err)
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
	network        db.Network
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
	originalSubscription := params.subscription // Keep a reference to the initially fetched one

	log.Printf("processSubscription: Re-fetching subscription %s for idempotency check within transaction.", originalSubscription.ID)
	currentDBSub, err := params.queries.GetSubscription(params.ctx, originalSubscription.ID)
	if err != nil {
		errMsg := fmt.Sprintf("processSubscription: Failed to re-fetch subscription %s for idempotency check: %v", originalSubscription.ID, err)
		log.Println(errMsg)
		if params.results != nil {
			params.results.Failed++
		}
		return result, errors.New(errMsg)
	}
	log.Printf("processSubscription: Fetched current state for %s: Status=%s, NextRedemptionDate=%v",
		currentDBSub.ID, currentDBSub.Status, currentDBSub.NextRedemptionDate.Time)

	// IDEMPOTENCY CHECKS:
	// 1. Check for Terminal Status (Completed or Failed by a previous run)
	if currentDBSub.Status == db.SubscriptionStatusCompleted {
		log.Printf("processSubscription: Subscription %s already marked as COMPLETED. Skipping.", currentDBSub.ID)
		if params.results != nil {
			params.results.Completed++
		}
		result.isProcessed = true
		result.isCompleted = true
		return result, nil
	}
	if currentDBSub.Status == db.SubscriptionStatusFailed {
		log.Printf("processSubscription: Subscription %s already marked as FAILED. Skipping.", currentDBSub.ID)
		if params.results != nil {
			params.results.Failed++
		}
		return result, nil
	}

	// 2. Check if Next Redemption Date has already been advanced past the current processing time
	if currentDBSub.NextRedemptionDate.Valid && currentDBSub.NextRedemptionDate.Time.After(params.now) {
		log.Printf("processSubscription: Subscription %s NextRedemptionDate (%v) is already past current processing time (%v). Likely processed by a concurrent/retried run. Skipping.",
			currentDBSub.ID, currentDBSub.NextRedemptionDate.Time, params.now)
		if params.results != nil {
			params.results.Succeeded++
		}
		result.isProcessed = true
		return result, nil
	}

	// 3. Safeguard: Ensure the subscription fetched for processing is still in a processable state
	if currentDBSub.Status != db.SubscriptionStatusActive && currentDBSub.Status != db.SubscriptionStatusOverdue {
		log.Printf("processSubscription: Subscription %s is no longer in a processable status (current: %s). Skipping.", currentDBSub.ID, currentDBSub.Status)
		return result, nil
	}

	log.Printf("processSubscription: Idempotency checks passed for subscription %s. Proceeding with redemption logic.", currentDBSub.ID)

	// NOTE: Use originalSubscription for data like ProductID, PriceInPennies for *this* processing attempt.
	// isFinalPayment should be based on originalSubscription.CurrentPeriodEnd
	isFinalPayment := originalSubscription.CurrentPeriodEnd.Time.Before(params.now)

	// Marshal delegation to JSON bytes for redemption
	delegationBytes, err := json.Marshal(params.delegationData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal delegation data for subscription %s: %v",
			originalSubscription.ID, err)
		log.Println(errMsg)

		// Create appropriate event based on transaction mode
		if params.tx != nil {
			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID: originalSubscription.ID,
				EventType:      db.SubscriptionEventTypeFailedRedemption,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				AmountInCents:  params.product.PriceInPennies,
				OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					originalSubscription.ID, eventErr)
			}
			// Update counters if tracking results
			if params.results != nil {
				params.results.Failed++
			}
		} else {
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: originalSubscription.ID,
				AmountInCents:  params.product.PriceInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					originalSubscription.ID, eventErr)
			}
		}
		result.isProcessed = false // Set isProcessed to false since increment failed
		return result, errors.New(errMsg)
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
				originalSubscription.ID, attempt+1, maxRetries, jitter) // Use originalSubscription.ID for consistent logging
			// Wait before retry
			time.Sleep(jitter)
		}

		// Attempt redemption
		// Get token info for redemption
		executionObject := dsClient.ExecutionObject{
			MerchantAddress:      params.merchantWallet.WalletAddress,
			TokenContractAddress: params.token.ContractAddress,
			TokenAmount:          params.subscription.TokenAmount.Int.Int64(),
			TokenDecimals:        params.token.Decimals,
			ChainID:              uint32(params.network.ChainID),
			NetworkName:          params.network.Name,
		}

		// Call the delegation client to redeem
		txHash, currentRedemptionErr := h.delegationClient.RedeemDelegationDirectly(
			params.ctx,
			delegationBytes,
			executionObject,
		)

		if currentRedemptionErr == nil {
			// Success!
			h.lastRedemptionTxHash = txHash
			redemptionSuccess = true
			result.txHash = txHash
			redemptionError = nil // Clear error
			break                 // Exit retry loop
		}

		// Error occurred: Store it, log it, and decide whether to break
		redemptionError = currentRedemptionErr
		detailedErr := redemptionError.Error()

		if isPermanentRedemptionError(redemptionError) {
			log.Printf("Permanent error redeeming delegation for subscription %s, won't retry: %v",
				originalSubscription.ID, detailedErr)
			break // Break loop for permanent error
		} else if strings.Contains(detailedErr, "AA25 invalid account nonce") {
			log.Printf("Temporary error (Nonce Collision AA25) redeeming delegation for subscription %s (attempt %d/%d): %v. Will retry if attempts remain.",
				originalSubscription.ID, attempt+1, maxRetries, detailedErr)
			// Continue loop
		} else {
			// Other temporary error
			log.Printf("Temporary error redeeming delegation for subscription %s (attempt %d/%d): %v. Will retry if attempts remain.",
				originalSubscription.ID, attempt+1, maxRetries, detailedErr)
			// Continue loop
		}
		// If not broken, the loop continues to the next attempt
	}

	// AFTER THE LOOP: Check redemptionSuccess and proceed...
	if redemptionSuccess {
		// Successfully redeemed delegation, but we still need to increment
		// result.isProcessed will be set to true after increment succeeds

		// Update next redemption date based on product interval
		var nextRedemptionDate pgtype.Timestamptz

		// Ensure IntervalType is valid before using it
		if !params.product.IntervalType.Valid {
			// Handle case where IntervalType is null for a recurring product (should not happen based on constraints)
			errMsg := fmt.Sprintf("IntervalType is null for recurring product %s (subscription %s)",
				params.product.ID, originalSubscription.ID) // Use originalSubscription.ID
			log.Println(errMsg)
			// Decide how to handle this - maybe default to monthly or fail?
			// For now, let's fail to highlight the issue.
			// You might choose to default: nextDate := CalculateNextRedemption(db.IntervalTypeMonth, params.now)
			return result, errors.New(errMsg)
		}

		// Calculate next redemption date using the product interval type
		nextDate := CalculateNextRedemption(params.product.IntervalType.IntervalType, params.now) // Access inner enum
		nextRedemptionDate = pgtype.Timestamptz{
			Time:  nextDate,
			Valid: true,
		}

		// Prepare update parameters for incrementing subscription
		incrementParams := db.IncrementSubscriptionRedemptionParams{
			ID:                 originalSubscription.ID, // Use originalSubscription.ID
			TotalAmountInCents: params.product.PriceInPennies,
			NextRedemptionDate: nextRedemptionDate,
		}

		// Update the subscription with new redemption data
		_, err := params.queries.IncrementSubscriptionRedemption(params.ctx, incrementParams)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to update subscription %s after successful redemption: %v",
				originalSubscription.ID, err) // Use originalSubscription.ID
			log.Println(errMsg)

			// Create failure event
			if params.tx != nil {
				_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
					SubscriptionID: originalSubscription.ID, // Use originalSubscription.ID
					EventType:      db.SubscriptionEventTypeFailedRedemption,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					AmountInCents:  params.product.PriceInPennies,
					OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					log.Printf("Failed to record failure event for subscription %s: %v",
						originalSubscription.ID, eventErr) // Use originalSubscription.ID
				}
				if params.results != nil {
					params.results.Failed++
				}
			} else {
				_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
					SubscriptionID: originalSubscription.ID, // Use originalSubscription.ID
					AmountInCents:  params.product.PriceInPennies,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					log.Printf("Failed to record failure event for subscription %s: %v",
						originalSubscription.ID, eventErr) // Use originalSubscription.ID
				}
			}
			result.isProcessed = false
			return result, fmt.Errorf("failed to update subscription: %w", err)
		}

		// Both redemption and increment succeeded
		result.isProcessed = true

		// If this was the final payment and it was successful, mark the subscription as completed
		// Use isFinalPayment calculated from originalSubscription's CurrentPeriodEnd
		if isFinalPayment {
			updateParams := db.UpdateSubscriptionStatusParams{
				ID:     originalSubscription.ID, // Use originalSubscription.ID
				Status: db.SubscriptionStatusCompleted,
			}
			if _, updateErr := params.queries.UpdateSubscriptionStatus(params.ctx, updateParams); updateErr != nil {
				log.Printf("Warning: Failed to mark subscription %s as completed: %v",
					originalSubscription.ID, updateErr) // Use originalSubscription.ID
			} else {
				result.isCompleted = true
				log.Printf("Marked subscription %s as completed after successful final payment", originalSubscription.ID) // Use originalSubscription.ID
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
				SubscriptionID:  originalSubscription.ID, // Use originalSubscription.ID
				EventType:       eventType,
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.product.PriceInPennies,
				OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:        metadataBytes,
			})
			if eventErr != nil {
				log.Printf("Warning: Failed to record success event for subscription %s: %v", originalSubscription.ID, eventErr) // Use originalSubscription.ID
				if params.results != nil {
					if result.isCompleted {
						params.results.Completed++
					} else {
						params.results.Succeeded++
					}
				}
			}
		} else {
			metadataBytes, _ := json.Marshal(map[string]interface{}{
				"next_redemption": nextRedemptionDate.Time,
				"is_final":        result.isCompleted,
			})

			_, eventErr := params.queries.CreateRedemptionEvent(params.ctx, db.CreateRedemptionEventParams{
				SubscriptionID:  originalSubscription.ID, // Use originalSubscription.ID
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.product.PriceInPennies,
				Metadata:        metadataBytes,
			})
			if eventErr != nil {
				log.Printf("Warning: Failed to record success event for subscription %s: %v",
					originalSubscription.ID, eventErr) // Use originalSubscription.ID
			}
		}

		log.Printf("Successfully processed subscription %s, next redemption at %s",
			originalSubscription.ID, nextRedemptionDate.Time) // Use originalSubscription.ID
	} else {
		// Redemption failed (either permanent error or exhausted retries for temporary errors)
		errMsg := fmt.Sprintf("Failed to redeem delegation for subscription %s after %d attempts: %v",
			originalSubscription.ID, maxRetries, redemptionError) // Use originalSubscription.ID and the last stored error
		log.Println(errMsg)

		// If this was the final payment and redemption failed, update status
		// Use isFinalPayment calculated from originalSubscription's CurrentPeriodEnd
		var updateErr error
		if isFinalPayment {
			// Update subscription to overdue status (or failed, depending on desired logic)
			updateParams := db.UpdateSubscriptionStatusParams{
				ID:     originalSubscription.ID,      // Use originalSubscription.ID
				Status: db.SubscriptionStatusOverdue, // Or db.SubscriptionStatusFailed
			}
			if _, updateErr = params.queries.UpdateSubscriptionStatus(params.ctx, updateParams); updateErr != nil {
				log.Printf("Failed to update subscription %s to overdue/failed status: %v",
					originalSubscription.ID, updateErr) // Use originalSubscription.ID
			} else {
				log.Printf("Marked subscription %s as %s due to failed final payment", originalSubscription.ID, updateParams.Status) // Use originalSubscription.ID
			}
		}
		// Create failure event
		if params.tx != nil {
			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID: originalSubscription.ID, // Use originalSubscription.ID
				EventType:      db.SubscriptionEventTypeFailedRedemption,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				AmountInCents:  params.product.PriceInPennies,
				OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					originalSubscription.ID, eventErr) // Use originalSubscription.ID
			}
		} else {
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: originalSubscription.ID, // Use originalSubscription.ID
				AmountInCents:  params.product.PriceInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					originalSubscription.ID, eventErr) // Use originalSubscription.ID
			}
		}

		result.isProcessed = false // Set isProcessed to false since redemption failed
		// Return the last error encountered during retries
		return result, fmt.Errorf("redemption failed after %d attempts: %w", maxRetries, redemptionError)
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

		// Skip subscriptions that are not active or overdue
		if subscription.Status != db.SubscriptionStatusActive && subscription.Status != db.SubscriptionStatusOverdue {
			log.Printf("Skipping non-active or overdue subscription %s with status %s", subscriptionID, subscription.Status)
			continue
		}

		// Get required data for processing
		product, err := h.common.db.GetProductWithoutWorkspaceId(ctx, subscription.ProductID)
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

		merchantWallet, err := h.common.db.GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          product.WalletID,
			WorkspaceID: product.WorkspaceID,
		})
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
	subscriptions, err := h.common.db.ListSubscriptionsDueForRedemption(ctx, now)
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
	log.Printf("Entering ProcessDueSubscriptions")
	results := ProcessDueSubscriptionsResult{}
	now := time.Now()

	// Start a transaction using the BeginTx helper
	tx, qtx, err := h.common.BeginTx(ctx)
	if err != nil {
		log.Printf("Error in ProcessDueSubscriptions: failed to begin transaction: %v", err)
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
	log.Printf("ProcessDueSubscriptions: Querying for subscriptions due before %v", now)
	subscriptions, err := qtx.ListSubscriptionsDueForRedemption(ctx, nowPgType)
	if err != nil {
		log.Printf("Error in ProcessDueSubscriptions: failed to fetch subscriptions due for redemption: %v", err)
		return results, fmt.Errorf("failed to fetch subscriptions due for redemption: %w", err)
	}

	// Update result count
	results.Total = len(subscriptions)
	if results.Total == 0 {
		log.Printf("ProcessDueSubscriptions: No subscriptions found due for renewal.")
		// No subscriptions to process, commit empty transaction
		if err := tx.Commit(ctx); err != nil {
			log.Printf("Error in ProcessDueSubscriptions: failed to commit empty transaction: %v", err)
			return results, fmt.Errorf("failed to commit empty transaction: %w", err)
		}
		tx = nil // Set to nil to avoid double rollback
		log.Printf("Exiting ProcessDueSubscriptions. Total: 0, Succeeded: 0, Failed: 0, Completed: 0")
		return results, nil
	}

	log.Printf("Found %d subscriptions due for redemption", results.Total)

	// Process each subscription within the transaction
	for i, subscription := range subscriptions {
		log.Printf("Processing subscription %d/%d: ID: %s, Status: %s, ProductID: %s, CurrentPeriodEnd: %v",
			i+1, results.Total, subscription.ID, subscription.Status, subscription.ProductID, subscription.CurrentPeriodEnd.Time)

		// Skip subscriptions that are not active
		if subscription.Status != db.SubscriptionStatusActive {
			log.Printf("Skipping non-active subscription %s with status %s", subscription.ID, subscription.Status)
			// results.Failed++ // Or some other counter for skipped if needed
			continue
		}

		// Get required data for processing
		log.Printf("Fetching product for subscription %s, ProductID: %s", subscription.ID, subscription.ProductID)
		product, err := qtx.GetProductWithoutWorkspaceId(ctx, subscription.ProductID)
		if err != nil {
			log.Printf("Failed to get product for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}
		log.Printf("Successfully fetched product %s for subscription %s", product.ID, subscription.ID)

		log.Printf("Fetching product token for subscription %s, ProductTokenID: %s", subscription.ID, subscription.ProductTokenID)
		productToken, err := qtx.GetProductToken(ctx, subscription.ProductTokenID)
		if err != nil {
			log.Printf("Failed to get product token for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}
		log.Printf("Successfully fetched product token %s for subscription %s", productToken.ID, subscription.ID)

		log.Printf("Fetching token for subscription %s, TokenID: %s", subscription.ID, productToken.TokenID)
		token, err := qtx.GetToken(ctx, productToken.TokenID)
		if err != nil {
			log.Printf("Failed to get token for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}
		log.Printf("Successfully fetched token %s (%s) for subscription %s", token.ID, token.Symbol, subscription.ID)

		log.Printf("Fetching network for token %s, NetworkID: %s", token.ID, token.NetworkID)
		network, err := qtx.GetNetwork(ctx, token.NetworkID)
		if err != nil {
			log.Printf("Failed to get network for token %s: %v", token.ID, err)
			results.Failed++
			continue
		}
		log.Printf("Successfully fetched network %s (ChainID: %d) for token %s", network.ID, network.ChainID, token.ID)

		log.Printf("Fetching merchant wallet for product %s, WalletID: %s, WorkspaceID: %s", product.ID, product.WalletID, product.WorkspaceID)
		merchantWallet, err := qtx.GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          product.WalletID,
			WorkspaceID: product.WorkspaceID,
		})
		if err != nil {
			log.Printf("Failed to get merchant wallet for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}
		log.Printf("Successfully fetched merchant wallet %s for product %s", merchantWallet.ID, product.ID)

		log.Printf("Fetching delegation data for subscription %s, DelegationID: %s", subscription.ID, subscription.DelegationID)
		delegationData, err := qtx.GetDelegationData(ctx, subscription.DelegationID)
		if err != nil {
			log.Printf("Failed to get delegation data for subscription %s: %v", subscription.ID, err)
			results.Failed++
			continue
		}
		log.Printf("Successfully fetched delegation data %s for subscription %s. Raw Delegation Data: %+v",
			delegationData.ID, subscription.ID, delegationData)

		// Check if this is the final payment
		isFinalPayment := subscription.CurrentPeriodEnd.Time.Before(now)
		log.Printf("Subscription %s: isFinalPayment determined to be %t (CurrentPeriodEnd: %v, Now: %v)",
			subscription.ID, isFinalPayment, subscription.CurrentPeriodEnd.Time, now)

		// Process the subscription
		params := processSubscriptionParams{
			ctx:            ctx,
			subscription:   subscription,
			product:        product,
			productToken:   productToken,
			network:        network,
			delegationData: delegationData,
			merchantWallet: merchantWallet,
			token:          token,
			isFinalPayment: isFinalPayment,
			now:            now,
			queries:        qtx,
			tx:             tx,
			results:        &results,
		}

		log.Printf("Calling h.processSubscription for subscription ID: %s", subscription.ID)
		_, err = h.processSubscription(params)
		if err != nil {
			// Error handling is done in processSubscription, which updates results.Failed
			// and logs the specific error. We just log that we are continuing.
			log.Printf("Error encountered by h.processSubscription for subscription ID %s (error logged within function): %v. Continuing.", subscription.ID, err)
			// The 'continue' is already part of the original logic if processSubscription returns an error.
			// If processSubscription itself handles logging and updates results.Failed,
			// then this log might be redundant or could be simplified.
			// For now, keeping it to show the flow.
			continue
		}
		log.Printf("Finished h.processSubscription call for subscription ID: %s. Current results - Succeeded: %d, Failed: %d, Completed: %d",
			subscription.ID, results.Succeeded, results.Failed, results.Completed)
	}

	log.Printf("Attempting to commit transaction. Current results - Succeeded: %d, Failed: %d, Completed: %d",
		results.Succeeded, results.Failed, results.Completed)
	// Commit the transaction if we got this far
	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error in ProcessDueSubscriptions: failed to commit transaction: %v", err)
		return results, fmt.Errorf("failed to commit transaction: %w", err)
	}
	tx = nil // Set to nil to avoid double rollback
	log.Printf("Transaction committed successfully.")

	log.Printf("Exiting ProcessDueSubscriptions. Total: %d, Succeeded: %d, Failed: %d, Completed: %d",
		results.Total, results.Succeeded, results.Failed, results.Completed)
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
