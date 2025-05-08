package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	awsclient "cyphera-api/internal/client/aws"
	dsClient "cyphera-api/internal/client/delegation_server"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"cyphera-api/internal/helpers"
	"cyphera-api/internal/logger"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Application holds all dependencies for the Lambda handler
type Application struct {
	subscriptionHandler *handlers.SubscriptionHandler
	// Add other dependencies here if HandleRequest needs them directly
	// e.g., dbPool *pgxpool.Pool, delegationClient *dsClient.DelegationClient
}

// HandleRequest is the actual Lambda handler function
// It now belongs to the Application struct to access dependencies.
func (app *Application) HandleRequest(ctx context.Context /*, event MyEvent - if you have a specific event type */) error {
	logger.Info("Entering HandleRequest for subscription processing") // Using structured logger

	// --- Process Subscriptions ---
	logger.Info("Starting subscription processing...")
	results, err := app.subscriptionHandler.ProcessDueSubscriptions(ctx)
	if err != nil {
		logger.Error("Error processing subscriptions in HandleRequest", zap.Error(err))
		return fmt.Errorf("HandleRequest: error from ProcessDueSubscriptions: %w", err) // Return error to Lambda runtime
	}

	logger.Info("Subscription processing results in HandleRequest",
		zap.Int("total", results.Total),
		zap.Int("succeeded", results.Succeeded),
		zap.Int("failed", results.Failed),
		zap.Int("completed", results.Completed),
	)

	logger.Info("Subscription processing finished successfully in HandleRequest.")
	return nil // Indicate successful execution to Lambda runtime
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
	if !handlers.IsAddressValid(cypheraSmartWalletAddress) {
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
		fullDelegationAddr = delegationHost + ":50051"
		useLocalModeForDelegation = true
	} else if stage == helpers.StageDev || stage == helpers.StageProd {
		fullDelegationAddr = delegationHost + ":443"
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
		logger.Warn("Failed to parse DELEGATION_RPC_TIMEOUT, using default 3m", zap.String("value", rpcTimeoutStr), zap.Error(err))
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
	// Assuming nil for rate limiter is fine for the subscription processor
	commonServices := handlers.NewCommonServices(dbQueries, cypheraSmartWalletAddress, nil)
	app := &Application{
		subscriptionHandler: handlers.NewSubscriptionHandler(commonServices, delegationClient),
		// Store connPool and delegationClient in App struct if HandleRequest needs to close them,
		// though typically you don't close them between warm invocations.
	}

	// --- Start the Lambda Handler ---
	// lambda.Start blocks and handles invocations using the HandleRequest method
	lambda.Start(app.HandleRequest)

}
