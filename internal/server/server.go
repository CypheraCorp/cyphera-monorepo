package server

import (
	"context"
	_ "cyphera-api/docs" // This will be generated
	"cyphera-api/internal/client/auth"
	awsclient "cyphera-api/internal/client/aws"
	"cyphera-api/internal/client/circle"
	dsClient "cyphera-api/internal/client/delegation_server"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"cyphera-api/internal/helpers" // Import helpers
	"cyphera-api/internal/logger"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
	accountHandler           *handlers.AccountHandler
	workspaceHandler         *handlers.WorkspaceHandler
	customerHandler          *handlers.CustomerHandler
	apiKeyHandler            *handlers.APIKeyHandler
	userHandler              *handlers.UserHandler
	networkHandler           *handlers.NetworkHandler
	tokenHandler             *handlers.TokenHandler
	productHandler           *handlers.ProductHandler
	walletHandler            *handlers.WalletHandler
	subscriptionHandler      *handlers.SubscriptionHandler
	subscriptionEventHandler *handlers.SubscriptionEventHandler
	delegationClient         *dsClient.DelegationClient
	redemptionProcessor      *handlers.RedemptionProcessor
	circleHandler            *handlers.CircleHandler

	// Database
	dbQueries *db.Queries

	// Clients
	authClient *auth.AuthClient
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
		dbSecretArn := os.Getenv("RDS_SECRET_ARN") // Renamed from dbUserSecretArn
		dbSSLMode := os.Getenv("DB_SSLMODE")

		if dbEndpoint == "" || dbName == "" || dbSecretArn == "" {
			logger.Fatal("Missing required DB environment variables for deployed stage (DB_HOST, DB_NAME, RDS_SECRET_ARN)")
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

		err = secretsClient.GetSecretJSON(ctx, dbSecretArn, "", &secretData)
		if err != nil {
			logger.Fatal("Failed to retrieve or parse RDS secret", zap.Error(err), zap.String("secretArn", dbSecretArn))
		}

		if secretData.Username == "" || secretData.Password == "" {
			logger.Fatal("Username or password not found in RDS secret data", zap.String("secretArn", dbSecretArn))
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

	// --- Supabase JWT Secret ---
	supabaseJwtSecret, err := secretsClient.GetSecretString(ctx, "SUPABASE_JWT_SECRET_ARN", "SUPABASE_JWT_SECRET")
	if err != nil || supabaseJwtSecret == "" {
		logger.Fatal("Failed to get Supabase JWT Secret", zap.Error(err))
	}

	// --- Auth Client ---
	authClient = auth.NewAuthClient(supabaseJwtSecret)

	// --- Circle API Key ---
	circleApiKey, err := secretsClient.GetSecretString(ctx, "CIRCLE_API_KEY_ARN", "CIRCLE_API_KEY")
	if err != nil || circleApiKey == "" {
		logger.Fatal("Failed to get Circle API Key", zap.Error(err))
	}

	// --- Circle Client ---
	circleClient := circle.NewCircleClient(circleApiKey)

	// --- Database Pool Initialization ---
	// Parse the DSN configuration first
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Unable to parse database DSN", zap.Error(err), zap.String("dsnUsed", dsn)) // Log the DSN used
	}

	// Configure the connection pool
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

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
	if !handlers.IsAddressValid(cypheraSmartWalletAddress) {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS is not a valid address")
	}

	// Initialize the delegation client
	delegationClient, err = dsClient.NewDelegationClient()
	if err != nil {
		logger.Fatal("Unable to create delegation client", zap.Error(err))
	}

	commonServices := handlers.NewCommonServices(
		dbQueries,
		cypheraSmartWalletAddress,
	)

	// API Handler initialization
	accountHandler = handlers.NewAccountHandler(commonServices)
	workspaceHandler = handlers.NewWorkspaceHandler(commonServices)
	customerHandler = handlers.NewCustomerHandler(commonServices)
	apiKeyHandler = handlers.NewAPIKeyHandler(commonServices)
	userHandler = handlers.NewUserHandler(commonServices)
	networkHandler = handlers.NewNetworkHandler(commonServices)
	tokenHandler = handlers.NewTokenHandler(commonServices)
	productHandler = handlers.NewProductHandler(commonServices, delegationClient)
	walletHandler = handlers.NewWalletHandler(commonServices)

	// Initialize subscription
	subscriptionHandler = handlers.NewSubscriptionHandler(commonServices, delegationClient)
	subscriptionEventHandler = handlers.NewSubscriptionEventHandler(commonServices)

	// 3rd party handlers
	circleHandler = handlers.NewCircleHandler(commonServices, circleClient)
}

func InitializeRoutes(router *gin.Engine) {
	// Logger is now initialized in InitializeHandlers

	// Configure and apply CORS middleware
	router.Use(configureCORS())

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
	redemptionProcessor = handlers.NewRedemptionProcessor(dbQueries, delegationClient, 3, 100)
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

	// if we are not in production, log the request body
	if os.Getenv("GIN_MODE") != "release" {
		router.Use(handlers.LogRequest())
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// No Public routes for now

		// Protected routes (authentication required)
		protected := v1.Group("/")
		protected.Use(authClient.EnsureValidAPIKeyOrToken(dbQueries))
		{
			// Admin-only routes
			admin := protected.Group("/admin")
			admin.Use(authClient.RequireRoles("admin"))
			{
				// public routes
				admin.GET("/public/products/:product_id", productHandler.GetPublicProductByID)

				// subscribe to a product
				admin.POST("/products/:product_id/subscribe", productHandler.SubscribeToProduct)

				// Account management
				admin.GET("/accounts", accountHandler.ListAccounts)
				admin.POST("/accounts", accountHandler.CreateAccount)
				admin.POST("/accounts/signin", accountHandler.SignInAccount)
				admin.DELETE("/accounts/:account_id", accountHandler.DeleteAccount)

				// User management
				admin.POST("/users", userHandler.CreateUser)
				admin.GET("/users/:user_id", userHandler.GetUser)
				admin.PUT("/users/:user_id", userHandler.UpdateUser)
				admin.DELETE("/users/:user_id", userHandler.DeleteUser)

				// Workspace management
				admin.GET("/workspaces", workspaceHandler.ListWorkspaces)
				admin.POST("/workspaces", workspaceHandler.CreateWorkspace)
				admin.GET("/workspaces/all", workspaceHandler.GetAllWorkspaces)
				admin.GET("/workspaces/:workspace_id", workspaceHandler.GetWorkspace)
				admin.PUT("/workspaces/:workspace_id", workspaceHandler.UpdateWorkspace)
				admin.DELETE("/workspaces/:workspace_id", workspaceHandler.DeleteWorkspace)
				admin.DELETE("/workspaces/:workspace_id/hard", workspaceHandler.HardDeleteWorkspace)
				admin.GET("/workspaces/:workspace_id/customers", workspaceHandler.ListWorkspaceCustomers)

				// API Key management
				admin.GET("/api-keys", apiKeyHandler.GetAllAPIKeys)
				admin.GET("/api-keys/expired", apiKeyHandler.GetExpiredAPIKeys)

				// Network management
				admin.POST("/networks", networkHandler.CreateNetwork)
				admin.PUT("/networks/:network_id", networkHandler.UpdateNetwork)
				admin.DELETE("/networks/:network_id", networkHandler.DeleteNetwork)

				// Token management
				admin.POST("/tokens", tokenHandler.CreateToken)
				admin.PUT("/tokens/:token_id", tokenHandler.UpdateToken)
				admin.DELETE("/tokens/:token_id", tokenHandler.DeleteToken)

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
						circleWallet.POST("/:workspace_id", circleHandler.CreateWallets)
						circleWallet.GET("/:workspace_id", circleHandler.ListWallets)
						circleWallet.GET("/get/:wallet_id", circleHandler.GetWallet)
						circleWallet.GET("/balances/:wallet_id", circleHandler.GetWalletBalance)
					}

					// Circle challenge endpoints
					circle.GET("/:workspace_id/challenges/:challenge_id", circleHandler.GetChallenge)
				}

				// Product Tokens
				products := protected.Group("/products")
				{
					products.GET("/:product_id/tokens", productHandler.GetProductTokensByProduct)
					products.GET("/:product_id/tokens/active", productHandler.GetActiveProductTokensByProduct)
					products.POST("/:product_id/tokens", productHandler.CreateProductToken)
					products.GET("/:product_id/tokens/:token_id", productHandler.GetProductToken)
					products.GET("/:product_id/networks/:network_id/tokens", productHandler.GetProductTokensByNetwork)
					products.GET("/:product_id/networks/:network_id/tokens/active", productHandler.GetActiveProductTokensByNetwork)
					products.DELETE("/:product_id/tokens", productHandler.DeleteProductTokensByProduct)
				}

				// Networks
				networks := protected.Group("/networks")
				{
					networks.GET("", networkHandler.ListNetworks)
					networks.GET("/active", networkHandler.ListActiveNetworks)
					networks.GET("/:network_id", networkHandler.GetNetwork)
					networks.GET("/chain/:chain_id", networkHandler.GetNetworkByChainID)
					networks.GET("/tokens", networkHandler.ListNetworksWithTokens)
				}

				// Tokens
				tokens := protected.Group("/tokens")
				{
					tokens.GET("", tokenHandler.ListTokens)
					tokens.GET("/:token_id", tokenHandler.GetToken)
					tokens.GET("/networks/:network_id", tokenHandler.ListTokensByNetwork)
					tokens.GET("/networks/:network_id/active", tokenHandler.ListActiveTokensByNetwork)
					tokens.GET("/networks/:network_id/gas", tokenHandler.GetGasToken)
					tokens.GET("/networks/:network_id/address/:address", tokenHandler.GetTokenByAddress)
				}
			}

			// Current Account routes
			accounts := protected.Group("/accounts")
			{
				accounts.PUT("/", accountHandler.UpdateAccount)
				accounts.POST("/onboard", accountHandler.OnboardAccount)
				accounts.GET("/details", accountHandler.GetAccountDetails)

				accounts.GET("/:account_id", accountHandler.GetAccount)
				accounts.PUT("/:account_id", accountHandler.UpdateAccount)
			}

			// Current User routes
			users := protected.Group("/users")
			{
				users.GET("/me", userHandler.GetCurrentUser)
				users.PUT("/me", userHandler.UpdateUser)
				users.GET("/supabase", userHandler.GetUserBySupabaseID)
			}

			// Customers
			customers := protected.Group("/customers")
			{
				customers.GET("", customerHandler.ListCustomers)
				customers.POST("", customerHandler.CreateCustomer)
				customers.GET("/:customer_id", customerHandler.GetCustomer)
				customers.PUT("/:customer_id", customerHandler.UpdateCustomer)
				customers.DELETE("/:customer_id", customerHandler.DeleteCustomer)

				// Customer subscriptions
				customers.GET("/:customer_id/subscriptions", subscriptionHandler.ListSubscriptionsByCustomer)
			}

			// API Keys
			apiKeys := protected.Group("/api-keys")
			{
				// Regular account routes (scoped to their workspace)
				apiKeys.GET("", apiKeyHandler.ListAPIKeys)
				apiKeys.POST("", apiKeyHandler.CreateAPIKey)
				apiKeys.GET("/count", apiKeyHandler.GetActiveAPIKeysCount)
				apiKeys.GET("/:api_key_id", apiKeyHandler.GetAPIKeyByID)
				apiKeys.PUT("/:api_key_id", apiKeyHandler.UpdateAPIKey)
				apiKeys.DELETE("/:api_key_id", apiKeyHandler.DeleteAPIKey)
			}

			// Products
			products := protected.Group("/products")
			{
				products.GET("", productHandler.ListProducts)
				products.POST("", productHandler.CreateProduct)
				products.POST("/:product_id/publish", productHandler.PublishProduct)
				products.GET("/:product_id", productHandler.GetProduct)
				products.PUT("/:product_id", productHandler.UpdateProduct)
				products.DELETE("/:product_id", productHandler.DeleteProduct)
				products.GET("/:product_id/networks/:network_id/tokens/:token_id", productHandler.GetProductTokenByIds)
				products.PUT("/:product_id/networks/:network_id/tokens/:token_id", productHandler.UpdateProductToken)
				products.DELETE("/:product_id/networks/:network_id/tokens/:token_id", productHandler.DeleteProductToken)

				// Product subscriptions
				products.GET("/:product_id/subscriptions", subscriptionHandler.ListSubscriptionsByProduct)
			}

			// Workspaces
			// workspaces := protected.Group("/workspaces")
			// {
			// 	workspaces.GET("", workspaceHandler.ListWorkspaces)
			// 	workspaces.POST("", workspaceHandler.CreateWorkspace)
			// 	workspaces.GET("/all", workspaceHandler.GetAllWorkspaces)
			// 	workspaces.GET("/:workspace_id", workspaceHandler.GetWorkspace)
			// 	workspaces.PUT("/:workspace_id", workspaceHandler.UpdateWorkspace)
			// 	workspaces.DELETE("/:workspace_id", workspaceHandler.DeleteWorkspace)
			// 	workspaces.DELETE("/:workspace_id/hard", workspaceHandler.HardDeleteWorkspace)
			// 	workspaces.GET("/:workspace_id/customers", workspaceHandler.ListWorkspaceCustomers)
			// 	workspaces.GET("/:workspace_id/products/active", productHandler.ListActiveProducts)
			// }

			// Wallets
			wallets := protected.Group("/wallets")
			{
				wallets.GET("", walletHandler.ListWallets)
				wallets.GET("/:wallet_id", walletHandler.GetWallet)
				wallets.POST("/:workspace_id", walletHandler.CreateWallet)
			}

			// Subscriptions
			subscriptions := protected.Group("/subscriptions")
			{
				subscriptions.GET("", subscriptionHandler.ListSubscriptions)
				// subscriptions.GET("/active", subscriptionHandler.ListActiveSubscriptions)
				// subscriptions.GET("/expired", subscriptionHandler.GetExpiredSubscriptions)
				// subscriptions.POST("", subscriptionHandler.CreateSubscription)
				// subscriptions.GET("/:subscription_id", subscriptionHandler.GetSubscription)
				// subscriptions.GET("/:subscription_id/details", subscriptionHandler.GetSubscriptionWithDetails)
				// subscriptions.PUT("/:subscription_id", subscriptionHandler.UpdateSubscription)
				// subscriptions.PATCH("/:subscription_id/status", subscriptionHandler.UpdateSubscriptionStatus)
				// subscriptions.POST("/:subscription_id/cancel", subscriptionHandler.CancelSubscription)
				// subscriptions.DELETE("/:subscription_id", subscriptionHandler.DeleteSubscription)
				// Subscription analytics
				// subscriptions.GET("/:subscription_id/total-amount", subscriptionEventHandler.GetTotalAmountBySubscription)
				// subscriptions.GET("/:subscription_id/redemption-count", subscriptionEventHandler.GetSuccessfulRedemptionCount)
				// subscriptions.GET("/:subscription_id/latest-event", subscriptionEventHandler.GetLatestSubscriptionEvent)
				// subscriptions.GET("/:subscription_id/events", subscriptionEventHandler.ListSubscriptionEventsBySubscription)
			}

			// Subscription events
			subEvents := protected.Group("/subscription-events")
			{
				subEvents.GET("/transactions", subscriptionEventHandler.ListSubscriptionEvents)
				// subEvents.POST("", subscriptionEventHandler.CreateSubscriptionEvent)
				// subEvents.GET("/:event_id", subscriptionEventHandler.GetSubscriptionEvent)
				// subEvents.PUT("/:event_id", subscriptionEventHandler.UpdateSubscriptionEvent)
				// subEvents.GET("/transaction/:tx_hash", subscriptionEventHandler.GetSubscriptionEventByTxHash)
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

			// Delegations
			// delegations := protected.Group("/delegations")
			// {
			// 	delegations.GET("/:delegation_id/subscriptions", subscriptionHandler.GetSubscriptionsByDelegation)
			// }
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
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Workspace-ID", "X-Account-ID"}
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
	}

	// Set credentials allowed
	corsConfig.AllowCredentials = os.Getenv("CORS_ALLOW_CREDENTIALS") == "true"

	return cors.New(corsConfig)
}
