package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	awsclient "cyphera-api/internal/client/aws"
	"cyphera-api/internal/client/payment_sync"
	"cyphera-api/internal/db"
	"cyphera-api/internal/helpers"
	"cyphera-api/internal/logger"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Application holds all dependencies for the webhook receiver Lambda handler
type Application struct {
	paymentSyncClient *payment_sync.PaymentSyncClient
	sqsClient         *sqs.Client
	sqsQueueURL       string
	dbQueries         *db.Queries
}

// HandleAPIGatewayRequest processes incoming webhook requests from API Gateway
// @godoc HandleAPIGatewayRequest processes payment provider webhook requests via API Gateway
func (app *Application) HandleAPIGatewayRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.Info("Webhook receiver handling API Gateway request",
		zap.String("path", request.Path),
		zap.String("method", request.HTTPMethod))

	// Extract provider from path (e.g., /webhooks/stripe)
	provider := request.PathParameters["provider"]
	if provider == "" {
		logger.Error("No provider specified in webhook path")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "provider not specified"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// Get signature header (provider-specific)
	var signatureHeader string
	switch provider {
	case "stripe":
		signatureHeader = request.Headers["stripe-signature"]
		if signatureHeader == "" {
			signatureHeader = request.Headers["Stripe-Signature"] // Try capitalized version
		}
	default:
		logger.Error("Unsupported provider", zap.String("provider", provider))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "unsupported provider"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	if signatureHeader == "" {
		logger.Error("Missing signature header",
			zap.String("provider", provider))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "missing signature header"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// For webhook processing, we need to determine the workspace
	// We'll do initial webhook validation to get the account ID, then resolve workspace
	workspaceID, err := app.resolveWorkspaceFromWebhook(ctx, provider, []byte(request.Body), signatureHeader)
	if err != nil {
		logger.Error("Failed to resolve workspace from webhook",
			zap.String("provider", provider),
			zap.Error(err))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "failed to resolve workspace"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// Get the provider service configured for this workspace
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, workspaceID, provider)
	if err != nil {
		logger.Error("Failed to get provider service",
			zap.String("provider", provider),
			zap.String("workspace_id", workspaceID),
			zap.Error(err))
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error": "provider service not available"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// Validate and parse webhook using the provider service
	webhookEvent, err := providerService.HandleWebhook(
		ctx,
		[]byte(request.Body),
		signatureHeader,
	)
	if err != nil {
		logger.Error("Failed to handle webhook",
			zap.String("provider", provider),
			zap.String("workspace_id", workspaceID),
			zap.Error(err))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "webhook validation failed"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// Add workspace context to the webhook event
	enhancedEvent := payment_sync.WebhookEvent{
		ProviderEventID: webhookEvent.ProviderEventID,
		Provider:        webhookEvent.Provider,
		EventType:       webhookEvent.EventType,
		ReceivedAt:      webhookEvent.ReceivedAt,
		Data:            webhookEvent.Data,
		RawData:         webhookEvent.RawData,
		SignatureValid:  webhookEvent.SignatureValid,
	}

	// Queue the webhook event for processing
	err = app.queueWebhookEvent(ctx, enhancedEvent, workspaceID)
	if err != nil {
		logger.Error("Failed to queue webhook event",
			zap.String("provider", provider),
			zap.String("workspace_id", workspaceID),
			zap.String("event_id", webhookEvent.ProviderEventID),
			zap.Error(err))
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error": "failed to queue event"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	logger.Info("Successfully processed webhook",
		zap.String("provider", provider),
		zap.String("workspace_id", workspaceID),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"status": "received"}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

// resolveWorkspaceFromWebhook determines the workspace for a webhook event
func (app *Application) resolveWorkspaceFromWebhook(ctx context.Context, provider string, body []byte, signature string) (string, error) {
	switch provider {
	case "stripe":
		return app.resolveWorkspaceFromStripeWebhook(ctx, body, signature)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// resolveWorkspaceFromStripeWebhook extracts Stripe account ID and resolves workspace
func (app *Application) resolveWorkspaceFromStripeWebhook(ctx context.Context, body []byte, signature string) (string, error) {
	// Parse the Stripe event to get the account ID
	var stripeEvent struct {
		Account string `json:"account"`
		ID      string `json:"id"`
	}

	if err := json.Unmarshal(body, &stripeEvent); err != nil {
		return "", fmt.Errorf("failed to parse stripe webhook body: %w", err)
	}

	// For Connect webhooks, the account field will contain the connected account ID
	// For direct webhooks, it may be empty (meaning it's for the platform account)
	accountID := stripeEvent.Account
	if accountID == "" {
		// This is a webhook for the main platform account
		// We need to determine the environment from other means or use a default
		// For now, we'll look for a "platform" account mapping
		accountID = "platform"
	}

	// Determine environment (live vs test) from the webhook
	// Stripe test events start with "evt_test_" while live events start with "evt_"
	environment := "live"
	if strings.HasPrefix(stripeEvent.ID, "evt_test_") {
		environment = "test"
	}

	logger.Info("Resolving workspace for Stripe webhook",
		zap.String("account_id", accountID),
		zap.String("environment", environment),
		zap.String("event_id", stripeEvent.ID))

	// Look up workspace using the provider accounts table
	providerAccount, err := app.dbQueries.GetWorkspaceProviderAccountForWebhook(ctx, db.GetWorkspaceProviderAccountForWebhookParams{
		ProviderName:      "stripe",
		ProviderAccountID: accountID,
		Environment:       environment,
	})
	if err != nil {
		return "", fmt.Errorf("no workspace found for stripe account %s in %s environment: %w", accountID, environment, err)
	}

	logger.Info("Workspace resolved successfully",
		zap.String("workspace_id", providerAccount.WorkspaceID.String()),
		zap.String("workspace_name", providerAccount.WorkspaceName),
		zap.String("account_type", providerAccount.AccountType))

	return providerAccount.WorkspaceID.String(), nil
}

// queueWebhookEvent sends the webhook event to SQS for processing
func (app *Application) queueWebhookEvent(ctx context.Context, webhookEvent payment_sync.WebhookEvent, workspaceID string) error {
	// Serialize webhook event
	eventBytes, err := json.Marshal(webhookEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook event: %w", err)
	}

	// Send to SQS
	_, err = app.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &app.sqsQueueURL,
		MessageBody: &[]string{string(eventBytes)}[0],
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Provider": {
				StringValue: &webhookEvent.Provider,
				DataType:    &[]string{"String"}[0],
			},
			"EventType": {
				StringValue: &webhookEvent.EventType,
				DataType:    &[]string{"String"}[0],
			},
			"WorkspaceID": {
				StringValue: &workspaceID,
				DataType:    &[]string{"String"}[0],
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}

	return nil
}

// LocalHandleRequest handles local testing
func (app *Application) LocalHandleRequest(ctx context.Context) error {
	logger.Info("Webhook receiver running in local mode")
	// For local testing, just log that the service is ready
	logger.Info("Webhook receiver initialized successfully")
	return nil
}

func main() {
	// Load .env file for local development
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v. Proceeding with environment variables/secrets.", err)
	}

	stage := os.Getenv("STAGE")
	if stage == "" {
		stage = helpers.StageLocal
		log.Printf("Warning: STAGE environment variable not set, defaulting to '%s'", stage)
	}
	if !helpers.IsValidStage(stage) {
		log.Fatalf("Invalid STAGE environment variable: '%s'. Must be one of: %s, %s, %s",
			stage, helpers.StageProd, helpers.StageDev, helpers.StageLocal)
	}

	// Initialize logger (AFTER stage validation)
	logger.InitLogger(stage)
	logger.Info("Lambda Cold Start: Initializing webhook receiver for stage", zap.String("stage", stage))
	defer func() {
		// Sync logger before exit (important for Lambda)
		_ = logger.Sync()
	}()

	ctx := context.Background()

	// --- Initialize AWS Secrets Manager Client ---
	secretsClient, err := awsclient.NewSecretsManagerClient(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize AWS Secrets Manager client", zap.Error(err))
	}

	// --- Database Connection Setup ---
	var dsn string
	if stage == helpers.StageProd || stage == helpers.StageDev {
		logger.Info("Running in deployed stage, fetching DB credentials from Secrets Manager", zap.String("stage", stage))
		dbEndpoint := os.Getenv("DB_HOST")
		dbName := os.Getenv("DB_NAME")
		dbSecretArn := os.Getenv("RDS_SECRET_ARN")
		dbSSLMode := os.Getenv("DB_SSLMODE")

		if dbEndpoint == "" || dbName == "" || dbSecretArn == "" {
			logger.Fatal("Missing required DB environment variables for deployed environment (DB_HOST, DB_NAME, RDS_SECRET_ARN)")
		}
		if dbSSLMode == "" {
			dbSSLMode = "require"
			logger.Warn("DB_SSLMODE not set, defaulting to 'require'")
		}

		type RdsSecret struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var secretData RdsSecret
		err = secretsClient.GetSecretJSON(ctx, "RDS_SECRET_ARN", "", &secretData)
		if err != nil {
			logger.Fatal("Failed to retrieve or parse RDS secret", zap.Error(err), zap.String("secretArnEnvVar", "RDS_SECRET_ARN"))
		}
		if secretData.Username == "" || secretData.Password == "" {
			logger.Fatal("Username or password not found in RDS secret data", zap.String("secretArnEnvVar", "RDS_SECRET_ARN"))
		}

		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
			url.QueryEscape(secretData.Username), url.QueryEscape(secretData.Password),
			dbEndpoint, dbName, dbSSLMode)
		logger.Info("Constructed DSN from Secrets Manager credentials")
	} else {
		// Local
		logger.Info("Running in local stage, using DATABASE_URL from env/secrets")
		dsn, err = secretsClient.GetSecretString(ctx, "DATABASE_URL_ARN", "DATABASE_URL")
		if err != nil {
			logger.Fatal("Failed to get DATABASE_URL", zap.Error(err))
		}
		if dsn == "" {
			logger.Fatal("DATABASE_URL is required for local development and not found")
		}
	}

	// --- Database Pool Initialization ---
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

	// --- Get Payment Sync Encryption Key ---
	// Only need encryption key for workspace credential management
	paymentSyncEncryptionKey, err := secretsClient.GetSecretString(ctx, "PAYMENT_SYNC_ENCRYPTION_KEY_ARN", "PAYMENT_SYNC_ENCRYPTION_KEY")
	if err != nil || paymentSyncEncryptionKey == "" {
		logger.Fatal("Failed to get Payment Sync Encryption Key", zap.Error(err))
	}

	// --- Initialize Payment Sync Client ---
	// Note: No global Stripe service needed - webhooks are routed by workspace resolution
	paymentSyncClient := payment_sync.NewPaymentSyncClient(dbQueries, logger.Log, paymentSyncEncryptionKey)

	// --- Initialize SQS Client (for deployed stages) ---
	var sqsClient *sqs.Client
	sqsQueueURL := os.Getenv("SQS_QUEUE_URL")

	if stage != helpers.StageLocal {
		if sqsQueueURL == "" {
			logger.Fatal("SQS_QUEUE_URL environment variable is required for deployed stages")
		}

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			logger.Fatal("Failed to load AWS config", zap.Error(err))
		}
		sqsClient = sqs.NewFromConfig(cfg)
	}

	// --- Create Application Instance ---
	app := &Application{
		paymentSyncClient: paymentSyncClient,
		sqsClient:         sqsClient,
		sqsQueueURL:       sqsQueueURL,
		dbQueries:         dbQueries,
	}

	if stage == helpers.StageLocal {
		err := app.LocalHandleRequest(ctx)
		if err != nil {
			logger.Fatal("Error in LocalHandleRequest", zap.Error(err))
		}
	} else {
		// --- Start the Lambda Handler ---
		lambda.Start(app.HandleAPIGatewayRequest)
	}
}
