package handlers

import (
	"context"
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// CommonServices holds common dependencies used across handlers
type CommonServices struct {
	db *db.Queries
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
func NewCommonServices(db *db.Queries) *CommonServices {
	return &CommonServices{
		db: db,
	}
}

// GetDBConn returns the underlying pgx.Conn or pgxpool.Pool from the db.Queries
// This is used for starting transactions
func (s *CommonServices) GetDBConn() (interface{}, error) {
	// Get the underlying DBTX interface
	dbtx := s.db.GetDBTX()

	// Check if it's a type that can begin transactions
	if _, ok := dbtx.(*pgxpool.Pool); ok {
		return dbtx, nil
	}

	if _, ok := dbtx.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}); ok {
		return dbtx, nil
	}

	return nil, errors.New("database connection does not support transactions")
}

// WithTx creates a new db.Queries instance that uses the provided transaction
func (s *CommonServices) WithTx(tx pgx.Tx) *db.Queries {
	return s.db.WithTx(tx)
}

// BeginTx starts a transaction and returns:
// - The transaction object (caller is responsible for committing or rolling back)
// - A new db.Queries instance that uses the transaction
// - Any error that occurred
func (s *CommonServices) BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	conn, err := s.GetDBConn()
	if err != nil {
		return nil, nil, err
	}

	var tx pgx.Tx

	// Try to cast the connection to different types that can begin a transaction
	if pool, ok := conn.(*pgxpool.Pool); ok {
		tx, err = pool.Begin(ctx)
	} else if pgxConn, ok := conn.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}); ok {
		tx, err = pgxConn.Begin(ctx)
	} else {
		return nil, nil, errors.New("database connection does not support transactions")
	}

	if err != nil {
		return nil, nil, err
	}

	// Create a queries instance that uses this transaction
	qtx := s.WithTx(tx)

	return tx, qtx, nil
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

// sendPaginatedSuccess sends a successful paginated response
func sendPaginatedSuccess(c *gin.Context, statusCode int, data interface{}, page, limit, total int) {
	response := gin.H{
		"data": data,
		"pagination": gin.H{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  (total + limit - 1) / limit,
		},
	}
	c.JSON(statusCode, response)
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
