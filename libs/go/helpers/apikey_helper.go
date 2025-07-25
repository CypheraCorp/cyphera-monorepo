package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/db"
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

// The GenerateAPIKey and HashAPIKey functions have been moved to the APIKeyService
// to avoid import cycles. Use the service methods instead.

// CompareAPIKeyHash compares a plain text API key with a bcrypt hash
func CompareAPIKeyHash(apiKey, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
}

// ExtractKeyPrefix extracts the prefix from a full API key for display purposes
func ExtractKeyPrefix(apiKey string) string {
	parts := strings.Split(apiKey, "_")
	if len(parts) < 2 {
		return "invalid"
	}

	// Return prefix + first 8 chars of the key
	keyPart := parts[1]
	if len(keyPart) >= 8 {
		return fmt.Sprintf("%s_%s", parts[0], keyPart[:8])
	}
	return fmt.Sprintf("%s_%s", parts[0], keyPart)
}

// APIKeyResponse represents the standardized API response for API key operations
type APIKeyResponse struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Name        string                 `json:"name"`
	AccessLevel string                 `json:"access_level"`
	ExpiresAt   *int64                 `json:"expires_at,omitempty"`
	LastUsedAt  *int64                 `json:"last_used_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
	KeyPrefix   string                 `json:"key_prefix,omitempty"` // Shows first part of key for identification
	Key         string                 `json:"key,omitempty"`        // Only included on creation
}

// ListAPIKeysResponse represents the paginated response for API key list operations
type ListAPIKeysResponse struct {
	Object  string           `json:"object"`
	Data    []APIKeyResponse `json:"data"`
	HasMore bool             `json:"has_more"`
	Total   int64            `json:"total"`
}

// ToAPIKeyResponse converts database model to API response
func ToAPIKeyResponse(a db.ApiKey) APIKeyResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(a.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling API key metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	var expiresAt *int64
	if a.ExpiresAt.Valid {
		unix := a.ExpiresAt.Time.Unix()
		expiresAt = &unix
	}

	var lastUsedAt *int64
	if a.LastUsedAt.Valid {
		unix := a.LastUsedAt.Time.Unix()
		lastUsedAt = &unix
	}

	return APIKeyResponse{
		ID:          a.ID.String(),
		Object:      "api_key",
		Name:        a.Name,
		AccessLevel: string(a.AccessLevel),
		ExpiresAt:   expiresAt,
		LastUsedAt:  lastUsedAt,
		Metadata:    metadata,
		CreatedAt:   a.CreatedAt.Time.Unix(),
		UpdatedAt:   a.UpdatedAt.Time.Unix(),
		KeyPrefix:   a.KeyPrefix.String,
	}
}
