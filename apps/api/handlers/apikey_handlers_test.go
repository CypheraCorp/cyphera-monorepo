package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/apps/api/handlers"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func init() {
	// Set Gin to test mode to reduce noise
	gin.SetMode(gin.TestMode)

	// Initialize logger for handlers to avoid nil pointer panics
	logger.InitLogger("test")
}

func setupAPIKeyHandler(t *testing.T) (*handlers.APIKeyHandler, *mocks.MockAPIKeyService, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockAPIKeyService := mocks.NewMockAPIKeyService(ctrl)

	logger := zap.NewNop()

	// Create CommonServices with the mock API key service
	commonServices := handlers.NewCommonServices(handlers.CommonServicesConfig{
		APIKeyService: mockAPIKeyService,
		Logger:        logger,
	})

	handler := handlers.NewAPIKeyHandler(commonServices, logger)
	return handler, mockAPIKeyService, ctrl
}

func createTestContext(method, url string, body interface{}, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	c.Request = req
	return c, w
}

func TestNewAPIKeyHandler(t *testing.T) {
	tests := []struct {
		name      string
		common    *handlers.CommonServices
		logger    *zap.Logger
		expectNil bool
	}{
		{
			name: "creates handler with valid parameters",
			common: handlers.NewCommonServices(handlers.CommonServicesConfig{
				APIKeyService: &mocks.MockAPIKeyService{},
				Logger:        zap.NewNop(),
			}),
			logger:    zap.NewNop(),
			expectNil: false,
		},
		{
			name:      "creates handler with nil common services",
			common:    nil,
			logger:    zap.NewNop(),
			expectNil: false,
		},
		{
			name: "creates handler with nil logger",
			common: handlers.NewCommonServices(handlers.CommonServicesConfig{
				APIKeyService: &mocks.MockAPIKeyService{},
				Logger:        zap.NewNop(),
			}),
			logger:    nil,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewAPIKeyHandler(tt.common, tt.logger)
			if tt.expectNil {
				assert.Nil(t, handler)
			} else {
				assert.NotNil(t, handler)
				assert.IsType(t, &handlers.APIKeyHandler{}, handler)
			}
		})
	}
}

func TestAPIKeyHandler_GetAPIKeyByID(t *testing.T) {
	workspaceID := uuid.New()
	apiKeyID := uuid.New()

	validAPIKey := db.ApiKey{
		ID:          apiKeyID,
		WorkspaceID: workspaceID,
		Name:        "Test API Key",
		AccessLevel: db.ApiKeyLevelRead,
		Metadata:    []byte("{}"),
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	tests := []struct {
		name             string
		apiKeyID         string
		workspaceID      string
		setupMocks       func(*mocks.MockAPIKeyService)
		expectedStatus   int
		expectedError    string
		validateResponse func(*testing.T, []byte)
	}{
		{
			name:        "successfully gets API key",
			apiKeyID:    apiKeyID.String(),
			workspaceID: workspaceID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					GetAPIKey(gomock.Any(), apiKeyID, workspaceID).
					Return(validAPIKey, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response responses.APIKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, validAPIKey.ID.String(), response.ID)
				assert.Equal(t, validAPIKey.Name, response.Name)
				assert.Equal(t, string(validAPIKey.AccessLevel), response.AccessLevel)
			},
		},
		{
			name:           "fails with invalid workspace ID",
			apiKeyID:       apiKeyID.String(),
			workspaceID:    "invalid-uuid",
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "fails with invalid API key ID",
			apiKeyID:       "invalid-uuid",
			workspaceID:    workspaceID.String(),
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid UUID format",
		},
		{
			name:        "fails when API key not found",
			apiKeyID:    apiKeyID.String(),
			workspaceID: workspaceID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					GetAPIKey(gomock.Any(), apiKeyID, workspaceID).
					Return(db.ApiKey{}, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "API key not found",
		},
		{
			name:        "fails with service error",
			apiKeyID:    apiKeyID.String(),
			workspaceID: workspaceID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					GetAPIKey(gomock.Any(), apiKeyID, workspaceID).
					Return(db.ApiKey{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "fails with missing workspace ID header",
			apiKeyID:       apiKeyID.String(),
			workspaceID:    "",
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService, ctrl := setupAPIKeyHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(mockService)

			headers := map[string]string{}
			if tt.workspaceID != "" {
				headers["X-Workspace-ID"] = tt.workspaceID
			}

			c, w := createTestContext(http.MethodGet, fmt.Sprintf("/api-keys/%s", tt.apiKeyID), nil, headers)
			c.Params = gin.Params{
				gin.Param{Key: "api_key_id", Value: tt.apiKeyID},
			}

			handler.GetAPIKeyByID(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse handlers.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIKeyHandler_ListAPIKeys(t *testing.T) {
	workspaceID := uuid.New()

	apiKeys := []db.ApiKey{
		{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			Name:        "API Key 1",
			AccessLevel: db.ApiKeyLevelRead,
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
		{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			Name:        "API Key 2",
			AccessLevel: db.ApiKeyLevelWrite,
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}

	tests := []struct {
		name             string
		workspaceID      string
		setupMocks       func(*mocks.MockAPIKeyService)
		expectedStatus   int
		expectedError    string
		validateResponse func(*testing.T, []byte)
	}{
		{
			name:        "successfully lists API keys",
			workspaceID: workspaceID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					ListAPIKeys(gomock.Any(), workspaceID).
					Return(apiKeys, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.ListAPIKeysResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "list", response.Object)
				assert.Equal(t, int64(2), response.Total)
				assert.False(t, response.HasMore)
				assert.Len(t, response.Data, 2)
				assert.Equal(t, apiKeys[0].ID.String(), response.Data[0].ID)
				assert.Equal(t, apiKeys[1].ID.String(), response.Data[1].ID)
			},
		},
		{
			name:        "successfully lists empty API keys",
			workspaceID: workspaceID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					ListAPIKeys(gomock.Any(), workspaceID).
					Return([]db.ApiKey{}, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.ListAPIKeysResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "list", response.Object)
				assert.Equal(t, int64(0), response.Total)
				assert.False(t, response.HasMore)
				assert.Len(t, response.Data, 0)
			},
		},
		{
			name:           "fails with invalid workspace ID",
			workspaceID:    "invalid-uuid",
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:        "fails with service error",
			workspaceID: workspaceID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					ListAPIKeys(gomock.Any(), workspaceID).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to retrieve API keys",
		},
		{
			name:           "fails with missing workspace ID header",
			workspaceID:    "",
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService, ctrl := setupAPIKeyHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(mockService)

			headers := map[string]string{}
			if tt.workspaceID != "" {
				headers["X-Workspace-ID"] = tt.workspaceID
			}

			c, w := createTestContext(http.MethodGet, "/api-keys", nil, headers)

			handler.ListAPIKeys(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse handlers.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIKeyHandler_GetAllAPIKeys(t *testing.T) {
	apiKeys := []db.ApiKey{
		{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Name:        "Global API Key 1",
			AccessLevel: db.ApiKeyLevelAdmin,
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
		{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Name:        "Global API Key 2",
			AccessLevel: db.ApiKeyLevelRead,
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}

	tests := []struct {
		name             string
		setupMocks       func(*mocks.MockAPIKeyService)
		expectedStatus   int
		expectedError    string
		validateResponse func(*testing.T, []byte)
	}{
		{
			name: "successfully gets all API keys",
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					GetAllAPIKeys(gomock.Any()).
					Return(apiKeys, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.ListAPIKeysResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "list", response.Object)
				assert.Equal(t, int64(2), response.Total)
				assert.False(t, response.HasMore)
				assert.Len(t, response.Data, 2)
			},
		},
		{
			name: "successfully gets empty list",
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					GetAllAPIKeys(gomock.Any()).
					Return([]db.ApiKey{}, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.ListAPIKeysResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "list", response.Object)
				assert.Equal(t, int64(0), response.Total)
				assert.Len(t, response.Data, 0)
			},
		},
		{
			name: "fails with service error",
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					GetAllAPIKeys(gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to retrieve API keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService, ctrl := setupAPIKeyHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(mockService)

			c, w := createTestContext(http.MethodGet, "/api-keys/all", nil, nil)

			handler.GetAllAPIKeys(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var ginError gin.H
				err := json.Unmarshal(w.Body.Bytes(), &ginError)
				require.NoError(t, err)
				assert.Contains(t, ginError["error"], tt.expectedError)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIKeyHandler_CreateAPIKey(t *testing.T) {
	workspaceID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	validRequest := handlers.CreateAPIKeyRequest{
		Name:        "Test API Key",
		Description: "Test Description",
		ExpiresAt:   &expiresAt,
		AccessLevel: "read",
		Metadata:    map[string]interface{}{"key": "value"},
	}

	createdAPIKey := db.ApiKey{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        validRequest.Name,
		AccessLevel: db.ApiKeyLevel(validRequest.AccessLevel),
		Metadata:    []byte("{}"),
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	tests := []struct {
		name             string
		workspaceID      string
		requestBody      interface{}
		setupMocks       func(*mocks.MockAPIKeyService)
		expectedStatus   int
		expectedError    string
		validateResponse func(*testing.T, []byte)
	}{
		{
			name:        "successfully creates API key",
			workspaceID: workspaceID.String(),
			requestBody: validRequest,
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					CreateAPIKey(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, params params.CreateAPIKeyParams) (db.ApiKey, string, string, error) {
						// Validate the important fields
						assert.Equal(t, workspaceID, params.WorkspaceID)
						assert.Equal(t, validRequest.Name, params.Name)
						assert.Equal(t, validRequest.Description, params.Description)
						assert.Equal(t, validRequest.AccessLevel, params.AccessLevel)
						// Check that the time is approximately correct (within 1 second)
						if validRequest.ExpiresAt != nil && params.ExpiresAt != nil {
							assert.WithinDuration(t, *validRequest.ExpiresAt, *params.ExpiresAt, time.Second)
						}
						return createdAPIKey, "cy_test_1234567890abcdef", "cy_test", nil
					})
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.APIKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, createdAPIKey.ID.String(), response.ID)
				assert.Equal(t, "cy_test_1234567890abcdef", response.Key)
				assert.Equal(t, "cy_test", response.KeyPrefix)
				assert.Equal(t, validRequest.Name, response.Name)
			},
		},
		{
			name:        "successfully creates API key with minimal data",
			workspaceID: workspaceID.String(),
			requestBody: handlers.CreateAPIKeyRequest{
				Name:        "Minimal Key",
				AccessLevel: "write",
			},
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					CreateAPIKey(gomock.Any(), gomock.Any()).
					Return(createdAPIKey, "cy_test_1234567890abcdef", "cy_test", nil)
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.APIKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Key)
				assert.NotEmpty(t, response.KeyPrefix)
			},
		},
		{
			name:           "fails with invalid workspace ID",
			workspaceID:    "invalid-uuid",
			requestBody:    validRequest,
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:        "fails with invalid request body",
			workspaceID: workspaceID.String(),
			requestBody: map[string]interface{}{
				"invalid": "request",
			},
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:        "fails with missing required fields",
			workspaceID: workspaceID.String(),
			requestBody: handlers.CreateAPIKeyRequest{
				Description: "Missing required fields",
			},
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:        "fails with invalid access level",
			workspaceID: workspaceID.String(),
			requestBody: handlers.CreateAPIKeyRequest{
				Name:        "Test Key",
				AccessLevel: "invalid",
			},
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:        "fails with service error",
			workspaceID: workspaceID.String(),
			requestBody: validRequest,
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					CreateAPIKey(gomock.Any(), gomock.Any()).
					Return(db.ApiKey{}, "", "", errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to create API key",
		},
		{
			name:           "fails with missing workspace ID header",
			workspaceID:    "",
			requestBody:    validRequest,
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService, ctrl := setupAPIKeyHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(mockService)

			headers := map[string]string{}
			if tt.workspaceID != "" {
				headers["X-Workspace-ID"] = tt.workspaceID
			}

			c, w := createTestContext(http.MethodPost, "/api-keys", tt.requestBody, headers)

			handler.CreateAPIKey(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse handlers.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIKeyHandler_UpdateAPIKey(t *testing.T) {
	workspaceID := uuid.New()
	apiKeyID := uuid.New()
	expiresAt := time.Now().Add(48 * time.Hour)

	validRequest := handlers.UpdateAPIKeyRequest{
		Name:        "Updated API Key",
		Description: "Updated Description",
		ExpiresAt:   &expiresAt,
		AccessLevel: "write",
		Metadata:    map[string]interface{}{"updated": "value"},
	}

	updatedAPIKey := db.ApiKey{
		ID:          apiKeyID,
		WorkspaceID: workspaceID,
		Name:        validRequest.Name,
		AccessLevel: db.ApiKeyLevel(validRequest.AccessLevel),
		Metadata:    []byte("{}"),
		CreatedAt:   pgtype.Timestamptz{Time: time.Now().Add(-24 * time.Hour), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	tests := []struct {
		name             string
		workspaceID      string
		apiKeyID         string
		requestBody      interface{}
		setupMocks       func(*mocks.MockAPIKeyService)
		expectedStatus   int
		expectedError    string
		validateResponse func(*testing.T, []byte)
	}{
		{
			name:        "successfully updates API key",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			requestBody: validRequest,
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					UpdateAPIKey(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, params params.UpdateAPIKeyParams) (db.ApiKey, error) {
						// Validate the important fields
						assert.Equal(t, workspaceID, params.WorkspaceID)
						assert.Equal(t, apiKeyID, params.ID)
						assert.Equal(t, &validRequest.Name, params.Name)
						assert.Equal(t, &validRequest.Description, params.Description)
						assert.Equal(t, validRequest.AccessLevel, params.AccessLevel)
						// Check that the time is approximately correct (within 1 second)
						if validRequest.ExpiresAt != nil && params.ExpiresAt != nil {
							assert.WithinDuration(t, *validRequest.ExpiresAt, *params.ExpiresAt, time.Second)
						}
						return updatedAPIKey, nil
					})
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.APIKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, updatedAPIKey.ID.String(), response.ID)
				assert.Equal(t, validRequest.Name, response.Name)
			},
		},
		{
			name:        "successfully updates with partial data",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			requestBody: handlers.UpdateAPIKeyRequest{
				Name: "Only Name Updated",
			},
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					UpdateAPIKey(gomock.Any(), gomock.Any()).
					Return(updatedAPIKey, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response handlers.APIKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, updatedAPIKey.ID.String(), response.ID)
			},
		},
		{
			name:           "fails with invalid workspace ID",
			workspaceID:    "invalid-uuid",
			apiKeyID:       apiKeyID.String(),
			requestBody:    validRequest,
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "fails with invalid API key ID",
			workspaceID:    workspaceID.String(),
			apiKeyID:       "invalid-uuid",
			requestBody:    validRequest,
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid UUID format",
		},
		{
			name:           "fails with invalid request body",
			workspaceID:    workspaceID.String(),
			apiKeyID:       apiKeyID.String(),
			requestBody:    "invalid json",
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:        "fails with invalid access level",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			requestBody: handlers.UpdateAPIKeyRequest{
				AccessLevel: "invalid",
			},
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:        "fails with service error",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			requestBody: validRequest,
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					UpdateAPIKey(gomock.Any(), gomock.Any()).
					Return(db.ApiKey{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to update API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService, ctrl := setupAPIKeyHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(mockService)

			headers := map[string]string{}
			if tt.workspaceID != "" {
				headers["X-Workspace-ID"] = tt.workspaceID
			}

			c, w := createTestContext(http.MethodPut, fmt.Sprintf("/api-keys/%s", tt.apiKeyID), tt.requestBody, headers)
			c.Params = gin.Params{
				gin.Param{Key: "api_key_id", Value: tt.apiKeyID},
			}

			handler.UpdateAPIKey(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse handlers.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIKeyHandler_DeleteAPIKey(t *testing.T) {
	workspaceID := uuid.New()
	apiKeyID := uuid.New()

	tests := []struct {
		name           string
		workspaceID    string
		apiKeyID       string
		setupMocks     func(*mocks.MockAPIKeyService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:        "successfully deletes API key",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					DeleteAPIKey(gomock.Any(), apiKeyID, workspaceID).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "fails with invalid workspace ID",
			workspaceID:    "invalid-uuid",
			apiKeyID:       apiKeyID.String(),
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "fails with invalid API key ID",
			workspaceID:    workspaceID.String(),
			apiKeyID:       "invalid-uuid",
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid UUID format",
		},
		{
			name:        "fails when API key not found",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					DeleteAPIKey(gomock.Any(), apiKeyID, workspaceID).
					Return(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Failed to delete API key",
		},
		{
			name:        "fails with service error",
			workspaceID: workspaceID.String(),
			apiKeyID:    apiKeyID.String(),
			setupMocks: func(mockService *mocks.MockAPIKeyService) {
				mockService.EXPECT().
					DeleteAPIKey(gomock.Any(), apiKeyID, workspaceID).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "fails with missing workspace ID header",
			workspaceID:    "",
			apiKeyID:       apiKeyID.String(),
			setupMocks:     func(mockService *mocks.MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService, ctrl := setupAPIKeyHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(mockService)

			headers := map[string]string{}
			if tt.workspaceID != "" {
				headers["X-Workspace-ID"] = tt.workspaceID
			}

			c, w := createTestContext(http.MethodDelete, fmt.Sprintf("/api-keys/%s", tt.apiKeyID), nil, headers)
			c.Params = gin.Params{
				gin.Param{Key: "api_key_id", Value: tt.apiKeyID},
			}

			handler.DeleteAPIKey(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse handlers.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			}

			if tt.expectedStatus == http.StatusNoContent {
				assert.Empty(t, w.Body.String())
			}
		})
	}
}

func TestAPIKeyHandler_EdgeCases(t *testing.T) {
	handler, mockService, ctrl := setupAPIKeyHandler(t)
	defer ctrl.Finish()

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockService.EXPECT().
			GetAllAPIKeys(gomock.Any()).
			Return(nil, context.Canceled)

		c, w := createTestContext(http.MethodGet, "/api-keys/all", nil, nil)
		c.Request = c.Request.WithContext(ctx)

		handler.GetAllAPIKeys(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles large request body", func(t *testing.T) {
		workspaceID := uuid.New()
		largeMetadata := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeMetadata[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		request := handlers.CreateAPIKeyRequest{
			Name:        "Large Metadata Key",
			AccessLevel: "read",
			Metadata:    largeMetadata,
		}

		createdAPIKey := db.ApiKey{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			Name:        request.Name,
			AccessLevel: db.ApiKeyLevel(request.AccessLevel),
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockService.EXPECT().
			CreateAPIKey(gomock.Any(), gomock.Any()).
			Return(createdAPIKey, "cy_test_1234567890abcdef", "cy_test", nil)

		headers := map[string]string{
			"X-Workspace-ID": workspaceID.String(),
		}

		c, w := createTestContext(http.MethodPost, "/api-keys", request, headers)

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("handles special characters in names", func(t *testing.T) {
		workspaceID := uuid.New()
		request := handlers.CreateAPIKeyRequest{
			Name:        "Test Key with émojis 🔑 and speciâl chars!",
			Description: "Description with unicode: ñáéíóú",
			AccessLevel: "write",
		}

		createdAPIKey := db.ApiKey{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			Name:        request.Name,
			AccessLevel: db.ApiKeyLevel(request.AccessLevel),
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockService.EXPECT().
			CreateAPIKey(gomock.Any(), gomock.Any()).
			Return(createdAPIKey, "cy_test_1234567890abcdef", "cy_test", nil)

		headers := map[string]string{
			"X-Workspace-ID": workspaceID.String(),
		}

		c, w := createTestContext(http.MethodPost, "/api-keys", request, headers)

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestAPIKeyHandler_ConcurrentRequests(t *testing.T) {
	handler, mockService, ctrl := setupAPIKeyHandler(t)
	defer ctrl.Finish()

	workspaceID := uuid.New()
	numRequests := 10

	// Setup expectations for concurrent requests
	for i := 0; i < numRequests; i++ {
		mockService.EXPECT().
			ListAPIKeys(gomock.Any(), workspaceID).
			Return([]db.ApiKey{}, nil)
	}

	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer func() { done <- true }()

			headers := map[string]string{
				"X-Workspace-ID": workspaceID.String(),
			}

			c, w := createTestContext(http.MethodGet, "/api-keys", nil, headers)

			handler.ListAPIKeys(c)

			assert.Equal(t, http.StatusOK, w.Code)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

func TestAPIKeyHandler_Integration(t *testing.T) {
	// Integration test using real Gin router
	gin.SetMode(gin.TestMode)

	handler, mockService, ctrl := setupAPIKeyHandler(t)
	defer ctrl.Finish()

	router := gin.New()

	// Register routes
	apiKeys := router.Group("/api-keys")
	apiKeys.GET("", handler.ListAPIKeys)
	apiKeys.GET("/all", handler.GetAllAPIKeys)
	apiKeys.GET("/:api_key_id", handler.GetAPIKeyByID)
	apiKeys.POST("", handler.CreateAPIKey)
	apiKeys.PUT("/:api_key_id", handler.UpdateAPIKey)
	apiKeys.DELETE("/:api_key_id", handler.DeleteAPIKey)

	workspaceID := uuid.New()

	// Test list endpoint
	mockService.EXPECT().
		ListAPIKeys(gomock.Any(), workspaceID).
		Return([]db.ApiKey{}, nil)

	server := httptest.NewServer(router)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api-keys", nil)
	req.Header.Set("X-Workspace-ID", workspaceID.String())

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response handlers.ListAPIKeysResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "list", response.Object)
}

// Benchmark tests
func BenchmarkAPIKeyHandler_ListAPIKeys(b *testing.B) {
	gin.SetMode(gin.TestMode)

	handler, mockService, ctrl := setupAPIKeyHandler(nil)
	defer ctrl.Finish()

	workspaceID := uuid.New()
	apiKeys := make([]db.ApiKey, 100)
	for i := range apiKeys {
		apiKeys[i] = db.ApiKey{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			Name:        fmt.Sprintf("API Key %d", i),
			AccessLevel: db.ApiKeyLevelRead,
			Metadata:    []byte("{}"),
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
	}

	// Setup expectations for benchmark
	mockService.EXPECT().
		ListAPIKeys(gomock.Any(), workspaceID).
		Return(apiKeys, nil).
		Times(b.N)

	headers := map[string]string{
		"X-Workspace-ID": workspaceID.String(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, _ := createTestContext(http.MethodGet, "/api-keys", nil, headers)
		handler.ListAPIKeys(c)
	}
}
