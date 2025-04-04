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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Handler Definitions
var (
	accountHandler                   *handlers.AccountHandler
	workspaceHandler                 *handlers.WorkspaceHandler
	customerHandler                  *handlers.CustomerHandler
	apiKeyHandler                    *handlers.APIKeyHandler
	userHandler                      *handlers.UserHandler
	networkHandler                   *handlers.NetworkHandler
	tokenHandler                     *handlers.TokenHandler
	productHandler                   *handlers.ProductHandler
	walletHandler                    *handlers.WalletHandler
	circleUserHandler                *handlers.CircleUserHandler
	subscriptionHandler              *handlers.SubscriptionHandler
	subscriptionEventHandler         *handlers.SubscriptionEventHandler
	failedSubscriptionAttemptHandler *handlers.FailedSubscriptionAttemptHandler
	delegationClient                 *dsClient.DelegationClient
	redemptionProcessor              *handlers.RedemptionProcessor
	circleClient                     *circle.CircleClient

	// Database
	dbQueries *db.Queries
)

func InitializeHandlers() {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Fatal("DATABASE_URL environment variable is required")
	}

	// Create a connection pool using pgxpool
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Fatal("Unable to parse database connection string", zap.Error(err))
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
	circleClient = circle.NewCircleClient(os.Getenv("CIRCLE_API_KEY"))

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
	circleUserHandler = handlers.NewCircleUserHandler(commonServices)

	// Initialize subscription handlers
	subscriptionHandler = handlers.NewSubscriptionHandler(commonServices, delegationClient)
	subscriptionEventHandler = handlers.NewSubscriptionEventHandler(commonServices)
	failedSubscriptionAttemptHandler = handlers.NewFailedSubscriptionAttemptHandler(commonServices)
}

func InitializeRoutes(router *gin.Engine) {
	// Initialize logger first
	logger.InitLogger()

	// Configure and apply CORS middleware
	router.Use(configureCORS())

	// Add Swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
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
			}

			// Current Account routes
			accounts := protected.Group("/accounts")
			{
				accounts.POST("/me/onboard", accountHandler.OnboardAccount)
				accounts.GET("/me/details", accountHandler.GetCurrentAccountDetails)
				accounts.PUT("/me", accountHandler.UpdateCurrentAccount)

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
			protected.GET("/customers", customerHandler.ListCustomers)
			protected.POST("/customers", customerHandler.CreateCustomer)
			protected.GET("/customers/:customer_id", customerHandler.GetCustomer)
			protected.PUT("/customers/:customer_id", customerHandler.UpdateCustomer)
			protected.DELETE("/customers/:customer_id", customerHandler.DeleteCustomer)

			// Customer subscriptions
			protected.GET("/customers/:customer_id/subscriptions", subscriptionHandler.ListSubscriptionsByCustomer)

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

			// Products
			products := protected.Group("/products")
			{
				products.GET("", productHandler.ListProducts)
				products.POST("", productHandler.CreateProduct)
				products.POST("/:product_id/publish", productHandler.PublishProduct)
				products.GET("/:product_id", productHandler.GetProduct)
				products.PUT("/:product_id", productHandler.UpdateProduct)
				products.DELETE("/:product_id", productHandler.DeleteProduct)

				// Product Tokens
				products.GET("/:product_id/tokens", productHandler.GetProductTokensByProduct)
				products.GET("/:product_id/tokens/active", productHandler.GetActiveProductTokensByProduct)
				products.POST("/:product_id/tokens", productHandler.CreateProductToken)
				products.GET("/:product_id/tokens/:token_id", productHandler.GetProductToken)
				products.GET("/:product_id/networks/:network_id/tokens", productHandler.GetProductTokensByNetwork)
				products.GET("/:product_id/networks/:network_id/tokens/active", productHandler.GetActiveProductTokensByNetwork)
				products.GET("/:product_id/networks/:network_id/tokens/:token_id", productHandler.GetProductTokenByIds)
				products.PUT("/:product_id/networks/:network_id/tokens/:token_id", productHandler.UpdateProductToken)
				products.DELETE("/:product_id/networks/:network_id/tokens/:token_id", productHandler.DeleteProductToken)
				products.DELETE("/:product_id/tokens", productHandler.DeleteProductTokensByProduct)

				// Product subscriptions
				products.GET("/:product_id/subscriptions", subscriptionHandler.ListSubscriptionsByProduct)
			}

			// Workspaces
			workspaces := protected.Group("/workspaces")
			{
				workspaces.GET("", workspaceHandler.ListWorkspaces)
				workspaces.POST("", workspaceHandler.CreateWorkspace)
				workspaces.GET("/all", workspaceHandler.GetAllWorkspaces)
				workspaces.GET("/:workspace_id", workspaceHandler.GetWorkspace)
				workspaces.PUT("/:workspace_id", workspaceHandler.UpdateWorkspace)
				workspaces.DELETE("/:workspace_id", workspaceHandler.DeleteWorkspace)
				workspaces.DELETE("/:workspace_id/hard", workspaceHandler.HardDeleteWorkspace)
				workspaces.GET("/:workspace_id/customers", workspaceHandler.ListWorkspaceCustomers)

				// Workspace Products
				workspaces.GET("/:workspace_id/products/active", productHandler.ListActiveProducts)
			}

			// Wallets
			wallets := protected.Group("/wallets")
			{
				// Basic CRUD operations
				wallets.POST("", walletHandler.CreateWallet)
				wallets.GET("", walletHandler.ListWallets)
				wallets.GET("/:wallet_id", walletHandler.GetWallet)
				wallets.PATCH("/:wallet_id", walletHandler.UpdateWallet)
				wallets.DELETE("/:wallet_id", walletHandler.DeleteWallet)
				wallets.GET("/address/:wallet_address", walletHandler.GetWalletByAddress)
				wallets.POST("/:wallet_id/primary", walletHandler.SetWalletAsPrimary)
			}

			// Circle Users
			circleUsers := protected.Group("/circle-users")
			{
				circleUsers.POST("", circleUserHandler.CreateCircleUser)
				circleUsers.GET("", circleUserHandler.ListCircleUsers)
				circleUsers.GET("/:id", circleUserHandler.GetCircleUserByID)
				circleUsers.GET("/account", circleUserHandler.GetCircleUserByAccountID)
				circleUsers.PATCH("/:id", circleUserHandler.UpdateCircleUser)
				circleUsers.PATCH("/account", circleUserHandler.UpdateCircleUserByAccountID)
				circleUsers.DELETE("/:id", circleUserHandler.DeleteCircleUser)
				circleUsers.DELETE("/account", circleUserHandler.DeleteCircleUserByAccountID)
				circleUsers.GET("/:id/wallets", circleUserHandler.GetCircleUserWithWallets)
				circleUsers.GET("/account/wallets", circleUserHandler.GetCircleUserWithWalletsByAccountID)
			}

			// Subscriptions
			subscriptions := protected.Group("/subscriptions")
			{
				subscriptions.GET("", subscriptionHandler.ListSubscriptions)
				subscriptions.GET("/active", subscriptionHandler.ListActiveSubscriptions)
				subscriptions.GET("/expired", subscriptionHandler.GetExpiredSubscriptions)
				subscriptions.POST("", subscriptionHandler.CreateSubscription)
				subscriptions.GET("/:subscription_id", subscriptionHandler.GetSubscription)
				subscriptions.GET("/:subscription_id/details", subscriptionHandler.GetSubscriptionWithDetails)
				subscriptions.PUT("/:subscription_id", subscriptionHandler.UpdateSubscription)
				subscriptions.PATCH("/:subscription_id/status", subscriptionHandler.UpdateSubscriptionStatus)
				subscriptions.POST("/:subscription_id/cancel", subscriptionHandler.CancelSubscription)
				subscriptions.DELETE("/:subscription_id", subscriptionHandler.DeleteSubscription)
				// Redemption endpoints
				subscriptions.POST("/redeem-due", subscriptionHandler.RedeemDueSubscriptionsHTTP)
				subscriptions.POST("/process-due", subscriptionHandler.ProcessDueSubscriptionsHTTP)
				// Subscription analytics
				subscriptions.GET("/:subscription_id/total-amount", subscriptionEventHandler.GetTotalAmountBySubscription)
				subscriptions.GET("/:subscription_id/redemption-count", subscriptionEventHandler.GetSuccessfulRedemptionCount)
				subscriptions.GET("/:subscription_id/latest-event", subscriptionEventHandler.GetLatestSubscriptionEvent)
				subscriptions.GET("/:subscription_id/events", subscriptionEventHandler.ListSubscriptionEventsBySubscription)
			}

			// Subscription events
			subEvents := protected.Group("/subscription-events")
			{
				subEvents.GET("", subscriptionEventHandler.ListSubscriptionEvents)
				subEvents.POST("", subscriptionEventHandler.CreateSubscriptionEvent)
				subEvents.GET("/:event_id", subscriptionEventHandler.GetSubscriptionEvent)
				subEvents.PUT("/:event_id", subscriptionEventHandler.UpdateSubscriptionEvent)
				subEvents.GET("/transaction/:tx_hash", subscriptionEventHandler.GetSubscriptionEventByTxHash)
				subEvents.GET("/type/:event_type", subscriptionEventHandler.ListSubscriptionEventsByType)
				subEvents.GET("/failed", subscriptionEventHandler.ListFailedSubscriptionEvents)
				subEvents.GET("/recent", subscriptionEventHandler.ListRecentSubscriptionEvents)
			}

			// Failed subscription attempts
			failedAttempts := protected.Group("/failed-subscription-attempts")
			{
				failedAttempts.GET("", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttempts)
				failedAttempts.GET("/:attempt_id", failedSubscriptionAttemptHandler.GetFailedSubscriptionAttempt)
				failedAttempts.GET("/customer/:customer_id", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttemptsByCustomer)
				failedAttempts.GET("/product/:product_id", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttemptsByProduct)
				failedAttempts.GET("/error-type/:error_type", failedSubscriptionAttemptHandler.ListFailedSubscriptionAttemptsByErrorType)
			}

			// Delegations
			delegations := protected.Group("/delegations")
			{
				delegations.GET("/:delegation_id/subscriptions", subscriptionHandler.GetSubscriptionsByDelegation)
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
