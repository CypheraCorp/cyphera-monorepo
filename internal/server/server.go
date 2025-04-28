package server

import (
	"context"
	_ "cyphera-api/docs" // This will be generated
	"cyphera-api/internal/auth"
	"cyphera-api/internal/client/circle"
	client "cyphera-api/internal/client/delegation_server"
	dsClient "cyphera-api/internal/client/delegation_server"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"cyphera-api/internal/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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
)

func InitializeHandlers() {
	var dsn string // Database Source Name (connection string)

	// Determine environment and construct DSN accordingly
	// Use GIN_MODE=release as indicator for deployed environment
	if os.Getenv("GIN_MODE") == "release" {
		logger.Info("Running in deployed environment (GIN_MODE=release), fetching DB credentials from Secrets Manager")

		secretArn := os.Getenv("RDS_SECRET_ARN")
		dbEndpoint := os.Getenv("DB_HOST") // This contains host:port
		dbName := os.Getenv("DB_NAME")
		dbSSLMode := os.Getenv("DB_SSLMODE")

		if secretArn == "" || dbEndpoint == "" || dbName == "" {
			logger.Fatal("Missing required environment variables for DB connection in deployed environment (RDS_SECRET_ARN, DB_HOST, DB_NAME)")
		}
		if dbSSLMode == "" {
			dbSSLMode = "require" // Sensible default for RDS
			logger.Warn("DB_SSLMODE not set, defaulting to 'require'")
		}

		// Fetch secret from Secrets Manager
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			logger.Fatal("Unable to load AWS SDK config", zap.Error(err))
		}
		svc := secretsmanager.NewFromConfig(cfg)
		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretArn),
		}
		logger.Info("Fetching secret from Secrets Manager", zap.String("secretArn", secretArn))
		result, err := svc.GetSecretValue(context.TODO(), input)
		if err != nil {
			logger.Fatal("Failed to retrieve secret from Secrets Manager", zap.Error(err), zap.String("secretArn", secretArn))
		}

		if result.SecretString == nil {
			logger.Fatal("Secret string is nil", zap.String("secretArn", secretArn))
		}

		// Parse the secret string (it's JSON)
		var secretData map[string]interface{} // Use interface{} for flexibility
		err = json.Unmarshal([]byte(*result.SecretString), &secretData)
		if err != nil {
			logger.Fatal("Failed to unmarshal secret JSON", zap.Error(err))
		}

		dbUser, okUser := secretData["username"].(string)
		dbPassword, okPassword := secretData["password"].(string)

		if !okUser || !okPassword || dbUser == "" || dbPassword == "" {
			logger.Fatal("Username or password not found or not a string in secret data", zap.Any("secretKeysFound", getMapKeys(secretData)))
		}

		// Construct DSN for deployed environment
		// DB_HOST already contains host:port from RDS endpoint
		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
			url.QueryEscape(dbUser),     // URL-encode username
			url.QueryEscape(dbPassword), // URL-encode password
			dbEndpoint, dbName, dbSSLMode)
		logger.Info("Constructed DSN from Secrets Manager credentials")

	} else {
		// --- Local Development Environment ---
		logger.Info("Running in local environment (GIN_MODE != release), using DATABASE_URL from .env")
		dsn = os.Getenv("DATABASE_URL")
		if dsn == "" {
			logger.Fatal("DATABASE_URL environment variable is required for local development")
		}
	}

	// Create a connection pool using the determined DSN
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Unable to parse database DSN", zap.Error(err), zap.String("dsnUsed", dsn)) // Log the DSN used
	}

	// Configure the connection pool
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	// Create the connection pool
	connPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool", zap.Error(err))
	}

	// Create queries instance with the connection pool
	dbQueries = db.New(connPool)

	cypheraSmartWalletAddress := os.Getenv("CYPHERA_SMART_WALLET_ADDRESS")
	if cypheraSmartWalletAddress == "" {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS environment variable is required")
	}

	// validate cyphera wallet address
	if !handlers.IsAddressValid(cypheraSmartWalletAddress) {
		logger.Fatal("CYPHERA_SMART_WALLET_ADDRESS is not a valid address")
	}

	// Initialize the delegation client
	delegationClient, err = client.NewDelegationClient()
	if err != nil {
		logger.Fatal("Unable to create delegation client", zap.Error(err))
	}

	// validate the circle api key
	if os.Getenv("CIRCLE_API_KEY") == "" {
		logger.Fatal("CIRCLE_API_KEY environment variable is required")
	}

	// Initialize the circle client
	circleClient := circle.NewCircleClient(os.Getenv("CIRCLE_API_KEY"))

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

	// Initialize subscription handlers
	subscriptionHandler = handlers.NewSubscriptionHandler(commonServices, delegationClient)
	subscriptionEventHandler = handlers.NewSubscriptionEventHandler(commonServices)

	// 3rd party handlers
	circleHandler = handlers.NewCircleHandler(commonServices, circleClient)
}

func InitializeRoutes(router *gin.Engine) {
	// Initialize logger  first
	logger.InitLogger()

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
		protected.Use(auth.EnsureValidAPIKeyOrToken(dbQueries))
		{
			// Admin-only routes
			admin := protected.Group("/admin")
			admin.Use(auth.RequireRoles("admin"))
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

// Helper function to get map keys for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
