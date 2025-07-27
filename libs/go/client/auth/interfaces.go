package auth

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CommonServicesInterface defines the interface for common services used by auth middleware
type CommonServicesInterface interface {
	GetDB() *db.Queries
	GetAPIKeyService() *services.APIKeyService
	BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error)
	WithTx(tx pgx.Tx) *db.Queries
}

// CommonServicesAdapter wraps handlers.CommonServices to implement our interface
type CommonServicesAdapter struct {
	db            *db.Queries
	apiKeyService *services.APIKeyService
}

// NewCommonServicesAdapter creates a new adapter
func NewCommonServicesAdapter(db *db.Queries) *CommonServicesAdapter {
	return &CommonServicesAdapter{
		db:            db,
		apiKeyService: services.NewAPIKeyService(db),
	}
}

// GetDB returns the database queries
func (a *CommonServicesAdapter) GetDB() *db.Queries {
	return a.db
}

// GetAPIKeyService returns the API key service
func (a *CommonServicesAdapter) GetAPIKeyService() *services.APIKeyService {
	return a.apiKeyService
}

// BeginTx is not implemented in the adapter - will return nil
func (a *CommonServicesAdapter) BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	return nil, nil, nil
}

// WithTx returns the db queries (no-op for adapter)
func (a *CommonServicesAdapter) WithTx(tx pgx.Tx) *db.Queries {
	return a.db
}

// APIKeyInfo contains information about an API key
type APIKeyInfo struct {
	Key         string
	WorkspaceID uuid.UUID
	AccountID   uuid.UUID
}