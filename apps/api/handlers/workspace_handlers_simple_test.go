package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/logger"
)

func init() {
	// Initialize logger for tests to avoid panic
	logger.Log = zap.NewNop()
}

// Test the workspace handler's HTTP behavior without database mocking
// This focuses on request/response validation and error handling

func TestWorkspaceHandler_HTTPBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetWorkspace - invalid UUID format", func(t *testing.T) {
		// Create a minimal handler for testing
		handler := &WorkspaceHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/workspaces/invalid-uuid", nil)
		c.Params = gin.Params{
			{Key: "workspace_id", Value: "invalid-uuid"},
		}

		handler.GetWorkspace(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid workspace ID format")
	})

	t.Run("ListWorkspaces - missing workspace header", func(t *testing.T) {
		handler := &WorkspaceHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/workspaces", nil)
		// No X-Workspace-ID header set

		handler.ListWorkspaces(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Workspace ID is required")
	})

	t.Run("ListWorkspaces - invalid workspace ID in header", func(t *testing.T) {
		handler := &WorkspaceHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/workspaces", nil)
		c.Request.Header.Set("X-Workspace-ID", "not-a-uuid")

		handler.ListWorkspaces(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid workspace ID format")
	})

	t.Run("CreateWorkspace - invalid JSON body", func(t *testing.T) {
		handler := &WorkspaceHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/workspaces", 
			bytes.NewBufferString("invalid json"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateWorkspace(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request body")
	})

	t.Run("CreateWorkspace - missing required fields", func(t *testing.T) {
		handler := &WorkspaceHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		// Missing required "name" field
		requestBody := CreateWorkspaceRequest{
			BusinessName: "Test Business",
		}
		jsonBody, _ := json.Marshal(requestBody)
		
		c.Request = httptest.NewRequest(http.MethodPost, "/workspaces", 
			bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateWorkspace(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request body")
	})
}

func TestWorkspaceHandler_PaginationParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		limitParam     string
		offsetParam    string
		expectedLimit  int32
		expectedOffset int32
		expectError    bool
	}{
		{
			name:           "default values",
			limitParam:     "",
			offsetParam:    "",
			expectedLimit:  10,
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "valid custom values",
			limitParam:     "25",
			offsetParam:    "50",
			expectedLimit:  25,
			expectedOffset: 50,
			expectError:    false,
		},
		{
			name:           "limit exceeds maximum",
			limitParam:     "200",
			offsetParam:    "0",
			expectedLimit:  100, // Should be capped at max
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "negative limit uses default",
			limitParam:     "-10",
			offsetParam:    "0",
			expectedLimit:  10, // Should use default
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "negative offset uses default",
			limitParam:     "10",
			offsetParam:    "-5",
			expectedLimit:  10,
			expectedOffset: 0, // Should use default
			expectError:    false,
		},
		{
			name:        "invalid limit format",
			limitParam:  "abc",
			offsetParam: "0",
			expectError: true,
		},
		{
			name:        "invalid offset format",
			limitParam:  "10",
			offsetParam: "xyz",
			expectError: true,
		},
		{
			name:        "overflow limit value",
			limitParam:  "9999999999999",
			offsetParam: "0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			if tt.limitParam != "" {
				q.Add("limit", tt.limitParam)
			}
			if tt.offsetParam != "" {
				q.Add("offset", tt.offsetParam)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req
			
			limit, offset, err := parsePaginationParams(c)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLimit, limit)
				assert.Equal(t, tt.expectedOffset, offset)
			}
		})
	}
}

func TestWorkspaceHandler_SafeParseInt32(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int32
		expectError bool
	}{
		{"valid positive", "123", 123, false},
		{"valid negative", "-456", -456, false},
		{"zero", "0", 0, false},
		{"max int32", "2147483647", 2147483647, false},
		{"min int32", "-2147483648", -2147483648, false},
		{"overflow positive", "2147483648", 0, true},
		{"overflow negative", "-2147483649", 0, true},
		{"invalid format", "abc", 0, true},
		{"empty string", "", 0, true},
		{"float format", "12.34", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeParseInt32(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestWorkspaceResponse_JSONSerialization(t *testing.T) {
	// Test that WorkspaceResponse serializes correctly
	response := WorkspaceResponse{
		ID:           "123e4567-e89b-12d3-a456-426614174000",
		Object:       "workspace",
		Name:         "Test Workspace",
		Description:  "A test workspace",
		BusinessName: "Test Corp",
		BusinessType: "Technology",
		WebsiteURL:   "https://test.com",
		SupportEmail: "support@test.com",
		SupportPhone: "+1234567890",
		AccountID:    "223e4567-e89b-12d3-a456-426614174000",
		Metadata:     map[string]interface{}{"key": "value"},
		Livemode:     false,
		CreatedAt:    1234567890,
		UpdatedAt:    1234567890,
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	// Verify JSON structure
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", parsed["id"])
	assert.Equal(t, "workspace", parsed["object"])
	assert.Equal(t, "Test Workspace", parsed["name"])
	assert.Equal(t, "A test workspace", parsed["description"])
	assert.Equal(t, false, parsed["livemode"])
	
	// Check metadata
	metadata, ok := parsed["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", metadata["key"])
}