//go:build !lambda
// +build !lambda

package main

import (
	"cyphera-api/internal/server"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	server.InitializeHandlers()
	server.InitializeRoutes(r)

	log.Printf("Server starting on :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
