package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	awsclient "github.com/cyphera/cyphera-api/libs/go/client/aws"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/apps/dunning-processor/internal/processor"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Application holds all dependencies for the Lambda handler
type Application struct {
	dunningProcessor *processor.DunningProcessor
	logger          *zap.Logger
}

// HandleRequest is the actual Lambda handler function
func (app *Application) HandleRequest(ctx context.Context) error {
	app.logger.Info("Starting dunning processor execution")

	// Process due dunning campaigns
	results, err := app.dunningProcessor.ProcessDueCampaigns(ctx)
	if err != nil {
		app.logger.Error("Error processing dunning campaigns", zap.Error(err))
		return fmt.Errorf("error processing dunning campaigns: %w", err)
	}

	app.logger.Info("Dunning processing results",
		zap.Int("total", results.Total),
		zap.Int("succeeded", results.Succeeded),
		zap.Int("failed", results.Failed),
		zap.Int("emails_sent", results.EmailsSent),
		zap.Int("payments_retried", results.PaymentsRetried),
	)

	return nil
}

// LocalHandleRequest is for local development testing
func (app *Application) LocalHandleRequest(ctx context.Context) error {
	return app.HandleRequest(ctx)
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

	// Initialize logger
	logger.InitLogger(stage)
	logger.Info("Lambda Cold Start: Initializing dunning processor for stage", zap.String("stage", stage))
	defer func() {
		_ = logger.Sync()
	}()

	ctx := context.Background()

	// Initialize AWS Secrets Manager Client
	secretsClient, err := awsclient.NewSecretsManagerClient(ctx)
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

	// Get Resend API configuration
	resendAPIKey, err := secretsClient.GetSecretString(ctx, "RESEND_API_KEY_ARN", "RESEND_API_KEY")
	if err != nil || resendAPIKey == "" {
		logger.Fatal("Failed to get RESEND_API_KEY", zap.Error(err))
	}

	fromEmail := os.Getenv("DUNNING_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = "noreply@cypherapay.com"
	}

	fromName := os.Getenv("DUNNING_FROM_NAME")
	if fromName == "" {
		fromName = "Cyphera"
	}

	// Create services
	dunningService := services.NewDunningService(dbQueries, logger.Log)
	emailService := services.NewEmailService(resendAPIKey, fromEmail, fromName, logger.Log)
	// Note: Delegation client is nil here as payment processing happens in the main API
	dunningRetryEngine := services.NewDunningRetryEngine(dbQueries, logger.Log, dunningService, emailService, nil)

	// Create the dunning processor
	dunningProcessor := processor.NewDunningProcessor(dunningRetryEngine, logger.Log)

	// Create application struct
	app := &Application{
		dunningProcessor: dunningProcessor,
		logger:          logger.Log,
	}

	if stage == helpers.StageLocal {
		// Local development - run once
		err := app.LocalHandleRequest(ctx)
		if err != nil {
			logger.Fatal("Error in LocalHandleRequest", zap.Error(err))
		}
	} else {
		// AWS Lambda environment
		lambda.Start(app.HandleRequest)
	}
}