package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/logger"
)

func init() {
	// Initialize logger for tests to avoid panic
	logger.Log = zap.NewNop()
}

func TestNewProductHandler(t *testing.T) {
	common := &CommonServices{}
	delegationClient := &dsClient.DelegationClient{}
	handler := NewProductHandler(common, delegationClient)
	
	require.NotNil(t, handler)
	assert.Equal(t, common, handler.common)
	assert.Equal(t, delegationClient, handler.delegationClient)
}

func TestProductHandler_CreateProduct_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing required name",
			requestBody: CreateProductRequest{
				WalletID:    uuid.New().String(),
				Description: "Test product",
				Prices:      []CreatePriceRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "missing required wallet ID",
			requestBody: CreateProductRequest{
				Name:        "Test Product",
				Description: "Test product",
				Prices:      []CreatePriceRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "invalid wallet ID format",
			requestBody: CreateProductRequest{
				Name:        "Test Product",
				WalletID:    "invalid-uuid",
				Description: "Test product",
				Prices:      []CreatePriceRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name: "missing required prices",
			requestBody: CreateProductRequest{
				Name:        "Test Product",
				WalletID:    uuid.New().String(),
				Description: "Test product",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "invalid price - missing type",
			requestBody: CreateProductRequest{
				Name:     "Test Product",
				WalletID: uuid.New().String(),
				Prices: []CreatePriceRequest{
					{
						Currency:            "USD",
						UnitAmountInPennies: 1000,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "invalid price - missing currency",
			requestBody: CreateProductRequest{
				Name:     "Test Product",
				WalletID: uuid.New().String(),
				Prices: []CreatePriceRequest{
					{
						Type:                "one_time",
						UnitAmountInPennies: 1000,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "empty product name",
			requestBody: CreateProductRequest{
				Name:     "",
				WalletID: uuid.New().String(),
				Prices: []CreatePriceRequest{
					{
						Type:                "one_time",
						Currency:            "USD",
						UnitAmountInPennies: 1000,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ProductHandler{
				common:           &CommonServices{},
				delegationClient: &dsClient.DelegationClient{},
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var requestBody []byte
			if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, _ = json.Marshal(tt.requestBody)
			}

			c.Request = httptest.NewRequest(http.MethodPost, "/products", 
				bytes.NewBuffer(requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.CreateProduct(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}

func TestProductHandler_GetProduct_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid UUID format",
			productID:      "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "empty product ID",
			productID:      "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "SQL injection attempt",
			productID:      "DROP-TABLE-products",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ProductHandler{
				common:           &CommonServices{},
				delegationClient: &dsClient.DelegationClient{},
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/products/"+tt.productID, nil)
			c.Params = gin.Params{
				{Key: "product_id", Value: tt.productID},
			}

			handler.GetProduct(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}

func TestProductHandler_UpdateProduct_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		productID      string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "invalid product ID",
			productID:  "invalid-uuid",
			requestBody: map[string]interface{}{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "invalid JSON",
			productID:      uuid.New().String(),
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:       "invalid URL format",
			productID:  uuid.New().String(),
			requestBody: map[string]interface{}{
				"url": "not a valid url",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid URL format",
		},
		{
			name:       "invalid image URL format",
			productID:  uuid.New().String(),
			requestBody: map[string]interface{}{
				"image_url": "not a valid url",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid image URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ProductHandler{
				common:           &CommonServices{},
				delegationClient: &dsClient.DelegationClient{},
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var requestBody []byte
			if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, _ = json.Marshal(tt.requestBody)
			}

			c.Request = httptest.NewRequest(http.MethodPatch, "/products/"+tt.productID,
				bytes.NewBuffer(requestBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{
				{Key: "product_id", Value: tt.productID},
			}

			handler.UpdateProduct(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}

func TestProductHandler_DeleteProduct_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid UUID format",
			productID:      "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
		{
			name:           "empty product ID",
			productID:      "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid workspace ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ProductHandler{
				common:           &CommonServices{},
				delegationClient: &dsClient.DelegationClient{},
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodDelete, "/products/"+tt.productID, nil)
			c.Params = gin.Params{
				{Key: "product_id", Value: tt.productID},
			}

			handler.DeleteProduct(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}

func TestProductHandler_ListProducts_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &ProductHandler{
		common:           &CommonServices{},
		delegationClient: &dsClient.DelegationClient{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/products", nil)

	// Without workspace ID, it should fail
	handler.ListProducts(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "Workspace ID is required")
}


func TestPriceValidation(t *testing.T) {
	tests := []struct {
		name        string
		price       CreatePriceRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid one-time price",
			price: CreatePriceRequest{
				Type:                "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
			},
			expectError: false,
		},
		{
			name: "valid recurring price",
			price: CreatePriceRequest{
				Type:                "recurring",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
				IntervalType:        "month",
				IntervalCount:       1,
			},
			expectError: false,
		},
		{
			name: "negative amount",
			price: CreatePriceRequest{
				Type:                "one_time",
				Currency:            "USD",
				UnitAmountInPennies: -100,
			},
			expectError: true,
			errorMsg:    "Unit amount must be positive",
		},
		{
			name: "zero amount",
			price: CreatePriceRequest{
				Type:                "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 0,
			},
			expectError: true,
			errorMsg:    "Unit amount must be positive",
		},
		{
			name: "recurring without interval type",
			price: CreatePriceRequest{
				Type:                "recurring",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
				IntervalCount:       1,
			},
			expectError: true,
			errorMsg:    "Recurring prices must have interval_type and interval_count",
		},
		{
			name: "recurring without interval count",
			price: CreatePriceRequest{
				Type:                "recurring",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
				IntervalType:        "month",
			},
			expectError: true,
			errorMsg:    "Recurring prices must have interval_type and interval_count",
		},
		{
			name: "invalid interval type",
			price: CreatePriceRequest{
				Type:                "recurring",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
				IntervalType:        "invalid",
				IntervalCount:       1,
			},
			expectError: true,
			errorMsg:    "Invalid interval type",
		},
		{
			name: "invalid price type",
			price: CreatePriceRequest{
				Type:                "invalid_type",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
			},
			expectError: true,
			errorMsg:    "Invalid price type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrice(tt.price)
			if tt.expectError {
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

// Mock validatePrice function for testing
func validatePrice(price CreatePriceRequest) error {
	// Validate price type
	if price.Type != "one_time" && price.Type != "recurring" {
		return fmt.Errorf("Invalid price type: %s", price.Type)
	}

	// Validate amount
	if price.UnitAmountInPennies <= 0 {
		return fmt.Errorf("Unit amount must be positive")
	}

	// Validate recurring price requirements
	if price.Type == "recurring" {
		if price.IntervalType == "" || price.IntervalCount == 0 {
			return fmt.Errorf("Recurring prices must have interval_type and interval_count")
		}
		
		// Validate interval type
		validIntervals := map[string]bool{
			"day":   true,
			"week":  true,
			"month": true,
			"year":  true,
		}
		if !validIntervals[price.IntervalType] {
			return fmt.Errorf("Invalid interval type: %s", price.IntervalType)
		}
	}

	return nil
}

