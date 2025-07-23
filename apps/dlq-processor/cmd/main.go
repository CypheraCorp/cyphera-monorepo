package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/client/aws"
	"github.com/cyphera/cyphera-api/libs/go/client/payment_sync"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"

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
	ProcessedAt           int64  `json:"processed_at"`
	ShouldRetry           bool   `json:"should_retry"`
}

// HandleSQSEvent processes messages from the DLQ
func (app *Application) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	logger.Info("DLQ processor handling SQS event",
		zap.Int("record_count", len(event.Records)))

	results := make([]DLQProcessingResult, 0, len(event.Records))
	hasFailures := false

	for _, record := range event.Records {
		result := app.processDLQRecord(ctx, record)
		results = append(results, result)

		if !result.ProcessedSuccessfully {
			hasFailures = true
		}
	}

	// Log summary
	successCount := 0
	for _, result := range results {
		if result.ProcessedSuccessfully {
			successCount++
		}
	}

	logger.Info("DLQ processing completed",
		zap.Int("total", len(results)),
		zap.Int("success", successCount),
		zap.Int("failed", len(results)-successCount),
		zap.Bool("has_failures", hasFailures))

	// Return error if any messages failed to process
	// This will trigger the DLQ retry mechanism
	if hasFailures {
		return fmt.Errorf("failed to process %d of %d messages", len(results)-successCount, len(results))
	}

	return nil
}

// processDLQRecord processes a single DLQ record
func (app *Application) processDLQRecord(ctx context.Context, record events.SQSMessage) DLQProcessingResult {
	result := DLQProcessingResult{
		MessageID:   record.MessageId,
		ProcessedAt: time.Now().Unix(),
	}

	logger.Info("Processing DLQ record",
		zap.String("message_id", record.MessageId),
		zap.String("source", record.EventSourceARN))

	// Parse the DLQ message
	var dlqMessage DLQMessage
	if err := json.Unmarshal([]byte(record.Body), &dlqMessage); err != nil {
		logger.Error("Failed to unmarshal DLQ message",
			zap.String("message_id", record.MessageId),
			zap.Error(err))
		result.Error = fmt.Sprintf("unmarshal error: %v", err)
		return result
	}

	result.WorkspaceID = dlqMessage.WorkspaceID
	result.Provider = dlqMessage.Provider
	result.RetryAttempt = dlqMessage.RetryAttempt + 1

	// Check if we've exceeded max retries
	if result.RetryAttempt > app.maxRetries {
		logger.Warn("Max retries exceeded for DLQ message",
			zap.String("message_id", record.MessageId),
			zap.String("workspace_id", dlqMessage.WorkspaceID),
			zap.Int("retry_attempt", result.RetryAttempt))
		result.Error = "max retries exceeded"
		result.ShouldRetry = false

		// Log to database as permanently failed
		app.logPermanentFailure(ctx, dlqMessage, result.Error)
		return result
	}

	// Apply exponential backoff
	timeSinceLastFailure := time.Now().Unix() - dlqMessage.LastFailedAt
	backoffSeconds := app.calculateBackoff(result.RetryAttempt)

	if timeSinceLastFailure < backoffSeconds {
		logger.Info("Skipping DLQ message due to backoff",
			zap.String("message_id", record.MessageId),
			zap.Int64("backoff_remaining", backoffSeconds-timeSinceLastFailure))
		result.Error = "still in backoff period"
		result.ShouldRetry = true
		result.ProcessedSuccessfully = false
		return result
	}

	// Attempt to reprocess the webhook event
	err := app.reprocessWebhookEvent(ctx, dlqMessage)
	if err != nil {
		logger.Error("Failed to reprocess webhook event",
			zap.String("message_id", record.MessageId),
			zap.String("workspace_id", dlqMessage.WorkspaceID),
			zap.Error(err))
		result.Error = fmt.Sprintf("reprocess error: %v", err)
		result.ShouldRetry = true
		result.ProcessedSuccessfully = false

		// Update failure count in database
		app.updateFailureCount(ctx, dlqMessage, err.Error())
		return result
	}

	// Success!
	logger.Info("Successfully reprocessed DLQ message",
		zap.String("message_id", record.MessageId),
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.Int("retry_attempt", result.RetryAttempt))

	result.ProcessedSuccessfully = true
	result.ShouldRetry = false

	// Log successful recovery in database
	app.logSuccessfulRecovery(ctx, dlqMessage)

	return result
}

// calculateBackoff calculates exponential backoff in seconds
func (app *Application) calculateBackoff(retryAttempt int) int64 {
	baseBackoff := app.retryBackoffMs / 1000 // Convert to seconds
	maxBackoff := int64(3600)                // 1 hour max

	backoff := int64(baseBackoff) * (1 << (retryAttempt - 1))
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// reprocessWebhookEvent attempts to reprocess a failed webhook event
func (app *Application) reprocessWebhookEvent(ctx context.Context, dlqMessage DLQMessage) error {
	// Get the provider service for the workspace
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, dlqMessage.WorkspaceID, dlqMessage.Provider)
	if err != nil {
		return fmt.Errorf("failed to get provider service: %w", err)
	}

	// Process the webhook event based on its type
	switch dlqMessage.Provider {
	case "stripe":
		return app.reprocessStripeWebhook(ctx, dlqMessage, providerService)
	default:
		return fmt.Errorf("unsupported provider: %s", dlqMessage.Provider)
	}
}

// reprocessStripeWebhook reprocesses a Stripe webhook event
func (app *Application) reprocessStripeWebhook(ctx context.Context, dlqMessage DLQMessage, providerService payment_sync.PaymentSyncService) error {
	// The webhook event data should already be parsed and validated
	// We just need to process it again

	// Since this is a retry, we might want to check if the data has already been processed
	// to avoid duplicates
	isDuplicate, err := app.checkIfAlreadyProcessed(ctx, dlqMessage.WorkspaceID, dlqMessage.WebhookEvent)
	if err != nil {
		return fmt.Errorf("failed to check for duplicates: %w", err)
	}

	if isDuplicate {
		logger.Info("Webhook event already processed, skipping",
			zap.String("workspace_id", dlqMessage.WorkspaceID),
			zap.String("event_id", dlqMessage.WebhookEvent.ProviderEventID))
		return nil
	}

	// Reprocess based on event type
	// This logic should match what's in the webhook processor
	// You might want to extract this into a shared function
	return fmt.Errorf("webhook reprocessing not yet implemented")
}

// checkIfAlreadyProcessed checks if a webhook event has already been processed
func (app *Application) checkIfAlreadyProcessed(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) (bool, error) {
	// This should check your database to see if the event has already been processed
	// For example, checking if a customer/product/subscription with the external ID already exists
	// This is a placeholder implementation
	return false, nil
}

// logPermanentFailure logs a webhook event that has permanently failed
func (app *Application) logPermanentFailure(ctx context.Context, dlqMessage DLQMessage, errorMessage string) {
	// Log to database that this webhook has permanently failed
	// This helps with monitoring and manual intervention if needed
	logger.Error("Webhook permanently failed",
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.String("event_type", dlqMessage.WebhookEvent.EventType),
		zap.String("event_id", dlqMessage.WebhookEvent.ProviderEventID),
		zap.String("error", errorMessage))
}

// updateFailureCount updates the failure count for a webhook event
func (app *Application) updateFailureCount(ctx context.Context, dlqMessage DLQMessage, errorMessage string) {
	// Update failure count in database
	// This helps track problematic webhooks
	logger.Warn("Webhook retry failed",
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.String("event_type", dlqMessage.WebhookEvent.EventType),
		zap.String("event_id", dlqMessage.WebhookEvent.ProviderEventID),
		zap.Int("retry_attempt", dlqMessage.RetryAttempt+1),
		zap.String("error", errorMessage))
}

// logSuccessfulRecovery logs that a webhook event was successfully recovered
func (app *Application) logSuccessfulRecovery(ctx context.Context, dlqMessage DLQMessage) {
	// Log successful recovery to database
	// This helps track recovery rates and patterns
	logger.Info("Webhook successfully recovered from DLQ",
		zap.String("workspace_id", dlqMessage.WorkspaceID),
		zap.String("provider", dlqMessage.Provider),
		zap.String("event_type", dlqMessage.WebhookEvent.EventType),
		zap.String("event_id", dlqMessage.WebhookEvent.ProviderEventID),
		zap.Int("retry_attempt", dlqMessage.RetryAttempt))
}

func main() {
	stage := os.Getenv("STAGE")
	if stage == "" {
		stage = helpers.StageLocal
	}
	if !helpers.IsValidStage(stage) {
		panic(fmt.Sprintf("Invalid STAGE environment variable: '%s'. Must be one of: %s, %s, %s",
			stage, helpers.StageProd, helpers.StageDev, helpers.StageLocal))
	}

	// Initialize logger
	logger.InitLogger(stage)
	logger.Info("Lambda Cold Start: Initializing DLQ processor for stage", zap.String("stage", stage))
	defer func() {
		_ = logger.Sync()
	}()

	ctx := context.Background()

	// Initialize AWS Secrets Manager Client
	secretsClient, err := aws.NewSecretsManagerClient(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize AWS Secrets Manager client", zap.Error(err))
	}

	// Database Connection Setup
	var dsn string
	if stage == helpers.StageProd || stage == helpers.StageDev {
		logger.Info("Running in deployed stage, fetching DB credentials from Secrets Manager", zap.String("stage", stage))
		dbEndpoint := os.Getenv("DB_HOST")
		dbName := os.Getenv("DB_NAME")
		dbSecretArn := os.Getenv("RDS_SECRET_ARN")
		dbSSLMode := os.Getenv("DB_SSLMODE")

		if dbEndpoint == "" || dbName == "" || dbSecretArn == "" {
			logger.Fatal("Missing required DB environment variables for deployed environment")
		}
		if dbSSLMode == "" {
			dbSSLMode = "require"
		}

		type RdsSecret struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var secretData RdsSecret
		err = secretsClient.GetSecretJSON(ctx, "RDS_SECRET_ARN", "", &secretData)
		if err != nil {
			logger.Fatal("Failed to retrieve or parse RDS secret", zap.Error(err))
		}

		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
			url.QueryEscape(secretData.Username), url.QueryEscape(secretData.Password),
			dbEndpoint, dbName, dbSSLMode)
	} else {
		// Local
		dsn, err = secretsClient.GetSecretString(ctx, "DATABASE_URL_ARN", "DATABASE_URL")
		if err != nil {
			logger.Fatal("Failed to get DATABASE_URL", zap.Error(err))
		}
	}

	// Database Pool Initialization
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Unable to parse database DSN", zap.Error(err))
	}
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 15
	connPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool", zap.Error(err))
	}

	dbQueries := db.New(connPool)

	// Get Payment Sync Encryption Key
	paymentSyncEncryptionKey, err := secretsClient.GetSecretString(ctx, "PAYMENT_SYNC_ENCRYPTION_KEY_ARN", "PAYMENT_SYNC_ENCRYPTION_KEY")
	if err != nil || paymentSyncEncryptionKey == "" {
		logger.Fatal("Failed to get Payment Sync Encryption Key", zap.Error(err))
	}

	// Initialize Payment Sync Client
	paymentSyncClient := payment_sync.NewPaymentSyncClient(dbQueries, logger.Log, paymentSyncEncryptionKey)

	// Get configuration from environment
	maxRetries := 5
	if maxRetriesStr := os.Getenv("DLQ_MAX_RETRIES"); maxRetriesStr != "" {
		if parsed, err := strconv.Atoi(maxRetriesStr); err == nil {
			maxRetries = parsed
		}
	}

	retryBackoffMs := 60000 // 1 minute default
	if backoffStr := os.Getenv("DLQ_RETRY_BACKOFF_MS"); backoffStr != "" {
		if parsed, err := strconv.Atoi(backoffStr); err == nil {
			retryBackoffMs = parsed
		}
	}

	// Create Application Instance
	app := &Application{
		db:                dbQueries,
		logger:            logger.Log,
		paymentSyncClient: paymentSyncClient,
		maxRetries:        maxRetries,
		retryBackoffMs:    retryBackoffMs,
	}

	// Start the Lambda Handler
	lambda.Start(app.HandleSQSEvent)
}
