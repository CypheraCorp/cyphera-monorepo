package handlers

import (
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionEventHandler manages subscription event-related HTTP endpoints
type SubscriptionEventHandler struct {
	common                   *CommonServices
	subscriptionEventService interfaces.SubscriptionEventService
}

// NewSubscriptionEventHandler creates a new subscription event handler with the required dependencies
func NewSubscriptionEventHandler(
	common *CommonServices,
	subscriptionEventService interfaces.SubscriptionEventService,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		common:                   common,
		subscriptionEventService: subscriptionEventService,
	}
}

// CreateEventRequest represents the request body for creating a subscription event
type CreateEventRequest struct {
	SubscriptionID  string `json:"subscription_id" binding:"required"`
	EventType       string `json:"event_type" binding:"required"`
	TransactionHash string `json:"transaction_hash"`
	AmountInCents   int32  `json:"amount_in_cents"`
	OccurredAt      int64  `json:"occurred_at"`
	ErrorMessage    string `json:"error_message"`
	Metadata        []byte `json:"metadata"`
}

// GetSubscriptionEvent godoc
// @Summary Get a subscription event by ID
// @Description Retrieves a subscription event by its ID
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {object} SubscriptionEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/{event_id} [get]
func (h *SubscriptionEventHandler) GetSubscriptionEvent(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	eventID := c.Param("event_id")
	parsedUUID, err := uuid.Parse(eventID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid event ID format", err)
		return
	}

	event, err := h.subscriptionEventService.GetSubscriptionEvent(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "subscription event not found" {
			sendError(c, http.StatusNotFound, "Subscription event not found", nil)
			return
		}
		if err.Error() == "unauthorized access to subscription event" {
			sendError(c, http.StatusForbidden, err.Error(), nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	eventResponse, err := helpers.ToSubscriptionEventResponse(c.Request.Context(), *event)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to format subscription event", err)
		return
	}

	sendSuccess(c, http.StatusOK, eventResponse)
}

// GetSubscriptionEventByTxHash godoc
// @Summary Get a subscription event by transaction hash
// @Description Retrieves a subscription event by its transaction hash
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param tx_hash path string true "Transaction Hash"
// @Success 200 {object} SubscriptionEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscription-events/transaction/{tx_hash} [get]
// @Tags exclude
func (h *SubscriptionEventHandler) GetSubscriptionEventByTxHash(c *gin.Context) {
	txHash := c.Param("tx_hash")
	if txHash == "" {
		sendError(c, http.StatusBadRequest, "Transaction hash is required", nil)
		return
	}

	event, err := h.subscriptionEventService.GetSubscriptionEventByTxHash(c.Request.Context(), txHash)
	if err != nil {
		if err.Error() == "subscription event not found" {
			sendError(c, http.StatusNotFound, "Subscription event not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	eventResponse, err := helpers.ToSubscriptionEventResponse(c.Request.Context(), *event)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to format subscription event", err)
		return
	}

	sendSuccess(c, http.StatusOK, eventResponse)
}

// ListSubscriptionEvents godoc
// @Summary List subscription events
// @Description Retrieves a paginated list of subscription events for the workspace
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param limit query int false "Number of events to return (default 20, max 100)"
// @Param page query int false "Page number (default 1)"
// @Success 200 {object} handlers.PaginatedResponse
// @Failure 400 {object} handlers.ErrorResponse "Invalid workspace ID format or pagination parameters"
// @Failure 500 {object} handlers.ErrorResponse "Failed to retrieve or count subscription events"
// @Router /subscription-events [get]
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

	result, err := h.subscriptionEventService.ListSubscriptionEvents(c.Request.Context(), services.ListSubscriptionEventsParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int(limit),
		Page:        int(page),
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	eventsResponse := make([]helpers.SubscriptionEventResponse, len(result.Events))
	for i, event := range result.Events {
		eventResponse, err := helpers.ToSubscriptionEventResponsePagination(c.Request.Context(), event)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to format subscription event", err)
			return
		}
		eventsResponse[i] = eventResponse
	}

	response := sendPaginatedSuccess(c, http.StatusOK, eventsResponse, int(page), int(limit), int(result.TotalCount))
	c.JSON(http.StatusOK, response)
}

// ListSubscriptionEventsBySubscription godoc
// @Summary List events for a subscription
// @Description Get a list of all events for a specific subscription
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {array} SubscriptionEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /subscriptions/{subscription_id}/events [get]
func (h *SubscriptionEventHandler) ListSubscriptionEventsBySubscription(c *gin.Context) {
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

	events, err := h.subscriptionEventService.ListSubscriptionEventsBySubscription(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "subscription not found" {
			sendError(c, http.StatusNotFound, "Subscription not found", nil)
			return
		}
		if err.Error() == "unauthorized access to subscription" {
			sendError(c, http.StatusForbidden, err.Error(), nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	eventsResponse := make([]helpers.SubscriptionEventResponse, len(events))
	for i, event := range events {
		eventResponse, err := helpers.ToSubscriptionEventResponse(c.Request.Context(), event)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to format subscription event", err)
			return
		}
		eventsResponse[i] = eventResponse
	}

	sendSuccess(c, http.StatusOK, eventsResponse)
}
