package handlers

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/client/auth"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/jackc/pgx/v5"
)

// AuthServicesAdapter adapts CommonServices to auth.CommonServicesInterface
type AuthServicesAdapter struct {
	common *CommonServices
}

// NewAuthServicesAdapter creates a new adapter
func NewAuthServicesAdapter(common *CommonServices) auth.CommonServicesInterface {
	return &AuthServicesAdapter{
		common: common,
	}
}

// GetDB returns the database queries
func (a *AuthServicesAdapter) GetDB() *db.Queries {
	return a.common.db
}

// GetAPIKeyService returns the API key service
func (a *AuthServicesAdapter) GetAPIKeyService() *services.APIKeyService {
	return a.common.APIKeyService
}

// BeginTx starts a transaction
func (a *AuthServicesAdapter) BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	return a.common.BeginTx(ctx)
}

// WithTx returns a new db.Queries instance that uses the provided transaction
func (a *AuthServicesAdapter) WithTx(tx pgx.Tx) *db.Queries {
	return a.common.WithTx(tx)
}