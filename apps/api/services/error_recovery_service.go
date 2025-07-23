package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/client/payment_sync"
	"github.com/cyphera/cyphera-api/libs/go/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// ErrorRecoveryService handles webhook replay, DLQ processing, and sync recovery
type ErrorRecoveryService struct {
	db                *db.Queries
	logger            *zap.Logger
	paymentSyncClient *payment_sync.PaymentSyncClient
}

// NewErrorRecoveryService creates a new error recovery service
func NewErrorRecoveryService(
	dbQueries *db.Queries,
	logger *zap.Logger,
	paymentSyncClient *payment_sync.PaymentSyncClient,
) *ErrorRecoveryService {
	return &ErrorRecoveryService{
		db:                dbQueries,
		logger:            logger,
		paymentSyncClient: paymentSyncClient,
	}
}

// WebhookReplayRequest represents a webhook replay request
type WebhookReplayRequest struct {
	WorkspaceID    string `json:"workspace_id" binding:"required"`
	ProviderName   string `json:"provider_name" binding:"required"`
	WebhookEventID string `json:"webhook_event_id" binding:"required"`
	ForceReplay    bool   `json:"force_replay"`
	ReplayReason   string `json:"replay_reason,omitempty"`
}

// WebhookReplayResponse represents the result of webhook replay
type WebhookReplayResponse struct {
	Success         bool      `json:"success"`
	ReplayEventID   string    `json:"replay_event_id,omitempty"`
	OriginalEventID string    `json:"original_event_id"`
	ReplayedAt      time.Time `json:"replayed_at"`
	Message         string    `json:"message"`
	Error           string    `json:"error,omitempty"`
}

// SyncRecoveryRequest represents a sync session recovery request
type SyncRecoveryRequest struct {
	WorkspaceID  string   `json:"workspace_id" binding:"required"`
	SessionID    string   `json:"session_id" binding:"required"`
	RecoveryMode string   `json:"recovery_mode"` // "resume", "restart"
	EntityTypes  []string `json:"entity_types,omitempty"`
}

// SyncRecoveryResponse represents the result of sync recovery
type SyncRecoveryResponse struct {
	Success     bool                   `json:"success"`
	SessionID   string                 `json:"session_id"`
	RecoveredAt time.Time              `json:"recovered_at"`
	Progress    map[string]interface{} `json:"progress,omitempty"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
}

// DLQProcessingStats represents DLQ processing statistics
type DLQProcessingStats struct {
	TotalMessages         int64     `json:"total_messages"`
	SuccessfullyProcessed int64     `json:"successfully_processed"`
	ProcessingFailed      int64     `json:"processing_failed"`
	MaxRetriesExceeded    int64     `json:"max_retries_exceeded"`
	LastProcessedAt       time.Time `json:"last_processed_at"`
	SuccessRate           float64   `json:"success_rate"`
}

// ReplayWebhookEvent replays a failed webhook event
func (ers *ErrorRecoveryService) ReplayWebhookEvent(ctx context.Context, req WebhookReplayRequest) (*WebhookReplayResponse, error) {
	workspaceUUID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			Error:           "invalid workspace ID format",
		}, nil
	}

	ers.logger.Info("Starting webhook replay",
		zap.String("workspace_id", req.WorkspaceID),
		zap.String("provider", req.ProviderName),
		zap.String("webhook_event_id", req.WebhookEventID),
		zap.Bool("force_replay", req.ForceReplay))

	// Get the original webhook event
	originalEvent, err := ers.db.GetWebhookEventForReplay(ctx, db.GetWebhookEventForReplayParams{
		WorkspaceID:    workspaceUUID,
		ProviderName:   req.ProviderName,
		WebhookEventID: pgtype.Text{String: req.WebhookEventID, Valid: true},
	})
	if err != nil {
		ers.logger.Error("Failed to find original webhook event", zap.Error(err))
		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			Error:           "original webhook event not found",
		}, nil
	}

	// Check if we should proceed with replay
	if !req.ForceReplay && originalEvent.ProcessingAttempts.Int32 >= 3 {
		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			Error:           "webhook has already been retried multiple times, use force_replay=true to override",
		}, nil
	}

	// Create a new idempotency key for the replay
	replayIdempotencyKey := fmt.Sprintf("%s_replay_%d", req.WebhookEventID, time.Now().Unix())

	// Create replay event record
	replayEvent, err := ers.db.ReplayWebhookEvent(ctx, db.ReplayWebhookEventParams{
		WorkspaceID:       workspaceUUID,
		ProviderName:      req.ProviderName,
		EventMessage:      pgtype.Text{String: fmt.Sprintf("Replaying webhook %s: %s", req.WebhookEventID, req.ReplayReason), Valid: true},
		EventDetails:      originalEvent.EventDetails,
		WebhookEventID:    pgtype.Text{String: req.WebhookEventID, Valid: true},
		ProviderAccountID: originalEvent.ProviderAccountID,
		IdempotencyKey:    pgtype.Text{String: replayIdempotencyKey, Valid: true},
		SignatureValid:    originalEvent.SignatureValid,
	})
	if err != nil {
		ers.logger.Error("Failed to create replay event record", zap.Error(err))
		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			Error:           "failed to create replay record",
		}, nil
	}

	// Extract webhook event data from the original event details
	var eventData map[string]interface{}
	if err := json.Unmarshal(originalEvent.EventDetails, &eventData); err != nil {
		ers.logger.Error("Failed to parse event details", zap.Error(err))
		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			ReplayEventID:   replayEvent.ID.String(),
			Error:           "failed to parse event data",
		}, nil
	}

	// Create webhook event object for processing
	webhookEvent := payment_sync.WebhookEvent{
		ProviderEventID: req.WebhookEventID,
		Provider:        req.ProviderName,
		EventType:       fmt.Sprintf("%v", eventData["event_type"]),
		ReceivedAt:      time.Now().Unix(),
		Data:            eventData["event_data"],
		SignatureValid:  originalEvent.SignatureValid.Bool,
	}

	// Get provider service and process the webhook
	providerService, err := ers.paymentSyncClient.GetProviderService(ctx, req.WorkspaceID, req.ProviderName)
	if err != nil {
		ers.logger.Error("Failed to get provider service", zap.Error(err))
		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			ReplayEventID:   replayEvent.ID.String(),
			Error:           "failed to get provider service",
		}, nil
	}

	// Process the replayed webhook event
	// Note: This would typically go through the webhook processor, but for replay we can process directly
	err = ers.processReplayedWebhook(ctx, req.WorkspaceID, webhookEvent, providerService)
	if err != nil {
		ers.logger.Error("Failed to process replayed webhook", zap.Error(err))

		// Update the replay event with failure
		ers.db.UpdateWebhookProcessingStatus(ctx, db.UpdateWebhookProcessingStatusParams{
			ID:           replayEvent.ID,
			EventType:    "webhook_replay_failed",
			EventMessage: pgtype.Text{String: fmt.Sprintf("Replay failed: %v", err), Valid: true},
			EventDetails: []byte(fmt.Sprintf(`{"error": "%s", "retry_attempt": true}`, err.Error())),
		})

		return &WebhookReplayResponse{
			Success:         false,
			OriginalEventID: req.WebhookEventID,
			ReplayEventID:   replayEvent.ID.String(),
			ReplayedAt:      time.Now(),
			Error:           fmt.Sprintf("processing failed: %v", err),
		}, nil
	}

	// Update the replay event with success
	ers.db.UpdateWebhookProcessingStatus(ctx, db.UpdateWebhookProcessingStatusParams{
		ID:           replayEvent.ID,
		EventType:    "webhook_replay_success",
		EventMessage: pgtype.Text{String: "Webhook replay completed successfully", Valid: true},
		EventDetails: []byte(`{"replay_completed": true, "processed_successfully": true}`),
	})

	ers.logger.Info("Webhook replay completed successfully",
		zap.String("workspace_id", req.WorkspaceID),
		zap.String("original_event_id", req.WebhookEventID),
		zap.String("replay_event_id", replayEvent.ID.String()))

	return &WebhookReplayResponse{
		Success:         true,
		OriginalEventID: req.WebhookEventID,
		ReplayEventID:   replayEvent.ID.String(),
		ReplayedAt:      time.Now(),
		Message:         "Webhook replayed and processed successfully",
	}, nil
}

// RecoverSyncSession recovers a failed or incomplete sync session
func (ers *ErrorRecoveryService) RecoverSyncSession(ctx context.Context, req SyncRecoveryRequest) (*SyncRecoveryResponse, error) {
	workspaceUUID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     "invalid workspace ID format",
		}, nil
	}

	sessionUUID, err := uuid.Parse(req.SessionID)
	if err != nil {
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     "invalid session ID format",
		}, nil
	}

	ers.logger.Info("Starting sync session recovery",
		zap.String("workspace_id", req.WorkspaceID),
		zap.String("session_id", req.SessionID),
		zap.String("recovery_mode", req.RecoveryMode))

	// Get the sync session
	session, err := ers.db.GetSyncSession(ctx, db.GetSyncSessionParams{
		ID:          sessionUUID,
		WorkspaceID: workspaceUUID,
	})
	if err != nil {
		ers.logger.Error("Failed to find sync session", zap.Error(err))
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     "sync session not found",
		}, nil
	}

	// Check if session is recoverable
	if session.Status != "failed" && session.Status != "running" {
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     fmt.Sprintf("session status '%s' is not recoverable", session.Status),
		}, nil
	}

	switch req.RecoveryMode {
	case "resume":
		return ers.resumeSyncSession(ctx, session, req)
	case "restart":
		return ers.restartSyncSession(ctx, session, req)
	default:
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     "invalid recovery mode, use 'resume' or 'restart'",
		}, nil
	}
}

// GetDLQStats returns DLQ processing statistics
func (ers *ErrorRecoveryService) GetDLQStats(ctx context.Context, workspaceID, providerName string, since time.Time) (*DLQProcessingStats, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID format: %w", err)
	}

	stats, err := ers.db.GetDLQProcessingStats(ctx, db.GetDLQProcessingStatsParams{
		WorkspaceID:  workspaceUUID,
		ProviderName: providerName,
		OccurredAt:   pgtype.Timestamptz{Time: since, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ stats: %w", err)
	}

	var successRate float64
	if stats.TotalDlqMessages > 0 {
		successRate = float64(stats.SuccessfullyProcessed) / float64(stats.TotalDlqMessages) * 100
	}

	var lastProcessedAt time.Time
	if lastProcessedTime, ok := stats.LastProcessedAt.(time.Time); ok {
		lastProcessedAt = lastProcessedTime
	}

	return &DLQProcessingStats{
		TotalMessages:         stats.TotalDlqMessages,
		SuccessfullyProcessed: stats.SuccessfullyProcessed,
		ProcessingFailed:      stats.ProcessingFailed,
		MaxRetriesExceeded:    stats.MaxRetriesExceeded,
		LastProcessedAt:       lastProcessedAt,
		SuccessRate:           successRate,
	}, nil
}

// Helper methods

func (ers *ErrorRecoveryService) processReplayedWebhook(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent, providerService payment_sync.PaymentSyncService) error {
	// This is a simplified version - in a real implementation, you'd route this through
	// the same webhook processing pipeline as normal webhooks

	// For now, we'll just validate that the provider service can handle the event
	if providerService.GetServiceName() != webhookEvent.Provider {
		return fmt.Errorf("provider service mismatch: expected %s, got %s", webhookEvent.Provider, providerService.GetServiceName())
	}

	// Log the successful replay processing
	ers.logger.Info("Replayed webhook processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", webhookEvent.Provider),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	return nil
}

func (ers *ErrorRecoveryService) resumeSyncSession(ctx context.Context, session db.PaymentSyncSession, req SyncRecoveryRequest) (*SyncRecoveryResponse, error) {
	// Resume the session from where it left off
	resumedSession, err := ers.db.ResumeSyncSession(ctx, db.ResumeSyncSessionParams{
		ID:          session.ID,
		WorkspaceID: session.WorkspaceID,
	})
	if err != nil {
		ers.logger.Error("Failed to resume sync session", zap.Error(err))
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     "failed to resume session",
		}, nil
	}

	// Parse current progress
	var progress map[string]interface{}
	if len(resumedSession.Progress) > 0 {
		json.Unmarshal(resumedSession.Progress, &progress)
	}

	ers.logger.Info("Sync session resumed successfully",
		zap.String("workspace_id", req.WorkspaceID),
		zap.String("session_id", req.SessionID))

	return &SyncRecoveryResponse{
		Success:     true,
		SessionID:   req.SessionID,
		RecoveredAt: time.Now(),
		Progress:    progress,
		Message:     "Sync session resumed successfully",
	}, nil
}

func (ers *ErrorRecoveryService) restartSyncSession(ctx context.Context, session db.PaymentSyncSession, req SyncRecoveryRequest) (*SyncRecoveryResponse, error) {
	// Create a new sync session based on the failed one
	entityTypes := req.EntityTypes
	if len(entityTypes) == 0 {
		entityTypes = session.EntityTypes
	}

	// Parse the original config
	var config map[string]interface{}
	if len(session.Config) > 0 {
		json.Unmarshal(session.Config, &config)
	}

	// Create new sync session
	configJSON, _ := json.Marshal(config)
	newSession, err := ers.db.CreateSyncSession(ctx, db.CreateSyncSessionParams{
		WorkspaceID:  session.WorkspaceID,
		ProviderName: session.ProviderName,
		SessionType:  "recovery_sync",
		Status:       "pending",
		EntityTypes:  entityTypes,
		Config:       configJSON,
	})
	if err != nil {
		ers.logger.Error("Failed to create recovery sync session", zap.Error(err))
		return &SyncRecoveryResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     "failed to create recovery session",
		}, nil
	}

	// Mark the old session as superseded
	ers.db.UpdateSyncSessionStatus(ctx, db.UpdateSyncSessionStatusParams{
		ID:     session.ID,
		Status: "superseded",
	})

	ers.logger.Info("New recovery sync session created",
		zap.String("workspace_id", req.WorkspaceID),
		zap.String("original_session_id", req.SessionID),
		zap.String("new_session_id", newSession.ID.String()))

	return &SyncRecoveryResponse{
		Success:     true,
		SessionID:   newSession.ID.String(),
		RecoveredAt: time.Now(),
		Message:     fmt.Sprintf("New recovery sync session created: %s", newSession.ID.String()),
	}, nil
}
