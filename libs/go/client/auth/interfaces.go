package auth

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CommonServicesInterface defines the interface for common services used by auth middleware
type CommonServicesInterface interface {
	GetDB() db.Querier
	GetAPIKeyService() interfaces.APIKeyService
	BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error)
	WithTx(tx pgx.Tx) *db.Queries
}

// CommonServicesAdapter wraps handlers.CommonServices to implement our interface
type CommonServicesAdapter struct {
	db            db.Querier
	apiKeyService interfaces.APIKeyService
}

// NewCommonServicesAdapter creates a new adapter
func NewCommonServicesAdapter(db db.Querier, apiKeyService interfaces.APIKeyService) *CommonServicesAdapter {
	return &CommonServicesAdapter{
		db:            db,
		apiKeyService: apiKeyService,
	}
}

// GetDB returns the database querier
func (a *CommonServicesAdapter) GetDB() db.Querier {
	return a.db
}

// GetAPIKeyService returns the API key service interface
func (a *CommonServicesAdapter) GetAPIKeyService() interfaces.APIKeyService {
	return a.apiKeyService
}

// BeginTx is not implemented in the adapter - will return nil
func (a *CommonServicesAdapter) BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	return nil, nil, nil
}

// WithTx returns the db queries (no-op for adapter)
func (a *CommonServicesAdapter) WithTx(tx pgx.Tx) *db.Queries {
	// This adapter doesn't support transactions, return nil
	return nil
}

// APIKeyInfo contains information about an API key
type APIKeyInfo struct {
	Key         string
	WorkspaceID uuid.UUID
	AccountID   uuid.UUID
}