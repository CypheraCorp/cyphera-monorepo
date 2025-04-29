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
	"cyphera-api/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load .env file for local development (optional, secrets manager preferred)
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v. Proceeding with environment variables/secrets.", err)
	}

	// Initialize logger
	logger.InitLogger()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	ctx := context.Background()

	// --- Initialize AWS Secrets Manager Client ---
	secretsClient, err := awsclient.NewSecretsManagerClient(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize AWS Secrets Manager client", zap.Error(err))
	}

	// Determine Stage (needed for parameter names)
	// Lambda environment variables should set STAGE
	stage := os.Getenv("STAGE")
	if stage == "" {
		stage = "dev" // Default to dev if not set (adjust as needed)
		logger.Warn("STAGE environment variable not set, defaulting to 'dev'")
	}
	logger.Info("Running for stage", zap.String("stage", stage))

	// --- Database Connection Setup ---
	var dsn string
	// Use GIN_MODE=release as indicator for deployed environment (set this in Lambda env)
	if os.Getenv("GIN_MODE") == "release" {
		logger.Info("Running in deployed environment (GIN_MODE=release), fetching DB credentials from Secrets Manager")

		// Use direct environment variables set by Lambda
		// Note: Using dbEndpoint variable name like in server.go
		dbEndpoint := os.Getenv("DB_HOST") // This should contain host:port based on server.go pattern
		dbName := os.Getenv("DB_NAME")
		dbSecretArn := os.Getenv("RDS_SECRET_ARN") // Use RDS_SECRET_ARN matching server.go and user's correction
		dbSSLMode := os.Getenv("DB_SSLMODE")

		// Validate required environment variables for release mode
		if dbEndpoint == "" || dbName == "" || dbSecretArn == "" {
			logger.Fatal("Missing required DB environment variables for deployed environment (DB_HOST, DB_NAME, RDS_SECRET_ARN)")
		}
		if dbSSLMode == "" {
			dbSSLMode = "require" // Sensible default for RDS
			logger.Warn("DB_SSLMODE not set, defaulting to 'require'")
		}

		// Define structure for RDS secret JSON (matching server.go)
		type RdsSecret struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var secretData RdsSecret

		// Fetch secret using the ARN (matching server.go)
		err = secretsClient.GetSecretJSON(ctx, dbSecretArn, "", &secretData)
		if err != nil {
			logger.Fatal("Failed to retrieve or parse RDS secret", zap.Error(err), zap.String("secretArn", dbSecretArn))
		}

		if secretData.Username == "" || secretData.Password == "" {
			logger.Fatal("Username or password not found in RDS secret data", zap.String("secretArn", dbSecretArn))
		}

		// Construct DSN for deployed environment (matching server.go)
		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
			url.QueryEscape(secretData.Username),
			url.QueryEscape(secretData.Password),
			dbEndpoint, // Assumes DB_HOST contains host:port
			dbName, dbSSLMode)
		logger.Info("Constructed DSN from Secrets Manager credentials")

	} else {
		// --- Local Development Environment --- (matching server.go)
		logger.Info("Running in local environment (GIN_MODE != release), using DATABASE_URL from env/secrets")
		// Try fetching DATABASE_URL from Secrets Manager first (using ARN), fallback to direct env var
		dsn, err = secretsClient.GetSecretString(ctx, "DATABASE_URL_ARN", "DATABASE_URL")
		if err != nil {
			logger.Fatal("Failed to get DATABASE_URL", zap.Error(err))
		}
		if dsn == "" {
			logger.Fatal("DATABASE_URL is required for local development and not found")
		}
	}

	// --- Get Cyphera Smart Wallet Address ---
	// Get the *value* directly from environment variable set by SAM template/deploy script
	cypheraSmartWalletAddress := os.Getenv("CYPHERA_SMART_WALLET_ADDRESS")
	if cypheraSmartWalletAddress == "" {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS environment variable is required and not set")
	}
	// Validate address format
	if !handlers.IsAddressValid(cypheraSmartWalletAddress) {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS is not a valid address", zap.String("address", cypheraSmartWalletAddress))
	}

	// --- Get Delegation Server gRPC Address ---
	// Get the *value* directly from environment variable set by SAM template/deploy script
	delegationGrpcAddr := os.Getenv("DELEGATION_GRPC_ADDR")
	if delegationGrpcAddr == "" {
		logger.Fatal("DELEGATION_GRPC_ADDR environment variable is required and not set")
	}

	// --- Set Environment Variable for Delegation Client ---
	// The delegation client reads DELEGATION_GRPC_ADDR from the environment, which is already set above.
	// DELEGATION_RPC_TIMEOUT is set directly in the SAM template's environment variables.
	// os.Setenv("DELEGATION_GRPC_ADDR", delegationGrpcAddr) // No longer needed, read directly above

	// --- Database Pool Initialization ---
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Unable to parse database DSN", zap.Error(err))
	}

	// Configure the connection pool (adjust sizes if needed for Lambda)
	poolConfig.MaxConns = 5 // Reduced for potentially concurrent Lambda executions
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 15

	connPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool", zap.Error(err))
	}
	defer connPool.Close()

	dbQueries := db.New(connPool)

	// --- Initialize Delegation Client ---
	delegationClient, err := dsClient.NewDelegationClient() // Now reads DELEGATION_GRPC_ADDR from env
	if err != nil {
		logger.Fatal("Failed to initialize delegation client", zap.Error(err))
	}
	defer delegationClient.Close()

	// --- Create Common Services & Handler ---
	commonServices := handlers.NewCommonServices(dbQueries, cypheraSmartWalletAddress)
	subscriptionHandler := handlers.NewSubscriptionHandler(commonServices, delegationClient)

	// --- Process Subscriptions ---
	log.Printf("Starting subscription processing...")
	processSubscriptions(ctx, subscriptionHandler) // Run the processing logic once
	log.Printf("Subscription processing finished.")

	// No loop, ticker, or signal handling needed for Lambda/cron job
}

// processSubscriptions runs the subscription processor and logs the results
func processSubscriptions(ctx context.Context, handler *handlers.SubscriptionHandler) {
	results, err := handler.ProcessDueSubscriptions(ctx)
	if err != nil {
		// Use structured logging
		logger.Error("Error processing subscriptions", zap.Error(err))
		return
	}

	// Use structured logging
	logger.Info("Subscription processing results",
		zap.Int("total", results.Total),
		zap.Int("succeeded", results.Succeeded),
		zap.Int("failed", results.Failed),
		zap.Int("completed", results.Completed),
	)
}
