package handlers

import (
	// "context" // Commented out: unused after commenting out logAndValidateEventCreation
	"errors"
	// "fmt" // Commented out: unused after commenting out logAndValidateEventCreation
	"net/http"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"

	// "github.com/cyphera/cyphera-api/libs/go/logger" // Commented out: unused after commenting out logAndValidateEventCreation
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
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
	common              *CommonServices
	delegationClient    *dsClient.DelegationClient
	subscriptionService interfaces.SubscriptionService
	paymentService      interfaces.PaymentService
	logger              *zap.Logger
	// lastRedemptionTxHash string // Commented out: unused field - Stores the transaction hash from the last successful redemption
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

// Commented out: unused function
/*
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
*/
