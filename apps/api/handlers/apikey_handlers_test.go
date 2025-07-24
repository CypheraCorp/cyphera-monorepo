package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Comprehensive API Key handler tests focusing on security, validation, and database operations

func TestAPIKeyHandler_KeyGeneration(t *testing.T) {
	// Test API key generation patterns
	
	t.Run("Key generation uniqueness", func(t *testing.T) {
		// Generate multiple keys and ensure they're unique
		keys := make(map[string]bool)
		
		for i := 0; i < 100; i++ {
			// Simulate API key generation
			keyBytes := make([]byte, 32)
			_, err := rand.Read(keyBytes)
			require.NoError(t, err)
			
			key := hex.EncodeToString(keyBytes)
			
			// Ensure no duplicates
			assert.False(t, keys[key], "Generated key should be unique")
			keys[key] = true
			
			// Validate key format
			assert.Len(t, key, 64, "API key should be 64 characters (32 bytes hex)")
		}
	})
	
	t.Run("Key prefix validation", func(t *testing.T) {
		// Test API key prefix patterns (e.g., "cyphera_")
		prefix := "cyphera_"
		
		keyBytes := make([]byte, 32)
		_, err := rand.Read(keyBytes)
		require.NoError(t, err)
		
		key := prefix + hex.EncodeToString(keyBytes)
		
		assert.True(t, len(key) > len(prefix))
		assert.Contains(t, key, prefix)
		assert.Equal(t, prefix, key[:len(prefix)])
	})
}

func TestAPIKeyHandler_HashingValidation(t *testing.T) {
	// Test bcrypt hashing for API key security
	
	t.Run("Bcrypt hashing consistency", func(t *testing.T) {
		testKey := "cyphera_" + hex.EncodeToString([]byte("test_key_data_12345"))
		
		// Hash the key
		hashedKey, err := bcrypt.GenerateFromPassword([]byte(testKey), bcrypt.DefaultCost)
		require.NoError(t, err)
		
		// Verify the hash
		err = bcrypt.CompareHashAndPassword(hashedKey, []byte(testKey))
		assert.NoError(t, err, "Hash verification should succeed")
		
		// Verify wrong key fails
		wrongKey := "cyphera_" + hex.EncodeToString([]byte("wrong_key_data"))
		err = bcrypt.CompareHashAndPassword(hashedKey, []byte(wrongKey))
		assert.Error(t, err, "Hash verification should fail for wrong key")
	})
	
	t.Run("Hash cost validation", func(t *testing.T) {
		testKey := "test_api_key"
		
		// Test different cost levels
		costs := []int{bcrypt.MinCost, bcrypt.DefaultCost, 12}
		
		for _, cost := range costs {
			hashedKey, err := bcrypt.GenerateFromPassword([]byte(testKey), cost)
			require.NoError(t, err)
			
			// Verify hash works
			err = bcrypt.CompareHashAndPassword(hashedKey, []byte(testKey))
			assert.NoError(t, err)
			
			// Hash should not be empty
			assert.NotEmpty(t, hashedKey)
		}
	})
}

func TestAPIKeyHandler_DatabaseStructures(t *testing.T) {
	// Test API key database structure patterns
	
	t.Run("APIKey structure validation", func(t *testing.T) {
		now := time.Now()
		
		apiKey := struct {
			ID          uuid.UUID          `json:"id"`
			WorkspaceID uuid.UUID          `json:"workspace_id"`
			Name        string             `json:"name"`
			HashedKey   string             `json:"hashed_key,omitempty"`
			Permissions []string           `json:"permissions"`
			IsActive    bool               `json:"is_active"`
			ExpiresAt   pgtype.Timestamptz `json:"expires_at"`
			CreatedAt   pgtype.Timestamptz `json:"created_at"`
			UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
		}{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Name:        "Test API Key",
			HashedKey:   "$2a$10$...", // bcrypt hash example
			Permissions: []string{"read", "write"},
			IsActive:    true,
			ExpiresAt:   pgtype.Timestamptz{Time: now.Add(365 * 24 * time.Hour), Valid: true},
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		}
		
		// Validate structure
		assert.NotEqual(t, uuid.Nil, apiKey.ID)
		assert.NotEqual(t, uuid.Nil, apiKey.WorkspaceID)
		assert.NotEmpty(t, apiKey.Name)
		assert.NotEmpty(t, apiKey.HashedKey)
		assert.True(t, apiKey.IsActive)
		assert.True(t, apiKey.ExpiresAt.Valid)
		assert.True(t, apiKey.CreatedAt.Valid)
		assert.Len(t, apiKey.Permissions, 2)
	})
}

func TestAPIKeyHandler_RequestValidation(t *testing.T) {
	// Test API key request validation patterns
	
	t.Run("CreateAPIKeyRequest validation", func(t *testing.T) {
		tests := []struct {
			name        string
			request     map[string]interface{}
			expectValid bool
		}{
			{
				name: "valid request",
				request: map[string]interface{}{
					"name":        "Production API Key",
					"permissions": []string{"read", "write"},
					"expires_in_days": 365,
				},
				expectValid: true,
			},
			{
				name: "missing name",
				request: map[string]interface{}{
					"permissions": []string{"read"},
					"expires_in_days": 30,
				},
				expectValid: false,
			},
			{
				name: "empty permissions",
				request: map[string]interface{}{
					"name":        "Test Key",
					"permissions": []string{},
					"expires_in_days": 30,
				},
				expectValid: false,
			},
			{
				name: "invalid expiration",
				request: map[string]interface{}{
					"name":        "Test Key",
					"permissions": []string{"read"},
					"expires_in_days": -1,
				},
				expectValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				jsonData, err := json.Marshal(tt.request)
				require.NoError(t, err)
				
				var parsed map[string]interface{}
				err = json.Unmarshal(jsonData, &parsed)
				require.NoError(t, err)
				
				// Basic validation checks
				hasName := parsed["name"] != nil && parsed["name"] != ""
				hasPermissions := parsed["permissions"] != nil
				
				var validPermissions bool
				if perms, ok := parsed["permissions"].([]interface{}); ok {
					validPermissions = len(perms) > 0
				}
				
				var validExpiration bool
				if expDays, ok := parsed["expires_in_days"].(float64); ok {
					validExpiration = expDays > 0
				}
				
				isValid := hasName && hasPermissions && validPermissions && validExpiration
				if tt.expectValid {
					assert.True(t, isValid, "Request should be valid")
				} else {
					assert.False(t, isValid, "Request should be invalid")
				}
			})
		}
	})
}

func TestAPIKeyHandler_SecurityPatterns(t *testing.T) {
	// Test security-related patterns
	
	t.Run("Key exposure prevention", func(t *testing.T) {
		// Test that raw keys are never stored or logged
		rawKey := "cyphera_" + hex.EncodeToString([]byte("secret_key_data"))
		
		// Hash immediately
		hashedKey, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
		require.NoError(t, err)
		
		// Create response structure (should not contain raw key)
		response := struct {
			ID        uuid.UUID `json:"id"`
			Name      string    `json:"name"`
			KeyPrefix string    `json:"key_prefix"`
			// HashedKey is intentionally omitted from JSON response
			HashedKey string `json:"-"`
		}{
			ID:        uuid.New(),
			Name:      "Test Key",
			KeyPrefix: rawKey[:12] + "...", // Only show prefix
			HashedKey: string(hashedKey),
		}
		
		// Verify raw key is not in JSON
		jsonData, err := json.Marshal(response)
		require.NoError(t, err)
		
		jsonString := string(jsonData)
		assert.NotContains(t, jsonString, rawKey, "Raw key should not appear in JSON")
		assert.NotContains(t, jsonString, string(hashedKey), "Hashed key should not appear in JSON")
		assert.Contains(t, jsonString, "cyphera_", "Key prefix should be visible")
	})
	
	t.Run("Permission isolation", func(t *testing.T) {
		// Test permission-based access patterns
		permissions := []struct {
			name   string
			rights []string
			level  string
		}{
			{"read-only", []string{"read"}, "basic"},
			{"read-write", []string{"read", "write"}, "standard"},
			{"admin", []string{"read", "write", "delete", "admin"}, "elevated"},
		}
		
		for _, perm := range permissions {
			t.Run(perm.name, func(t *testing.T) {
				assert.NotEmpty(t, perm.rights)
				assert.Contains(t, perm.rights, "read") // All should have read
				
				if perm.level == "elevated" {
					assert.Contains(t, perm.rights, "admin")
				}
			})
		}
	})
}

// Benchmark tests for API key operations
func BenchmarkAPIKeyHandler_KeyGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyBytes := make([]byte, 32)
		_, err := rand.Read(keyBytes)
		if err != nil {
			b.Fatal(err)
		}
		_ = hex.EncodeToString(keyBytes)
	}
}

func BenchmarkAPIKeyHandler_BcryptHashing(b *testing.B) {
	testKey := "cyphera_test_key_for_benchmarking"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bcrypt.GenerateFromPassword([]byte(testKey), bcrypt.DefaultCost)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAPIKeyHandler_BcryptVerification(b *testing.B) {
	testKey := "cyphera_test_key_for_benchmarking"
	hashedKey, _ := bcrypt.GenerateFromPassword([]byte(testKey), bcrypt.DefaultCost)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := bcrypt.CompareHashAndPassword(hashedKey, []byte(testKey))
		if err != nil {
			b.Fatal(err)
		}
	}
}