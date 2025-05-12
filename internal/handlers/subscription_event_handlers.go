package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"

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
// @Success 200 {object} PaginatedResponse{data=[]db.ListSubscriptionEventDetailsWithPaginationRow}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/transactions [get]
// @exclude
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
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	}

	events, err := h.common.db.ListSubscriptionEventDetailsWithPagination(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscription events", err)
		return
	}

	eventsResponse := make([]SubscriptionEventResponse, len(events))
	for i, event := range events {
		eventResponse, err := toSubscriptionEventResponse(c.Request.Context(), event)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve subscription event", err)
			return
		}
		eventsResponse[i] = eventResponse
	}

	// Get the total count for pagination metadata
	totalCount, err := h.common.db.CountSubscriptionEventDetails(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count subscription events", err)
		return
	}

	sendPaginatedSuccess(c, http.StatusOK, eventsResponse, int(page), int(limit), int(totalCount))
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
