package server

import (
	"context"
	_ "cyphera-api/docs" // This will be generated
	"cyphera-api/internal/auth"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"cyphera-api/internal/pkg/actalink"
	"log"
	"net/http"
	"os"

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
	// Actalink Handler initialization
	actalinkHandler = handlers.NewActalinkHandler(commonServices)
}

func InitializeRoutes(router *gin.Engine) {
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
		router.Use(handlers.LogRequestBody())
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
				// Account management
				admin.GET("/accounts", accountHandler.ListAccounts)
				admin.POST("/accounts", accountHandler.CreateAccount)
				admin.POST("/accounts/signin", accountHandler.SignInAccount)
				admin.DELETE("/accounts/:id", accountHandler.DeleteAccount)

				// User management
				admin.POST("/users", userHandler.CreateUser)
				admin.GET("/users/:id", userHandler.GetUser)
				admin.PUT("/users/:id", userHandler.UpdateUser)
				admin.DELETE("/users/:id", userHandler.DeleteUser)

				// Workspace management
				admin.GET("/workspaces", workspaceHandler.ListWorkspaces)
				admin.POST("/workspaces", workspaceHandler.CreateWorkspace)
				admin.GET("/workspaces/all", workspaceHandler.GetAllWorkspaces)
				admin.GET("/workspaces/:id", workspaceHandler.GetWorkspace)
				admin.PUT("/workspaces/:id", workspaceHandler.UpdateWorkspace)
				admin.DELETE("/workspaces/:id", workspaceHandler.DeleteWorkspace)
				admin.DELETE("/workspaces/:id/hard", workspaceHandler.HardDeleteWorkspace)
				admin.GET("/workspaces/:id/customers", workspaceHandler.ListWorkspaceCustomers)

				// API Key management
				admin.GET("/api-keys", apiKeyHandler.GetAllAPIKeys)
				admin.GET("/api-keys/expired", apiKeyHandler.GetExpiredAPIKeys)

				// Network management
				admin.POST("/networks", networkHandler.CreateNetwork)
				admin.PUT("/networks/:id", networkHandler.UpdateNetwork)
				admin.DELETE("/networks/:id", networkHandler.DeleteNetwork)

				// Token management
				admin.POST("/tokens", tokenHandler.CreateToken)
				admin.PUT("/tokens/:id", tokenHandler.UpdateToken)
				admin.DELETE("/tokens/:id", tokenHandler.DeleteToken)
			}

			// Current Account routes
			accounts := protected.Group("/accounts")
			{
				accounts.GET("/me/details", accountHandler.GetCurrentAccountDetails)
				accounts.PUT("/me", accountHandler.UpdateCurrentAccount)

				accounts.GET("/:id", accountHandler.GetAccount)
				accounts.PUT("/:id", accountHandler.UpdateAccount)
			}

			// Current User routes
			users := protected.Group("/users")
			{
				users.GET("/me", userHandler.GetCurrentUser)
				users.PUT("/me", userHandler.UpdateUser)
				users.GET("/auth0", userHandler.GetUserByAuth0ID)
			}

			// Customers
			protected.GET("/customers", customerHandler.ListCustomers)
			protected.POST("/customers", customerHandler.CreateCustomer)
			protected.GET("/customers/:id", customerHandler.GetCustomer)
			protected.PUT("/customers/:id", customerHandler.UpdateCustomer)
			protected.DELETE("/customers/:id", customerHandler.DeleteCustomer)

			// API Keys
			apiKeys := protected.Group("/api-keys")
			{
				// Regular account routes (scoped to their workspace)
				apiKeys.GET("", apiKeyHandler.ListAPIKeys)
				apiKeys.POST("", apiKeyHandler.CreateAPIKey)
				apiKeys.GET("/count", apiKeyHandler.GetActiveAPIKeysCount)
				apiKeys.GET("/:id", apiKeyHandler.GetAPIKeyByID)
				apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
				apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
			}

			// Networks
			networks := protected.Group("/networks")
			{
				networks.GET("", networkHandler.ListNetworks)
				networks.GET("/active", networkHandler.ListActiveNetworks)
				networks.GET("/:id", networkHandler.GetNetwork)
				networks.GET("/chain/:chain_id", networkHandler.GetNetworkByChainID)
			}

			// Tokens
			tokens := protected.Group("/tokens")
			{
				tokens.GET("", tokenHandler.ListTokens)
				tokens.GET("/:id", tokenHandler.GetToken)
				tokens.GET("/network/:network_id", tokenHandler.ListTokensByNetwork)
				tokens.GET("/network/:network_id/active", tokenHandler.ListActiveTokensByNetwork)
				tokens.GET("/network/:network_id/gas", tokenHandler.GetGasToken)
				tokens.GET("/network/:network_id/address/:address", tokenHandler.GetTokenByAddress)
			}

			// Products
			products := protected.Group("/products")
			{
				products.GET("", productHandler.ListProducts)
				products.POST("", productHandler.CreateProduct)
				products.GET("/:id", productHandler.GetProduct)
				products.PUT("/:id", productHandler.UpdateProduct)
				products.DELETE("/:id", productHandler.DeleteProduct)

				// Product Tokens
				products.GET("/:product_id/tokens", productHandler.GetProductTokensByProduct)
				products.GET("/:product_id/tokens/active", productHandler.GetActiveProductTokensByProduct)
				products.POST("/:product_id/tokens", productHandler.CreateProductToken)
				products.GET("/:product_id/tokens/:id", productHandler.GetProductToken)
				products.GET("/:product_id/networks/:network_id/tokens", productHandler.GetProductTokensByNetwork)
				products.GET("/:product_id/networks/:network_id/tokens/active", productHandler.GetActiveProductTokensByNetwork)
				products.GET("/:product_id/networks/:network_id/tokens/:token_id", productHandler.GetProductTokenByIds)
				products.PUT("/:product_id/networks/:network_id/tokens/:token_id", productHandler.UpdateProductToken)
				products.DELETE("/:product_id/networks/:network_id/tokens/:token_id", productHandler.DeleteProductToken)
			}

			// Workspaces
			workspaces := protected.Group("/workspaces")
			{
				workspaces.GET("", workspaceHandler.ListWorkspaces)
				workspaces.POST("", workspaceHandler.CreateWorkspace)
				workspaces.GET("/all", workspaceHandler.GetAllWorkspaces)
				workspaces.GET("/:id", workspaceHandler.GetWorkspace)
				workspaces.PUT("/:id", workspaceHandler.UpdateWorkspace)
				workspaces.DELETE("/:id", workspaceHandler.DeleteWorkspace)
				workspaces.DELETE("/:id/hard", workspaceHandler.HardDeleteWorkspace)
				workspaces.GET("/:id/customers", workspaceHandler.ListWorkspaceCustomers)

				// Workspace Products
				workspaces.GET("/:workspace_id/products", productHandler.ListProducts)
				workspaces.GET("/:workspace_id/products/active", productHandler.ListActiveProducts)
			}

			// ActaLink routes
			actalink := protected.Group("/actalink")
			{
				// Nonce
				actalink.GET("/nonce", actalinkHandler.GetNonce)

				// Account
				actalink.GET("/isuseravailable", actalinkHandler.CheckUserAvailability)
				actalink.POST("/accounts/register", actalinkHandler.RegisterActalinkUser)
				actalink.POST("/accounts/login", actalinkHandler.LoginActalinkUser)

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
