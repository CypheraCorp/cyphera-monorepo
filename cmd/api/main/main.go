//go:build lambda
// +build lambda

package main

import (
	"context"

	_ "cyphera-api/docs" // This will be generated
	"cyphera-api/internal/logger"
	"cyphera-api/internal/server"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

var ginLambda *ginadapter.GinLambda

func init() {
	// Initialize logger
	logger.InitLogger()

	// Initialize your Gin router
	r := gin.Default()

	// Initialize Handlers
	server.InitializeHandlers()

	// Initialize routes
	server.InitializeRoutes(r)

	ginLambda = ginadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add debug logging
	logger.Debug("Received Lambda request",
		zap.String("path", req.Path),
		zap.Any("request", spew.Sdump(req)),
	)

	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	defer logger.Sync()
	lambda.Start(Handler)
}
