package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const (
	// APIKeyLength is the length of the random part of the API key (in bytes before base64 encoding)
	APIKeyLength = 32
	// APIKeyPrefix is the prefix for all API keys
	APIKeyPrefix = "cyk"
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 10
)

// APIKeyService handles business logic for API key operations
type APIKeyService struct {
	db db.Querier
}

// NewAPIKeyService creates a new instance of APIKeyService
func NewAPIKeyService(database db.Querier) *APIKeyService {
	return &APIKeyService{
		db: database,
	}
}

// generateAPIKey generates a new secure API key
// Returns the full key (to be shown once to the user) and the key prefix for identification
func (s *APIKeyService) generateAPIKey() (fullKey string, keyPrefix string, err error) {
	// Generate random bytes
	randomBytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64 URL-safe
	encodedKey := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Create the full key with prefix
	fullKey = fmt.Sprintf("%s_%s", APIKeyPrefix, encodedKey)

	// Extract prefix for storage (first 8 chars after the standard prefix)
	if len(encodedKey) >= 8 {
		keyPrefix = fmt.Sprintf("%s_%s", APIKeyPrefix, encodedKey[:8])
	} else {
		keyPrefix = fullKey // fallback, shouldn't happen with 32 bytes
	}

	return fullKey, keyPrefix, nil
}

// hashAPIKey hashes an API key using bcrypt
func (s *APIKeyService) hashAPIKey(apiKey string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedBytes), nil
}

// CompareAPIKeyHash compares a plain text API key with a bcrypt hash
func (s *APIKeyService) CompareAPIKeyHash(apiKey, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
}

// CreateAPIKeyParams represents the parameters for creating an API key
type CreateAPIKeyParams struct {
	WorkspaceID uuid.UUID
	Name        string
	Description string
	ExpiresAt   *time.Time
	AccessLevel string
	Metadata    map[string]interface{}
}

// UpdateAPIKeyParams represents the parameters for updating an API key
type UpdateAPIKeyParams struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	ExpiresAt   *time.Time
	AccessLevel string
	Metadata    map[string]interface{}
}

// GetAPIKey retrieves an API key by ID and workspace
func (s *APIKeyService) GetAPIKey(ctx context.Context, id, workspaceID uuid.UUID) (db.ApiKey, error) {
	return s.db.GetAPIKey(ctx, db.GetAPIKeyParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// ListAPIKeys retrieves all API keys for a workspace
func (s *APIKeyService) ListAPIKeys(ctx context.Context, workspaceID uuid.UUID) ([]db.ApiKey, error) {
	return s.db.ListAPIKeys(ctx, workspaceID)
}

// GetAllAPIKeys retrieves all API keys (admin function)
func (s *APIKeyService) GetAllAPIKeys(ctx context.Context) ([]db.ApiKey, error) {
	return s.db.GetAllAPIKeys(ctx)
}

// CreateAPIKey creates a new API key with proper key generation and hashing
func (s *APIKeyService) CreateAPIKey(ctx context.Context, params CreateAPIKeyParams) (db.ApiKey, string, string, error) {
	// Generate the API key
	fullKey, keyPrefix, err := s.generateAPIKey()
	if err != nil {
		return db.ApiKey{}, "", "", err
	}

	// Hash the key for storage
	hashedKey, err := s.hashAPIKey(fullKey)
	if err != nil {
		return db.ApiKey{}, "", "", err
	}

	// Marshal metadata
	metadata, err := json.Marshal(params.Metadata)
	if err != nil {
		return db.ApiKey{}, "", "", err
	}

	// Prepare expires_at
	var expiresAt pgtype.Timestamptz
	if params.ExpiresAt != nil {
		expiresAt.Time = *params.ExpiresAt
		expiresAt.Valid = true
	}

	// Prepare key prefix for storage
	keyPrefixPgText := pgtype.Text{
		String: keyPrefix,
		Valid:  true,
	}

	// Create the API key in database
	apiKey, err := s.db.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		WorkspaceID: params.WorkspaceID,
		Name:        params.Name,
		KeyHash:     hashedKey,
		KeyPrefix:   keyPrefixPgText,
		AccessLevel: db.ApiKeyLevel(params.AccessLevel),
		ExpiresAt:   expiresAt,
		Metadata:    metadata,
	})
	if err != nil {
		return db.ApiKey{}, "", "", err
	}

	return apiKey, fullKey, keyPrefix, nil
}

// UpdateAPIKey updates an existing API key
func (s *APIKeyService) UpdateAPIKey(ctx context.Context, params UpdateAPIKeyParams) (db.ApiKey, error) {
	// Marshal metadata
	metadata, err := json.Marshal(params.Metadata)
	if err != nil {
		return db.ApiKey{}, err
	}

	// Prepare expires_at
	var expiresAt pgtype.Timestamptz
	if params.ExpiresAt != nil {
		expiresAt.Time = *params.ExpiresAt
		expiresAt.Valid = true
	}

	return s.db.UpdateAPIKey(ctx, db.UpdateAPIKeyParams{
		WorkspaceID: params.WorkspaceID,
		ID:          params.ID,
		Name:        params.Name,
		AccessLevel: db.ApiKeyLevel(params.AccessLevel),
		ExpiresAt:   expiresAt,
		Metadata:    metadata,
	})
}

// DeleteAPIKey soft deletes an API key
func (s *APIKeyService) DeleteAPIKey(ctx context.Context, id, workspaceID uuid.UUID) error {
	return s.db.DeleteAPIKey(ctx, db.DeleteAPIKeyParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}