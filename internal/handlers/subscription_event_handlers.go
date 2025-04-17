package handlers

import (
	"bytes"
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionEventHandler manages subscription event-related HTTP endpoints
type SubscriptionEventHandler struct {
	common *CommonServices
}

// NewSubscriptionEventHandler creates a new subscription event handler with the required dependencies
func NewSubscriptionEventHandler(common *CommonServices) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		common: common,
	}
}

// CreateEventRequest represents the request body for creating a subscription event
type CreateEventRequest struct {
	SubscriptionID  string          `json:"subscription_id" binding:"required"`
	EventType       string          `json:"event_type" binding:"required"`
	TransactionHash string          `json:"transaction_hash"`
	AmountInCents   int32           `json:"amount_in_cents"`
	OccurredAt      int64           `json:"occurred_at"`
	ErrorMessage    string          `json:"error_message"`
	Metadata        json.RawMessage `json:"metadata"`
}

// GetSubscriptionEvent godoc
// @Summary Get a subscription event by ID
// @Description Retrieves a subscription event by its ID
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {object} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/{event_id} [get]
func (h *SubscriptionEventHandler) GetSubscriptionEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	parsedUUID, err := uuid.Parse(eventID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid event ID format", err)
		return
	}

	event, err := h.common.db.GetSubscriptionEvent(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Subscription event not found")
		return
	}

	sendSuccess(c, http.StatusOK, event)
}

// GetSubscriptionEventByTxHash godoc
// @Summary Get a subscription event by transaction hash
// @Description Retrieves a subscription event by its transaction hash
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param tx_hash path string true "Transaction Hash"
// @Success 200 {object} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/transaction/{tx_hash} [get]
func (h *SubscriptionEventHandler) GetSubscriptionEventByTxHash(c *gin.Context) {
	txHash := c.Param("tx_hash")
	if txHash == "" {
		sendError(c, http.StatusBadRequest, "Transaction hash is required", nil)
		return
	}

	txHashPg := pgtype.Text{
		String: txHash,
		Valid:  true,
	}

	event, err := h.common.db.GetSubscriptionEventByTransactionHash(c.Request.Context(), txHashPg)
	if err != nil {
		handleDBError(c, err, "Subscription event not found")
		return
	}

	sendSuccess(c, http.StatusOK, event)
}

// ListSubscriptionEvents godoc
// @Summary List all subscription events
// @Description Get a list of all subscription events
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {array} db.SubscriptionEventDetails
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/transactions [get]
func (h *SubscriptionEventHandler) ListSubscriptionEvents(c *gin.Context) {
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

	params := db.ListSubscriptionEventDetailsWithPaginationParams{
		Limit:       int32(limit),
		Offset:      int32(offset),
		WorkspaceID: parsedWorkspaceID,
	}

	events, err := h.common.db.ListSubscriptionEventDetailsWithPagination(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscription events", err)
		return
	}

	// Get the total count for pagination metadata
	totalCount, err := h.common.db.CountSubscriptionEventDetails(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count subscription events", err)
		return
	}

	sendPaginatedSuccess(c, http.StatusOK, events, int(page), int(limit), int(totalCount))
}

// ListSubscriptionEventsBySubscription godoc
// @Summary List events for a subscription
// @Description Get a list of all events for a specific subscription
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {array} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/events [get]
func (h *SubscriptionEventHandler) ListSubscriptionEventsBySubscription(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	events, err := h.common.db.ListSubscriptionEventsBySubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscription events", err)
		return
	}

	sendSuccess(c, http.StatusOK, events)
}

// ListSubscriptionEventsByType godoc
// @Summary List events by type
// @Description Get a list of all events of a specific type
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param event_type path string true "Event Type"
// @Success 200 {array} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/type/{event_type} [get]
func (h *SubscriptionEventHandler) ListSubscriptionEventsByType(c *gin.Context) {
	eventTypeStr := c.Param("event_type")

	// Validate event type
	var eventType db.SubscriptionEventType
	switch eventTypeStr {
	case string(db.SubscriptionEventTypeCreated), string(db.SubscriptionEventTypeRedeemed), string(db.SubscriptionEventTypeFailed), string(db.SubscriptionEventTypeCanceled):
		eventType = db.SubscriptionEventType(eventTypeStr)
	default:
		sendError(c, http.StatusBadRequest, "Invalid event type", nil)
		return
	}

	events, err := h.common.db.ListSubscriptionEventsByType(c.Request.Context(), eventType)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve events by type", err)
		return
	}

	sendSuccess(c, http.StatusOK, events)
}

// ListFailedSubscriptionEvents godoc
// @Summary List failed subscription events
// @Description Get a list of all failed subscription events
// @Tags subscription-events
// @Accept json
// @Produce json
// @Success 200 {array} db.SubscriptionEvent
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/failed [get]
func (h *SubscriptionEventHandler) ListFailedSubscriptionEvents(c *gin.Context) {
	events, err := h.common.db.ListFailedSubscriptionEvents(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve failed events", err)
		return
	}

	sendSuccess(c, http.StatusOK, events)
}

// ListRecentSubscriptionEvents godoc
// @Summary List recent subscription events
// @Description Get a list of events that occurred after a specified time
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param since query int false "Timestamp (Unix epoch in seconds) to filter from"
// @Success 200 {array} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/recent [get]
func (h *SubscriptionEventHandler) ListRecentSubscriptionEvents(c *gin.Context) {
	sinceStr := c.DefaultQuery("since", "")
	var since time.Time

	if sinceStr != "" {
		sinceTimestamp, err := strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid since timestamp", err)
			return
		}
		since = time.Unix(sinceTimestamp, 0)
	} else {
		// Default to events in the last 24 hours
		since = time.Now().Add(-24 * time.Hour)
	}

	sincePg := pgtype.Timestamptz{
		Time:  since,
		Valid: true,
	}

	events, err := h.common.db.ListRecentSubscriptionEvents(c.Request.Context(), sincePg)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve recent events", err)
		return
	}

	sendSuccess(c, http.StatusOK, events)
}

// CreateSubscriptionEvent godoc
// @Summary Create a new subscription event
// @Description Creates a new subscription event with the provided details
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param event body CreateEventRequest true "Event details"
// @Success 201 {object} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events [post]
func (h *SubscriptionEventHandler) CreateSubscriptionEvent(c *gin.Context) {
	var request CreateEventRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Parse subscription ID
	subscriptionID, err := uuid.Parse(request.SubscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID", err)
		return
	}

	// Validate event type
	var eventType db.SubscriptionEventType
	switch request.EventType {
	case string(db.SubscriptionEventTypeCreated), string(db.SubscriptionEventTypeRedeemed), string(db.SubscriptionEventTypeFailed), string(db.SubscriptionEventTypeCanceled):
		eventType = db.SubscriptionEventType(request.EventType)
	default:
		sendError(c, http.StatusBadRequest, "Invalid event type", nil)
		return
	}

	// Set occurred at time (use current time if not provided)
	var occurredAt time.Time
	if request.OccurredAt > 0 {
		occurredAt = time.Unix(request.OccurredAt, 0)
	} else {
		occurredAt = time.Now()
	}

	// Create database params
	params := db.CreateSubscriptionEventParams{
		SubscriptionID: subscriptionID,
		EventType:      eventType,
		TransactionHash: pgtype.Text{
			String: request.TransactionHash,
			Valid:  request.TransactionHash != "",
		},
		AmountInCents: request.AmountInCents,
		OccurredAt: pgtype.Timestamptz{
			Time:  occurredAt,
			Valid: true,
		},
		ErrorMessage: pgtype.Text{
			String: request.ErrorMessage,
			Valid:  request.ErrorMessage != "",
		},
		Metadata: request.Metadata,
	}

	event, err := h.common.db.CreateSubscriptionEvent(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create subscription event", err)
		return
	}

	sendSuccess(c, http.StatusCreated, event)
}

// UpdateSubscriptionEvent godoc
// @Summary Update an existing subscription event
// @Description Updates a subscription event with the provided details
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param event_id path string true "Event ID"
// @Param event body CreateEventRequest true "Updated event details"
// @Success 200 {object} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/{event_id} [put]
func (h *SubscriptionEventHandler) UpdateSubscriptionEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	parsedUUID, err := uuid.Parse(eventID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid event ID format", err)
		return
	}

	// Get existing event to use its values for fields not specified in the request
	existingEvent, err := h.common.db.GetSubscriptionEvent(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Subscription event not found")
		return
	}

	var request CreateEventRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Initialize update parameters with all existing values
	params := db.UpdateSubscriptionEventParams{
		ID:              parsedUUID,
		EventType:       existingEvent.EventType,
		TransactionHash: existingEvent.TransactionHash,
		AmountInCents:   existingEvent.AmountInCents,
		ErrorMessage:    existingEvent.ErrorMessage,
		Metadata:        existingEvent.Metadata,
	}

	// Update event type if provided and different
	if request.EventType != "" && string(existingEvent.EventType) != request.EventType {
		// Validate event type
		switch request.EventType {
		case string(db.SubscriptionEventTypeCreated), string(db.SubscriptionEventTypeRedeemed), string(db.SubscriptionEventTypeFailed), string(db.SubscriptionEventTypeCanceled):
			params.EventType = db.SubscriptionEventType(request.EventType)
		default:
			sendError(c, http.StatusBadRequest, "Invalid event type", nil)
			return
		}
	}

	// Update transaction hash if provided and different
	if request.TransactionHash != "" &&
		(!existingEvent.TransactionHash.Valid || existingEvent.TransactionHash.String != request.TransactionHash) {
		params.TransactionHash = pgtype.Text{
			String: request.TransactionHash,
			Valid:  true,
		}
	}

	// Update amount if provided and different
	if request.AmountInCents != existingEvent.AmountInCents {
		params.AmountInCents = request.AmountInCents
	}

	// Update error message if provided and different
	if request.ErrorMessage != "" &&
		(!existingEvent.ErrorMessage.Valid || existingEvent.ErrorMessage.String != request.ErrorMessage) {
		params.ErrorMessage = pgtype.Text{
			String: request.ErrorMessage,
			Valid:  true,
		}
	}

	// Update metadata if provided and different
	if request.Metadata != nil && !bytes.Equal(request.Metadata, existingEvent.Metadata) {
		params.Metadata = request.Metadata
	}

	updatedEvent, err := h.common.db.UpdateSubscriptionEvent(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update subscription event", err)
		return
	}

	sendSuccess(c, http.StatusOK, updatedEvent)
}

// GetTotalAmountBySubscription godoc
// @Summary Get total amount processed for a subscription
// @Description Retrieves the total amount processed for a subscription
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/total-amount [get]
func (h *SubscriptionEventHandler) GetTotalAmountBySubscription(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	totalAmount, err := h.common.db.GetTotalAmountBySubscription(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to calculate total amount", err)
		return
	}

	sendSuccess(c, http.StatusOK, gin.H{"total_amount": totalAmount})
}

// GetSuccessfulRedemptionCount godoc
// @Summary Get count of successful redemptions
// @Description Gets the count of successful redemptions for a subscription
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/redemption-count [get]
func (h *SubscriptionEventHandler) GetSuccessfulRedemptionCount(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	count, err := h.common.db.GetSuccessfulRedemptionCount(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count redemptions", err)
		return
	}

	sendSuccess(c, http.StatusOK, gin.H{"redemption_count": count})
}

// GetLatestSubscriptionEvent godoc
// @Summary Get latest event for a subscription
// @Description Retrieves the most recent event for a subscription
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} db.SubscriptionEvent
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/latest-event [get]
func (h *SubscriptionEventHandler) GetLatestSubscriptionEvent(c *gin.Context) {
	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	event, err := h.common.db.GetLatestSubscriptionEvent(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Latest subscription event not found")
		return
	}

	sendSuccess(c, http.StatusOK, event)
}
