package server

import (
	"context"
	_ "cyphera-api/docs" // This will be generated
	"cyphera-api/internal/auth"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"cyphera-api/internal/logger"
	"cyphera-api/internal/pkg/actalink"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handler Definitions
var (
	accountHandler   *handlers.AccountHandler
	workspaceHandler *handlers.WorkspaceHandler
	customerHandler  *handlers.CustomerHandler
	apiKeyHandler    *handlers.APIKeyHandler
	userHandler      *handlers.UserHandler
	networkHandler   *handlers.NetworkHandler
	tokenHandler     *handlers.TokenHandler
	productHandler   *handlers.ProductHandler
	walletHandler    *handlers.WalletHandler

	// Actalink
	actalinkHandler *handlers.ActalinkHandler

	// Database
	dbQueries *db.Queries
)

func InitializeHandlers() {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Create queries instance
	dbQueries = db.New(conn)

	apiKey := os.Getenv("ACTALINK_API_KEY")
	if apiKey == "" {
		log.Fatal("ACTALINK_API_KEY environment variable is required")
	}

	// common services initialization
	actalinkClient := actalink.NewActaLinkClient(apiKey)

	commonServices := handlers.NewCommonServices(
		dbQueries,
		actalinkClient,
	)

	// API Handler initialization
	accountHandler = handlers.NewAccountHandler(commonServices)
	workspaceHandler = handlers.NewWorkspaceHandler(commonServices)
	customerHandler = handlers.NewCustomerHandler(commonServices)
	apiKeyHandler = handlers.NewAPIKeyHandler(commonServices)
	userHandler = handlers.NewUserHandler(commonServices)
	networkHandler = handlers.NewNetworkHandler(commonServices)
	tokenHandler = handlers.NewTokenHandler(commonServices)
	productHandler = handlers.NewProductHandler(commonServices)
	walletHandler = handlers.NewWalletHandler(commonServices)
	// Actalink Handler initialization
	actalinkHandler = handlers.NewActalinkHandler(commonServices)
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

				// Specialized endpoints
				wallets.GET("/stats", walletHandler.GetWalletStats)
				wallets.GET("/recent", walletHandler.GetRecentlyUsedWallets)
				wallets.GET("/ens", walletHandler.GetWalletsByENS)
				wallets.GET("/search", walletHandler.SearchWallets)
				wallets.GET("/network/:network_type", walletHandler.ListWalletsByNetworkType)
			}

			// ActaLink routes
			actalink := protected.Group("/actalink")
			{
				// Nonce
				actalink.GET("/nonce", actalinkHandler.GetNonce)

				// Account
				actalink.GET("/isuseravailable", actalinkHandler.CheckUserAvailability)
				actalink.POST("/register", actalinkHandler.RegisterActalinkUser)
				actalink.POST("/login", actalinkHandler.LoginActalinkUser)

				// Subscription
				actalink.POST("/subscriptions", actalinkHandler.CreateSubscription)
				actalink.DELETE("/subscriptions", actalinkHandler.DeleteSubscription)
				actalink.GET("/subscriptions", actalinkHandler.GetAllSubscriptions)

				// Subscribers
				actalink.GET("/subscribers", actalinkHandler.GetSubscribers)

				// Operations
				actalink.GET("/operations", actalinkHandler.GetOperations)

				// Tokens
				actalink.GET("/tokens", actalinkHandler.GetTokens)

				// Networks
				actalink.GET("/networks", actalinkHandler.GetNetworks)
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
