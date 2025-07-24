package handlers

import (
	"context"
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// SubscriptionHandler manages subscription-related HTTP endpoints
type SubscriptionHandler struct {
	common               *CommonServices
	delegationClient     *dsClient.DelegationClient
	lastRedemptionTxHash string // Stores the transaction hash from the last successful redemption
}

// SubscriptionExistsError is a custom error for when a subscription already exists
type SubscriptionExistsError struct {
	Subscription *db.Subscription
}

func (e *SubscriptionExistsError) Error() string {
	return fmt.Sprintf("subscription already exists with ID: %s", e.Subscription.ID)
}

// NewSubscriptionHandler creates a new subscription handler with the required dependencies
func NewSubscriptionHandler(common *CommonServices, delegationClient *dsClient.DelegationClient) *SubscriptionHandler {
	return &SubscriptionHandler{
		common:           common,
		delegationClient: delegationClient,
	}
}

// SubscribeRequest represents the request body for subscribing to a product
type SubscribeRequest struct {
	SubscriberAddress string           `json:"subscriber_address" binding:"required"`
	PriceID           string           `json:"price_id" binding:"required"`
	ProductTokenID    string           `json:"product_token_id" binding:"required"`
	TokenAmount       string           `json:"token_amount" binding:"required"`
	Delegation        DelegationStruct `json:"delegation" binding:"required"`
}

// SubscriptionResponse represents a subscription along with its associated price and product details.
type SubscriptionResponse struct {
	ID                     uuid.UUID              `json:"id"`
	WorkspaceID            uuid.UUID              `json:"workspace_id"`
	CustomerID             uuid.UUID              `json:"customer_id,omitempty"`
	CustomerName           string                 `json:"customer_name,omitempty"`
	CustomerEmail          string                 `json:"customer_email,omitempty"`
	Status                 string                 `json:"status"`
	CurrentPeriodStart     time.Time              `json:"current_period_start"`
	CurrentPeriodEnd       time.Time              `json:"current_period_end"`
	NextRedemptionDate     *time.Time             `json:"next_redemption_date,omitempty"`
	TotalRedemptions       int32                  `json:"total_redemptions"`
	TotalAmountInCents     int32                  `json:"total_amount_in_cents"`
	TokenAmount            int32                  `json:"token_amount"`
	DelegationID           uuid.UUID              `json:"delegation_id"`
	CustomerWalletID       *uuid.UUID             `json:"customer_wallet_id,omitempty"`
	ExternalID             string                 `json:"external_id,omitempty"`
	PaymentSyncStatus      string                 `json:"payment_sync_status,omitempty"`
	PaymentSyncedAt        *time.Time             `json:"payment_synced_at,omitempty"`
	PaymentSyncVersion     int32                  `json:"payment_sync_version,omitempty"`
	PaymentProvider        string                 `json:"payment_provider,omitempty"`
	InitialTransactionHash string                 `json:"initial_transaction_hash,omitempty"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
	Price                  PriceResponse          `json:"price"`
	Product                ProductResponse        `json:"product"`
	ProductToken           ProductTokenResponse   `json:"product_token"`
}

type CreateSubscriptionParams struct {
	Customer       db.Customer
	CustomerWallet db.CustomerWallet
	WorkspaceID    uuid.UUID
	ProductID      uuid.UUID
	ProductTokenID uuid.UUID
	Price          db.Price
	TokenAmount    int64
	DelegationData db.DelegationDatum
	PeriodStart    time.Time
	PeriodEnd      time.Time
	NextRedemption time.Time
}

// GetSubscription godoc
// @Summary Get a subscription by ID
// @Description Retrieves a subscription by its ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} SubscriptionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id} [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	subscription, err := h.common.db.GetSubscriptionWithWorkspace(c.Request.Context(), db.GetSubscriptionWithWorkspaceParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	sendSuccess(c, http.StatusOK, subscription)
}

// ListSubscriptions godoc
// @Summary List all subscriptions
// @Description Get a list of all non-deleted subscriptions
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} PaginatedResponse{data=[]SubscriptionResponse}
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

	subscriptionResponses := make([]SubscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		subscription, err := toSubscriptionResponse(sub)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to convert subscription to response", err)
			return
		}
		subscriptionResponses[i] = subscription
	}

	response := sendPaginatedSuccess(c, http.StatusOK, subscriptionResponses, int(page), int(limit), int(totalCount))
	c.JSON(http.StatusOK, response)
}

// ListSubscriptionsByCustomer godoc
// @Summary List subscriptions by customer
// @Description Get a list of all subscriptions for a specific customer
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Success 200 {object} PaginatedResponse{data=[]SubscriptionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{customer_id}/subscriptions [get]
func (h *SubscriptionHandler) ListSubscriptionsByCustomer(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	customerID := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByCustomer(c.Request.Context(), db.ListSubscriptionsByCustomerParams{
		CustomerID:  parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve customer subscriptions", err)
		return
	}

	subscriptionResponses := make([]SubscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		subscription, err := toSubscriptionResponseFromDBSubscription(sub)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to convert subscription to response", err)
			return
		}
		subscriptionResponses[i] = subscription
	}
	sendSuccess(c, http.StatusOK, subscriptionResponses)
}

// ListSubscriptionsByProduct godoc
// @Summary List subscriptions by product
// @Description Get a list of all subscriptions for a specific product
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} PaginatedResponse{data=[]SubscriptionResponse}
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
// @Summary Update a subscription
// @Description Updates an existing subscription with the specified details
// @Tags subscriptions
// @Accept json
// @Produce json
// @Tags exclude
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
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	subscription, err := h.common.db.GetSubscriptionWithWorkspace(c.Request.Context(), db.GetSubscriptionWithWorkspaceParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	if subscription.WorkspaceID != parsedWorkspaceID {
		sendError(c, http.StatusBadRequest, "Subscription does not belong to this workspace", nil)
		return
	}

	if subscription.Status != db.SubscriptionStatusCanceled && subscription.Status != db.SubscriptionStatusExpired {
		sendError(c, http.StatusBadRequest, "Subscription is not canceled or expired", nil)
		return
	}

	err = h.common.db.DeleteSubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete subscription")
		return
	}

	sendSuccessMessage(c, http.StatusOK, "Subscription successfully deleted")
}

// CalculateNextRedemption computes the next scheduled redemption date based on interval type.
// This is the central function for all interval calculations in the system.
// For testing and development purposes, 1min and 5mins intervals are supported.
func CalculateNextRedemption(intervalType string, currentTime time.Time) time.Time {
	switch intervalType {
	case "1min":
		return currentTime.Add(1 * time.Minute)
	case "5mins":
		return currentTime.Add(5 * time.Minute)
	case "daily":
		return currentTime.AddDate(0, 0, 1) // Next day
	case "week":
		return currentTime.AddDate(0, 0, 7) // Next week
	case "month":
		return currentTime.AddDate(0, 1, 0) // Next month
	case "year":
		return currentTime.AddDate(1, 0, 0) // Next year
	default:
		return currentTime.AddDate(0, 1, 0) // Default to monthly
	}
}

// CalculatePeriodEnd determines the end date of a subscription period based on interval type and term length.
// This function is used when creating or updating subscription periods.
func CalculatePeriodEnd(start time.Time, intervalType string, termLength int32) time.Time {
	switch intervalType {
	case "1min":
		return start.Add(time.Duration(termLength) * time.Minute)
	case "5mins":
		return start.Add(time.Duration(termLength*5) * time.Minute)
	case "daily":
		return start.AddDate(0, 0, int(termLength))
	case "week":
		return start.AddDate(0, 0, int(termLength*7))
	case "month":
		return start.AddDate(0, int(termLength), 0)
	case "year":
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
}

// processSubscriptionParams contains all parameters needed to process a subscription
type processSubscriptionParams struct {
	ctx            context.Context
	subscription   db.ListSubscriptionsDueForRedemptionRow
	product        db.Product
	Price          db.Price
	productToken   db.GetProductTokenRow
	delegationData db.DelegationDatum
	merchantWallet db.Wallet
	token          db.Token
	network        db.Network
	now            time.Time
	queries        db.Querier                     // Database queries interface (could be transaction or regular)
	tx             pgx.Tx                         // Optional transaction for atomic operations
	results        *ProcessDueSubscriptionsResult // Optional results tracker for ProcessDueSubscriptions
}

// processSubscriptionResult contains the result of processing a subscription
type processSubscriptionResult struct {
	isProcessed bool   // Successfully processed (payment redeemed)
	isCompleted bool   // Subscription was completed (final payment processed) - internal flag, not in final results struct
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
			params.results.Failed++ // Count as failed for this run if fetch fails
		}
		return result, errors.New(errMsg)
	}
	log.Printf("processSubscription: Fetched current state for %s: Status=%s, NextRedemptionDate=%v, TotalRedemptions=%d",
		currentDBSub.ID, currentDBSub.Status, currentDBSub.NextRedemptionDate.Time, currentDBSub.TotalRedemptions)

	// IDEMPOTENCY CHECKS:
	// 1. Check for Terminal Status (Completed or Failed by a previous run)
	if currentDBSub.Status == db.SubscriptionStatusCompleted {
		log.Printf("processSubscription: Subscription %s already marked as COMPLETED. Skipping this run's counters.", currentDBSub.ID)
		// Do not increment Succeeded/Failed for *this* run if already completed
		result.isProcessed = true // Considered processed (by a previous run)
		result.isCompleted = true
		return result, nil
	}
	if currentDBSub.Status == db.SubscriptionStatusFailed {
		log.Printf("processSubscription: Subscription %s already marked as FAILED. Skipping, but counting as Failed for this run.", currentDBSub.ID)
		if params.results != nil {
			params.results.Failed++ // Count as Failed for this run as we found it failed
		}
		return result, nil
	}

	// 2. Check if Next Redemption Date has already been advanced past the current processing time
	if currentDBSub.NextRedemptionDate.Valid && currentDBSub.NextRedemptionDate.Time.After(params.now) {
		log.Printf("processSubscription: Subscription %s NextRedemptionDate (%v) is already past current processing time (%v). Likely processed by a concurrent/retried run. Skipping, counting as Succeeded for this run.",
			currentDBSub.ID, currentDBSub.NextRedemptionDate.Time, params.now)
		if params.results != nil {
			params.results.Succeeded++ // Count as Succeeded as it appears processed by another run
		}
		result.isProcessed = true
		return result, nil
	}

	// 3. Safeguard: Ensure the subscription fetched for processing is still in a processable state
	if currentDBSub.Status != db.SubscriptionStatusActive && currentDBSub.Status != db.SubscriptionStatusOverdue {
		log.Printf("processSubscription: Subscription %s is no longer in a processable status (current: %s). Skipping this run's counters.", currentDBSub.ID, currentDBSub.Status)
		// Do not increment Succeeded/Failed for this run, it's just a state mismatch found
		return result, nil
	}

	log.Printf("processSubscription: Idempotency checks passed for subscription %s. Proceeding with redemption logic.", currentDBSub.ID)

	// IMPORTANT FIX: Determine if this will be the final payment using the CURRENT subscription state
	// instead of the potentially stale data passed in params.isFinalPayment
	isActualFinalPayment := false
	if params.Price.Type == db.PriceTypeRecurring {
		// Check if the current redemption will fulfill the term length using CURRENT TotalRedemptions
		if params.Price.TermLength > 0 && (currentDBSub.TotalRedemptions+1) >= params.Price.TermLength {
			isActualFinalPayment = true
		}
	} else if params.Price.Type == db.PriceTypeOneOff {
		// For one-off, it's complete if this payment makes total_redemptions reach 1
		if (currentDBSub.TotalRedemptions + 1) >= 1 {
			isActualFinalPayment = true
		}
	}

	log.Printf("processSubscription: Final payment determination for %s: TermLength=%d, CurrentTotalRedemptions=%d, WillBeFinal=%t",
		currentDBSub.ID, params.Price.TermLength, currentDBSub.TotalRedemptions, isActualFinalPayment)

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
				AmountInCents:  params.Price.UnitAmountInPennies,
				OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					originalSubscription.ID, eventErr)
			}
			// Update counters if tracking results
			if params.results != nil {
				params.results.Failed++ // Increment Failed here
			}
		} else {
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: originalSubscription.ID,
				AmountInCents:  params.Price.UnitAmountInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v",
					originalSubscription.ID, eventErr)
			}
			// Non-transactional path might not update results struct directly
		}
		result.isProcessed = false
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
			TokenAmount:          int64(params.subscription.TokenAmount),
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
		// Successfully redeemed delegation, now update the database

		// Update next redemption date based on product interval
		var nextRedemptionDate pgtype.Timestamptz
		if params.Price.IntervalType == "" {
			errMsg := fmt.Sprintf("IntervalType is null for recurring price %s (subscription %s)", params.Price.ID, originalSubscription.ID)
			log.Println(errMsg)
			// Record failure event and update results before returning
			if params.tx != nil && params.results != nil {
				params.results.Failed++
				// Optionally record event here too
			}
			return result, errors.New(errMsg) // isProcessed remains false
		}
		nextDate := CalculateNextRedemption(string(params.Price.IntervalType), params.now)
		nextRedemptionDate = pgtype.Timestamptz{Time: nextDate, Valid: true}

		// Prepare update parameters for incrementing subscription
		incrementParams := db.IncrementSubscriptionRedemptionParams{
			ID:                 originalSubscription.ID,
			TotalAmountInCents: params.Price.UnitAmountInPennies,
			NextRedemptionDate: nextRedemptionDate,
		}

		// Update the subscription with new redemption data
		_, err := params.queries.IncrementSubscriptionRedemption(params.ctx, incrementParams)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to update subscription %s after successful redemption: %v", originalSubscription.ID, err)
			log.Println(errMsg)
			// Even though redemption worked, the overall process failed if DB update fails.
			if params.tx != nil { // Record failure event if in transaction
				_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
					SubscriptionID: originalSubscription.ID,
					EventType:      db.SubscriptionEventTypeFailedRedemption,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					AmountInCents:  params.Price.UnitAmountInPennies,
					OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					log.Printf("Failed to record DB update failure event for %s: %v", originalSubscription.ID, eventErr)
				}
				if params.results != nil {
					params.results.Failed++
				}
			} else { // Non-transactional path might have different event recording
				_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
					SubscriptionID: originalSubscription.ID,
					AmountInCents:  params.Price.UnitAmountInPennies,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					log.Printf("Failed to record DB update failure event for %s: %v", originalSubscription.ID, eventErr)
				}
			}
			result.isProcessed = false
			return result, fmt.Errorf("failed to update subscription after redemption: %w", err)
		}

		// Both redemption and DB update succeeded
		result.isProcessed = true
		if params.results != nil {
			params.results.Succeeded++ // Increment Succeeded here
		}

		// Use the ACTUAL final payment determination, not the potentially stale one from params
		if isActualFinalPayment {
			// Call CompleteSubscription which sets status and nullifies next_redemption_date
			if _, updateErr := params.queries.CompleteSubscription(params.ctx, originalSubscription.ID); updateErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to mark subscription %s as completed after successful final payment: %v", originalSubscription.ID, updateErr)
				log.Printf(errMsg)
				// This is critical for final payments - we need to ensure completion status is set
				result.isProcessed = false
				if params.results != nil {
					params.results.Failed++
				}
				return result, fmt.Errorf(errMsg)
			} else {
				result.isCompleted = true // Internal flag for event type
				log.Printf("Marked subscription %s as completed (via CompleteSubscription) after successful final payment. TotalRedemptions will be %d", originalSubscription.ID, currentDBSub.TotalRedemptions+1)
			}
		}

		// Record successful event(s)
		// ALWAYS create a "redeemed" event for the successful redemption with transaction hash
		redemptionMetadataBytes, _ := json.Marshal(map[string]interface{}{
			"next_redemption":         nextRedemptionDate.Time,
			"total_redemptions_after": currentDBSub.TotalRedemptions + 1,
			"term_length":             params.Price.TermLength,
			"is_final_payment":        isActualFinalPayment,
		})

		if params.tx != nil { // Transactional path
			// 1. Create the "redeemed" event (always)
			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID:  originalSubscription.ID,
				EventType:       db.SubscriptionEventTypeRedeemed,
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.Price.UnitAmountInPennies,
				OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:        redemptionMetadataBytes,
			})
			if eventErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to record redeemed event for subscription %s after successful on-chain redemption. Transaction will be rolled back: %v", originalSubscription.ID, eventErr)
				log.Printf(errMsg)
				result.isProcessed = false
				if params.results != nil {
					params.results.Failed++
				}
				return result, fmt.Errorf(errMsg)
			}

			// Validate that the redeemed event was properly created
			if validationErr := h.logAndValidateEventCreation(params.ctx, params.queries, originalSubscription.ID, db.SubscriptionEventTypeRedeemed, result.txHash); validationErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Redeemed event creation validation failed for subscription %s: %v", originalSubscription.ID, validationErr)
				log.Printf(errMsg)
				result.isProcessed = false
				if params.results != nil {
					params.results.Failed++
				}
				return result, fmt.Errorf(errMsg)
			}

			// 2. If this is the final payment, ALSO create a "completed" event (separate from redemption)
			if result.isCompleted {
				completionMetadataBytes, _ := json.Marshal(map[string]interface{}{
					"final_total_redemptions": currentDBSub.TotalRedemptions + 1,
					"term_length":             params.Price.TermLength,
					"subscription_completed":  time.Now(),
				})

				_, completionEventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
					SubscriptionID:  originalSubscription.ID,
					EventType:       db.SubscriptionEventTypeCompleted,
					TransactionHash: pgtype.Text{Valid: false}, // No transaction hash for completion event
					AmountInCents:   0,                         // Completion event doesn't represent a payment
					OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
					Metadata:        completionMetadataBytes,
				})
				if completionEventErr != nil {
					errMsg := fmt.Sprintf("CRITICAL: Failed to record completion event for subscription %s after successful final payment. Transaction will be rolled back: %v", originalSubscription.ID, completionEventErr)
					log.Printf(errMsg)
					result.isProcessed = false
					if params.results != nil {
						params.results.Failed++
					}
					return result, fmt.Errorf(errMsg)
				}

				log.Printf("Created both redeemed and completed events for final payment of subscription %s", originalSubscription.ID)
			}
		} else { // Non-transactional path
			// 1. Create the "redeemed" event (always)
			_, eventErr := params.queries.CreateRedemptionEvent(params.ctx, db.CreateRedemptionEventParams{
				SubscriptionID:  originalSubscription.ID,
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.Price.UnitAmountInPennies,
				Metadata:        redemptionMetadataBytes,
			})
			if eventErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to record non-tx redeemed event for subscription %s after successful on-chain redemption: %v", originalSubscription.ID, eventErr)
				log.Printf(errMsg)
				result.isProcessed = false
				return result, fmt.Errorf(errMsg)
			}

			// Validate that the redeemed event was properly created
			if validationErr := h.logAndValidateEventCreation(params.ctx, params.queries, originalSubscription.ID, db.SubscriptionEventTypeRedeemed, result.txHash); validationErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Non-tx redeemed event creation validation failed for subscription %s: %v", originalSubscription.ID, validationErr)
				log.Printf(errMsg)
				result.isProcessed = false
				return result, fmt.Errorf(errMsg)
			}

			// 2. If this is the final payment, ALSO create a "completed" event (separate from redemption)
			if result.isCompleted {
				completionMetadataBytes, _ := json.Marshal(map[string]interface{}{
					"final_total_redemptions": currentDBSub.TotalRedemptions + 1,
					"term_length":             params.Price.TermLength,
					"subscription_completed":  time.Now(),
				})

				// For non-transactional path, we need to use CreateSubscriptionEvent manually since CreateRedemptionEvent hardcodes "redeemed" type
				_, completionEventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
					SubscriptionID:  originalSubscription.ID,
					EventType:       db.SubscriptionEventTypeCompleted,
					TransactionHash: pgtype.Text{Valid: false}, // No transaction hash for completion event
					AmountInCents:   0,                         // Completion event doesn't represent a payment
					OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
					Metadata:        completionMetadataBytes,
				})
				if completionEventErr != nil {
					errMsg := fmt.Sprintf("CRITICAL: Failed to record non-tx completion event for subscription %s after successful final payment: %v", originalSubscription.ID, completionEventErr)
					log.Printf(errMsg)
					result.isProcessed = false
					return result, fmt.Errorf(errMsg)
				}

				log.Printf("Created both redeemed and completed events for final payment of subscription %s (non-tx path)", originalSubscription.ID)
			}
		}

		log.Printf("Successfully processed subscription %s, next redemption at %s, isCompleted: %t", originalSubscription.ID, nextRedemptionDate.Time, result.isCompleted)

	} else {
		// Redemption failed (exhausted retries or hit permanent error)
		errMsg := fmt.Sprintf("Failed to redeem delegation for subscription %s after %d attempts: %v",
			originalSubscription.ID, maxRetries, redemptionError)
		log.Println(errMsg)
		if params.results != nil {
			params.results.Failed++ // Increment Failed here
		}

		// If this was the final payment and redemption failed, update status
		var updateErr error
		if isActualFinalPayment {
			updateParams := db.UpdateSubscriptionStatusParams{
				ID:     originalSubscription.ID,
				Status: db.SubscriptionStatusOverdue, // Or Failed, depending on logic
			}
			if _, updateErr = params.queries.UpdateSubscriptionStatus(params.ctx, updateParams); updateErr != nil {
				log.Printf("Failed to update subscription %s to overdue/failed status after redemption failure: %v", originalSubscription.ID, updateErr)
			} else {
				log.Printf("Marked subscription %s as %s due to failed final payment redemption", originalSubscription.ID, updateParams.Status)
			}
		}

		// Create failure event
		if params.tx != nil { // Transactional path
			_, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID: originalSubscription.ID,
				EventType:      db.SubscriptionEventTypeFailedRedemption,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				AmountInCents:  params.Price.UnitAmountInPennies,
				OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record failure event for subscription %s: %v", originalSubscription.ID, eventErr)
			}
		} else { // Non-transactional path
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: originalSubscription.ID,
				AmountInCents:  params.Price.UnitAmountInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				log.Printf("Failed to record non-tx failure event for subscription %s: %v", originalSubscription.ID, eventErr)
			}
		}

		result.isProcessed = false
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

// ProcessDueSubscriptions finds and processes all subscriptions that are due for redemption
// It uses a transaction for atomicity and updates subscription status based on the result
func (h *SubscriptionHandler) ProcessDueSubscriptions(ctx context.Context) (ProcessDueSubscriptionsResult, error) {
	log.Printf("Entering ProcessDueSubscriptions")
	results := ProcessDueSubscriptionsResult{}
	now := time.Now()

	// Get database pool
	pool, err := h.common.GetDBPool()
	if err != nil {
		log.Printf("Error in ProcessDueSubscriptions: failed to get database pool: %v", err)
		return results, fmt.Errorf("failed to get database pool: %w", err)
	}

	var processingTx pgx.Tx // Need this outside the transaction func for processSubscription

	// Execute within transaction
	err = helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		processingTx = tx // Store tx for use in processSubscription
		qtx := h.common.WithTx(tx)

		// Query for subscriptions due for redemption and lock them for processing
		nowPgType := pgtype.Timestamptz{Time: now, Valid: true}
		log.Printf("ProcessDueSubscriptions: Querying for subscriptions due before %v", now)
		subscriptions, err := qtx.ListSubscriptionsDueForRedemption(ctx, nowPgType)
		if err != nil {
			log.Printf("Error in ProcessDueSubscriptions: failed to fetch subscriptions due for redemption: %v", err)
			return fmt.Errorf("failed to fetch subscriptions due for redemption: %w", err)
		}

		// Update result count
		results.Total = len(subscriptions)
		if results.Total == 0 {
			log.Printf("ProcessDueSubscriptions: No subscriptions found due for renewal.")
			// No subscriptions to process, transaction will commit automatically
			log.Printf("Exiting ProcessDueSubscriptions. Total: 0, Succeeded: 0, Failed: 0")
			return nil
		}

		log.Printf("Found %d subscriptions due for redemption", results.Total)

		// Process each subscription within the transaction
		for i, subscription := range subscriptions {
		log.Printf("Processing subscription %d/%d: ID: %s, Status: %s, ProductID: %s, CurrentPeriodEnd: %v",
			i+1, results.Total, subscription.ID, subscription.Status, subscription.ProductID, subscription.CurrentPeriodEnd.Time)

		// Skip subscriptions that are not in a processable state (active or overdue)
		if !(subscription.Status == db.SubscriptionStatusActive || subscription.Status == db.SubscriptionStatusOverdue) {
			log.Printf("Skipping subscription %s with non-processable status %s", subscription.ID, subscription.Status)
			continue
		}

		// Get required data for processing
		product, err := qtx.GetProductWithoutWorkspaceId(ctx, subscription.ProductID)
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

		network, err := qtx.GetNetwork(ctx, token.NetworkID)
		if err != nil {
			log.Printf("Failed to get network for token %s: %v", token.ID, err)
			results.Failed++
			continue
		}

		merchantWallet, err := qtx.GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          product.WalletID,
			WorkspaceID: product.WorkspaceID,
		})
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

		// Fetch the Price for the current subscription to determine term length and other price details
		price, err := qtx.GetPrice(ctx, subscription.PriceID)
		if err != nil {
			log.Printf("Failed to get price %s for subscription %s: %v", subscription.PriceID, subscription.ID, err)
			results.Failed++
			continue
		}

		// Process the subscription
		params := processSubscriptionParams{
			ctx:            ctx,
			subscription:   subscription,
			product:        product,
			Price:          price,
			productToken:   productToken,
			network:        network,
			delegationData: delegationData,
			merchantWallet: merchantWallet,
			token:          token,
			now:            now,
			queries:        qtx,
			tx:             processingTx,
			results:        &results,
		}

		log.Printf("Calling h.processSubscription for subscription ID: %s", subscription.ID)
		_, err = h.processSubscription(params)
		if err != nil {
			// Error handling is done in processSubscription, which updates results.Failed (potentially)
			// and logs the specific error. We just log that we are continuing based on the error return.
			log.Printf("Error returned by h.processSubscription for subscription ID %s: %v. Continuing.", subscription.ID, err)
			// Note: The result counters (Failed/Completed/Succeeded) are primarily managed within processSubscription
			// based on its internal logic and idempotency checks.
			continue
		}
		// If processSubscription didn't return an error, we rely on the counters updated within it.
		// No need for explicit Succeeded++ here anymore, it's handled based on idempotency/completion inside.
		log.Printf("Finished h.processSubscription call for subscription ID: %s (Results updated internally).", subscription.ID)

		}

		log.Printf("ProcessDueSubscriptions transaction scope complete. Results - Succeeded: %d, Failed: %d",
			results.Succeeded, results.Failed)
		return nil
	})

	if err != nil {
		log.Printf("Error in ProcessDueSubscriptions: transaction failed: %v", err)
		return results, err
	}

	log.Printf("Exiting ProcessDueSubscriptions. Total Found: %d, Succeeded: %d, Failed: %d",
		results.Total, results.Succeeded, results.Failed)
	return results, nil
}

// toSubscriptionResponse converts a db.ListSubscriptionDetailsWithPaginationRow to a SubscriptionResponse.
func toSubscriptionResponse(subDetails db.ListSubscriptionDetailsWithPaginationRow) (SubscriptionResponse, error) {
	// Helper to convert pgtype.Text to string for enums
	helperPgEnumTextToString := func(pgText pgtype.Text) string {
		if pgText.Valid {
			return pgText.String
		}
		return ""
	}

	// Prepare PriceResponse.CreatedAt and PriceResponse.UpdatedAt (assuming they are int64 in PriceResponse)
	var priceCreatedAtUnix int64
	if subDetails.PriceCreatedAt.Valid {
		priceCreatedAtUnix = subDetails.PriceCreatedAt.Time.Unix()
	}
	var priceUpdatedAtUnix int64
	if subDetails.PriceUpdatedAt.Valid {
		priceUpdatedAtUnix = subDetails.PriceUpdatedAt.Time.Unix()
	}

	// Prepare nullable fields for PriceResponse
	var intervalTypeStr string
	if subDetails.PriceIntervalType != "" {
		intervalTypeStr = string(subDetails.PriceIntervalType)
	}

	resp := SubscriptionResponse{
		ID:                 subDetails.SubscriptionID,
		WorkspaceID:        subDetails.ProductWorkspaceID,
		Status:             string(subDetails.SubscriptionStatus),
		CurrentPeriodStart: subDetails.SubscriptionCurrentPeriodStart.Time,
		CurrentPeriodEnd:   subDetails.SubscriptionCurrentPeriodEnd.Time,
		CreatedAt:          subDetails.SubscriptionCreatedAt.Time,
		UpdatedAt:          subDetails.SubscriptionUpdatedAt.Time,
		CustomerID:         subDetails.CustomerID,
		TokenAmount:        subDetails.SubscriptionTokenAmount,
		CustomerName:       subDetails.CustomerName.String,
		CustomerEmail:      subDetails.CustomerEmail.String,
		Price: PriceResponse{
			ID:                  subDetails.PriceID.String(),
			Object:              "price", // Typically static for PriceResponse
			ProductID:           subDetails.PriceProductID.String(),
			Active:              subDetails.PriceActive,
			Type:                string(subDetails.PriceType),
			Nickname:            helperPgEnumTextToString(subDetails.PriceNickname),
			Currency:            string(subDetails.PriceCurrency),
			UnitAmountInPennies: subDetails.PriceUnitAmountInPennies,
			IntervalType:        intervalTypeStr,
			TermLength:          subDetails.PriceTermLength,
			Metadata:            subDetails.PriceMetadata, // Expecting json.RawMessage
			CreatedAt:           priceCreatedAtUnix,
			UpdatedAt:           priceUpdatedAtUnix,
		},
		Product: ProductResponse{
			ID:          subDetails.ProductID.String(),
			Name:        subDetails.ProductName,
			Description: subDetails.ProductDescription.String,
			ImageURL:    subDetails.ProductImageUrl.String,
			Active:      subDetails.ProductActive,
			Metadata:    subDetails.ProductMetadata, // Expecting json.RawMessage
		},
		ProductToken: ProductTokenResponse{
			ID:          subDetails.ProductTokenID.String(),
			TokenID:     subDetails.ProductTokenTokenID.String(),
			TokenSymbol: subDetails.TokenSymbol,
			NetworkID:   subDetails.ProductTokenNetworkID.String(),
			CreatedAt:   subDetails.ProductTokenCreatedAt.Time.Unix(),
			UpdatedAt:   subDetails.ProductTokenUpdatedAt.Time.Unix(),
		},
	}
	return resp, nil
}

// toSubscriptionResponseFromDBSubscription converts a db.Subscription to a SubscriptionResponse.
func toSubscriptionResponseFromDBSubscription(sub db.Subscription) (SubscriptionResponse, error) {
	resp := SubscriptionResponse{
		ID:          sub.ID,
		CustomerID:  sub.CustomerID,
		WorkspaceID: sub.WorkspaceID,
		Status:      string(sub.Status),
		TokenAmount: sub.TokenAmount,
		CreatedAt:   sub.CreatedAt.Time,
		UpdatedAt:   sub.UpdatedAt.Time,
	}

	if sub.CurrentPeriodStart.Valid {
		resp.CurrentPeriodStart = sub.CurrentPeriodStart.Time
	}
	if sub.CurrentPeriodEnd.Valid {
		resp.CurrentPeriodEnd = sub.CurrentPeriodEnd.Time
	}

	return resp, nil
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

	// Get database pool
	pool, err := h.common.GetDBPool()
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get database pool", err)
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

		periodStart, periodEnd, nextRedemption := h.calculateSubscriptionPeriods(price)

		subscription, err = h.createSubscription(ctx, qtx, CreateSubscriptionParams{
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
		sendError(c, http.StatusInternalServerError, "Transaction failed", err)
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

		sendError(c, http.StatusInternalServerError, "Initial redemption failed, subscription marked as failed and soft-deleted", err)
		return
	}
	// Create comprehensive response with all subscription fields and initial transaction hash
	comprehensiveResponse, err := h.toComprehensiveSubscriptionResponse(ctx, updatedSubscription)
	if err != nil {
		logger.Error("Failed to create comprehensive subscription response",
			zap.Error(err),
			zap.String("subscription_id", updatedSubscription.ID.String()))
		sendError(c, http.StatusInternalServerError, "Failed to create subscription response", err)
		return
	}

	sendSuccess(c, http.StatusCreated, comprehensiveResponse)
}

// logAndValidateEventCreation helps debug event creation issues by logging details and optionally validating the event was created
func (h *SubscriptionHandler) logAndValidateEventCreation(ctx context.Context, queries db.Querier, subscriptionID uuid.UUID, eventType db.SubscriptionEventType, txHash string) error {
	// Get the latest event for this subscription to validate it was created properly
	latestEvent, err := queries.GetLatestSubscriptionEvent(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to retrieve latest event for validation: %w", err)
	}

	// Validate the event matches what we just tried to create
	if latestEvent.EventType != eventType {
		return fmt.Errorf("event type mismatch: expected %s, got %s", eventType, latestEvent.EventType)
	}

	if txHash != "" && (!latestEvent.TransactionHash.Valid || latestEvent.TransactionHash.String != txHash) {
		return fmt.Errorf("transaction hash mismatch: expected %s, got %v", txHash, latestEvent.TransactionHash)
	}

	log.Printf("Event creation validated: subscription %s, event_type %s, tx_hash %s, event_id %s",
		subscriptionID, eventType, txHash, latestEvent.ID)

	return nil
}
