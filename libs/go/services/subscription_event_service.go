package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// SubscriptionEventService handles business logic for subscription event operations
type SubscriptionEventService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewSubscriptionEventService creates a new subscription event service
func NewSubscriptionEventService(queries db.Querier) *SubscriptionEventService {
	return &SubscriptionEventService{
		queries: queries,
		logger:  logger.Log,
	}
}

// GetSubscriptionEvent retrieves a subscription event by ID with workspace validation
func (s *SubscriptionEventService) GetSubscriptionEvent(ctx context.Context, eventID, workspaceID uuid.UUID) (*db.SubscriptionEvent, error) {
	event, err := s.queries.GetSubscriptionEvent(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription event not found")
		}
		s.logger.Error("Failed to get subscription event",
			zap.String("event_id", eventID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve subscription event: %w", err)
	}

	// Validate workspace access
	subscription, err := s.queries.GetSubscription(ctx, event.SubscriptionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	product, err := s.queries.GetProduct(ctx, db.GetProductParams{
		ID:          subscription.ProductID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to retrieve product: %w", err)
	}

	if product.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("unauthorized access to subscription event")
	}

	return &event, nil
}

// GetSubscriptionEventByTxHash retrieves a subscription event by transaction hash
func (s *SubscriptionEventService) GetSubscriptionEventByTxHash(ctx context.Context, txHash string) (*db.SubscriptionEvent, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash is required")
	}

	event, err := s.queries.GetSubscriptionEventByTransactionHash(ctx, pgtype.Text{
		String: txHash,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription event not found")
		}
		s.logger.Error("Failed to get subscription event by tx hash",
			zap.String("tx_hash", txHash),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve subscription event: %w", err)
	}

	return &event, nil
}

// ListSubscriptionEvents retrieves a paginated list of subscription events for a workspace
func (s *SubscriptionEventService) ListSubscriptionEvents(ctx context.Context, params params.ListSubscriptionEventsParams) (*responses.ListSubscriptionEventsResult, error) {
	// Validate pagination parameters
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// Get events
	s.logger.Info("Fetching subscription events with pagination",
		zap.String("workspace_id", params.WorkspaceID.String()),
		zap.Int32("limit", params.Limit),
		zap.Int32("offset", params.Offset))

	events, err := s.queries.ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
		WorkspaceID: params.WorkspaceID,
		Limit:       params.Limit,
		Offset:      params.Offset,
	})
	if err != nil {
		s.logger.Error("Failed to list subscription events",
			zap.String("workspace_id", params.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve subscription events: %w", err)
	}

	s.logger.Info("Retrieved subscription events from database",
		zap.String("workspace_id", params.WorkspaceID.String()),
		zap.Int("events_count", len(events)),
		zap.Any("first_event_sample", func() interface{} {
			if len(events) > 0 {
				return map[string]interface{}{
					"id":         events[0].SubscriptionEventID.String(),
					"event_type": events[0].EventType,
					"tx_hash":    events[0].TransactionHash,
				}
			}
			return nil
		}()))

	// Get total count
	totalCount, err := s.queries.CountSubscriptionEventDetails(ctx, params.WorkspaceID)
	if err != nil {
		s.logger.Error("Failed to count subscription events",
			zap.String("workspace_id", params.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to count subscription events: %w", err)
	}

	// Convert events to response format using the helper for full response
	responseEvents := make([]responses.SubscriptionEventFullResponse, len(events))
	for i, event := range events {
		fullResponse, err := helpers.ToSubscriptionEventResponsePagination(ctx, event)
		if err != nil {
			s.logger.Error("Failed to convert subscription event to response",
				zap.String("event_id", event.SubscriptionEventID.String()),
				zap.Error(err))
			return nil, fmt.Errorf("failed to convert subscription event to response: %w", err)
		}
		responseEvents[i] = fullResponse
	}

	s.logger.Info("Converted events to response format",
		zap.String("workspace_id", params.WorkspaceID.String()),
		zap.Int("response_events_count", len(responseEvents)),
		zap.Any("first_response_sample", func() interface{} {
			if len(responseEvents) > 0 {
				return map[string]interface{}{
					"id":         responseEvents[0].ID.String(),
					"event_type": responseEvents[0].EventType,
					"tx_hash":    responseEvents[0].TransactionHash,
				}
			}
			return nil
		}()))

	return &responses.ListSubscriptionEventsResult{
		Events: responseEvents,
		Total:  totalCount,
	}, nil
}

// ListSubscriptionEventsBySubscription retrieves all events for a specific subscription
func (s *SubscriptionEventService) ListSubscriptionEventsBySubscription(ctx context.Context, subscriptionID, workspaceID uuid.UUID) ([]db.SubscriptionEvent, error) {
	// Validate workspace access
	subscription, err := s.queries.GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	if subscription.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("unauthorized access to subscription")
	}

	// Get events
	events, err := s.queries.ListSubscriptionEventsBySubscription(ctx, subscriptionID)
	if err != nil {
		s.logger.Error("Failed to list subscription events by subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve subscription events: %w", err)
	}

	return events, nil
}

// CreateSubscriptionEvent creates a new subscription event
func (s *SubscriptionEventService) CreateSubscriptionEvent(ctx context.Context, params params.CreateSubscriptionEventParams) (*db.SubscriptionEvent, error) {
	// Validate subscription exists
	_, err := s.queries.GetSubscription(ctx, params.SubscriptionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create the event
	event, err := s.queries.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
		SubscriptionID: params.SubscriptionID,
		EventType:      params.EventType,
		TransactionHash: pgtype.Text{
			String: func() string {
				if params.TransactionHash != nil {
					return *params.TransactionHash
				} else {
					return ""
				}
			}(),
			Valid: params.TransactionHash != nil && *params.TransactionHash != "",
		},
		AmountInCents: params.AmountInCents,
		OccurredAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true}, // Use current time
		ErrorMessage: pgtype.Text{
			String: func() string {
				if params.FailureReason != nil {
					return *params.FailureReason
				} else {
					return ""
				}
			}(),
			Valid: params.FailureReason != nil,
		},
		Metadata: metadataJSON,
	})
	if err != nil {
		s.logger.Error("Failed to create subscription event",
			zap.String("subscription_id", params.SubscriptionID.String()),
			zap.String("event_type", string(params.EventType)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create subscription event: %w", err)
	}

	s.logger.Info("Subscription event created successfully",
		zap.String("event_id", event.ID.String()),
		zap.String("event_type", string(params.EventType)))

	return &event, nil
}
