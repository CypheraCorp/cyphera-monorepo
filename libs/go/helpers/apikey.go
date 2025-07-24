package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

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

// GenerateAPIKey generates a new secure API key
// Returns the full key (to be shown once to the user) and the key prefix for identification
func GenerateAPIKey() (fullKey string, keyPrefix string, err error) {
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

// HashAPIKey hashes an API key using bcrypt
func HashAPIKey(apiKey string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedBytes), nil
}

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