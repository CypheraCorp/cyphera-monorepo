//go:build !lambda
// +build !lambda

package main

import (
	"cyphera-api/internal/logger"
	"cyphera-api/internal/server"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		// It's often okay if the .env file is missing, especially in production
		// where variables might be set directly in the environment.
		// Log it but don't necessarily stop the application.
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Initialize logger first
	logger.InitLogger()

	r := gin.Default()
	server.InitializeHandlers()
	server.InitializeRoutes(r)

	log.Printf("Server starting on :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
