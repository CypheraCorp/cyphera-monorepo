package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/apps/subscription-processor/internal/processor"
	awsclient "github.com/cyphera/cyphera-api/libs/go/client/aws"
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Application holds all dependencies for the Lambda handler
type Application struct {
	subscriptionProcessor     *processor.SubscriptionProcessor
	scheduledChangesProcessor *processor.ScheduledChangesProcessor
	// Add other dependencies here if HandleRequest needs them directly
	// e.g., dbPool *pgxpool.Pool, delegationClient *dsClient.DelegationClient
	failureDetector *services.PaymentFailureDetector
	dunningService  *services.DunningService
}

// HandleRequest is the actual Lambda handler function
// It now belongs to the Application struct to access dependencies.
func (app *Application) HandleRequest(ctx context.Context /*, event MyEvent - if you have a specific event type */) error {
	logger.Info("Entering HandleRequest for subscription processing") // Using structured logger

	// --- Process Subscriptions ---
	logger.Info("Starting subscription processing...")
	results, err := app.subscriptionProcessor.ProcessDueSubscriptions(ctx)
	if err != nil {
		logger.Error("Error processing subscriptions in HandleRequest", zap.Error(err))
		return fmt.Errorf("HandleRequest: error from ProcessDueSubscriptions: %w", err) // Return error to Lambda runtime
	}

	logger.Info("Subscription processing results in HandleRequest",
		zap.Int("total", results.Total),
		zap.Int("succeeded", results.Succeeded),
		zap.Int("failed", results.Failed),
	)

	// --- Detect Failed Payments and Create Dunning Campaigns ---
	if results.Failed > 0 {
		logger.Info("Detecting failed payments and creating dunning campaigns...")

		// Look back 10 minutes for failed payments (adjust based on Lambda schedule)
		detectionResult, err := app.failureDetector.DetectAndCreateCampaigns(ctx, 10)
		if err != nil {
			logger.Error("Error detecting failed payments", zap.Error(err))
			// Don't fail the entire Lambda execution, just log the error
		} else {
			logger.Info("Failed payment detection results",
				zap.Int("new_campaigns", detectionResult.NewCampaigns),
				zap.Int("updated_campaigns", detectionResult.UpdatedCampaigns),
				zap.Int("failed_detections", detectionResult.FailedDetections),
				zap.Int("campaign_ids", len(detectionResult.CampaignIDs)),
				zap.Int("errors", len(detectionResult.Errors)),
			)
		}
	}

	// --- Process Scheduled Subscription Changes ---
	if app.scheduledChangesProcessor != nil {
		logger.Info("Processing scheduled subscription changes...")
		app.scheduledChangesProcessor.ProcessChanges()
	}

	logger.Info("Subscription processing finished successfully in HandleRequest.")
	return nil // Indicate successful execution to Lambda runtime
}

func (a *Application) LocalHandleRequest(ctx context.Context) error {
	logger.Info("Entering LocalHandleRequest for subscription processing")

	// --- Process Subscriptions ---
	logger.Info("Starting subscription processing...")
	results, err := a.subscriptionProcessor.ProcessDueSubscriptions(ctx)
	if err != nil {
		logger.Error("Error processing subscriptions in LocalHandleRequest", zap.Error(err))
		return fmt.Errorf("LocalHandleRequest: error from ProcessDueSubscriptions: %w", err) // Return error to Lambda runtime
	}

	logger.Info("Subscription processing results in LocalHandleRequest",
		zap.Int("total", results.Total),
		zap.Int("succeeded", results.Succeeded),
		zap.Int("failed", results.Failed),
	)

	// --- Detect Failed Payments and Create Dunning Campaigns ---
	if results.Failed > 0 {
		logger.Info("Detecting failed payments and creating dunning campaigns...")

		// Look back 10 minutes for failed payments (adjust based on Lambda schedule)
		detectionResult, err := a.failureDetector.DetectAndCreateCampaigns(ctx, 10)
		if err != nil {
			logger.Error("Error detecting failed payments", zap.Error(err))
			// Don't fail the entire execution, just log the error
		} else {
			logger.Info("Failed payment detection results",
				zap.Int("new_campaigns", detectionResult.NewCampaigns),
				zap.Int("updated_campaigns", detectionResult.UpdatedCampaigns),
				zap.Int("failed_detections", detectionResult.FailedDetections),
				zap.Int("campaign_ids", len(detectionResult.CampaignIDs)),
				zap.Int("errors", len(detectionResult.Errors)),
			)
		}
	}

	// --- Process Scheduled Subscription Changes ---
	if a.scheduledChangesProcessor != nil {
		logger.Info("Processing scheduled subscription changes...")
		a.scheduledChangesProcessor.ProcessChanges()
	}

	logger.Info("Subscription processing finished successfully in LocalHandleRequest.")
	return nil // Indicate successful execution to Lambda runtime
}

func main() {
	// Load .env file for local development
	err := godotenv.Load("../../.env")
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
	logger.Info("Lambda Cold Start: Initializing subscription processor for stage", zap.String("stage", stage))
	defer func() {
		// Sync logger before exit (important for Lambda)
		// This will run when the Lambda execution environment shuts down.
		_ = logger.Sync()
	}()

	ctx := context.Background()

	// --- Initialize AWS Secrets Manager Client ---
	secretsClient, err := awsclient.NewSecretsManagerClient(ctx)
	if err != nil {
		// Use Fatal logging which will call os.Exit(1) after logging
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
		// Assuming "RDS_SECRET_ARN" is the env var *name* holding the ARN
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
	} else { // Local
		logger.Info("Running in local stage, using DATABASE_URL from env/secrets")
		dsn, err = secretsClient.GetSecretString(ctx, "DATABASE_URL_ARN", "DATABASE_URL")
		if err != nil {
			logger.Fatal("Failed to get DATABASE_URL", zap.Error(err))
		}
		if dsn == "" {
			logger.Fatal("DATABASE_URL is required for local development and not found")
		}
	}

	cypheraSmartWalletAddress := os.Getenv("CYPHERA_SMART_WALLET_ADDRESS")
	if cypheraSmartWalletAddress == "" {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS environment variable is required and not set")
	}
	if !helpers.IsAddressValid(cypheraSmartWalletAddress) {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS is not a valid address", zap.String("address", cypheraSmartWalletAddress))
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
	// IMPORTANT: For Lambda, connections in the pool should ideally be managed across invocations.
	// Do NOT defer connPool.Close() here in main if using lambda.Start, as main finishes quickly.
	// The pool will persist for warm starts. AWS handles container shutdown.

	dbQueries := db.New(connPool)

	// --- Delegation Client Initialization ---
	delegationHost := os.Getenv("DELEGATION_SERVER_URL")
	if delegationHost == "" {
		logger.Fatal("DELEGATION_SERVER_URL environment variable is required and not set")
	}
	var fullDelegationAddr string
	var useLocalModeForDelegation bool
	if stage == helpers.StageLocal {
		// Check if delegationHost already contains a port (has colon)
		if strings.Contains(delegationHost, ":") {
			fullDelegationAddr = delegationHost
		} else {
			fullDelegationAddr = delegationHost + ":50051"
		}
		useLocalModeForDelegation = true
	} else if stage == helpers.StageDev || stage == helpers.StageProd {
		// Check if delegationHost already contains a port (has colon)
		if strings.Contains(delegationHost, ":") {
			fullDelegationAddr = delegationHost
		} else {
			fullDelegationAddr = delegationHost + ":443"
		}
		useLocalModeForDelegation = false
	} else {
		logger.Fatal("Invalid STAGE for delegation server connection configuration", zap.String("stage", stage))
	}
	logger.Info("Delegation server connection details",
		zap.String("address", fullDelegationAddr),
		zap.Bool("useLocalMode", useLocalModeForDelegation),
	)

	// Read RPC timeout from environment, default if not set
	rpcTimeoutStr := os.Getenv("DELEGATION_RPC_TIMEOUT") // As defined in template.yaml
	rpcTimeout, err := time.ParseDuration(rpcTimeoutStr)
	if err != nil {
		rpcTimeout = 3 * time.Minute
	}

	delegationClientConfig := dsClient.DelegationClientConfig{
		DelegationGRPCAddr: fullDelegationAddr,
		RPCTimeout:         rpcTimeout, // Use parsed or default
		UseLocalMode:       useLocalModeForDelegation,
	}
	delegationClient, err := dsClient.NewDelegationClient(delegationClientConfig)
	if err != nil {
		logger.Fatal("Failed to initialize delegation client", zap.Error(err))
	}
	// Similarly, don't defer delegationClient.Close() here.

	// --- Create Handler and Application Struct ---
	// Initialize email service if API key is available
	var emailService *services.EmailService
	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey != "" {
		fromEmail := os.Getenv("EMAIL_FROM_ADDRESS")
		if fromEmail == "" {
			fromEmail = "noreply@cypherapay.com"
		}
		fromName := os.Getenv("EMAIL_FROM_NAME")
		if fromName == "" {
			fromName = "Cyphera"
		}
		emailService = services.NewEmailService(resendAPIKey, fromEmail, fromName, logger.Log)
		logger.Info("Email service initialized",
			zap.String("from_email", fromEmail),
			zap.String("from_name", fromName))
	}

	// Create the dunning service
	dunningService := services.NewDunningService(dbQueries, logger.Log)

	// Create the payment failure detector
	failureDetector := services.NewPaymentFailureDetector(dbQueries, logger.Log, dunningService)

	// Initialize payment service for subscription management
	cmcApiKey := os.Getenv("CMC_API_KEY")
	paymentService := services.NewPaymentService(dbQueries, cmcApiKey)

	// Initialize customer service
	customerService := services.NewCustomerService(dbQueries)

	// Initialize subscription service
	subscriptionService := services.NewSubscriptionService(dbQueries, delegationClient, paymentService, customerService)

	// Create the scheduled changes processor
	var scheduledChangesProcessor *processor.ScheduledChangesProcessor
	if emailService != nil {
		scheduledChangesProcessor = processor.NewScheduledChangesProcessor(dbQueries, paymentService, emailService, 5*time.Minute)
	}

	// Create the subscription processor using the subscription service
	app := &Application{
		subscriptionProcessor:     processor.NewSubscriptionProcessor(subscriptionService),
		scheduledChangesProcessor: scheduledChangesProcessor,
		failureDetector:           failureDetector,
		dunningService:            dunningService,
		// Store connPool and delegationClient in App struct if HandleRequest needs to close them,
		// though typically you don't close them between warm invocations.
	}

	if stage == helpers.StageLocal {
		err := app.LocalHandleRequest(ctx)
		if err != nil {
			logger.Fatal("Error in LocalHandleRequest", zap.Error(err))
		}
	} else {
		// --- Start the Lambda Handler ---
		// lambda.Start blocks and handles invocations using the HandleRequest method
		lambda.Start(app.HandleRequest)
	}
}
