package handlers

import (
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"cyphera-api/internal/pkg/actalink"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// CommonServices holds common dependencies used across handlers
type CommonServices struct {
	db       *db.Queries
	actalink *actalink.ActaLinkClient
	// other shared dependencies
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// NewCommonServices creates a new instance of CommonServices
func NewCommonServices(db *db.Queries, actalink *actalink.ActaLinkClient) *CommonServices {
	return &CommonServices{
		db:       db,
		actalink: actalink,
	}
}

// sendError is a helper function that combines logging and error response
// It logs the error with the given message and sends a JSON error response
func sendError(c *gin.Context, statusCode int, message string, err error) {
	logger.Error(message,
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
	)
	c.JSON(statusCode, ErrorResponse{Error: message})
}

// handleDBError is a helper function that handles database errors and returns appropriate HTTP status codes
func handleDBError(c *gin.Context, err error, notFoundMsg string) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		sendError(c, http.StatusNotFound, notFoundMsg, err)
	default:
		sendError(c, http.StatusInternalServerError, "Internal server error", err)
	}
}

// sendSuccess is a helper function that sends a success response
func sendSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// sendSuccessMessage is a helper function that sends a success message
func sendSuccessMessage(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, SuccessResponse{Message: message})
}

// sendList is a helper function that sends a paginated list response
func sendList(c *gin.Context, items interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   items,
	})
}
