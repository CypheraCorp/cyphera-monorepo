package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strings"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	typeParams "github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Use types from the centralized packages
type (
	PriceResponse   = responses.PriceResponse
	ProductResponse = responses.ProductResponse
)

// SubscriptionHandler manages subscription-related HTTP endpoints
type SubscriptionHandler struct {
	common               *CommonServices
	delegationClient     *dsClient.DelegationClient
	subscriptionService  interfaces.SubscriptionService
	paymentService       interfaces.PaymentService
	logger               *zap.Logger
	lastRedemptionTxHash string // Stores the transaction hash from the last successful redemption
}

// NewSubscriptionHandler creates a handler with interface dependencies
func NewSubscriptionHandler(
	common *CommonServices,
	delegationClient *dsClient.DelegationClient,
	subscriptionService interfaces.SubscriptionService,
	paymentService interfaces.PaymentService,
	logger *zap.Logger,
) *SubscriptionHandler {
	if logger == nil {
		logger = zap.L()
	}
	return &SubscriptionHandler{
		common:              common,
		delegationClient:    delegationClient,
		subscriptionService: subscriptionService,
		paymentService:      paymentService,
		logger:              logger,
	}
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
		h.common.HandleError(c, err, "Invalid workspace ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		h.common.HandleError(c, err, "Invalid subscription ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscription, err := h.common.db.GetSubscriptionWithWorkspace(c.Request.Context(), db.GetSubscriptionWithWorkspaceParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Subscription not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get subscription", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	c.JSON(http.StatusOK, subscription)
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
		h.common.HandleError(c, err, "Invalid workspace ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Get pagination parameters
	params, err := helpers.ParsePaginationParams(c)
	if err != nil {
		h.common.HandleError(c, err, err.Error(), http.StatusBadRequest, h.common.GetLogger())
		return
	}
	limit, page := params.Limit, params.Page

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	dbParams := db.ListSubscriptionDetailsWithPaginationParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	}

	subscriptions, err := h.common.db.ListSubscriptionDetailsWithPagination(c.Request.Context(), dbParams)
	if err != nil {
		h.common.HandleError(c, err, "Failed to retrieve subscriptions", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	// Get the total count for pagination metadata
	totalCount, err := h.common.db.CountSubscriptions(c.Request.Context())
	if err != nil {
		h.common.HandleError(c, err, "Failed to count subscriptions", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	subscriptionResponses := make([]responses.SubscriptionResponse, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subscription, err := toSubscriptionResponse(sub)
		if err != nil {
			h.common.HandleError(c, err, "Failed to convert subscription to response", http.StatusInternalServerError, h.common.GetLogger())
			return
		}
		subscriptionResponses = append(subscriptionResponses, subscription)
	}

	hasMore := (int(totalCount)+int(limit)-1)/int(limit) > int(page)
	response := PaginatedResponse{
		Data:    subscriptionResponses,
		Object:  "list",
		HasMore: hasMore,
		Pagination: Pagination{
			CurrentPage: int(page),
			PerPage:     int(limit),
			TotalItems:  int(totalCount),
			TotalPages:  (int(totalCount) + int(limit) - 1) / int(limit),
		},
	}
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
		h.common.HandleError(c, err, "Invalid workspace ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	customerID := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerID)
	if err != nil {
		h.common.HandleError(c, err, "Invalid customer ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByCustomer(c.Request.Context(), db.ListSubscriptionsByCustomerParams{
		CustomerID:  parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		h.common.HandleError(c, err, "Failed to retrieve customer subscriptions", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	subscriptionResponses := make([]responses.SubscriptionResponse, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subscription, err := toSubscriptionResponseFromDBSubscription(sub)
		if err != nil {
			h.common.HandleError(c, err, "Failed to convert subscription to response", http.StatusInternalServerError, h.common.GetLogger())
			return
		}
		subscriptionResponses = append(subscriptionResponses, subscription)
	}
	c.JSON(http.StatusOK, subscriptionResponses)
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
		h.common.HandleError(c, err, "Invalid workspace ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}
	productID := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productID)
	if err != nil {
		h.common.HandleError(c, err, "Invalid product ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscriptions, err := h.common.db.ListSubscriptionsByProduct(c.Request.Context(), db.ListSubscriptionsByProductParams{
		ProductID:   parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		h.common.HandleError(c, err, "Failed to retrieve product subscriptions", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusOK, subscriptions)
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
		h.common.HandleError(c, err, "Invalid subscription ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Check if subscription exists
	existingSubscription, err := h.common.db.GetSubscription(ctx, parsedSubscriptionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Subscription not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get subscription", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	var request params.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.common.HandleError(c, err, "Invalid request format", http.StatusBadRequest, h.common.GetLogger())
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
			h.common.HandleError(c, err, "Invalid customer ID format", http.StatusBadRequest, h.common.GetLogger())
			return
		}
		params.CustomerID = parsedCustomerID
	}

	if request.ProductID != "" {
		parsedProductID, err := uuid.Parse(request.ProductID)
		if err != nil {
			h.common.HandleError(c, err, "Invalid product ID format", http.StatusBadRequest, h.common.GetLogger())
			return
		}
		params.ProductID = parsedProductID
	}

	if request.ProductTokenID != "" {
		parsedProductTokenID, err := uuid.Parse(request.ProductTokenID)
		if err != nil {
			h.common.HandleError(c, err, "Invalid product token ID format", http.StatusBadRequest, h.common.GetLogger())
			return
		}
		params.ProductTokenID = parsedProductTokenID
	}

	if request.DelegationID != "" {
		parsedDelegationID, err := uuid.Parse(request.DelegationID)
		if err != nil {
			h.common.HandleError(c, err, "Invalid delegation ID format", http.StatusBadRequest, h.common.GetLogger())
			return
		}
		params.DelegationID = parsedDelegationID
	}

	if request.CustomerWalletID != "" {
		parsedCustomerWalletID, err := uuid.Parse(request.CustomerWalletID)
		if err != nil {
			h.common.HandleError(c, err, "Invalid customer wallet ID format", http.StatusBadRequest, h.common.GetLogger())
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
			h.common.HandleError(c, nil, "Invalid status value", http.StatusBadRequest, h.common.GetLogger())
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
		h.common.HandleError(c, err, "Failed to update subscription", http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusOK, subscription)
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
		h.common.HandleError(c, err, "Invalid workspace ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		h.common.HandleError(c, err, "Invalid subscription ID format", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscription, err := h.common.db.GetSubscriptionWithWorkspace(c.Request.Context(), db.GetSubscriptionWithWorkspaceParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Subscription not found", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to get subscription", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	if subscription.WorkspaceID != parsedWorkspaceID {
		h.common.HandleError(c, nil, "Subscription does not belong to this workspace", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	if subscription.Status != db.SubscriptionStatusCanceled && subscription.Status != db.SubscriptionStatusExpired {
		h.common.HandleError(c, nil, "Subscription is not canceled or expired", http.StatusBadRequest, h.common.GetLogger())
		return
	}

	err = h.common.db.DeleteSubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.common.HandleError(c, err, "Failed to delete subscription", http.StatusNotFound, h.common.GetLogger())
		} else {
			h.common.HandleError(c, err, "Failed to delete subscription", http.StatusInternalServerError, h.common.GetLogger())
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription successfully deleted"})
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
	queries        db.Querier                               // Database queries interface (could be transaction or regular)
	tx             pgx.Tx                                   // Optional transaction for atomic operations
	results        *responses.ProcessDueSubscriptionsResult // Optional results tracker for ProcessDueSubscriptions
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

	logger.Info("processSubscription: Re-fetching subscription for idempotency check within transaction",
		zap.String("subscription_id", originalSubscription.ID.String()))
	currentDBSub, err := params.queries.GetSubscription(params.ctx, originalSubscription.ID)
	if err != nil {
		errMsg := fmt.Sprintf("processSubscription: Failed to re-fetch subscription %s for idempotency check: %v", originalSubscription.ID, err)
		logger.Error(errMsg)
		if params.results != nil {
			params.results.FailedCount++ // Count as failed for this run if fetch fails
		}
		return result, errors.New(errMsg)
	}
	logger.Info("processSubscription: Fetched current state",
		zap.String("subscription_id", currentDBSub.ID.String()),
		zap.String("status", string(currentDBSub.Status)),
		zap.Time("next_redemption_date", currentDBSub.NextRedemptionDate.Time),
		zap.Int32("total_redemptions", currentDBSub.TotalRedemptions))

	// IDEMPOTENCY CHECKS:
	// 1. Check for Terminal Status (Completed or Failed by a previous run)
	if currentDBSub.Status == db.SubscriptionStatusCompleted {
		logger.Info("processSubscription: Subscription already marked as COMPLETED. Skipping this run's counters",
			zap.String("subscription_id", currentDBSub.ID.String()))
		// Do not increment Succeeded/Failed for *this* run if already completed
		result.isProcessed = true // Considered processed (by a previous run)
		result.isCompleted = true
		return result, nil
	}
	if currentDBSub.Status == db.SubscriptionStatusFailed {
		logger.Info("processSubscription: Subscription already marked as FAILED. Skipping, but counting as Failed for this run",
			zap.String("subscription_id", currentDBSub.ID.String()))
		if params.results != nil {
			params.results.FailedCount++ // Count as Failed for this run as we found it failed
		}
		return result, nil
	}

	// 2. Check if Next Redemption Date has already been advanced past the current processing time
	if currentDBSub.NextRedemptionDate.Valid && currentDBSub.NextRedemptionDate.Time.After(params.now) {
		logger.Info("processSubscription: Subscription NextRedemptionDate is already past current processing time. Likely processed by concurrent/retried run. Skipping, counting as Succeeded",
			zap.String("subscription_id", currentDBSub.ID.String()),
			zap.Time("next_redemption_date", currentDBSub.NextRedemptionDate.Time),
			zap.Time("current_processing_time", params.now))
		if params.results != nil {
			params.results.SuccessfulCount++ // Count as Succeeded as it appears processed by another run
		}
		result.isProcessed = true
		return result, nil
	}

	// 3. Safeguard: Ensure the subscription fetched for processing is still in a processable state
	if currentDBSub.Status != db.SubscriptionStatusActive && currentDBSub.Status != db.SubscriptionStatusOverdue {
		logger.Info("processSubscription: Subscription is no longer in a processable status. Skipping this run's counters",
			zap.String("subscription_id", currentDBSub.ID.String()),
			zap.String("current_status", string(currentDBSub.Status)))
		// Do not increment Succeeded/Failed for this run, it's just a state mismatch found
		return result, nil
	}

	logger.Info("processSubscription: Idempotency checks passed. Proceeding with redemption logic",
		zap.String("subscription_id", currentDBSub.ID.String()))

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

	logger.Info("processSubscription: Final payment determination",
		zap.String("subscription_id", currentDBSub.ID.String()),
		zap.Int32("term_length", params.Price.TermLength),
		zap.Int32("current_total_redemptions", currentDBSub.TotalRedemptions),
		zap.Bool("will_be_final", isActualFinalPayment))

	// Marshal delegation to JSON bytes for redemption
	delegationBytes, err := json.Marshal(params.delegationData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal delegation data for subscription %s: %v",
			originalSubscription.ID, err)
		logger.Error(errMsg)

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
				logger.Info("Failed to record failure event for subscription",
					zap.String("subscription_id", originalSubscription.ID.String()),
					zap.Error(eventErr))
			}
			// Update counters if tracking results
			if params.results != nil {
				params.results.FailedCount++ // Increment Failed here
			}
		} else {
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: originalSubscription.ID,
				AmountInCents:  params.Price.UnitAmountInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				logger.Info("Failed to record failure event for subscription",
					zap.String("subscription_id", originalSubscription.ID.String()),
					zap.Error(eventErr))
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
		jitterFactor, _ := rand.Int(rand.Reader, big.NewInt(40))
		jitter := time.Duration(float64(backoff) * (0.8 + float64(jitterFactor.Int64())/100.0))

		if attempt > 0 {
			logger.Info("Retrying delegation redemption for subscription",
				zap.String("subscription_id", originalSubscription.ID.String()),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.Duration("backoff", jitter)) // Use originalSubscription.ID for consistent logging
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
			logger.Info("Permanent error redeeming delegation, won't retry",
				zap.String("subscription_id", originalSubscription.ID.String()),
				zap.String("error", detailedErr))
			break // Break loop for permanent error
		} else if strings.Contains(detailedErr, "AA25 invalid account nonce") {
			logger.Info("Temporary error (Nonce Collision AA25) redeeming delegation. Will retry if attempts remain",
				zap.String("subscription_id", originalSubscription.ID.String()),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.String("error", detailedErr))
			// Continue loop
		} else {
			// Other temporary error
			logger.Info("Temporary error redeeming delegation. Will retry if attempts remain",
				zap.String("subscription_id", originalSubscription.ID.String()),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.String("error", detailedErr))
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
			logger.Error(errMsg)
			// Record failure event and update results before returning
			if params.tx != nil && params.results != nil {
				params.results.FailedCount++
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
			logger.Error(errMsg)
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
					logger.Info("Failed to record DB update failure event",
						zap.String("subscription_id", originalSubscription.ID.String()),
						zap.Error(eventErr))
				}
				if params.results != nil {
					params.results.FailedCount++
				}
			} else { // Non-transactional path might have different event recording
				_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
					SubscriptionID: originalSubscription.ID,
					AmountInCents:  params.Price.UnitAmountInPennies,
					ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
					Metadata:       nil,
				})
				if eventErr != nil {
					logger.Info("Failed to record DB update failure event",
						zap.String("subscription_id", originalSubscription.ID.String()),
						zap.Error(eventErr))
				}
			}
			result.isProcessed = false
			return result, fmt.Errorf("failed to update subscription after redemption: %w", err)
		}

		// Both redemption and DB update succeeded
		result.isProcessed = true
		if params.results != nil {
			params.results.SuccessfulCount++ // Increment Succeeded here
		}

		// Use the ACTUAL final payment determination, not the potentially stale one from params
		if isActualFinalPayment {
			// Call CompleteSubscription which sets status and nullifies next_redemption_date
			if _, updateErr := params.queries.CompleteSubscription(params.ctx, originalSubscription.ID); updateErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to mark subscription %s as completed after successful final payment: %v", originalSubscription.ID, updateErr)
				logger.Info(errMsg)
				// This is critical for final payments - we need to ensure completion status is set
				result.isProcessed = false
				if params.results != nil {
					params.results.FailedCount++
				}
				return result, fmt.Errorf(errMsg)
			} else {
				result.isCompleted = true // Internal flag for event type
				logger.Info("Marked subscription as completed (via CompleteSubscription) after successful final payment",
					zap.String("subscription_id", originalSubscription.ID.String()),
					zap.Int32("total_redemptions", currentDBSub.TotalRedemptions+1))
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
			redeemedEvent, eventErr := params.queries.CreateSubscriptionEvent(params.ctx, db.CreateSubscriptionEventParams{
				SubscriptionID:  originalSubscription.ID,
				EventType:       db.SubscriptionEventTypeRedeemed,
				TransactionHash: pgtype.Text{String: result.txHash, Valid: true},
				AmountInCents:   params.Price.UnitAmountInPennies,
				OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
				Metadata:        redemptionMetadataBytes,
			})
			if eventErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to record redeemed event for subscription %s after successful on-chain redemption. Transaction will be rolled back: %v", originalSubscription.ID, eventErr)
				logger.Info(errMsg)
				result.isProcessed = false
				if params.results != nil {
					params.results.FailedCount++
				}
				return result, fmt.Errorf(errMsg)
			}

			// Validate that the redeemed event was properly created
			if validationErr := h.logAndValidateEventCreation(params.ctx, params.queries, originalSubscription.ID, db.SubscriptionEventTypeRedeemed, result.txHash); validationErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Redeemed event creation validation failed for subscription %s: %v", originalSubscription.ID, validationErr)
				logger.Info(errMsg)
				result.isProcessed = false
				if params.results != nil {
					params.results.FailedCount++
				}
				return result, fmt.Errorf(errMsg)
			}

			// Create payment record for the redeemed event
			payment, paymentErr := h.paymentService.CreatePaymentFromSubscriptionEvent(
				params.ctx,
				typeParams.CreatePaymentFromSubscriptionEventParams{
					SubscriptionEvent: &redeemedEvent,
					Subscription:      &db.Subscription{ID: originalSubscription.ID, WorkspaceID: params.product.WorkspaceID, CustomerID: originalSubscription.CustomerID},
					Product:           &params.product,
					Price:             &params.Price,
					Customer:          &db.Customer{ID: originalSubscription.CustomerID},
					TransactionHash:   result.txHash,
					NetworkID:         params.productToken.NetworkID,
					TokenID:           params.productToken.TokenID,
				},
			)
			if paymentErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to create payment record for subscription %s after successful redemption. Transaction will be rolled back: %v", originalSubscription.ID, paymentErr)
				logger.Info(errMsg)
				result.isProcessed = false
				if params.results != nil {
					params.results.FailedCount++
				}
				return result, fmt.Errorf(errMsg)
			}
			logger.Info("Created payment record for subscription redemption",
				zap.String("payment_id", payment.ID.String()),
				zap.String("subscription_id", originalSubscription.ID.String()))

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
					logger.Info(errMsg)
					result.isProcessed = false
					if params.results != nil {
						params.results.FailedCount++
					}
					return result, fmt.Errorf(errMsg)
				}

				logger.Info("Created both redeemed and completed events for final payment",
					zap.String("subscription_id", originalSubscription.ID.String()))
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
				logger.Info(errMsg)
				result.isProcessed = false
				return result, fmt.Errorf(errMsg)
			}

			// Validate that the redeemed event was properly created
			if validationErr := h.logAndValidateEventCreation(params.ctx, params.queries, originalSubscription.ID, db.SubscriptionEventTypeRedeemed, result.txHash); validationErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Non-tx redeemed event creation validation failed for subscription %s: %v", originalSubscription.ID, validationErr)
				logger.Info(errMsg)
				result.isProcessed = false
				return result, fmt.Errorf(errMsg)
			}

			// Fetch the created event to get its ID for payment creation
			redeemedEvents, evtErr := params.queries.GetLatestSubscriptionEventByType(params.ctx, db.GetLatestSubscriptionEventByTypeParams{
				SubscriptionID: originalSubscription.ID,
				EventType:      db.SubscriptionEventTypeRedeemed,
			})
			if evtErr != nil || len(redeemedEvents) == 0 {
				errMsg := fmt.Sprintf("CRITICAL: Failed to fetch redeemed event for payment creation for subscription %s: %v", originalSubscription.ID, evtErr)
				logger.Info(errMsg)
				result.isProcessed = false
				return result, fmt.Errorf(errMsg)
			}

			// Create payment record for the redeemed event
			payment, paymentErr := h.paymentService.CreatePaymentFromSubscriptionEvent(
				params.ctx,
				typeParams.CreatePaymentFromSubscriptionEventParams{
					SubscriptionEvent: &redeemedEvents[0],
					Subscription:      &db.Subscription{ID: originalSubscription.ID, WorkspaceID: params.product.WorkspaceID, CustomerID: originalSubscription.CustomerID},
					Product:           &params.product,
					Price:             &params.Price,
					Customer:          &db.Customer{ID: originalSubscription.CustomerID},
					TransactionHash:   result.txHash,
					NetworkID:         params.productToken.NetworkID,
					TokenID:           params.productToken.TokenID,
				},
			)
			if paymentErr != nil {
				errMsg := fmt.Sprintf("CRITICAL: Failed to create payment record for subscription %s after successful redemption (non-tx): %v", originalSubscription.ID, paymentErr)
				logger.Info(errMsg)
				result.isProcessed = false
				return result, fmt.Errorf(errMsg)
			}
			logger.Info("Created payment record for subscription redemption (non-tx path)",
				zap.String("payment_id", payment.ID.String()),
				zap.String("subscription_id", originalSubscription.ID.String()))

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
					logger.Info(errMsg)
					result.isProcessed = false
					return result, fmt.Errorf(errMsg)
				}

				logger.Info("Created both redeemed and completed events for final payment (non-tx path)",
					zap.String("subscription_id", originalSubscription.ID.String()))
			}
		}

		logger.Info("Successfully processed subscription, next redemption scheduled",
			zap.String("subscription_id", originalSubscription.ID.String()),
			zap.Time("next_redemption", nextRedemptionDate.Time),
			zap.Bool("is_completed", result.isCompleted))

	} else {
		// Redemption failed (exhausted retries or hit permanent error)
		errMsg := fmt.Sprintf("Failed to redeem delegation for subscription %s after %d attempts: %v",
			originalSubscription.ID, maxRetries, redemptionError)
		logger.Error(errMsg)
		if params.results != nil {
			params.results.FailedCount++ // Increment Failed here
		}

		// If this was the final payment and redemption failed, update status
		var updateErr error
		if isActualFinalPayment {
			updateParams := db.UpdateSubscriptionStatusParams{
				ID:     originalSubscription.ID,
				Status: db.SubscriptionStatusOverdue, // Or Failed, depending on logic
			}
			if _, updateErr = params.queries.UpdateSubscriptionStatus(params.ctx, updateParams); updateErr != nil {
				logger.Info("Failed to update subscription to overdue/failed status after redemption failure",
					zap.String("subscription_id", originalSubscription.ID.String()),
					zap.Error(updateErr))
			} else {
				logger.Info("Marked subscription as failed due to final payment redemption failure",
					zap.String("subscription_id", originalSubscription.ID.String()),
					zap.String("status", string(updateParams.Status)))
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
				logger.Info("Failed to record failure event for subscription", zap.String("subscription_id", originalSubscription.ID.String()), zap.Error(eventErr))
			}
		} else { // Non-transactional path
			_, eventErr := params.queries.CreateFailedRedemptionEvent(params.ctx, db.CreateFailedRedemptionEventParams{
				SubscriptionID: originalSubscription.ID,
				AmountInCents:  params.Price.UnitAmountInPennies,
				ErrorMessage:   pgtype.Text{String: errMsg, Valid: true},
				Metadata:       nil,
			})
			if eventErr != nil {
				logger.Info("Failed to record non-tx failure event for subscription",
					zap.String("subscription_id", originalSubscription.ID.String()),
					zap.Error(eventErr))
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
func (h *SubscriptionHandler) ProcessDueSubscriptions(ctx context.Context) (responses.ProcessDueSubscriptionsResult, error) {
	logger.Info("Entering ProcessDueSubscriptions")
	results := responses.ProcessDueSubscriptionsResult{}
	now := time.Now()

	// Get database pool
	pool, err := h.common.GetDBPool()
	if err != nil {
		logger.Info("Error in ProcessDueSubscriptions: failed to get database pool",
			zap.Error(err))
		return results, fmt.Errorf("failed to get database pool: %w", err)
	}

	var processingTx pgx.Tx // Need this outside the transaction func for processSubscription

	// Execute within transaction
	err = helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		processingTx = tx // Store tx for use in processSubscription
		qtx := h.common.WithTx(tx)

		// Query for subscriptions due for redemption and lock them for processing
		nowPgType := pgtype.Timestamptz{Time: now, Valid: true}
		logger.Info("ProcessDueSubscriptions: Querying for subscriptions due before",
			zap.Time("before", now))
		subscriptions, err := qtx.ListSubscriptionsDueForRedemption(ctx, nowPgType)
		if err != nil {
			logger.Info("Error in ProcessDueSubscriptions: failed to fetch subscriptions due for redemption",
				zap.Error(err))
			return fmt.Errorf("failed to fetch subscriptions due for redemption: %w", err)
		}

		// Update result count
		results.ProcessedCount = len(subscriptions)
		if results.ProcessedCount == 0 {
			logger.Info("ProcessDueSubscriptions: No subscriptions found due for renewal.")
			// No subscriptions to process, transaction will commit automatically
			logger.Info("Exiting ProcessDueSubscriptions. Total: 0, Succeeded: 0, Failed: 0")
			return nil
		}

		logger.Info("Found subscriptions due for redemption",
			zap.Int("count", results.ProcessedCount))

		// Process each subscription within the transaction
		for i, subscription := range subscriptions {
			logger.Info("Processing subscription",
				zap.Int("current", i+1),
				zap.Int("total", results.ProcessedCount),
				zap.String("subscription_id", subscription.ID.String()),
				zap.String("status", string(subscription.Status)),
				zap.String("product_id", subscription.ProductID.String()),
				zap.Time("current_period_end", subscription.CurrentPeriodEnd.Time))

			// Skip subscriptions that are not in a processable state (active or overdue)
			if !(subscription.Status == db.SubscriptionStatusActive || subscription.Status == db.SubscriptionStatusOverdue) {
				logger.Info("Skipping subscription with non-processable status",
					zap.String("subscription_id", subscription.ID.String()),
					zap.String("status", string(subscription.Status)))
				continue
			}

			// Get required data for processing
			product, err := qtx.GetProductWithoutWorkspaceId(ctx, subscription.ProductID)
			if err != nil {
				logger.Info("Failed to get product for subscription",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				results.FailedCount++
				continue
			}

			productToken, err := qtx.GetProductToken(ctx, subscription.ProductTokenID)
			if err != nil {
				logger.Info("Failed to get product token for subscription",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				results.FailedCount++
				continue
			}

			token, err := qtx.GetToken(ctx, productToken.TokenID)
			if err != nil {
				logger.Info("Failed to get token for subscription",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				results.FailedCount++
				continue
			}

			network, err := qtx.GetNetwork(ctx, token.NetworkID)
			if err != nil {
				logger.Info("Failed to get network for token",
					zap.String("token_id", token.ID.String()),
					zap.Error(err))
				results.FailedCount++
				continue
			}

			merchantWallet, err := qtx.GetWalletByID(ctx, db.GetWalletByIDParams{
				ID:          product.WalletID,
				WorkspaceID: product.WorkspaceID,
			})
			if err != nil {
				logger.Info("Failed to get merchant wallet for subscription",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				results.FailedCount++
				continue
			}

			delegationData, err := qtx.GetDelegationData(ctx, subscription.DelegationID)
			if err != nil {
				logger.Info("Failed to get delegation data for subscription",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				results.FailedCount++
				continue
			}

			// Fetch the Price for the current subscription to determine term length and other price details
			price, err := qtx.GetPrice(ctx, subscription.PriceID)
			if err != nil {
				logger.Info("Failed to get price for subscription",
					zap.String("price_id", subscription.PriceID.String()),
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				results.FailedCount++
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

			logger.Info("Calling h.processSubscription for subscription",
				zap.String("subscription_id", subscription.ID.String()))
			_, err = h.processSubscription(params)
			if err != nil {
				// Error handling is done in processSubscription, which updates results.FailedCount (potentially)
				// and logs the specific error. We just log that we are continuing based on the error return.
				logger.Info("Error returned by h.processSubscription. Continuing",
					zap.String("subscription_id", subscription.ID.String()),
					zap.Error(err))
				// Note: The result counters (Failed/Completed/Succeeded) are primarily managed within processSubscription
				// based on its internal logic and idempotency checks.
				continue
			}
			// If processSubscription didn't return an error, we rely on the counters updated within it.
			// No need for explicit Succeeded++ here anymore, it's handled based on idempotency/completion inside.
			logger.Info("Finished h.processSubscription call (Results updated internally)",
				zap.String("subscription_id", subscription.ID.String()))

		}

		logger.Info("ProcessDueSubscriptions transaction scope complete",
			zap.Int("succeeded", results.SuccessfulCount),
			zap.Int("failed", results.FailedCount))
		return nil
	})

	if err != nil {
		logger.Info("Error in ProcessDueSubscriptions: transaction failed",
			zap.Error(err))
		return results, err
	}

	logger.Info("Exiting ProcessDueSubscriptions",
		zap.Int("total_found", results.ProcessedCount),
		zap.Int("succeeded", results.SuccessfulCount),
		zap.Int("failed", results.FailedCount))
	return results, nil
}

// toSubscriptionResponse converts a db.ListSubscriptionDetailsWithPaginationRow to a SubscriptionResponse.
func toSubscriptionResponse(subDetails db.ListSubscriptionDetailsWithPaginationRow) (responses.SubscriptionResponse, error) {
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

	resp := responses.SubscriptionResponse{
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
			UnitAmountInPennies: int64(subDetails.PriceUnitAmountInPennies), // Convert int32 to int64
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
		ProductToken: responses.ProductTokenResponse{
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
func toSubscriptionResponseFromDBSubscription(sub db.Subscription) (responses.SubscriptionResponse, error) {
	resp := responses.SubscriptionResponse{
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

	logger.Info("Event creation validated",
		zap.String("subscription_id", subscriptionID.String()),
		zap.String("event_type", string(eventType)),
		zap.String("tx_hash", txHash),
		zap.String("event_id", latestEvent.ID.String()))

	return nil
}
