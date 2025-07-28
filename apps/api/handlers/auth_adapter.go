package handlers

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AuthServicesAdapter adapts CommonServices to interfaces.CommonServicesInterface
type AuthServicesAdapter struct {
	common *CommonServices
}

// NewAuthServicesAdapter creates a new adapter
func NewAuthServicesAdapter(common *CommonServices) interfaces.CommonServicesInterface {
	return &AuthServicesAdapter{
		common: common,
	}
}

// GetDB returns the database querier
func (a *AuthServicesAdapter) GetDB() db.Querier {
	return a.common.db
}

// GetDBPool returns the database connection pool
func (a *AuthServicesAdapter) GetDBPool() (*pgxpool.Pool, error) {
	return a.common.GetDBPool()
}

// WithTx returns a new db.Queries instance that uses the provided transaction
func (a *AuthServicesAdapter) WithTx(tx pgx.Tx) *db.Queries {
	return a.common.WithTx(tx)
}

// BeginTx starts a transaction
func (a *AuthServicesAdapter) BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	return a.common.BeginTx(ctx)
}

// RunInTransaction executes a function within a database transaction
func (a *AuthServicesAdapter) RunInTransaction(ctx context.Context, fn func(qtx *db.Queries) error) error {
	return a.common.RunInTransaction(ctx, fn)
}

// RunInTransactionWithRetry executes a function within a database transaction with retry logic
func (a *AuthServicesAdapter) RunInTransactionWithRetry(ctx context.Context, maxRetries int, fn func(qtx *db.Queries) error) error {
	return a.common.RunInTransactionWithRetry(ctx, maxRetries, fn)
}

// GetLogger returns the logger instance
func (a *AuthServicesAdapter) GetLogger() *zap.Logger {
	return a.common.GetLogger()
}

// GetAPIKeyService returns the API key service interface
func (a *AuthServicesAdapter) GetAPIKeyService() interfaces.APIKeyService {
	return a.common.GetAPIKeyService()
}

// GetTaxService returns the tax service interface
func (a *AuthServicesAdapter) GetTaxService() interfaces.TaxService {
	return a.common.GetTaxService()
}

// GetDiscountService returns the discount service interface
func (a *AuthServicesAdapter) GetDiscountService() interfaces.DiscountService {
	return a.common.GetDiscountService()
}

// GetGasSponsorshipService returns the gas sponsorship service interface
func (a *AuthServicesAdapter) GetGasSponsorshipService() interfaces.GasSponsorshipService {
	return a.common.GetGasSponsorshipService()
}

// GetCurrencyService returns the currency service interface
func (a *AuthServicesAdapter) GetCurrencyService() interfaces.CurrencyService {
	return a.common.GetCurrencyService()
}

// GetExchangeRateService returns the exchange rate service interface
func (a *AuthServicesAdapter) GetExchangeRateService() interfaces.ExchangeRateService {
	return a.common.GetExchangeRateService()
}

// GetCypheraSmartWalletAddress returns the smart wallet address
func (a *AuthServicesAdapter) GetCypheraSmartWalletAddress() string {
	return a.common.GetCypheraSmartWalletAddress()
}
