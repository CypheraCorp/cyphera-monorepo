package handlers

import (
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// Error messages for subscription events
	errMsgInvalidEventIDFormat        = "Invalid event ID format"
	errMsgInvalidSubscriptionIDFormat = "Invalid subscription ID format"
	errMsgTransactionHashRequired     = "Transaction hash is required"
	errMsgSubscriptionEventNotFound   = "Subscription event not found"
	errMsgSubscriptionNotFound        = "Subscription not found"
	errMsgFailedToFormatEvent         = "Failed to format subscription event"
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
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	eventID := c.Param("event_id")
	parsedUUID, err := uuid.Parse(eventID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidEventIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	event, err := h.subscriptionEventService.GetSubscriptionEvent(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "subscription event not found" {
			h.common.HandleError(c, nil, errMsgSubscriptionEventNotFound, http.StatusNotFound, h.common.GetLogger())
			return
		}
		if err.Error() == "unauthorized access to subscription event" {
			h.common.HandleError(c, nil, err.Error(), http.StatusForbidden, h.common.GetLogger())
			return
		}
		h.common.HandleError(c, err, err.Error(), http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	eventResponse, err := helpers.ToSubscriptionEventResponse(c.Request.Context(), *event)
	if err != nil {
		h.common.HandleError(c, err, errMsgFailedToFormatEvent, http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusOK, eventResponse)
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
func (h *SubscriptionEventHandler) GetSubscriptionEventByTxHash(c *gin.Context) {
	txHash := c.Param("tx_hash")
	if txHash == "" {
		h.common.HandleError(c, nil, errMsgTransactionHashRequired, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	event, err := h.subscriptionEventService.GetSubscriptionEventByTxHash(c.Request.Context(), txHash)
	if err != nil {
		if err.Error() == "subscription event not found" {
			h.common.HandleError(c, nil, errMsgSubscriptionEventNotFound, http.StatusNotFound, h.common.GetLogger())
			return
		}
		h.common.HandleError(c, err, err.Error(), http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	eventResponse, err := helpers.ToSubscriptionEventResponse(c.Request.Context(), *event)
	if err != nil {
		h.common.HandleError(c, err, errMsgFailedToFormatEvent, http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	c.JSON(http.StatusOK, eventResponse)
}

// ListSubscriptionEvents godoc
// @Summary List subscription events transactions
// @Description Retrieves a paginated list of subscription events for the workspace with full details including network and customer information
// @Tags subscription-events
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param limit query int false "Number of events to return (default 20, max 100)"
// @Param page query int false "Page number (default 1)"
// @Success 200 {object} responses.PaginatedResponse{data=[]responses.SubscriptionEventFullResponse} "Paginated list of subscription events with full network and customer details"
// @Failure 400 {object} handlers.ErrorResponse "Invalid workspace ID format or pagination parameters"
// @Failure 500 {object} handlers.ErrorResponse "Failed to retrieve or count subscription events"
// @Router /subscription-events/transactions [get]
// @Security ApiKeyAuth
func (h *SubscriptionEventHandler) ListSubscriptionEvents(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	// Get pagination parameters
	pageParams, err := helpers.ParsePaginationParams(c)
	if err != nil {
		h.common.HandleError(c, err, err.Error(), http.StatusBadRequest, h.common.GetLogger())
		return
	}
	limit, page := pageParams.Limit, pageParams.Page

	result, err := h.subscriptionEventService.ListSubscriptionEvents(c.Request.Context(), params.ListSubscriptionEventsParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32((page - 1) * limit), // Convert page to offset (page 1 = offset 0)
	})
	if err != nil {
		h.common.HandleError(c, err, err.Error(), http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	// Pre-allocate slice with capacity for better memory efficiency
	eventsResponse := make([]responses.SubscriptionEventFullResponse, 0, len(result.Events))
	for _, event := range result.Events {
		eventsResponse = append(eventsResponse, event)
	}

	response := sendPaginatedSuccess(c, http.StatusOK, eventsResponse, int(page), int(limit), int(result.Total))
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
		h.common.HandleError(c, err, errMsgInvalidWorkspaceIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	subscriptionID := c.Param("subscription_id")
	parsedUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidSubscriptionIDFormat, http.StatusBadRequest, h.common.GetLogger())
		return
	}

	events, err := h.subscriptionEventService.ListSubscriptionEventsBySubscription(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "subscription not found" {
			h.common.HandleError(c, nil, errMsgSubscriptionNotFound, http.StatusNotFound, h.common.GetLogger())
			return
		}
		if err.Error() == "unauthorized access to subscription" {
			h.common.HandleError(c, nil, err.Error(), http.StatusForbidden, h.common.GetLogger())
			return
		}
		h.common.HandleError(c, err, err.Error(), http.StatusInternalServerError, h.common.GetLogger())
		return
	}

	eventsResponse := make([]responses.SubscriptionEventResponse, 0, len(events))
	for _, event := range events {
		eventResponse, err := helpers.ToSubscriptionEventResponse(c.Request.Context(), event)
		if err != nil {
			h.common.HandleError(c, err, errMsgFailedToFormatEvent, http.StatusInternalServerError, h.common.GetLogger())
			return
		}
		eventsResponse = append(eventsResponse, eventResponse)
	}

	sendSuccess(c, http.StatusOK, eventsResponse)
}
