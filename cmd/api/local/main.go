//go:build !lambda
// +build !lambda

package main

import (
	"cyphera-api/internal/server"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	server.InitializeHandlers()
	server.InitializeRoutes(r)
	r.Run(":8000")
}
