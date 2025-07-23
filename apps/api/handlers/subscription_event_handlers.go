package handlers

import (
	"context"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"net/http"
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
	SubscriptionID  string `json:"subscription_id" binding:"required"`
	EventType       string `json:"event_type" binding:"required"`
	TransactionHash string `json:"transaction_hash"`
	AmountInCents   int32  `json:"amount_in_cents"`
	OccurredAt      int64  `json:"occurred_at"`
	ErrorMessage    string `json:"error_message"`
	Metadata        []byte `json:"metadata"`
}

// SubscriptionEventPriceInfo contains essential price information relevant to a subscription event.
// Simplified compared to a full PriceResponse for embedding.
type SubscriptionEventPriceInfo struct {
	ID                  uuid.UUID `json:"id"`
	Type                string    `json:"type"` // e.g., recurring, one_off
	Currency            string    `json:"currency"`
	UnitAmountInPennies int32     `json:"unit_amount_in_pennies"`
	IntervalType        string    `json:"interval_type,omitempty"`  // Use db.NullIntervalType to handle potential nulls
	IntervalCount       int32     `json:"interval_count,omitempty"` // Use pgtype.Int4 to handle potential nulls
	TermLength          int32     `json:"term_length,omitempty"`    // Use pgtype.Int4 to handle potential nulls
}

// SubscriptionEventResponse represents a detailed view of a subscription event.
// It includes denormalized information from related entities like subscription, product, and price.
type SubscriptionEventResponse struct {
	ID                 uuid.UUID                  `json:"id"`
	SubscriptionID     uuid.UUID                  `json:"subscription_id"`
	EventType          string                     `json:"event_type"`
	TransactionHash    string                     `json:"transaction_hash,omitempty"`
	EventAmountInCents int32                      `json:"event_amount_in_cents"`
	EventOccurredAt    time.Time                  `json:"event_occurred_at"`
	ErrorMessage       string                     `json:"error_message,omitempty"`
	EventMetadata      []byte                     `json:"event_metadata,omitempty"`
	EventCreatedAt     time.Time                  `json:"event_created_at"`
	CustomerID         uuid.UUID                  `json:"customer_id"`
	SubscriptionStatus string                     `json:"subscription_status"`
	ProductID          uuid.UUID                  `json:"product_id"`
	ProductName        string                     `json:"product_name"`
	PriceInfo          SubscriptionEventPriceInfo `json:"price_info,omitempty"`
	Subscription       SubscriptionResponse       `json:"subscription,omitempty"`
	ProductToken       ProductTokenResponse       `json:"product_token,omitempty"`
	Network            NetworkResponse            `json:"network,omitempty"`
	Customer           CustomerResponse           `json:"customer,omitempty"`
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

	event, err := h.common.db.GetSubscriptionEvent(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Subscription event not found")
		return
	}

	subscription, err := h.common.db.GetSubscription(c.Request.Context(), event.SubscriptionID)
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	product, err := h.common.db.GetProduct(c.Request.Context(), db.GetProductParams{
		ID:          subscription.ProductID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	if product.WorkspaceID != parsedWorkspaceID {
		sendError(c, http.StatusForbidden, "You are not authorized to access this subscription event", nil)
		return
	}

	eventResponse, err := toSubscriptionEventResponse(c.Request.Context(), event)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve subscription event", err)
		return
	}

	sendSuccess(c, http.StatusOK, eventResponse)
}

func toSubscriptionEventResponse(ctx context.Context, eventDetails db.SubscriptionEvent) (SubscriptionEventResponse, error) {
	return SubscriptionEventResponse{
		ID:                 eventDetails.ID,
		SubscriptionID:     eventDetails.SubscriptionID,
		EventType:          string(eventDetails.EventType),
		TransactionHash:    eventDetails.TransactionHash.String,
		EventAmountInCents: eventDetails.AmountInCents,
		EventOccurredAt:    eventDetails.OccurredAt.Time,
		ErrorMessage:       eventDetails.ErrorMessage.String,
		EventMetadata:      eventDetails.Metadata,
		EventCreatedAt:     eventDetails.CreatedAt.Time,
	}, nil
}

// // toSubscriptionEventResponse converts a db.ListSubscriptionEventDetailsWithPaginationRow to a SubscriptionEventResponse.
// // It maps fields from the sqlc-generated struct, which includes joined data from subscriptions, products, and prices.
func toSubscriptionEventResponsePagination(ctx context.Context, eventDetails db.ListSubscriptionEventDetailsWithPaginationRow) (SubscriptionEventResponse, error) {
	resp := SubscriptionEventResponse{
		ID:                 eventDetails.SubscriptionEventID,
		SubscriptionID:     eventDetails.SubscriptionID,
		EventType:          string(eventDetails.EventType),
		TransactionHash:    eventDetails.TransactionHash.String,
		EventAmountInCents: eventDetails.EventAmountInCents,
		EventOccurredAt:    eventDetails.EventOccurredAt.Time,
		ErrorMessage:       eventDetails.ErrorMessage.String,
		EventMetadata:      eventDetails.EventMetadata,
		EventCreatedAt:     eventDetails.EventCreatedAt.Time,
		SubscriptionStatus: string(eventDetails.SubscriptionStatus),
		ProductID:          eventDetails.ProductID,
		ProductName:        eventDetails.ProductName,
		Customer: CustomerResponse{
			ID:    eventDetails.CustomerID.String(),
			Email: eventDetails.CustomerEmail.String,
			Name:  eventDetails.CustomerName.String,
		},
		PriceInfo: SubscriptionEventPriceInfo{
			ID:                  eventDetails.PriceID,
			Type:                string(eventDetails.PriceType),
			Currency:            string(eventDetails.PriceCurrency),
			UnitAmountInPennies: eventDetails.PriceUnitAmountInPennies,
			IntervalType:        string(eventDetails.PriceIntervalType), // db.NullIntervalType from sqlc
			TermLength:          int32(eventDetails.PriceTermLength),    // pgtype.Int4 from sqlc
		},
		ProductToken: ProductTokenResponse{
			ID:          eventDetails.ProductTokenID.String(),
			TokenID:     eventDetails.ProductTokenTokenID.String(),
			TokenSymbol: eventDetails.ProductTokenSymbol,
		},
		Network: NetworkResponse{
			ID:      eventDetails.NetworkID.String(),
			ChainID: eventDetails.NetworkChainID,
			Name:    eventDetails.NetworkName,
		},
	}
	return resp, nil
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
		eventResponse, err := toSubscriptionEventResponsePagination(c.Request.Context(), event)
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

	response := sendPaginatedSuccess(c, http.StatusOK, eventsResponse, int(page), int(limit), int(totalCount))
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

	subscription, err := h.common.db.GetSubscriptionWithWorkspace(c.Request.Context(), db.GetSubscriptionWithWorkspaceParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Subscription not found")
		return
	}

	if subscription.WorkspaceID != parsedWorkspaceID {
		sendError(c, http.StatusForbidden, "You are not authorized to access this subscription", nil)
		return
	}

	events, err := h.common.db.ListSubscriptionEventsBySubscription(c.Request.Context(), parsedUUID)
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

	sendSuccess(c, http.StatusOK, eventsResponse)
}
