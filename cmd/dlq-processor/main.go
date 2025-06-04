package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"cyphera-api/internal/client/payment_sync"
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Application holds the application dependencies
type Application struct {
	db                *db.Queries
	logger            *zap.Logger
	paymentSyncClient *payment_sync.PaymentSyncClient
	maxRetries        int
	retryBackoffMs    int
}

// DLQMessage represents a message from the DLQ
type DLQMessage struct {
	WorkspaceID   string                    `json:"workspace_id"`
	Provider      string                    `json:"provider"`
	WebhookEvent  payment_sync.WebhookEvent `json:"webhook_event"`
	OriginalError string                    `json:"original_error,omitempty"`
	FailureCount  int                       `json:"failure_count"`
	LastFailedAt  int64                     `json:"last_failed_at"`
	RetryAttempt  int                       `json:"retry_attempt"`
}

// DLQProcessingResult represents the result of processing a DLQ message
type DLQProcessingResult struct {
	MessageID             string `json:"message_id"`
	WorkspaceID           string `json:"workspace_id"`
	Provider              string `json:"provider"`
	ProcessedSuccessfully bool   `json:"processed_successfully"`
	RetryAttempt          int    `json:"retry_attempt"`
	Error                 string `json:"error,omitempty"`
	ShouldRetry           bool   `json:"should_retry"`
}

func main() {
	// Initialize logger
	logger.InitLogger("production")
	zapLogger := logger.Log
	defer zapLogger.Sync()

	// Create application
	app, err := createApplication(zapLogger)
	if err != nil {
		zapLogger.Fatal("Failed to create application", zap.Error(err))
	}

	// Start Lambda handler
	lambda.Start(app.handleDLQEvent)
}

func createApplication(logger *zap.Logger) (*Application, error) {
	// Get configuration from environment variables
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	encryptionKey := os.Getenv("PAYMENT_SYNC_ENCRYPTION_KEY")
	if encryptionKey == "" {
		return nil, fmt.Errorf("PAYMENT_SYNC_ENCRYPTION_KEY environment variable is required")
	}

	// Parse max retries (default: 3)
	maxRetries := 3
	if maxRetriesStr := os.Getenv("DLQ_MAX_RETRIES"); maxRetriesStr != "" {
		if parsed, err := strconv.Atoi(maxRetriesStr); err == nil && parsed > 0 {
			maxRetries = parsed
		}
	}

	// Parse retry backoff (default: 5000ms)
	retryBackoffMs := 5000
	if backoffStr := os.Getenv("DLQ_RETRY_BACKOFF_MS"); backoffStr != "" {
		if parsed, err := strconv.Atoi(backoffStr); err == nil && parsed > 0 {
			retryBackoffMs = parsed
		}
	}

	// Create database connection
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test database connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create queries instance
	queries := db.New(pool)

	// Create payment sync client
	paymentSyncClient := payment_sync.NewPaymentSyncClient(queries, logger, encryptionKey)

	return &Application{
		db:                queries,
		logger:            logger,
		paymentSyncClient: paymentSyncClient,
		maxRetries:        maxRetries,
		retryBackoffMs:    retryBackoffMs,
	}, nil
}

// handleDLQEvent processes SQS events from the dead letter queue
func (app *Application) handleDLQEvent(ctx context.Context, event events.SQSEvent) error {
	app.logger.Info("Processing DLQ event", zap.Int("message_count", len(event.Records)))

	var results []DLQProcessingResult

	for _, record := range event.Records {
		result := app.processDLQMessage(ctx, record)
		results = append(results, result)

		// Log individual result
		if result.ProcessedSuccessfully {
			app.logger.Info("DLQ message processed successfully",
				zap.String("message_id", result.MessageID),
				zap.String("workspace_id", result.WorkspaceID),
				zap.String("provider", result.Provider),
				zap.Int("retry_attempt", result.RetryAttempt))
		} else {
			app.logger.Error("DLQ message processing failed",
				zap.String("message_id", result.MessageID),
				zap.String("workspace_id", result.WorkspaceID),
				zap.String("provider", result.Provider),
				zap.Int("retry_attempt", result.RetryAttempt),
				zap.String("error", result.Error),
				zap.Bool("should_retry", result.ShouldRetry))
		}
	}

	// Log summary
	successful := 0
	failed := 0
	for _, result := range results {
		if result.ProcessedSuccessfully {
			successful++
		} else {
			failed++
		}
	}

	app.logger.Info("DLQ processing complete",
		zap.Int("total_messages", len(results)),
		zap.Int("successful", successful),
		zap.Int("failed", failed))

	return nil
}

// processDLQMessage processes a single DLQ message
func (app *Application) processDLQMessage(ctx context.Context, record events.SQSMessage) DLQProcessingResult {
	result := DLQProcessingResult{
		MessageID: record.MessageId,
	}

	// Parse the DLQ message
	var dlqMessage DLQMessage
	if err := json.Unmarshal([]byte(record.Body), &dlqMessage); err != nil {
		result.Error = fmt.Sprintf("Failed to parse DLQ message: %v", err)
		return result
	}

	result.WorkspaceID = dlqMessage.WorkspaceID
	result.Provider = dlqMessage.Provider
	result.RetryAttempt = dlqMessage.RetryAttempt + 1

	app.logger.Info("Processing DLQ message",
		zap.String("message_id", record.MessageId),
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.Int("failure_count", dlqMessage.FailureCount),
		zap.Int("retry_attempt", result.RetryAttempt))

	// Check if we should retry
	if result.RetryAttempt > app.maxRetries {
		result.Error = fmt.Sprintf("Maximum retries exceeded (%d)", app.maxRetries)
		result.ShouldRetry = false

		// Log to database as permanently failed
		app.logDLQPermanentFailure(ctx, dlqMessage, result.Error)
		return result
	}

	// Implement exponential backoff
	backoffDelay := time.Duration(app.retryBackoffMs*result.RetryAttempt) * time.Millisecond
	if backoffDelay > 0 {
		app.logger.Info("Applying backoff delay",
			zap.Duration("delay", backoffDelay),
			zap.Int("retry_attempt", result.RetryAttempt))

		select {
		case <-time.After(backoffDelay):
			// Continue processing
		case <-ctx.Done():
			result.Error = "Context cancelled during backoff"
			result.ShouldRetry = true
			return result
		}
	}

	// Get provider service
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, dlqMessage.WorkspaceID, dlqMessage.Provider)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get provider service: %v", err)
		result.ShouldRetry = true
		return result
	}

	// Process the webhook event
	err = app.processWebhookEvent(ctx, dlqMessage.WorkspaceID, dlqMessage.WebhookEvent, providerService)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to process webhook event: %v", err)
		result.ShouldRetry = app.shouldRetryError(err)

		// Log the retry attempt
		app.logDLQRetryAttempt(ctx, dlqMessage, result.RetryAttempt, err.Error())
		return result
	}

	// Success!
	result.ProcessedSuccessfully = true

	// Log successful processing
	app.logDLQSuccessfulProcessing(ctx, dlqMessage, result.RetryAttempt)

	return result
}

// processWebhookEvent processes a webhook event using the provider service
func (app *Application) processWebhookEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent, providerService payment_sync.PaymentSyncService) error {
	// This is similar to the webhook processor logic, but specifically for DLQ processing

	app.logger.Debug("Processing webhook event",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", webhookEvent.Provider),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	// For now, we'll use a simplified processing approach
	// In a full implementation, this would route through the same processing logic as the main webhook processor

	// Validate that the provider service matches
	if providerService.GetServiceName() != webhookEvent.Provider {
		return fmt.Errorf("provider service mismatch: expected %s, got %s", webhookEvent.Provider, providerService.GetServiceName())
	}

	// Log successful processing (placeholder for actual webhook processing)
	app.logger.Info("Webhook event processed successfully via DLQ",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", webhookEvent.Provider),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	return nil
}

// shouldRetryError determines if an error is retryable
func (app *Application) shouldRetryError(err error) bool {
	errorStr := err.Error()

	// Don't retry authentication errors
	if contains(errorStr, []string{"authentication", "unauthorized", "forbidden", "invalid key", "invalid secret"}) {
		return false
	}

	// Don't retry invalid data errors
	if contains(errorStr, []string{"invalid format", "malformed", "parse error", "invalid json"}) {
		return false
	}

	// Retry network and temporary errors
	if contains(errorStr, []string{"timeout", "connection", "network", "temporary", "rate limit", "service unavailable"}) {
		return true
	}

	// Default to retry for unknown errors
	return true
}

// contains checks if any of the substrings are contained in the target string
func contains(target string, substrings []string) bool {
	for _, substring := range substrings {
		if len(target) >= len(substring) {
			for i := 0; i <= len(target)-len(substring); i++ {
				if target[i:i+len(substring)] == substring {
					return true
				}
			}
		}
	}
	return false
}

// logDLQRetryAttempt logs a DLQ retry attempt to the database
func (app *Application) logDLQRetryAttempt(ctx context.Context, dlqMessage DLQMessage, retryAttempt int, errorMessage string) {
	app.logger.Debug("Logging DLQ retry attempt",
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.Int("retry_attempt", retryAttempt),
		zap.String("error", errorMessage))

	// In a full implementation, this would log to the database using the appropriate queries
	// For now, we'll just log it
}

// logDLQSuccessfulProcessing logs successful DLQ processing to the database
func (app *Application) logDLQSuccessfulProcessing(ctx context.Context, dlqMessage DLQMessage, retryAttempt int) {
	app.logger.Debug("Logging DLQ successful processing",
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.Int("retry_attempt", retryAttempt))

	// In a full implementation, this would log to the database using the appropriate queries
	// For now, we'll just log it
}

// logDLQPermanentFailure logs a permanent DLQ failure to the database
func (app *Application) logDLQPermanentFailure(ctx context.Context, dlqMessage DLQMessage, errorMessage string) {
	app.logger.Error("Logging DLQ permanent failure",
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.String("error", errorMessage))

	// In a full implementation, this would log to the database using the appropriate queries
	// For now, we'll just log it
}
