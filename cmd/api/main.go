package main

import (
	"context"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "cyphera-api/docs" // This will be generated

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handler Definitions
var handlerClient *handlers.HandlerClient

// @title           Cyphera API
// @version         1.0
// @description     API Server for Cyphera application
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8000
// @BasePath  /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v\n", err)
	}

	// Initialize router
	router := gin.Default()

	// Initialize Handlers
	initializeHandlers()

	// Initialize routes
	initializeRoutes(router)

	// Get port from environment variable or use default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8000"
	}

	// Configure server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}
	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func initializeHandlers() {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to the database
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Create queries instance
	dbQueries := db.New(conn)

	apiKey := os.Getenv("ACTALINK_API_KEY")
	if apiKey == "" {
		log.Fatal("ACTALINK_API_KEY environment variable is required")
	}

	handlerClient = handlers.NewHandlerClient(apiKey, dbQueries)
}

func initializeRoutes(router *gin.Engine) {
	// Add Swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{

		//
		v1.GET("/customers/:id", handlerClient.GetCustomerByID)
		v1.GET("/api-keys/:id", handlerClient.GetAPIKeyByID)

		// actalink group
		actalink := v1.Group("/actalink")
		{
			// Nonce
			actalink.GET("/nonce", handlerClient.GetNonce)

			// User
			actalink.GET("/users", handlerClient.CheckUserAvailability)
			actalink.POST("/users/register", handlerClient.RegisterUser)
			actalink.POST("/users/login", handlerClient.LoginUser)

			// Subscription
			actalink.POST("/subscriptions", handlerClient.CreateSubscription)
			actalink.DELETE("/subscriptions", handlerClient.DeleteSubscription)
			actalink.GET("/subscriptions", handlerClient.GetAllSubscriptions)

			// Subscribers
			actalink.GET("/subscribers", handlerClient.GetSubscribers)

			// Operations
			actalink.GET("/operations", handlerClient.GetOperations)

			// Tokens
			actalink.GET("/tokens", handlerClient.GetTokens)

			// Networks
			actalink.GET("/networks", handlerClient.GetNetworks)
		}
	}
}
