package middleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)


func TestValidateInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		config         ValidationConfig
		body           interface{}
		expectedStatus int
		expectedErrors []string
	}{
		{
			name: "Valid product creation",
			config: ValidationConfig{
				MaxBodySize: 1024,
				Rules: []ValidationRule{
					{Field: "name", Required: true, Type: "string", MinLength: 1},
					{Field: "amount", Required: true, Type: "number", Min: float64Ptr(0)},
				},
			},
			body: map[string]interface{}{
				"name":   "Test Product",
				"amount": 100.0,
			},
			expectedStatus: 200,
		},
		{
			name: "Missing required field",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "name", Required: true, Type: "string"},
				},
			},
			body:           map[string]interface{}{},
			expectedStatus: 400,
			expectedErrors: []string{"name is required"},
		},
		{
			name: "Invalid type",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "amount", Required: true, Type: "number"},
				},
			},
			body: map[string]interface{}{
				"amount": "not-a-number",
			},
			expectedStatus: 400,
			expectedErrors: []string{"must be a number"},
		},
		{
			name: "String too short",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "name", Required: true, Type: "string", MinLength: 5},
				},
			},
			body: map[string]interface{}{
				"name": "abc",
			},
			expectedStatus: 400,
			expectedErrors: []string{"must be at least 5 characters long"},
		},
		{
			name: "Invalid email",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "email", Required: true, Type: "email"},
				},
			},
			body: map[string]interface{}{
				"email": "not-an-email",
			},
			expectedStatus: 400,
			expectedErrors: []string{"must be a valid email address"},
		},
		{
			name: "Invalid UUID",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "id", Required: true, Type: "uuid"},
				},
			},
			body: map[string]interface{}{
				"id": "not-a-uuid",
			},
			expectedStatus: 400,
			expectedErrors: []string{"must be a valid UUID"},
		},
		{
			name: "Request too large",
			config: ValidationConfig{
				MaxBodySize: 10, // Very small limit
				Rules:       []ValidationRule{},
			},
			body: map[string]interface{}{
				"data": "This is a much longer string than 10 bytes",
			},
			expectedStatus: 413,
		},
		{
			name: "Unknown field not allowed",
			config: ValidationConfig{
				AllowUnknownFields: false,
				Rules: []ValidationRule{
					{Field: "name", Required: true, Type: "string"},
				},
			},
			body: map[string]interface{}{
				"name":    "Test",
				"unknown": "field",
			},
			expectedStatus: 400,
			expectedErrors: []string{"unknown field"},
		},
		{
			name: "Sanitize HTML in string",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "description", Required: true, Type: "string", Sanitize: true},
				},
			},
			body: map[string]interface{}{
				"description": "<script>alert('xss')</script>",
			},
			expectedStatus: 200,
		},
		{
			name: "Allowed values validation",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "type", Required: true, Type: "string", AllowedValues: []string{"A", "B", "C"}},
				},
			},
			body: map[string]interface{}{
				"type": "D",
			},
			expectedStatus: 400,
			expectedErrors: []string{"must be one of: A, B, C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()
			router.POST("/test", ValidateInput(tt.config), func(c *gin.Context) {
				// Get validated body
				validatedBody, exists := c.Get("validatedBody")
				assert.True(t, exists)
				
				// Check if sanitization worked
				if bodyMap, ok := validatedBody.(map[string]interface{}); ok {
					if desc, exists := bodyMap["description"]; exists && tt.name == "Sanitize HTML in string" {
						assert.NotContains(t, desc, "<script>")
						// The sanitizer encodes special chars including the & in &lt;
						assert.Contains(t, desc, "script")
					}
				}
				
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Create request
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tt.config.MaxBodySize > 0 {
				req.ContentLength = int64(len(bodyBytes))
			}

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check errors if expected
			if tt.expectedStatus != 200 {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				if errors, ok := response["errors"].([]interface{}); ok && len(tt.expectedErrors) > 0 {
					// Check that expected error messages are present
					for _, expectedErr := range tt.expectedErrors {
						found := false
						for _, err := range errors {
							if errMap, ok := err.(map[string]interface{}); ok {
								if msg, ok := errMap["message"].(string); ok && msg == expectedErr {
									found = true
									break
								}
							}
						}
						assert.True(t, found, "Expected error message not found: %s", expectedErr)
					}
				}
			}
		})
	}
}

func TestValidateQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		config         ValidationConfig
		query          string
		expectedStatus int
	}{
		{
			name: "Valid pagination params",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "page", Type: "number", Min: float64Ptr(1)},
					{Field: "limit", Type: "number", Min: float64Ptr(1), Max: float64Ptr(100)},
				},
			},
			query:          "page=2&limit=50",
			expectedStatus: 200,
		},
		{
			name: "Invalid page number",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "page", Type: "number", Min: float64Ptr(1)},
				},
			},
			query:          "page=0",
			expectedStatus: 400,
		},
		{
			name: "Limit too high",
			config: ValidationConfig{
				Rules: []ValidationRule{
					{Field: "limit", Type: "number", Max: float64Ptr(100)},
				},
			},
			query:          "limit=200",
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", ValidateQueryParams(tt.config), func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest("GET", "/test?"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}