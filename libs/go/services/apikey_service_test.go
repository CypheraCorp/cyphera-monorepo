package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	logger.InitLogger("test")
}

func TestAPIKeyService_CreateAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	futureTime := time.Now().Add(30 * 24 * time.Hour)

	tests := []struct {
		name           string
		params         params.CreateAPIKeyParams
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(db.ApiKey, string, string)
	}{
		{
			name: "successfully creates API key",
			params: params.CreateAPIKeyParams{
				WorkspaceID: workspaceID,
				Name:        "Test API Key",
				Description: "Test description",
				ExpiresAt:   &futureTime,
				AccessLevel: "read",
				Metadata:    map[string]interface{}{"environment": "test"},
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: workspaceID,
					Name:        "Test API Key",
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			wantErr: false,
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, "Test API Key", apiKey.Name)
				assert.Equal(t, workspaceID, apiKey.WorkspaceID)
				assert.True(t, strings.HasPrefix(fullKey, "cyk_"))
				assert.True(t, strings.HasPrefix(keyPrefix, "cyk_"))
				assert.True(t, len(fullKey) > len(keyPrefix))
			},
		},
		{
			name: "successfully creates API key without expiration",
			params: params.CreateAPIKeyParams{
				WorkspaceID: workspaceID,
				Name:        "Permanent Key",
				Description: "No expiration",
				ExpiresAt:   nil,
				AccessLevel: "write",
				Metadata:    nil,
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: workspaceID,
					Name:        "Permanent Key",
					AccessLevel: "write",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			wantErr: false,
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, "Permanent Key", apiKey.Name)
				assert.True(t, strings.HasPrefix(fullKey, "cyk_"))
			},
		},
		{
			name: "handles database error",
			params: params.CreateAPIKeyParams{
				WorkspaceID: workspaceID,
				Name:        "Test Key",
				AccessLevel: "read",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(db.ApiKey{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "database error",
		},
		{
			name: "handles invalid metadata",
			params: params.CreateAPIKeyParams{
				WorkspaceID: workspaceID,
				Name:        "Test Key",
				AccessLevel: "read",
				Metadata:    map[string]interface{}{"invalid": make(chan int)}, // Invalid JSON
			},
			setupMocks: func() {
				// No database call expected due to JSON marshal failure
			},
			wantErr:     true,
			errorString: "json: unsupported type",
		},
		{
			name: "creates admin access level key",
			params: params.CreateAPIKeyParams{
				WorkspaceID: workspaceID,
				Name:        "Admin Key",
				AccessLevel: "admin",
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: workspaceID,
					Name:        "Admin Key",
					AccessLevel: "admin",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			wantErr: false,
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, "Admin Key", apiKey.Name)
				assert.Equal(t, db.ApiKeyLevel("admin"), apiKey.AccessLevel)
			},
		},
		{
			name: "creates key with complex metadata",
			params: params.CreateAPIKeyParams{
				WorkspaceID: workspaceID,
				Name:        "Complex Key",
				AccessLevel: "read",
				Metadata: map[string]interface{}{
					"environment": "production",
					"version":     "1.0",
					"config": map[string]interface{}{
						"rate_limit": 1000,
						"features":   []string{"api", "webhooks"},
					},
				},
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: workspaceID,
					Name:        "Complex Key",
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			wantErr: false,
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, "Complex Key", apiKey.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			apiKey, fullKey, keyPrefix, err := service.CreateAPIKey(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Empty(t, fullKey)
				assert.Empty(t, keyPrefix)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, fullKey)
				assert.NotEmpty(t, keyPrefix)
				if tt.validateResult != nil {
					tt.validateResult(apiKey, fullKey, keyPrefix)
				}
			}
		})
	}
}

func TestAPIKeyService_GetAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	apiKeyID := uuid.New()
	workspaceID := uuid.New()
	expectedAPIKey := db.ApiKey{
		ID:          apiKeyID,
		WorkspaceID: workspaceID,
		Name:        "Test API Key",
		AccessLevel: "read",
	}

	tests := []struct {
		name        string
		apiKeyID    uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantAPIKey  *db.ApiKey
	}{
		{
			name:        "successfully gets API key",
			apiKeyID:    apiKeyID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAPIKey(ctx, db.GetAPIKeyParams{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
				}).Return(expectedAPIKey, nil)
			},
			wantErr:    false,
			wantAPIKey: &expectedAPIKey,
		},
		{
			name:        "API key not found",
			apiKeyID:    apiKeyID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAPIKey(ctx, db.GetAPIKeyParams{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
				}).Return(db.ApiKey{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "no rows",
		},
		{
			name:        "database error",
			apiKeyID:    apiKeyID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAPIKey(ctx, db.GetAPIKeyParams{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
				}).Return(db.ApiKey{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "database error",
		},
		{
			name:        "wrong workspace ID",
			apiKeyID:    apiKeyID,
			workspaceID: uuid.New(), // Different workspace
			setupMocks: func() {
				mockQuerier.EXPECT().GetAPIKey(ctx, gomock.Any()).Return(db.ApiKey{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			apiKey, err := service.GetAPIKey(ctx, tt.apiKeyID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				if tt.wantAPIKey != nil {
					assert.Equal(t, tt.wantAPIKey.ID, apiKey.ID)
					assert.Equal(t, tt.wantAPIKey.WorkspaceID, apiKey.WorkspaceID)
					assert.Equal(t, tt.wantAPIKey.Name, apiKey.Name)
				}
			}
		})
	}
}

func TestAPIKeyService_ListAPIKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	expectedAPIKeys := []db.ApiKey{
		{ID: uuid.New(), WorkspaceID: workspaceID, Name: "Key 1", AccessLevel: "read"},
		{ID: uuid.New(), WorkspaceID: workspaceID, Name: "Key 2", AccessLevel: "write"},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:        "successfully lists API keys",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListAPIKeys(ctx, workspaceID).Return(expectedAPIKeys, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:        "returns empty list",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListAPIKeys(ctx, workspaceID).Return([]db.ApiKey{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListAPIKeys(ctx, workspaceID).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			apiKeys, err := service.ListAPIKeys(ctx, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, apiKeys)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, apiKeys, tt.wantCount)
			}
		})
	}
}

func TestAPIKeyService_UpdateAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	apiKeyID := uuid.New()
	workspaceID := uuid.New()
	futureTime := time.Now().Add(30 * 24 * time.Hour)

	tests := []struct {
		name        string
		params      params.UpdateAPIKeyParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates API key",
			params: params.UpdateAPIKeyParams{
				WorkspaceID: workspaceID,
				ID:          apiKeyID,
				Name:        aws.String("Updated Key"),
				Description: aws.String("Updated description"),
				ExpiresAt:   &futureTime,
				AccessLevel: "write",
				Metadata:    map[string]interface{}{"updated": true},
			},
			setupMocks: func() {
				updatedAPIKey := db.ApiKey{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
					Name:        "Updated Key",
					AccessLevel: "write",
				}
				mockQuerier.EXPECT().UpdateAPIKey(ctx, gomock.Any()).Return(updatedAPIKey, nil)
			},
			wantErr: false,
		},
		{
			name: "updates API key without expiration",
			params: params.UpdateAPIKeyParams{
				WorkspaceID: workspaceID,
				ID:          apiKeyID,
				Name:        aws.String("No Expiry Key"),
				AccessLevel: "read",
				ExpiresAt:   nil,
			},
			setupMocks: func() {
				updatedAPIKey := db.ApiKey{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
					Name:        "No Expiry Key",
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().UpdateAPIKey(ctx, gomock.Any()).Return(updatedAPIKey, nil)
			},
			wantErr: false,
		},
		{
			name: "handles invalid metadata",
			params: params.UpdateAPIKeyParams{
				WorkspaceID: workspaceID,
				ID:          apiKeyID,
				Name:        aws.String("Invalid Key"),
				AccessLevel: "read",
				Metadata:    map[string]interface{}{"invalid": make(chan int)},
			},
			setupMocks: func() {
				// No database call expected due to JSON marshal failure
			},
			wantErr:     true,
			errorString: "json: unsupported type",
		},
		{
			name: "database update error",
			params: params.UpdateAPIKeyParams{
				WorkspaceID: workspaceID,
				ID:          apiKeyID,
				Name:        aws.String("Test Key"),
				AccessLevel: "read",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateAPIKey(ctx, gomock.Any()).Return(db.ApiKey{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "update error",
		},
		{
			name: "API key not found for update",
			params: params.UpdateAPIKeyParams{
				WorkspaceID: workspaceID,
				ID:          apiKeyID,
				Name:        aws.String("Non-existent Key"),
				AccessLevel: "read",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateAPIKey(ctx, gomock.Any()).Return(db.ApiKey{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			apiKey, err := service.UpdateAPIKey(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, apiKey.ID)
			}
		})
	}
}

func TestAPIKeyService_DeleteAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	apiKeyID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name        string
		apiKeyID    uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully deletes API key",
			apiKeyID:    apiKeyID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().DeleteAPIKey(ctx, db.DeleteAPIKeyParams{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
				}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "API key not found for deletion",
			apiKeyID:    apiKeyID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().DeleteAPIKey(ctx, db.DeleteAPIKeyParams{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
				}).Return(pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "no rows",
		},
		{
			name:        "database delete error",
			apiKeyID:    apiKeyID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().DeleteAPIKey(ctx, db.DeleteAPIKeyParams{
					ID:          apiKeyID,
					WorkspaceID: workspaceID,
				}).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "delete error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteAPIKey(ctx, tt.apiKeyID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIKeyService_GetAllAPIKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	expectedAPIKeys := []db.ApiKey{
		{ID: uuid.New(), WorkspaceID: uuid.New(), Name: "Key 1", AccessLevel: "read"},
		{ID: uuid.New(), WorkspaceID: uuid.New(), Name: "Key 2", AccessLevel: "admin"},
	}

	tests := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name: "successfully gets all API keys",
			setupMocks: func() {
				mockQuerier.EXPECT().GetAllAPIKeys(ctx).Return(expectedAPIKeys, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "returns empty list",
			setupMocks: func() {
				mockQuerier.EXPECT().GetAllAPIKeys(ctx).Return([]db.ApiKey{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "database error",
			setupMocks: func() {
				mockQuerier.EXPECT().GetAllAPIKeys(ctx).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			apiKeys, err := service.GetAllAPIKeys(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, apiKeys)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, apiKeys, tt.wantCount)
			}
		})
	}
}

func TestAPIKeyService_CompareAPIKeyHash(t *testing.T) {
	service := services.NewAPIKeyService(nil) // No database needed for this test

	// Create a valid hash for testing
	testKey := "cyk_testkey12345"
	validHash, err := bcrypt.GenerateFromPassword([]byte(testKey), 10)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		apiKey  string
		hash    string
		wantErr bool
	}{
		{
			name:    "valid API key and hash match",
			apiKey:  testKey,
			hash:    string(validHash),
			wantErr: false,
		},
		{
			name:    "invalid API key does not match",
			apiKey:  "cyk_wrongkey",
			hash:    string(validHash),
			wantErr: true,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			hash:    string(validHash),
			wantErr: true,
		},
		{
			name:    "empty hash",
			apiKey:  testKey,
			hash:    "",
			wantErr: true,
		},
		{
			name:    "invalid hash format",
			apiKey:  testKey,
			hash:    "invalid-hash",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CompareAPIKeyHash(tt.apiKey, tt.hash)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIKeyService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name      string
		operation func() error
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "nil context handling in GetAPIKey",
			operation: func() error {
				// Even with nil context, the mock expectation is needed
				mockQuerier.EXPECT().GetAPIKey(gomock.Any(), gomock.Any()).Return(db.ApiKey{}, errors.New("context error"))
				_, err := service.GetAPIKey(nil, uuid.New(), uuid.New())
				return err
			},
			wantErr: true,
		},
		{
			name: "empty UUID in GetAPIKey",
			operation: func() error {
				mockQuerier.EXPECT().GetAPIKey(gomock.Any(), gomock.Any()).Return(db.ApiKey{}, pgx.ErrNoRows)
				_, err := service.GetAPIKey(ctx, uuid.Nil, uuid.New())
				return err
			},
			wantErr: true,
		},
		{
			name: "very long API key name",
			operation: func() error {
				longName := strings.Repeat("a", 1000)
				params := params.CreateAPIKeyParams{
					WorkspaceID: uuid.New(),
					Name:        longName,
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(db.ApiKey{}, nil)
				_, _, _, err := service.CreateAPIKey(ctx, params)
				return err
			},
			wantErr: false, // Should handle long names gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIKeyService_KeyGeneration(t *testing.T) {
	t.Run("generated keys have correct format", func(t *testing.T) {
		// Use reflection to access private method for testing
		// In a real scenario, you might want to expose this as a public method for testing
		// or test it indirectly through CreateAPIKey
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewAPIKeyService(mockQuerier)

		params := params.CreateAPIKeyParams{
			WorkspaceID: uuid.New(),
			Name:        "Test Key",
			AccessLevel: "read",
		}

		expectedAPIKey := db.ApiKey{
			ID:          uuid.New(),
			WorkspaceID: params.WorkspaceID,
			Name:        params.Name,
			AccessLevel: "read",
		}

		mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)

		_, fullKey, keyPrefix, err := service.CreateAPIKey(ctx, params)

		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(fullKey, "cyk_"))
		assert.True(t, strings.HasPrefix(keyPrefix, "cyk_"))
		assert.True(t, len(fullKey) > len(keyPrefix))
		assert.True(t, len(keyPrefix) >= 12) // cyk_ + at least 8 chars
	})
}

func TestAPIKeyService_BoundaryConditions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name           string
		params         params.CreateAPIKeyParams
		setupMocks     func()
		validateResult func(db.ApiKey, string, string)
	}{
		{
			name: "empty name API key",
			params: params.CreateAPIKeyParams{
				WorkspaceID: uuid.New(),
				Name:        "",
				AccessLevel: "read",
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: uuid.New(),
					Name:        "",
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, "", apiKey.Name)
				assert.True(t, strings.HasPrefix(fullKey, "cyk_"))
			},
		},
		{
			name: "past expiration date",
			params: params.CreateAPIKeyParams{
				WorkspaceID: uuid.New(),
				Name:        "Past Expiry Key",
				AccessLevel: "read",
				ExpiresAt:   &[]time.Time{time.Now().Add(-24 * time.Hour)}[0],
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: uuid.New(),
					Name:        "Past Expiry Key",
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, "Past Expiry Key", apiKey.Name)
			},
		},
		{
			name: "nil workspace ID",
			params: params.CreateAPIKeyParams{
				WorkspaceID: uuid.Nil,
				Name:        "Nil Workspace Key",
				AccessLevel: "read",
			},
			setupMocks: func() {
				expectedAPIKey := db.ApiKey{
					ID:          uuid.New(),
					WorkspaceID: uuid.Nil,
					Name:        "Nil Workspace Key",
					AccessLevel: "read",
				}
				mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
			},
			validateResult: func(apiKey db.ApiKey, fullKey string, keyPrefix string) {
				assert.Equal(t, uuid.Nil, apiKey.WorkspaceID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			apiKey, fullKey, keyPrefix, err := service.CreateAPIKey(ctx, tt.params)

			assert.NoError(t, err)
			if tt.validateResult != nil {
				tt.validateResult(apiKey, fullKey, keyPrefix)
			}
		})
	}
}

func TestAPIKeyService_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAPIKeyService(mockQuerier)
	ctx := context.Background()

	// Test concurrent API key creation
	t.Run("concurrent API key creation", func(t *testing.T) {
		const numGoroutines = 10
		resultsChan := make(chan struct {
			apiKey    db.ApiKey
			fullKey   string
			keyPrefix string
			err       error
		}, numGoroutines)

		workspaceID := uuid.New()

		// Setup expectations for all concurrent calls
		for i := 0; i < numGoroutines; i++ {
			expectedAPIKey := db.ApiKey{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Name:        "Concurrent Key",
				AccessLevel: "read",
			}
			mockQuerier.EXPECT().CreateAPIKey(ctx, gomock.Any()).Return(expectedAPIKey, nil)
		}

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				params := params.CreateAPIKeyParams{
					WorkspaceID: workspaceID,
					Name:        "Concurrent Key",
					AccessLevel: "read",
				}

				apiKey, fullKey, keyPrefix, err := service.CreateAPIKey(ctx, params)
				resultsChan <- struct {
					apiKey    db.ApiKey
					fullKey   string
					keyPrefix string
					err       error
				}{apiKey, fullKey, keyPrefix, err}
			}(i)
		}

		// Collect results
		var results []struct {
			apiKey    db.ApiKey
			fullKey   string
			keyPrefix string
			err       error
		}

		for i := 0; i < numGoroutines; i++ {
			result := <-resultsChan
			results = append(results, result)
		}

		// Verify all operations completed without errors
		assert.Len(t, results, numGoroutines)

		keySet := make(map[string]bool)
		for _, result := range results {
			assert.NoError(t, result.err)
			assert.NotEmpty(t, result.fullKey)
			assert.NotEmpty(t, result.keyPrefix)

			// Verify all keys are unique
			assert.False(t, keySet[result.fullKey], "Duplicate key generated: %s", result.fullKey)
			keySet[result.fullKey] = true
		}
	})
}
