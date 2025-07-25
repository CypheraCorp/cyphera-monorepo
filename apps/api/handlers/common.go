package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// CommonServices holds common dependencies used across handlers
type CommonServices struct {
	db                        *db.Queries
	cypheraSmartWalletAddress string
	CMCClient                 *coinmarketcap.Client
	APIKeyService             *services.APIKeyService
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
func NewCommonServices(db *db.Queries, cypheraSmartWalletAddress string, cmcClient *coinmarketcap.Client) *CommonServices {
	return &CommonServices{
		db:                        db,
		cypheraSmartWalletAddress: cypheraSmartWalletAddress,
		CMCClient:                 cmcClient,
		APIKeyService:             services.NewAPIKeyService(db),
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

// GetDBPool returns the underlying pgxpool.Pool from the db.Queries
// Returns an error if the underlying connection is not a pool
func (s *CommonServices) GetDBPool() (*pgxpool.Pool, error) {
	dbtx := s.db.GetDBTX()
	pool, ok := dbtx.(*pgxpool.Pool)
	if !ok {
		return nil, errors.New("database connection is not a pool")
	}
	return pool, nil
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

// RunInTransaction executes a function within a database transaction using the helper
// It automatically handles commit/rollback and provides a queries instance that uses the transaction
func (s *CommonServices) RunInTransaction(ctx context.Context, fn func(qtx *db.Queries) error) error {
	pool, err := s.GetDBPool()
	if err != nil {
		return err
	}

	return helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		// Create queries instance that uses this transaction
		qtx := s.db.WithTx(tx)
		return fn(qtx)
	})
}

// RunInTransactionWithRetry executes a function within a database transaction with retry logic
func (s *CommonServices) RunInTransactionWithRetry(ctx context.Context, maxRetries int, fn func(qtx *db.Queries) error) error {
	pool, err := s.GetDBPool()
	if err != nil {
		return err
	}

	return helpers.WithTransactionRetry(ctx, pool, maxRetries, func(tx pgx.Tx) error {
		qtx := s.db.WithTx(tx)
		return fn(qtx)
	})
}

// GetCypheraSmartWalletAddress returns the Cyphera smart wallet address
func (s *CommonServices) GetCypheraSmartWalletAddress() string {
	return s.cypheraSmartWalletAddress
}

// IsAddressValid checks if the provided string is a valid Ethereum address
// It verifies:
// 1. The address is exactly 42 characters long
// 2. The address starts with "0x"
// 3. The remaining 40 characters are valid hexadecimal
func IsAddressValid(address string) bool {
	// Check length
	if len(address) != 42 {
		return false
	}

	// Check "0x" prefix
	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Check if the address contains only hex characters after the 0x prefix
	for _, c := range address[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// IsPrivateKeyValid checks if the provided string is a valid Ethereum private key
// It verifies:
// 1. The key is exactly 66 characters long (including 0x prefix)
// 2. The key starts with "0x"
// 3. The remaining 64 characters are valid hexadecimal
func IsPrivateKeyValid(key string) bool {
	// Check length (32 bytes = 64 hex chars + 2 chars for "0x")
	if len(key) != 66 {
		return false
	}

	// Check "0x" prefix
	if !strings.HasPrefix(key, "0x") {
		return false
	}

	// Check if the key contains only hex characters after the 0x prefix
	for _, c := range key[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// sendError is a helper function that combines logging and error response
// It logs the error with the given message and sends a JSON error response
func sendError(c *gin.Context, statusCode int, message string, err error) {
	// Get correlation ID from context
	correlationID := ""
	if id, exists := c.Get("correlationID"); exists {
		correlationID, _ = id.(string)
	}

	logger.Error(message,
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("correlation_id", correlationID),
	)

	// Include correlation ID in error response for debugging
	response := struct {
		Error         string `json:"error"`
		CorrelationID string `json:"correlation_id,omitempty"`
	}{
		Error:         message,
		CorrelationID: correlationID,
	}

	c.JSON(statusCode, response)
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

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Object     string      `json:"object"`
	HasMore    bool        `json:"has_more"`
	Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	TotalItems  int `json:"total_items"`
	TotalPages  int `json:"total_pages"`
}

// sendPaginatedSuccess sends a successful paginated response
func sendPaginatedSuccess(c *gin.Context, statusCode int, data interface{}, page, limit, total int) PaginatedResponse {
	hasMore := (total+limit-1)/limit > page
	response := PaginatedResponse{
		Data:    data,
		Object:  "list",
		HasMore: hasMore,
		Pagination: Pagination{
			CurrentPage: page,
			PerPage:     limit,
			TotalItems:  total,
			TotalPages:  (total + limit - 1) / limit,
		},
	}
	return response
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

// validatePaginationParams validates and returns limit and page parameters
func validatePaginationParams(c *gin.Context) (limit int32, page int32, err error) {
	const maxLimit int32 = 100
	limit = 10

	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit parameter")
		}
		if parsedLimit > int64(maxLimit) {
			limit = maxLimit
		} else if parsedLimit > 0 {
			limit = int32(parsedLimit)
		}
	}

	if pageStr := c.Query("page"); pageStr != "" {
		parsedPage, err := strconv.ParseInt(pageStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid page parameter")
		}
		if parsedPage > 0 {
			page = int32(parsedPage)
		}
	}

	return limit, page, nil
}
