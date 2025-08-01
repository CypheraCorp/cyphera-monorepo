package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/apps/api/handlers"
	"github.com/cyphera/cyphera-api/libs/go/client/auth"
	awsclient "github.com/cyphera/cyphera-api/libs/go/client/aws"
	"github.com/cyphera/cyphera-api/libs/go/client/circle"
	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap" // Import CMC client
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/client/payment_sync"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers" // Import helpers
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/middleware"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Handler Definitions
var (
	accountHandler                *handlers.AccountHandler
	workspaceHandler              *handlers.WorkspaceHandler
	customerHandler               *handlers.CustomerHandler
	apiKeyHandler                 *handlers.APIKeyHandler
	userHandler                   *handlers.UserHandler
	networkHandler                *handlers.NetworkHandler
	tokenHandler                  *handlers.TokenHandler
	productHandler                *handlers.ProductHandler
	walletHandler                 *handlers.WalletHandler
	subscriptionHandler           *handlers.SubscriptionHandler
	subscriptionEventHandler      *handlers.SubscriptionEventHandler
	subscriptionManagementHandler *handlers.SubscriptionManagementHandler
	paymentSyncHandler            *handlers.PaymentSyncHandlers
	delegationClient              *dsClient.DelegationClient
	redemptionProcessor           *services.RedemptionProcessor
	circleHandler                 *handlers.CircleHandler
	currencyHandler               *handlers.CurrencyHandler
	analyticsHandler              *handlers.AnalyticsHandler
	gasSponsorshipHandler         *handlers.GasSponsorshipHandler
	invoiceHandler                *handlers.InvoiceHandler
	paymentLinkHandler            *handlers.PaymentLinkHandler
	paymentPageHandler            *handlers.PaymentPageHandler
	dunningHandler                *handlers.DunningHandler

	// Database
	dbQueries *db.Queries

	// Clients
	authClient *auth.AuthClient

	// Services
	commonServices *handlers.CommonServices
	cmcApiKey      string
	// dunningRetryEngine *services.DunningRetryEngine // Commented out: unused variable
	handlerFactory *handlers.HandlerFactory
)

func InitializeHandlers() {
	var dsn string // Database Source Name (connection string)

	// Load environment variables from .env file for local development
	// Note: .env might still set STAGE=local, which is now the preferred way
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v", err) // Use basic log before logger init
	}

	// --- Determine and Validate Stage ---
	stage := os.Getenv("STAGE")
	if stage == "" {
		stage = helpers.StageLocal // Default to local if not set
		log.Printf("Warning: STAGE environment variable not set, defaulting to '%s'", stage)
	}
	if !helpers.IsValidStage(stage) {
		log.Fatalf("Invalid STAGE environment variable: '%s'. Must be one of: %s, %s, %s",
			stage, helpers.StageProd, helpers.StageDev, helpers.StageLocal)
	}

	// --- Initialize Logger (AFTER stage validation) ---
	logger.InitLogger(stage)
	logger.Info("Initializing handlers for stage", zap.String("stage", stage))

	ctx := context.Background()

	// --- Initialize AWS Secrets Manager Client ---
	secretsClient, err := awsclient.NewSecretsManagerClient(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize AWS Secrets Manager client", zap.Error(err))
	}

	// --- Database Connection Setup ---
	// Use stage variable to determine connection method
	if stage == helpers.StageProd || stage == helpers.StageDev {
		// Deployed environment logic (prod or dev)
		logger.Info("Running in deployed stage, fetching DB credentials from Secrets Manager", zap.String("stage", stage))

		// This code block remains largely the same as before...
		// It reads DB_HOST, DB_NAME, RDS_SECRET_ARN from env (set by SAM template)
		dbEndpoint := os.Getenv("DB_HOST")
		dbName := os.Getenv("DB_NAME")
		dbSSLMode := os.Getenv("DB_SSLMODE")

		if dbEndpoint == "" || dbName == "" {
			logger.Fatal("Missing required DB environment variables for deployed stage (DB_HOST, DB_NAME)")
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
			url.QueryEscape(secretData.Username),
			url.QueryEscape(secretData.Password),
			dbEndpoint, dbName, dbSSLMode)
		logger.Info("Constructed DSN from Secrets Manager credentials")

	} else {
		// --- Local Development Environment (stage == helpers.StageLocal) ---
		logger.Info("Running in local stage, using DATABASE_URL from env/secrets")
		// Use GetSecretString for DATABASE_URL as it might be set directly or via an ARN
		dsn, err = secretsClient.GetSecretString(ctx, "DATABASE_URL_ARN", "DATABASE_URL")
		if err != nil {
			logger.Fatal("Failed to get DATABASE_URL", zap.Error(err))
		}
		if dsn == "" {
			// This check might be redundant if GetSecretString returns an error when empty, but good for clarity
			logger.Fatal("DATABASE_URL is required for local development")
		}
	}

	// --- Web3Auth Configuration ---
	web3AuthClientID, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_CLIENT_ID_ARN", "WEB3AUTH_CLIENT_ID")
	if err != nil || web3AuthClientID == "" {
		logger.Fatal("Failed to get Web3Auth Client ID", zap.Error(err))
	}

	web3AuthJWKSEndpoint, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_JWKS_ENDPOINT_ARN", "WEB3AUTH_JWKS_ENDPOINT")
	if err != nil || web3AuthJWKSEndpoint == "" {
		logger.Fatal("Failed to get Web3Auth JWKS Endpoint", zap.Error(err))
	}

	web3AuthIssuer, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_ISSUER_ARN", "WEB3AUTH_ISSUER")
	if err != nil || web3AuthIssuer == "" {
		logger.Fatal("Failed to get Web3Auth Issuer", zap.Error(err))
	}

	web3AuthAudience, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_AUDIENCE_ARN", "WEB3AUTH_AUDIENCE")
	if err != nil || web3AuthAudience == "" {
		logger.Fatal("Failed to get Web3Auth Audience", zap.Error(err))
	}

	// Set environment variables for AuthClient
	os.Setenv("WEB3AUTH_CLIENT_ID", web3AuthClientID)
	os.Setenv("WEB3AUTH_JWKS_ENDPOINT", web3AuthJWKSEndpoint)
	os.Setenv("WEB3AUTH_ISSUER", web3AuthIssuer)
	os.Setenv("WEB3AUTH_AUDIENCE", web3AuthAudience)

	// --- Auth Client ---
	authClient = auth.NewAuthClient()

	// --- Circle API Key ---
	circleApiKey, err := secretsClient.GetSecretString(ctx, "CIRCLE_API_KEY_ARN", "CIRCLE_API_KEY")
	if err != nil || circleApiKey == "" {
		logger.Fatal("Failed to get Circle API Key", zap.Error(err))
	}

	// --- Circle Client ---
	circleClient := circle.NewCircleClient(circleApiKey)

	// --- CoinMarketCap API Key --- (Add this section)
	cmcApiKey, err = secretsClient.GetSecretString(ctx, "COIN_MARKET_CAP_API_KEY_ARN", "COIN_MARKET_CAP_API_KEY") // Use appropriate names
	if err != nil || cmcApiKey == "" {
		// Log a warning instead of fatal if price checks are optional
		logger.Log.Warn("Failed to get CoinMarketCap API Key (COIN_MARKET_CAP_API_KEY_ARN or COIN_MARKET_CAP_API_KEY). Price conversions will fail.", zap.Error(err))
		// Set cmcApiKey to empty string or handle the error based on requirements
		cmcApiKey = "" // Allow initialization but calls will fail
	} else {
		logger.Log.Info("Successfully retrieved CoinMarketCap API Key")
	}

	// --- CoinMarketCap Client ---
	cmcClient := coinmarketcap.NewClient(cmcApiKey)

	// --- Payment Sync Encryption Key ---
	paymentSyncEncryptionKey, err := secretsClient.GetSecretString(ctx, "PAYMENT_SYNC_ENCRYPTION_KEY_ARN", "PAYMENT_SYNC_ENCRYPTION_KEY")
	if err != nil || paymentSyncEncryptionKey == "" {
		logger.Fatal("Failed to get Payment Sync Encryption Key", zap.Error(err))
	}

	// --- Resend API Key ---
	resendAPIKey, err := secretsClient.GetSecretString(ctx, "RESEND_API_KEY_ARN", "RESEND_API_KEY")
	if err != nil || resendAPIKey == "" {
		logger.Log.Warn("Failed to get Resend API Key. Email functionality will be disabled.", zap.Error(err))
		resendAPIKey = "" // Allow initialization but email sending will be skipped
	} else {
		logger.Log.Info("Successfully retrieved Resend API Key")
	}

	// --- Database Pool Initialization ---
	// Parse the DSN configuration first
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Unable to parse database DSN", zap.Error(err), zap.String("dsnUsed", dsn)) // Log the DSN used
	}

	// Configure the connection pool
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Minute * 30 // Shorter lifetime to prevent cached plan issues
	poolConfig.MaxConnIdleTime = time.Minute * 15 // Shorter idle time

	// Create the connection pool using the config
	dbpool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool with config", zap.Error(err))
	}

	// Create queries instance with the connection pool
	dbQueries = db.New(dbpool)

	cypheraSmartWalletAddress := os.Getenv("CYPHERA_SMART_WALLET_ADDRESS")
	if cypheraSmartWalletAddress == "" {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS environment variable is required")
	}

	// validate cyphera wallet address
	if !helpers.IsAddressValid(cypheraSmartWalletAddress) {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS is not a valid address")
	}

	// --- Delegation Client Configuration ---
	delegationHost := os.Getenv("DELEGATION_SERVER_URL")
	if delegationHost == "" {
		logger.Fatal("DELEGATION_SERVER_URL environment variable is required")
	}

	var fullDelegationAddr string
	var useLocalModeForDelegation bool

	// STAGE is already determined and validated earlier in InitializeHandlers
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
		// This case should ideally not be reached if STAGE validation is robust
		logger.Fatal("Invalid STAGE for delegation server connection configuration", zap.String("stage", stage))
	}
	logger.Info("Delegation server connection details",
		zap.String("address", fullDelegationAddr),
		zap.Bool("useLocalMode", useLocalModeForDelegation),
	)

	delegationClientConfig := dsClient.DelegationClientConfig{
		DelegationGRPCAddr: fullDelegationAddr,        // Use the constructed full address
		RPCTimeout:         3 * time.Minute,           // Existing timeout
		UseLocalMode:       useLocalModeForDelegation, // Set based on STAGE
	}

	// Initialize the delegation client
	delegationClient, err = dsClient.NewDelegationClient(delegationClientConfig)
	if err != nil {
		logger.Fatal("Unable to create delegation client", zap.Error(err))
	}

	// Initialize PaymentSyncClient with encryption key
	paymentSyncClient := payment_sync.NewPaymentSyncClient(dbQueries, logger.Log, paymentSyncEncryptionKey)

	// Get additional configurations
	fromEmail := os.Getenv("EMAIL_FROM_ADDRESS")
	if fromEmail == "" {
		fromEmail = "noreply@cypherapay.com"
	}
	fromName := os.Getenv("EMAIL_FROM_NAME")
	if fromName == "" {
		fromName = "Cyphera"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://pay.cyphera.com"
	}
	rpcAPIKey := os.Getenv("RPC_API_KEY")
	if rpcAPIKey == "" {
		logger.Warn("RPC_API_KEY not set, blockchain service functionality may be limited")
	}

	// Create the handler factory with all dependencies
	handlerFactory = handlers.CreateDefaultFactory(
		dbQueries,
		dbpool,
		cypheraSmartWalletAddress,
		cmcClient,
		cmcApiKey,
		resendAPIKey,
		fromEmail,
		fromName,
		baseURL,
		rpcAPIKey,
		delegationClient,
		paymentSyncClient,
	)

	// Get common services from factory
	commonServices = handlerFactory.GetCommonServices()

	// API Handler initialization using factory
	accountHandler = handlerFactory.NewAccountHandler()
	workspaceHandler = handlerFactory.NewWorkspaceHandler()
	customerHandler = handlerFactory.NewCustomerHandler()
	apiKeyHandler = handlerFactory.NewAPIKeyHandler()
	userHandler = handlerFactory.NewUserHandler()
	networkHandler = handlerFactory.NewNetworkHandler()
	tokenHandler = handlerFactory.NewTokenHandler()
	productHandler = handlerFactory.NewProductHandler(delegationClient)
	walletHandler = handlerFactory.NewWalletHandler()

	// Initialize currency handler
	currencyHandler = handlerFactory.NewCurrencyHandler()

	// Initialize subscription handlers
	subscriptionHandler = handlerFactory.NewSubscriptionHandler(delegationClient)
	subscriptionEventHandler = handlerFactory.NewSubscriptionEventHandler()

	// Payment Sync Service and Handlers
	// Note: Stripe services are now configured per-workspace dynamically,
	// no global Stripe service configuration needed
	paymentSyncHandler = handlers.NewPaymentSyncHandlers(commonServices, paymentSyncClient)

	// Analytics handler
	analyticsHandler = handlerFactory.NewAnalyticsHandler()
	// Gas sponsorship handler
	gasSponsorshipHandler = handlerFactory.NewGasSponsorshipHandler()
	// Invoice handler
	invoiceHandler = handlerFactory.NewInvoiceHandler()
	// Payment link handler
	paymentLinkHandler = handlerFactory.NewPaymentLinkHandler()
	// Payment page handler
	paymentPageHandler = handlerFactory.NewPaymentPageHandler()

	// Dunning management handler
	dunningHandler = handlerFactory.CreateDunningHandler()

	// Initialize subscription management handler
	subscriptionManagementHandler = handlerFactory.NewSubscriptionManagementHandler()

	// 3rd party handlers
	circleHandler = handlers.NewCircleHandler(commonServices, circleClient)
}

func InitializeRoutes(router *gin.Engine) {
	// Logger is now initialized in InitializeHandlers

	// Configure and apply CORS middleware
	router.Use(configureCORS())

	// Add correlation ID middleware for request tracing
	router.Use(middleware.CorrelationIDMiddleware())

	// Apply rate limiting middleware globally
	// This provides a default rate limit for all endpoints
	router.Use(middleware.DefaultRateLimiter.Middleware())

	// Add enhanced logging in development mode
	isDevelopment := os.Getenv("GIN_MODE") != "release"
	router.Use(middleware.EnhancedLoggingMiddleware(isDevelopment))

	// Add basic request logging for production
	if !isDevelopment {
		router.Use(middleware.RequestLoggingMiddleware())
	}

	// Add Swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health for raw lambda url check
	router.GET("/:stage/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Initialize and start the redemption processor with 3 workers and a buffer size of 100
	redemptionProcessor = handlerFactory.CreateRedemptionProcessor(delegationClient, 3, 100)
	redemptionProcessor.Start()

	// Ensure we gracefully stop the redemption processor when the server shuts down
	router.GET("/shutdown", func(c *gin.Context) {
		go func() {
			time.Sleep(1 * time.Second)
			redemptionProcessor.Stop()
			logger.Info("Server is shutting down...")
			os.Exit(0)
		}()
		c.JSON(http.StatusOK, gin.H{"message": "Server is shutting down..."})
	})

	// Request logging is now handled by the enhanced logging middleware added earlier

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		// Payment link by slug - public endpoint for customers to view payment links
		v1.GET("/payment-links/slug/:slug", paymentLinkHandler.GetPaymentLinkBySlug)

		// Payment page endpoints - public endpoints for payment processing
		v1.GET("/payment-pages/:slug", paymentPageHandler.GetPaymentPageData)
		v1.POST("/payment-pages/:slug/intent", paymentPageHandler.CreatePaymentIntent)

		// Protected routes (authentication required)
		protected := v1.Group("/")
		authAdapter := handlers.NewAuthServicesAdapter(commonServices)
		protected.Use(authClient.EnsureValidAPIKeyOrToken(authAdapter))
		{
			// Admin-only routes
			admin := protected.Group("/admin")
			admin.Use(authClient.RequireRoles("admin"))
			{
				// Apply stricter rate limiting for auth endpoints
				admin.POST("/accounts/signin", middleware.StrictRateLimiter.Middleware(), accountHandler.SignInRegisterAccount)
				admin.POST("/customers/signin", middleware.StrictRateLimiter.Middleware(), customerHandler.SignInRegisterCustomer)
				admin.GET("/products/:product_id", productHandler.GetPublicProductByID)

				// subscribe to a product
				admin.POST("/products/:product_id/subscribe", middleware.ValidateInput(middleware.CreateDelegationSubscriptionValidation), productHandler.SubscribeToProductByID)

				// Account management
				admin.GET("/accounts", accountHandler.ListAccounts)
				admin.POST("/accounts", accountHandler.CreateAccount)
				admin.DELETE("/accounts/:account_id", accountHandler.DeleteAccount)

				// User management
				admin.POST("/users", userHandler.CreateUser)
				admin.GET("/users/:user_id", userHandler.GetUserByID)
				admin.PUT("/users/:user_id", middleware.ValidateInput(middleware.UpdateUserValidation), userHandler.UpdateUser)
				admin.DELETE("/users/:user_id", userHandler.DeleteUser)

				// Workspace management
				admin.GET("/workspaces", workspaceHandler.ListWorkspaces)
				admin.POST("/workspaces", workspaceHandler.CreateWorkspace)
				admin.GET("/workspaces/all", workspaceHandler.GetAllWorkspaces)
				admin.GET("/workspaces/:workspace_id", workspaceHandler.GetWorkspace)
				admin.PUT("/workspaces/:workspace_id", workspaceHandler.UpdateWorkspace)
				admin.DELETE("/workspaces/:workspace_id", workspaceHandler.DeleteWorkspace)
				admin.GET("/workspaces/:workspace_id/stats", workspaceHandler.GetWorkspaceStats)

				// API Key management
				admin.GET("/api-keys", apiKeyHandler.GetAllAPIKeys)

				// Network management
				admin.POST("/networks", networkHandler.CreateNetwork)
				admin.PUT("/networks/:network_id", networkHandler.UpdateNetwork)
				admin.DELETE("/networks/:network_id", networkHandler.DeleteNetwork)

				// Circle API endpoints
				circle := admin.Group("/circle")
				{
					// Circle user endpoints
					circleUser := circle.Group("/users")
					{
						circleUser.POST("/:workspace_id", circleHandler.CreateUser)
						circleUser.POST("/:workspace_id/token", circleHandler.CreateUserToken)
						circleUser.GET("/:workspace_id/token", circleHandler.GetUserByToken)
						circleUser.GET("/:workspace_id", circleHandler.GetUserByID)
						circleUser.POST("/:workspace_id/initialize", circleHandler.InitializeUser)

						// PIN management
						circleUser.POST("/:workspace_id/pin/create", circleHandler.CreatePinChallenge)
						circleUser.PUT("/:workspace_id/pin/update", circleHandler.UpdatePinChallenge)
						circleUser.POST("/:workspace_id/pin/restore", circleHandler.CreatePinRestoreChallenge)
					}

					// Circle wallet endpoints
					circleWallet := circle.Group("/wallets")
					{
						circleWallet.POST("/:workspace_id", middleware.ValidateInput(middleware.CircleWalletValidation), circleHandler.CreateWallets)
						circleWallet.GET("/:workspace_id", circleHandler.ListWallets)
						circleWallet.GET("/get/:wallet_id", circleHandler.GetWallet)
						circleWallet.GET("/balances/:wallet_id", circleHandler.GetWalletBalance)
					}

					// Circle challenge endpoints
					circle.GET("/:workspace_id/challenges/:challenge_id", circleHandler.GetChallenge)
				}

				// Networks
				networks := protected.Group("/networks")
				{
					networks.GET("", networkHandler.ListNetworks)
					networks.GET("/:network_id", networkHandler.GetNetwork)
					networks.GET("/chain/:chain_id", networkHandler.GetNetworkByChainID)
				}

				// Tokens
				tokens := protected.Group("/tokens")
				{
					tokens.POST("/quote", tokenHandler.GetTokenQuote)
				}
			}

			// Current Account routes
			accounts := protected.Group("/accounts")
			{
				accounts.GET("/:account_id", accountHandler.GetAccount)
				accounts.POST("/onboard", accountHandler.OnboardAccount)
			}

			// Customers
			customers := protected.Group("/customers")
			{
				customers.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), customerHandler.ListCustomers)
				customers.POST("", middleware.ValidateInput(middleware.CreateCustomerValidation), customerHandler.CreateCustomer)
				customers.GET("/:customer_id", customerHandler.GetCustomer)
				customers.PUT("/:customer_id", middleware.ValidateInput(middleware.CreateCustomerValidation), customerHandler.UpdateCustomer)
				customers.DELETE("/:customer_id", customerHandler.DeleteCustomer)

				// Customer onboarding status
				customers.PATCH("/:customer_id/onboarding", customerHandler.UpdateCustomerOnboardingStatus)

				// Customer subscriptions
				customers.GET("/:customer_id/subscriptions", subscriptionHandler.ListSubscriptionsByCustomer)
			}

			// API Keys
			apiKeys := protected.Group("/api-keys")
			{
				// Regular account routes (scoped to their workspace)
				apiKeys.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), apiKeyHandler.ListAPIKeys)
				apiKeys.POST("", middleware.ValidateInput(middleware.CreateAPIKeyValidation), apiKeyHandler.CreateAPIKey)
				apiKeys.GET("/:api_key_id", apiKeyHandler.GetAPIKeyByID)
				apiKeys.PUT("/:api_key_id", middleware.ValidateInput(middleware.CreateAPIKeyValidation), apiKeyHandler.UpdateAPIKey)
				apiKeys.DELETE("/:api_key_id", apiKeyHandler.DeleteAPIKey)
			}

			// Products
			products := protected.Group("/products")
			{
				products.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), productHandler.ListProducts)
				products.POST("", middleware.ValidateInput(middleware.CreateProductValidation), productHandler.CreateProduct)
				products.GET("/:product_id", productHandler.GetProduct)
				products.PUT("/:product_id", middleware.ValidateInput(middleware.CreateProductValidation), productHandler.UpdateProduct)
				products.DELETE("/:product_id", productHandler.DeleteProduct)

				// Product subscriptions
				products.GET("/:product_id/subscriptions", subscriptionHandler.ListSubscriptionsByProduct)
			}

			// Currencies
			currencies := protected.Group("/currencies")
			{
				currencies.GET("", currencyHandler.ListActiveCurrencies)
				currencies.GET("/:code", currencyHandler.GetCurrency)
				currencies.POST("/format", currencyHandler.FormatAmount)
			}

			// Workspace currency settings
			workspacesCurrent := protected.Group("/workspaces/current")
			{
				workspacesCurrent.GET("/currency-settings", currencyHandler.GetWorkspaceCurrencySettings)
				workspacesCurrent.PUT("/currency-settings", currencyHandler.UpdateWorkspaceCurrencySettings)
				workspacesCurrent.GET("/currencies", currencyHandler.ListWorkspaceSupportedCurrencies)
			}

			// Workspaces (non-admin routes for regular users)
			workspaces := protected.Group("/workspaces")
			{
				workspaces.GET("", workspaceHandler.ListWorkspaces) // List workspaces user has access to
				workspaces.GET("/:workspace_id", workspaceHandler.GetWorkspace)
				// TODO: Implement ListWorkspaceCustomers
				// workspaces.GET("/:workspace_id/customers", workspaceHandler.ListWorkspaceCustomers)
				workspaces.GET("/:workspace_id/stats", workspaceHandler.GetWorkspaceStats)
			}

			// Wallets
			wallets := protected.Group("/wallets")
			{
				wallets.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), walletHandler.ListWallets)
				wallets.POST("", middleware.ValidateInput(middleware.CreateWalletValidation), walletHandler.CreateWallet)
				wallets.GET("/:wallet_id", walletHandler.GetWallet)
				wallets.DELETE("/:wallet_id", walletHandler.DeleteWallet)
			}

			// Subscriptions
			subscriptions := protected.Group("/subscriptions")
			{
				subscriptions.GET("", subscriptionHandler.ListSubscriptions)
				// TODO: Implement these subscription methods
				// subscriptions.GET("/active", subscriptionHandler.ListActiveSubscriptions)
				// subscriptions.GET("/expired", subscriptionHandler.GetExpiredSubscriptions)
				// subscriptions.POST("", subscriptionHandler.CreateSubscription)
				subscriptions.GET("/:subscription_id", subscriptionHandler.GetSubscription)
				// TODO: Implement GetSubscriptionWithDetails
				// subscriptions.GET("/:subscription_id/details", subscriptionHandler.GetSubscriptionWithDetails)
				subscriptions.PUT("/:subscription_id", subscriptionHandler.UpdateSubscription)
				// TODO: Implement these subscription status methods
				// subscriptions.PATCH("/:subscription_id/status", subscriptionHandler.UpdateSubscriptionStatus)
				// subscriptions.POST("/:subscription_id/cancel", subscriptionHandler.CancelSubscription)
				subscriptions.DELETE("/:subscription_id", subscriptionHandler.DeleteSubscription)

				// Subscription management endpoints
				subscriptions.POST("/:subscription_id/upgrade", subscriptionManagementHandler.UpgradeSubscription)
				subscriptions.POST("/:subscription_id/downgrade", subscriptionManagementHandler.DowngradeSubscription)
				subscriptions.POST("/:subscription_id/cancel", subscriptionManagementHandler.CancelSubscription)
				subscriptions.POST("/:subscription_id/pause", subscriptionManagementHandler.PauseSubscription)
				subscriptions.POST("/:subscription_id/resume", subscriptionManagementHandler.ResumeSubscription)
				subscriptions.POST("/:subscription_id/reactivate", subscriptionManagementHandler.ReactivateSubscription)
				subscriptions.POST("/:subscription_id/preview-change", subscriptionManagementHandler.PreviewChange)
				subscriptions.GET("/:subscription_id/history", subscriptionManagementHandler.GetSubscriptionHistory)

				// Subscription analytics
				// TODO: Implement these subscription event analytics methods
				// subscriptions.GET("/:subscription_id/total-amount", subscriptionEventHandler.GetTotalAmountBySubscription)
				// subscriptions.GET("/:subscription_id/redemption-count", subscriptionEventHandler.GetSuccessfulRedemptionCount)
				// subscriptions.GET("/:subscription_id/latest-event", subscriptionEventHandler.GetLatestSubscriptionEvent)
				subscriptions.GET("/:subscription_id/events", subscriptionEventHandler.ListSubscriptionEventsBySubscription)
			}

			// Subscription events
			subEvents := protected.Group("/subscription-events")
			{
				subEvents.GET("/transactions", subscriptionEventHandler.ListSubscriptionEvents)
				subEvents.GET("/:event_id", subscriptionEventHandler.GetSubscriptionEvent)
				// TODO: Implement CreateSubscriptionEvent and UpdateSubscriptionEvent
				// subEvents.POST("", subscriptionEventHandler.CreateSubscriptionEvent)
				// subEvents.PUT("/:event_id", subscriptionEventHandler.UpdateSubscriptionEvent)
				subEvents.GET("/transaction/:tx_hash", subscriptionEventHandler.GetSubscriptionEventByTxHash)
				// TODO: Implement these subscription event filtering methods
				// subEvents.GET("/type/:event_type", subscriptionEventHandler.ListSubscriptionEventsByType)
				// subEvents.GET("/failed", subscriptionEventHandler.ListFailedSubscriptionEvents)
				// subEvents.GET("/recent", subscriptionEventHandler.ListRecentSubscriptionEvents)
			}

			// Failed subscription attempts
			// failedAttempts := protected.Group("/failed-subscription-attempts")
			// {
			// 	failedAttempts.GET("", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttempts)
			// 	failedAttempts.GET("/:attempt_id", failedSubscriptionAttemptHandler.GetFailedSubscriptionAttempt)
			// 	failedAttempts.GET("/customer/:customer_id", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttemptsByCustomer)
			// 	failedAttempts.GET("/product/:product_id", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttemptsByProduct)
			// 	failedAttempts.GET("/error-type/:error_type", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttemptsByErrorType)
			// }

			// Payment sync routes
			sync := protected.Group("/sync")
			{
				// Configuration management routes
				config := sync.Group("/config")
				{
					config.POST("", paymentSyncHandler.CreateConfiguration)               // Create new configuration
					config.GET("", paymentSyncHandler.ListConfigurations)                 // List all configurations for workspace
					config.GET("/:provider", paymentSyncHandler.GetConfiguration)         // Get config by provider name
					config.GET("/id/:config_id", paymentSyncHandler.GetConfigurationByID) // Get config by ID
					config.PUT("/:config_id", paymentSyncHandler.UpdateConfiguration)     // Update configuration
					config.DELETE("/:config_id", paymentSyncHandler.DeleteConfiguration)  // Delete configuration
					config.POST("/:config_id/test", paymentSyncHandler.TestConnection)    // Test connection
				}

				// Provider account management routes (for webhook routing)
				accounts := sync.Group("/accounts")
				{
					accounts.POST("", paymentSyncHandler.CreateProviderAccount) // Create provider account mapping
					accounts.GET("", paymentSyncHandler.GetProviderAccounts)    // List provider accounts for workspace
				}

				// Provider information
				sync.GET("/providers", paymentSyncHandler.GetAvailableProviders) // List available providers

				// Initial sync for any provider
				sync.POST("/:provider/initial", paymentSyncHandler.StartInitialSync)

				// Sync session management
				sessions := sync.Group("/sessions")
				{
					sessions.GET("", paymentSyncHandler.ListSyncSessions)                // List all sessions for workspace
					sessions.GET("/:id", paymentSyncHandler.GetSyncSession)              // Get session details
					sessions.GET("/:id/status", paymentSyncHandler.GetSyncSessionStatus) // Get session status and progress
				}
			}

			// Analytics routes
			analytics := protected.Group("/analytics")
			{
				// Dashboard overview
				analytics.GET("/dashboard", analyticsHandler.GetDashboardSummary)

				// Chart endpoints
				analytics.GET("/revenue-chart", analyticsHandler.GetRevenueChart)
				analytics.GET("/customer-chart", analyticsHandler.GetCustomerChart)
				analytics.GET("/subscription-chart", analyticsHandler.GetSubscriptionChart)
				analytics.GET("/mrr-chart", analyticsHandler.GetMRRChart)

				// Metrics endpoints
				analytics.GET("/payment-metrics", analyticsHandler.GetPaymentMetrics)
				analytics.GET("/network-breakdown", analyticsHandler.GetNetworkBreakdown)
				analytics.GET("/gas-fee-pie", analyticsHandler.GetGasFeePieChart)
				analytics.GET("/hourly", analyticsHandler.GetHourlyMetrics)

				// Refresh metrics
				analytics.POST("/refresh", analyticsHandler.RefreshMetrics)
			}

			// Gas sponsorship routes
			gasSponsorship := protected.Group("/gas-sponsorship")
			{
				// Configuration management
				gasSponsorship.GET("/config", gasSponsorshipHandler.GetGasSponsorshipConfig)
				gasSponsorship.PUT("/config", gasSponsorshipHandler.UpdateGasSponsorshipConfig)

				// Budget status
				gasSponsorship.GET("/budget-status", gasSponsorshipHandler.GetGasSponsorshipBudgetStatus)
			}

			// Invoice routes
			invoices := protected.Group("/invoices")
			{
				// Invoice management
				invoices.GET("", invoiceHandler.ListInvoices)
				invoices.POST("", middleware.ValidateInput(middleware.CreateInvoiceValidation), invoiceHandler.CreateInvoice)
				invoices.GET("/:invoice_id", invoiceHandler.GetInvoice)
				invoices.GET("/:invoice_id/preview", invoiceHandler.PreviewInvoice)
				invoices.POST("/:invoice_id/finalize", invoiceHandler.FinalizeInvoice)
				invoices.POST("/:invoice_id/send", invoiceHandler.SendInvoice)
				invoices.GET("/:invoice_id/payment-link", invoiceHandler.GetInvoicePaymentLink)
				// Create payment link for invoice
				invoices.POST("/:invoice_id/payment-link", paymentLinkHandler.CreateInvoicePaymentLink)
			}

			// Payment Links
			paymentLinks := protected.Group("/payment-links")
			{
				// Payment link management
				paymentLinks.GET("", paymentLinkHandler.ListPaymentLinks)
				paymentLinks.POST("", paymentLinkHandler.CreatePaymentLink)
				paymentLinks.GET("/:link_id", paymentLinkHandler.GetPaymentLink)
				paymentLinks.PUT("/:link_id", paymentLinkHandler.UpdatePaymentLink)
				paymentLinks.POST("/:link_id/deactivate", paymentLinkHandler.DeactivatePaymentLink)
				paymentLinks.GET("/stats", paymentLinkHandler.GetPaymentLinkStats)
			}

			// Dunning Management
			dunning := protected.Group("/dunning")
			{
				// Configuration management
				dunning.GET("/configurations", dunningHandler.ListConfigurations)
				dunning.POST("/configurations", dunningHandler.CreateConfiguration)
				dunning.GET("/configurations/:id", dunningHandler.GetConfiguration)

				// Campaign management
				dunning.GET("/campaigns", dunningHandler.ListCampaigns)
				dunning.GET("/campaigns/:id", dunningHandler.GetCampaign)
				dunning.POST("/campaigns/:id/pause", dunningHandler.PauseCampaign)
				dunning.POST("/campaigns/:id/resume", dunningHandler.ResumeCampaign)

				// Email template management
				dunning.GET("/email-templates", dunningHandler.ListEmailTemplates)
				dunning.POST("/email-templates", dunningHandler.CreateEmailTemplate)

				// Analytics
				dunning.GET("/stats", dunningHandler.GetCampaignStats)

				// Manual processing (for testing)
				dunning.POST("/process", dunningHandler.ProcessDueCampaigns)
			}

			// Webhook routes for payment failures
			webhooks := protected.Group("/webhooks")
			{
				paymentFailureHandler := handlerFactory.NewPaymentFailureWebhookHandler()
				webhooks.POST("/payment-failure", paymentFailureHandler.HandlePaymentFailure)
				webhooks.POST("/payment-failures/batch", paymentFailureHandler.HandleBatchPaymentFailures)
			}
		}
	}
}

// configureCORS returns a configured CORS middleware
func configureCORS() gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()

	// Get allowed origins from environment variable
	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if originsEnv == "" {
		// Default to localhost if not set
		corsConfig.AllowOrigins = []string{"http://localhost:3000"}
	} else {
		// Split and trim the origins
		origins := strings.Split(originsEnv, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		corsConfig.AllowOrigins = origins
	}

	// Get allowed methods from environment variable
	methodsEnv := os.Getenv("CORS_ALLOWED_METHODS")
	if methodsEnv == "" {
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	} else {
		methods := strings.Split(methodsEnv, ",")
		for i, method := range methods {
			methods[i] = strings.TrimSpace(method)
		}
		corsConfig.AllowMethods = methods
	}

	// Get allowed headers from environment variable
	headersEnv := os.Getenv("CORS_ALLOWED_HEADERS")
	if headersEnv == "" {
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Workspace-ID", "X-Account-ID", "X-Correlation-ID"}
	} else {
		headers := strings.Split(headersEnv, ",")
		for i, header := range headers {
			headers[i] = strings.TrimSpace(header)
		}
		corsConfig.AllowHeaders = headers
	}

	// Get exposed headers from environment variable
	exposedHeadersEnv := os.Getenv("CORS_EXPOSED_HEADERS")
	if exposedHeadersEnv != "" {
		exposedHeaders := strings.Split(exposedHeadersEnv, ",")
		for i, header := range exposedHeaders {
			exposedHeaders[i] = strings.TrimSpace(header)
		}
		corsConfig.ExposeHeaders = exposedHeaders
	} else {
		// Default exposed headers including rate limit headers
		corsConfig.ExposeHeaders = []string{
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"Retry-After",
			"X-Correlation-ID",
		}
	}

	// Set credentials allowed
	corsConfig.AllowCredentials = os.Getenv("CORS_ALLOW_CREDENTIALS") == "true"

	return cors.New(corsConfig)
}
