package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test customer handler's HTTP behavior without database dependencies
// Focuses on request validation and error handling

func TestCustomerHandler_HTTPValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetCustomer - invalid UUID format", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/customers/invalid-uuid", nil)
		c.Params = gin.Params{
			{Key: "customer_id", Value: "invalid-uuid"},
		}

		handler.GetCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid customer ID format")
	})

	t.Run("GetCustomer - empty customer ID", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/customers/", nil)
		c.Params = gin.Params{
			{Key: "customer_id", Value: ""},
		}

		handler.GetCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ListCustomers - invalid workspace ID", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/customers", nil)
		c.Request.Header.Set("X-Workspace-ID", "not-a-uuid")

		// This will call listWorkspaceCustomers which validates the workspace ID
		handler.ListCustomers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid workspace ID format")
	})

	t.Run("CreateCustomer - invalid JSON body", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/customers", 
			bytes.NewBufferString("invalid json"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request body")
	})

	t.Run("CreateCustomer - missing required email", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		requestBody := CreateCustomerRequest{
			Name: "Test Customer",
			// Missing required Email field
		}
		jsonBody, _ := json.Marshal(requestBody)
		
		c.Request = httptest.NewRequest(http.MethodPost, "/customers", 
			bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("CreateCustomer - invalid email format", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		requestBody := CreateCustomerRequest{
			Email: "not-an-email",
			Name:  "Test Customer",
		}
		jsonBody, _ := json.Marshal(requestBody)
		
		c.Request = httptest.NewRequest(http.MethodPost, "/customers", 
			bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("UpdateCustomer - invalid customer ID", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		requestBody := UpdateCustomerRequest{
			Name: strPtr("Updated Name"),
		}
		jsonBody, _ := json.Marshal(requestBody)
		
		c.Request = httptest.NewRequest(http.MethodPatch, "/customers/invalid-uuid",
			bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{
			{Key: "customer_id", Value: "invalid-uuid"},
		}

		handler.UpdateCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("UpdateCustomer - invalid email in update", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		requestBody := UpdateCustomerRequest{
			Email: strPtr("invalid-email"),
		}
		jsonBody, _ := json.Marshal(requestBody)
		
		c.Request = httptest.NewRequest(http.MethodPatch, "/customers/"+uuid.New().String(),
			bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{
			{Key: "customer_id", Value: uuid.New().String()},
		}

		handler.UpdateCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("DeleteCustomer - invalid UUID", func(t *testing.T) {
		handler := &CustomerHandler{
			common: &CommonServices{},
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/customers/invalid-uuid", nil)
		c.Params = gin.Params{
			{Key: "customer_id", Value: "invalid-uuid"},
		}

		handler.DeleteCustomer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Removed pagination test since it requires database access

// Removed SignInRegisterCustomerRequest validation test since it requires database access

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}

func TestCustomerHandler_NetworkTypeParsing(t *testing.T) {
	handler := &CustomerHandler{}

	tests := []struct {
		input       string
		expectError bool
	}{
		{"evm", false},
		{"solana", false},
		{"cosmos", false},
		{"bitcoin", false},
		{"polkadot", false},
		{"invalid", true},
		{"", true},
		{"EVM", true}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := handler.parseNetworkType(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}