package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCorrelationIDMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                string
		requestCorrelationID string
		expectNewID         bool
	}{
		{
			name:                "New ID generated when header not present",
			requestCorrelationID: "",
			expectNewID:         true,
		},
		{
			name:                "Existing ID preserved when header present",
			requestCorrelationID: "test-correlation-id-123",
			expectNewID:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new router with the middleware
			router := gin.New()
			router.Use(CorrelationIDMiddleware())
			
			// Add a test route that returns the correlation ID
			router.GET("/test", func(c *gin.Context) {
				correlationID := GetCorrelationID(c)
				c.JSON(http.StatusOK, gin.H{
					"correlation_id": correlationID,
				})
			})

			// Create a request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.requestCorrelationID != "" {
				req.Header.Set(CorrelationIDHeader, tt.requestCorrelationID)
			}

			// Perform the request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check the response
			assert.Equal(t, http.StatusOK, w.Code)

			// Check the response header
			responseCorrelationID := w.Header().Get(CorrelationIDHeader)
			assert.NotEmpty(t, responseCorrelationID)

			if tt.expectNewID {
				// When no ID provided, a new one should be generated
				assert.NotEqual(t, "", responseCorrelationID)
				// Check it's a valid UUID format (36 chars with dashes)
				assert.Len(t, responseCorrelationID, 36)
			} else {
				// When ID provided, it should be preserved
				assert.Equal(t, tt.requestCorrelationID, responseCorrelationID)
			}
		})
	}
}

func TestCorrelationIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CorrelationIDMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		// Get correlation ID from context
		correlationID := CorrelationIDFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"correlation_id": correlationID,
		})
	})

	// Test with provided correlation ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	testID := "test-correlation-id-456"
	req.Header.Set(CorrelationIDHeader, testID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testID, w.Header().Get(CorrelationIDHeader))
}