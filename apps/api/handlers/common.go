package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Common error messages used across handlers
const (
	errMsgInvalidWorkspaceIDFormat = "Invalid workspace ID format"
)

// CommonServices holds all the common services and dependencies used by handlers
type CommonServices struct {
	db                        db.Querier
	dbPool                    *pgxpool.Pool // Optional: for transaction support
	cypheraSmartWalletAddress string
	CMCClient                 *coinmarketcap.Client
	CMCAPIKey                 string
	APIKeyService             interfaces.APIKeyService
	logger                    *zap.Logger
	TaxService                interfaces.TaxService
	DiscountService           interfaces.DiscountService
	GasSponsorshipService     interfaces.GasSponsorshipService
	CurrencyService           interfaces.CurrencyService
	ExchangeRateService       interfaces.ExchangeRateService
	// other shared dependencies
}

// Use types from the centralized responses package
type ErrorResponse = responses.ErrorResponse
type SuccessResponse = responses.SuccessResponse
type PaginatedResponse = responses.PaginatedResponse
type Pagination = responses.Pagination

// CommonServicesConfig contains all dependencies needed to create CommonServices
type CommonServicesConfig struct {
	DB                        db.Querier
	DBPool                    *pgxpool.Pool // Optional: for transaction support
	CypheraSmartWalletAddress string
	CMCClient                 *coinmarketcap.Client
	CMCAPIKey                 string
	APIKeyService             interfaces.APIKeyService
	Logger                    *zap.Logger
	TaxService                interfaces.TaxService
	DiscountService           interfaces.DiscountService
	GasSponsorshipService     interfaces.GasSponsorshipService
	CurrencyService           interfaces.CurrencyService
	ExchangeRateService       interfaces.ExchangeRateService
}

// NewCommonServices creates a new instance of CommonServices with interface dependencies
func NewCommonServices(config CommonServicesConfig) *CommonServices {
	if config.Logger == nil {
		config.Logger = logger.Log
	}

	return &CommonServices{
		db:                        config.DB,
		dbPool:                    config.DBPool,
		cypheraSmartWalletAddress: config.CypheraSmartWalletAddress,
		CMCClient:                 config.CMCClient,
		CMCAPIKey:                 config.CMCAPIKey,
		APIKeyService:             config.APIKeyService,
		logger:                    config.Logger,
		TaxService:                config.TaxService,
		DiscountService:           config.DiscountService,
		GasSponsorshipService:     config.GasSponsorshipService,
		CurrencyService:           config.CurrencyService,
		ExchangeRateService:       config.ExchangeRateService,
	}
}

// NewCommonServicesWithPool creates CommonServices with database pool for transaction support
// This is the recommended constructor when you need transaction support
func NewCommonServicesWithPool(db *db.Queries, pool *pgxpool.Pool, cypheraSmartWalletAddress string, cmcClient *coinmarketcap.Client, cmcAPIKey string) *CommonServices {
	// Initialize logger
	log := logger.Log

	// Initialize services
	currencyService := services.NewCurrencyService(db)
	exchangeRateService := services.NewExchangeRateService(db, cmcAPIKey)
	taxService := services.NewTaxService(db)
	discountService := services.NewDiscountService(db)
	gasSponsorshipService := services.NewGasSponsorshipService(db)

	return &CommonServices{
		db:                        db,
		dbPool:                    pool,
		cypheraSmartWalletAddress: cypheraSmartWalletAddress,
		CMCClient:                 cmcClient,
		CMCAPIKey:                 cmcAPIKey,
		APIKeyService:             services.NewAPIKeyService(db),
		logger:                    log,
		TaxService:                taxService,
		DiscountService:           discountService,
		GasSponsorshipService:     gasSponsorshipService,
		CurrencyService:           currencyService,
		ExchangeRateService:       exchangeRateService,
	}
}

// GetDB returns the database querier
func (s *CommonServices) GetDB() db.Querier {
	return s.db
}

// GetDBConn returns the underlying database connection
// This is a temporary method for compatibility - should be refactored
func (s *CommonServices) GetDBConn() (interface{}, error) {
	// If db is actually a *db.Queries, it should have access to the underlying connection
	// For now, returning the db itself
	return s.db, nil
}

// GetDBPool returns the underlying database pool
// This is a temporary method for compatibility - should be refactored
func (s *CommonServices) GetDBPool() (*pgxpool.Pool, error) {
	if s.dbPool != nil {
		return s.dbPool, nil
	}
	// This is a limitation of using the Querier interface
	// In production, you should pass the pool separately or refactor this
	return nil, errors.New("pool not available - please provide DBPool in CommonServicesConfig")
}

// WithTx creates a new db.Queries instance that uses the provided transaction
func (s *CommonServices) WithTx(tx pgx.Tx) *db.Queries {
	// Since we're using the Querier interface, we need to type assert to *db.Queries
	if queries, ok := s.db.(*db.Queries); ok {
		return queries.WithTx(tx)
	}
	// Return nil if not a *db.Queries (shouldn't happen in production)
	return nil
}

// BeginTx starts a transaction and returns:
// - The transaction object (caller is responsible for committing or rolling back)
// - A new db.Queries instance that uses the transaction
// - Any error that occurred
func (s *CommonServices) BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	if s.dbPool == nil {
		return nil, nil, errors.New("database pool not configured - please provide DBPool in CommonServicesConfig")
	}

	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Create a queries instance that uses this transaction
	qtx := s.WithTx(tx)
	if qtx == nil {
		// Rollback if we can't create queries with transaction
		_ = tx.Rollback(ctx)
		return nil, nil, errors.New("failed to create queries with transaction")
	}

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
		qtx := s.WithTx(tx)
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
		qtx := s.WithTx(tx)
		return fn(qtx)
	})
}

// GetCypheraSmartWalletAddress returns the Cyphera smart wallet address
func (s *CommonServices) GetCypheraSmartWalletAddress() string {
	return s.cypheraSmartWalletAddress
}

// GetLogger returns the logger
func (s *CommonServices) GetLogger() *zap.Logger {
	return s.logger
}

// GetAPIKeyService returns the API key service interface
func (s *CommonServices) GetAPIKeyService() interfaces.APIKeyService {
	return s.APIKeyService
}

// GetTaxService returns the tax service interface
func (s *CommonServices) GetTaxService() interfaces.TaxService {
	return s.TaxService
}

// GetDiscountService returns the discount service interface
func (s *CommonServices) GetDiscountService() interfaces.DiscountService {
	return s.DiscountService
}

// GetGasSponsorshipService returns the gas sponsorship service interface
func (s *CommonServices) GetGasSponsorshipService() interfaces.GasSponsorshipService {
	return s.GasSponsorshipService
}

// GetCurrencyService returns the currency service interface
func (s *CommonServices) GetCurrencyService() interfaces.CurrencyService {
	return s.CurrencyService
}

// GetExchangeRateService returns the exchange rate service interface
func (s *CommonServices) GetExchangeRateService() interfaces.ExchangeRateService {
	return s.ExchangeRateService
}

// HandleError is a helper method to handle errors consistently
func (s *CommonServices) HandleError(c *gin.Context, err error, message string, statusCode int, logger *zap.Logger) {
	if err != nil {
		logger.Error(message,
			zap.Error(err),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method))
	}

	c.JSON(statusCode, ErrorResponse{
		Error: message,
	})
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

// CreateMockCommonServices creates a CommonServices instance with mock interfaces for testing
// This is useful when you want to test handlers without actual database connections
func CreateMockCommonServices(
	db db.Querier,
	apiKeyService interfaces.APIKeyService,
	taxService interfaces.TaxService,
	discountService interfaces.DiscountService,
	gasSponsorshipService interfaces.GasSponsorshipService,
	currencyService interfaces.CurrencyService,
	exchangeRateService interfaces.ExchangeRateService,
) *CommonServices {
	return &CommonServices{
		db:                        db,
		dbPool:                    nil, // No pool for mocks
		cypheraSmartWalletAddress: "0xMockAddress",
		CMCClient:                 nil,
		CMCAPIKey:                 "mock-api-key",
		APIKeyService:             apiKeyService,
		logger:                    zap.NewNop(), // No-op logger for tests
		TaxService:                taxService,
		DiscountService:           discountService,
		GasSponsorshipService:     gasSponsorshipService,
		CurrencyService:           currencyService,
		ExchangeRateService:       exchangeRateService,
	}
}
