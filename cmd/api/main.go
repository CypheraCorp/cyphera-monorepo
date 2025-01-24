package main

import (
	"context"
	"cyphera-api/internal/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "cyphera-api/docs" // This will be generated

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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
	// Initialize router
	router := gin.Default()

	// Initialize routes
	initializeRoutes(router)

	// Configure server
	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
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
		// Nonce
		v1.GET("/nonce", handlers.GetNonce)

		// User
		v1.GET("/user", handlers.CheckUserAvailability)
		v1.POST("/user/register", handlers.RegisterUser)
		v1.POST("/user/login", handlers.LoginUser)
		v1.POST("/user/subscription", handlers.CreateSubscription)
		v1.DELETE("/user/subscription", handlers.DeleteSubscription)

		// Subscriptions
		v1.GET("/subscription", handlers.GetAllSubscriptions)

		// Subscribers
		v1.GET("/subscribers", handlers.GetSubscribers)

		// Operations
		v1.GET("/operations/", handlers.GetOperations)

		// Tokens
		v1.GET("/tokens", handlers.GetTokens)

		// Networks
		v1.GET("/networks", handlers.GetNetworks)
	}
}
