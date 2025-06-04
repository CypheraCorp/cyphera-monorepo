package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	awsclient "cyphera-api/internal/client/aws"
	"cyphera-api/internal/client/payment_sync"
	ps "cyphera-api/internal/client/payment_sync"
	"cyphera-api/internal/client/payment_sync/stripe"
	"cyphera-api/internal/db"
	"cyphera-api/internal/helpers"
	"cyphera-api/internal/logger"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Application holds all dependencies for the webhook processor Lambda handler
type Application struct {
	paymentSyncClient *payment_sync.PaymentSyncClient
	dbQueries         *db.Queries
	logger            *zap.Logger
}

// HandleSQSEvent processes webhook events from SQS
// @godoc HandleSQSEvent processes payment provider webhook events from SQS queue
func (app *Application) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	logger.Info("Webhook processor handling SQS event",
		zap.Int("record_count", len(event.Records)))

	for _, record := range event.Records {
		err := app.processWebhookRecord(ctx, record)
		if err != nil {
			logger.Error("Failed to process webhook record",
				zap.String("message_id", record.MessageId),
				zap.Error(err))
			// Continue processing other records, but return error to indicate partial failure
			// SQS will handle retries for failed messages
			return fmt.Errorf("failed to process message %s: %w", record.MessageId, err)
		}
	}

	logger.Info("Successfully processed all webhook records",
		zap.Int("count", len(event.Records)))
	return nil
}

// processWebhookRecord processes a single SQS record containing a webhook event
func (app *Application) processWebhookRecord(ctx context.Context, record events.SQSMessage) error {
	logger.Info("Processing webhook record",
		zap.String("message_id", record.MessageId),
		zap.String("source_queue", record.EventSourceARN))

	// Extract workspace ID from message attributes
	var workspaceID string
	if workspaceAttr, exists := record.MessageAttributes["WorkspaceID"]; exists && workspaceAttr.StringValue != nil {
		workspaceID = *workspaceAttr.StringValue
	} else {
		return fmt.Errorf("missing workspace ID in message attributes")
	}

	// Parse webhook event from message body
	var webhookEvent payment_sync.WebhookEvent
	err := json.Unmarshal([]byte(record.Body), &webhookEvent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal webhook event: %w", err)
	}

	logger.Info("Processing webhook event",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", webhookEvent.Provider),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	// Process based on event type and data
	err = app.processWebhookEventData(ctx, workspaceID, webhookEvent)
	if err != nil {
		return fmt.Errorf("failed to process webhook event data: %w", err)
	}

	logger.Info("Successfully processed webhook event",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", webhookEvent.Provider),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	return nil
}

// processWebhookEventData processes the webhook event data and updates the database
func (app *Application) processWebhookEventData(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Check for duplicate webhook events using idempotency key
	if err := app.checkAndLogWebhookEvent(ctx, workspaceID, webhookEvent); err != nil {
		if err.Error() == "duplicate_event" {
			// Event already processed, skip it
			logger.Info("Skipping duplicate webhook event",
				zap.String("workspace_id", workspaceID),
				zap.String("provider", webhookEvent.Provider),
				zap.String("event_id", webhookEvent.ProviderEventID))
			return nil
		}
		return fmt.Errorf("failed to check webhook event: %w", err)
	}

	switch webhookEvent.Provider {
	case "stripe":
		return app.processStripeWebhookEvent(ctx, workspaceID, webhookEvent)
	default:
		return fmt.Errorf("unsupported provider: %s", webhookEvent.Provider)
	}
}

// checkAndLogWebhookEvent checks for duplicates and logs the webhook event
func (app *Application) checkAndLogWebhookEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Create idempotency key: workspace_id + provider + event_id
	idempotencyKey := fmt.Sprintf("%s_%s_%s", workspaceID, webhookEvent.Provider, webhookEvent.ProviderEventID)

	// Check for existing event by idempotency key
	_, err = app.dbQueries.GetWebhookEventByIdempotencyKey(ctx, db.GetWebhookEventByIdempotencyKeyParams{
		WorkspaceID:    wsID,
		ProviderName:   webhookEvent.Provider,
		IdempotencyKey: pgtype.Text{String: idempotencyKey, Valid: true},
	})

	if err == nil {
		// Event already exists, return duplicate error
		return fmt.Errorf("duplicate_event")
	}

	// Check for existing event by provider event ID as backup
	_, err = app.dbQueries.GetWebhookEventByProviderEventID(ctx, db.GetWebhookEventByProviderEventIDParams{
		WorkspaceID:    wsID,
		ProviderName:   webhookEvent.Provider,
		WebhookEventID: pgtype.Text{String: webhookEvent.ProviderEventID, Valid: true},
	})

	if err == nil {
		// Event already exists, return duplicate error
		return fmt.Errorf("duplicate_event")
	}

	// Log the incoming webhook event (this prevents duplicates)
	eventDetails, _ := json.Marshal(map[string]interface{}{
		"event_data":      webhookEvent.Data,
		"signature_valid": webhookEvent.SignatureValid,
		"received_at":     webhookEvent.ReceivedAt,
	})

	_, err = app.dbQueries.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
		WorkspaceID:        wsID,
		ProviderName:       webhookEvent.Provider,
		EntityType:         "webhook",
		EventType:          "webhook_received",
		EventMessage:       pgtype.Text{String: fmt.Sprintf("Received %s webhook event", webhookEvent.EventType), Valid: true},
		EventDetails:       eventDetails,
		WebhookEventID:     pgtype.Text{String: webhookEvent.ProviderEventID, Valid: true},
		IdempotencyKey:     pgtype.Text{String: idempotencyKey, Valid: true},
		ProcessingAttempts: pgtype.Int4{Int32: 0, Valid: true},
		SignatureValid:     pgtype.Bool{Bool: webhookEvent.SignatureValid, Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to log webhook event: %w", err)
	}

	return nil
}

// processStripeWebhookEvent processes Stripe-specific webhook events
func (app *Application) processStripeWebhookEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	logger.Info("Processing Stripe webhook event",
		zap.String("workspace_id", workspaceID),
		zap.String("event_type", webhookEvent.EventType),
		zap.String("event_id", webhookEvent.ProviderEventID))

	// Get the configured Stripe service for this workspace
	_, err := app.paymentSyncClient.GetProviderService(ctx, workspaceID, "stripe")
	if err != nil {
		return fmt.Errorf("failed to get stripe service for workspace %s: %w", workspaceID, err)
	}

	switch webhookEvent.EventType {
	case "customer.created", "customer.updated":
		return app.processCustomerEvent(ctx, workspaceID, webhookEvent)
	case "customer.deleted":
		return app.processCustomerDeletedEvent(ctx, workspaceID, webhookEvent)
	case "product.created", "product.updated":
		return app.processProductEvent(ctx, workspaceID, webhookEvent)
	case "price.created", "price.updated":
		return app.processPriceEvent(ctx, workspaceID, webhookEvent)
	case "invoice.created", "invoice.updated", "invoice.paid", "invoice.payment_failed":
		return app.processInvoiceEvent(ctx, workspaceID, webhookEvent)
	case "subscription.created", "subscription.updated", "subscription.deleted":
		return app.processSubscriptionEvent(ctx, workspaceID, webhookEvent)
	default:
		logger.Info("Ignoring unhandled event type",
			zap.String("workspace_id", workspaceID),
			zap.String("event_type", webhookEvent.EventType))
		return nil
	}
}

// processCustomerEvent handles customer creation and updates
func (app *Application) processCustomerEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Type assertion to get customer data
	var customer payment_sync.Customer

	// The webhook data might come as a map[string]interface{} from JSON unmarshaling
	// or might already be a payment_sync.Customer from the HandleWebhook mapping
	switch data := webhookEvent.Data.(type) {
	case payment_sync.Customer:
		customer = data
	case map[string]interface{}:
		// Convert map to Customer struct via JSON marshaling
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal customer data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &customer); err != nil {
			return fmt.Errorf("failed to unmarshal customer data: %w", err)
		}
	default:
		return fmt.Errorf("invalid customer data type: %T", webhookEvent.Data)
	}

	logger.Info("Processing customer event",
		zap.String("workspace_id", workspaceID),
		zap.String("customer_id", customer.ExternalID),
		zap.String("email", customer.Email))

	// Get the Stripe service instance for this workspace
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, workspaceID, "stripe")
	if err != nil {
		return fmt.Errorf("failed to get stripe service: %w", err)
	}

	// Type assert to StripeService to access upsert methods
	stripeService, ok := providerService.(*stripe.StripeService)
	if !ok {
		return fmt.Errorf("invalid stripe service type")
	}

	// Parse workspace ID for session
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Create a minimal session for webhook processing
	// Note: In production, you might want to create/reuse a session per webhook batch
	session := &db.PaymentSyncSession{
		ID:           uuid.New(), // Generate a new ID for this session
		WorkspaceID:  wsID,
		ProviderName: "stripe",
		SessionType:  "webhook_sync",
		Status:       "running",
	}

	// Call the existing upsert function
	if err := stripeService.UpsertCustomer(ctx, session, customer); err != nil {
		return fmt.Errorf("failed to upsert customer: %w", err)
	}

	logger.Info("Customer event processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("customer_id", customer.ExternalID))

	return nil
}

// processCustomerDeletedEvent handles customer deletion
func (app *Application) processCustomerDeletedEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Type assertion to get customer data
	var customer payment_sync.Customer

	switch data := webhookEvent.Data.(type) {
	case payment_sync.Customer:
		customer = data
	case map[string]interface{}:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal customer data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &customer); err != nil {
			return fmt.Errorf("failed to unmarshal customer data: %w", err)
		}
	default:
		return fmt.Errorf("invalid customer data type: %T", webhookEvent.Data)
	}

	logger.Info("Processing customer deletion event",
		zap.String("workspace_id", workspaceID),
		zap.String("customer_id", customer.ExternalID))

	// Parse workspace ID
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Find the customer by external ID and soft delete
	existingCustomer, err := app.dbQueries.GetCustomerByExternalID(ctx, db.GetCustomerByExternalIDParams{
		WorkspaceID: wsID,
		ExternalID:  pgtype.Text{String: customer.ExternalID, Valid: true},
	})
	if err != nil {
		logger.Warn("Customer not found for deletion",
			zap.String("workspace_id", workspaceID),
			zap.String("external_id", customer.ExternalID))
		return nil // Not an error if customer doesn't exist
	}

	// Soft delete the customer
	err = app.dbQueries.DeleteCustomer(ctx, db.DeleteCustomerParams{
		ID:          existingCustomer.ID,
		WorkspaceID: wsID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	logger.Info("Customer deletion event processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("customer_id", customer.ExternalID))

	return nil
}

// processProductEvent handles product creation and updates
func (app *Application) processProductEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Type assertion to get product data
	var product payment_sync.Product

	switch data := webhookEvent.Data.(type) {
	case payment_sync.Product:
		product = data
	case map[string]interface{}:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal product data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &product); err != nil {
			return fmt.Errorf("failed to unmarshal product data: %w", err)
		}
	default:
		return fmt.Errorf("invalid product data type: %T", webhookEvent.Data)
	}

	logger.Info("Processing product event",
		zap.String("workspace_id", workspaceID),
		zap.String("product_id", product.ExternalID),
		zap.String("name", product.Name))

	// Get the Stripe service instance
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, workspaceID, "stripe")
	if err != nil {
		return fmt.Errorf("failed to get stripe service: %w", err)
	}

	stripeService, ok := providerService.(*stripe.StripeService)
	if !ok {
		return fmt.Errorf("invalid stripe service type")
	}

	// Parse workspace ID for session
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Create a minimal session for webhook processing
	session := &db.PaymentSyncSession{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ProviderName: "stripe",
		SessionType:  "webhook_sync",
		Status:       "running",
	}

	// Call the existing upsert function
	if err := stripeService.UpsertProduct(ctx, session, product); err != nil {
		return fmt.Errorf("failed to upsert product: %w", err)
	}

	logger.Info("Product event processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("product_id", product.ExternalID))

	return nil
}

// processPriceEvent handles price creation and updates
func (app *Application) processPriceEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Type assertion to get price data
	var price payment_sync.Price

	switch data := webhookEvent.Data.(type) {
	case payment_sync.Price:
		price = data
	case map[string]interface{}:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal price data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &price); err != nil {
			return fmt.Errorf("failed to unmarshal price data: %w", err)
		}
	default:
		return fmt.Errorf("invalid price data type: %T", webhookEvent.Data)
	}

	logger.Info("Processing price event",
		zap.String("workspace_id", workspaceID),
		zap.String("price_id", price.ExternalID),
		zap.String("product_id", price.ProductID))

	// Get the Stripe service instance
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, workspaceID, "stripe")
	if err != nil {
		return fmt.Errorf("failed to get stripe service: %w", err)
	}

	stripeService, ok := providerService.(*stripe.StripeService)
	if !ok {
		return fmt.Errorf("invalid stripe service type")
	}

	// Parse workspace ID for session
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Create a minimal session for webhook processing
	session := &db.PaymentSyncSession{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ProviderName: "stripe",
		SessionType:  "webhook_sync",
		Status:       "running",
	}

	// Call the existing upsert function
	if err := stripeService.UpsertPrice(ctx, session, price); err != nil {
		return fmt.Errorf("failed to upsert price: %w", err)
	}

	logger.Info("Price event processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("price_id", price.ExternalID))

	return nil
}

// processInvoiceEvent handles invoice events
func (app *Application) processInvoiceEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Type assertion to get invoice data
	var invoice payment_sync.Invoice

	switch data := webhookEvent.Data.(type) {
	case payment_sync.Invoice:
		invoice = data
	case map[string]interface{}:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal invoice data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &invoice); err != nil {
			return fmt.Errorf("failed to unmarshal invoice data: %w", err)
		}
	default:
		return fmt.Errorf("invalid invoice data type: %T", webhookEvent.Data)
	}

	logger.Info("Processing invoice event",
		zap.String("workspace_id", workspaceID),
		zap.String("invoice_id", invoice.ExternalID),
		zap.String("status", invoice.Status),
		zap.String("customer_id", invoice.CustomerID))

	// For now, we'll just log invoice events as they might need special handling
	// based on the invoice status (paid, failed, etc.)
	// You can implement invoice-specific logic here

	logger.Info("Invoice event processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("invoice_id", invoice.ExternalID))

	return nil
}

// processSubscriptionEvent handles subscription events
func (app *Application) processSubscriptionEvent(ctx context.Context, workspaceID string, webhookEvent payment_sync.WebhookEvent) error {
	// Type assertion to get subscription data
	var subscription payment_sync.Subscription

	switch data := webhookEvent.Data.(type) {
	case payment_sync.Subscription:
		subscription = data
	case map[string]interface{}:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal subscription data: %w", err)
		}
		if err := json.Unmarshal(jsonData, &subscription); err != nil {
			return fmt.Errorf("failed to unmarshal subscription data: %w", err)
		}
	default:
		return fmt.Errorf("invalid subscription data type: %T", webhookEvent.Data)
	}

	logger.Info("Processing subscription event",
		zap.String("workspace_id", workspaceID),
		zap.String("subscription_id", subscription.ExternalID),
		zap.String("status", subscription.Status),
		zap.String("customer_id", subscription.CustomerID))

	// Get the Stripe service instance
	providerService, err := app.paymentSyncClient.GetProviderService(ctx, workspaceID, "stripe")
	if err != nil {
		return fmt.Errorf("failed to get stripe service: %w", err)
	}

	stripeService, ok := providerService.(*stripe.StripeService)
	if !ok {
		return fmt.Errorf("invalid stripe service type")
	}

	// Parse workspace ID for session
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Create a minimal session for webhook processing
	session := &db.PaymentSyncSession{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ProviderName: "stripe",
		SessionType:  "webhook_sync",
		Status:       "running",
	}

	// Call the existing upsert function
	if err := stripeService.UpsertSubscription(ctx, session, subscription); err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}

	logger.Info("Subscription event processed successfully",
		zap.String("workspace_id", workspaceID),
		zap.String("subscription_id", subscription.ExternalID))

	return nil
}

// LocalHandleRequest handles local testing
func (app *Application) LocalHandleRequest(ctx context.Context) error {
	logger.Info("Webhook processor running in local mode")
	// For local testing, just log that the service is ready
	logger.Info("Webhook processor initialized successfully")
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
	logger.Info("Lambda Cold Start: Initializing webhook processor for stage", zap.String("stage", stage))
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
	poolConfig.MaxConns = 10 // More connections for processing
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30
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
	// Note: No global Stripe service needed - workspace-specific services are created dynamically
	paymentSyncClient := payment_sync.NewPaymentSyncClient(dbQueries, logger.Log, paymentSyncEncryptionKey)

	// --- Create Application Instance ---
	app := &Application{
		paymentSyncClient: paymentSyncClient,
		dbQueries:         dbQueries,
		logger:            logger.Log,
	}

	if stage == helpers.StageLocal {
		err := app.LocalHandleRequest(ctx)
		if err != nil {
			logger.Fatal("Error in LocalHandleRequest", zap.Error(err))
		}
	} else {
		// --- Start the Lambda Handler ---
		lambda.Start(app.HandleSQSEvent)
	}
}

// convertStripeInvoiceToPaymentSync converts Stripe invoice data to payment sync Invoice format
func convertStripeInvoiceToPaymentSync(invoiceData map[string]interface{}) (*ps.Invoice, error) {
	// Extract required fields
	id, ok := invoiceData["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing invoice id")
	}

	status, ok := invoiceData["status"].(string)
	if !ok {
		return nil, fmt.Errorf("missing invoice status")
	}

	currency, ok := invoiceData["currency"].(string)
	if !ok {
		return nil, fmt.Errorf("missing invoice currency")
	}

	// Helper function to safely extract int64 from interface{}
	extractInt64 := func(data map[string]interface{}, key string) int64 {
		if val, ok := data[key]; ok {
			switch v := val.(type) {
			case float64:
				return int64(v)
			case int64:
				return v
			case int:
				return int64(v)
			}
		}
		return 0
	}

	// Helper function to safely extract string from interface{}
	extractString := func(data map[string]interface{}, key string) string {
		if val, ok := data[key].(string); ok {
			return val
		}
		return ""
	}

	// Helper function to safely extract bool from interface{}
	extractBool := func(data map[string]interface{}, key string) bool {
		if val, ok := data[key].(bool); ok {
			return val
		}
		return false
	}

	// Helper function to safely extract Unix timestamp
	extractUnixTimestamp := func(data map[string]interface{}, key string) int64 {
		if val, ok := data[key]; ok && val != nil {
			return extractInt64(data, key)
		}
		return 0
	}

	// Extract optional fields
	amountDue := extractInt64(invoiceData, "amount_due")
	amountPaid := extractInt64(invoiceData, "amount_paid")
	amountRemaining := extractInt64(invoiceData, "amount_remaining")
	tax := extractInt64(invoiceData, "tax")
	attemptCount := extractInt64(invoiceData, "attempt_count")

	customerID := extractString(invoiceData, "customer")
	subscriptionID := extractString(invoiceData, "subscription")
	collectionMethod := extractString(invoiceData, "collection_method")
	invoicePDF := extractString(invoiceData, "invoice_pdf")
	hostedInvoiceURL := extractString(invoiceData, "hosted_invoice_url")
	chargeID := extractString(invoiceData, "charge")
	paymentIntentID := extractString(invoiceData, "payment_intent")
	billingReason := extractString(invoiceData, "billing_reason")

	paidOutOfBand := extractBool(invoiceData, "paid_out_of_band")

	dueDate := extractUnixTimestamp(invoiceData, "due_date")
	paidAt := extractUnixTimestamp(invoiceData, "status_transitions.paid_at")
	nextPaymentAttempt := extractUnixTimestamp(invoiceData, "next_payment_attempt")

	// Extract line items
	var lines []ps.InvoiceLineItem
	if linesData, ok := invoiceData["lines"].(map[string]interface{}); ok {
		if dataArray, ok := linesData["data"].([]interface{}); ok {
			for _, lineData := range dataArray {
				if lineMap, ok := lineData.(map[string]interface{}); ok {
					line := ps.InvoiceLineItem{
						ID:          extractString(lineMap, "id"),
						Amount:      extractInt64(lineMap, "amount"),
						Description: extractString(lineMap, "description"),
						Quantity:    int(extractInt64(lineMap, "quantity")),
					}
					lines = append(lines, line)
				}
			}
		}
	}

	// Extract total tax amounts
	var totalTaxAmounts []ps.TaxAmount
	if taxAmountsData, ok := invoiceData["total_tax_amounts"].([]interface{}); ok {
		for _, taxData := range taxAmountsData {
			if taxMap, ok := taxData.(map[string]interface{}); ok {
				taxAmount := ps.TaxAmount{
					Amount: extractInt64(taxMap, "amount"),
					RateID: extractString(taxMap, "tax_rate"),
				}
				totalTaxAmounts = append(totalTaxAmounts, taxAmount)
			}
		}
	}

	// Extract metadata and convert to map[string]string
	metadata := make(map[string]string)
	if metadataData, ok := invoiceData["metadata"].(map[string]interface{}); ok {
		for key, value := range metadataData {
			if strValue, ok := value.(string); ok {
				metadata[key] = strValue
			}
		}
	}

	return &ps.Invoice{
		ExternalID:         id,
		CustomerID:         customerID,
		SubscriptionID:     subscriptionID,
		Status:             status,
		CollectionMethod:   collectionMethod,
		AmountDue:          amountDue,
		AmountPaid:         amountPaid,
		AmountRemaining:    amountRemaining,
		Currency:           currency,
		DueDate:            dueDate,
		PaidAt:             paidAt,
		InvoicePDF:         invoicePDF,
		HostedInvoiceURL:   hostedInvoiceURL,
		ChargeID:           chargeID,
		PaymentIntentID:    paymentIntentID,
		Lines:              lines,
		Tax:                tax,
		TotalTaxAmounts:    totalTaxAmounts,
		BillingReason:      billingReason,
		PaidOutOfBand:      paidOutOfBand,
		AttemptCount:       int(attemptCount),
		NextPaymentAttempt: nextPaymentAttempt,
		Metadata:           metadata,
	}, nil
}
