package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler()
	require.NotNil(t, handler)
	assert.IsType(t, &HealthHandler{}, handler)
}

func TestHealthHandler_Health(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)
	
	// Create handler
	handler := NewHealthHandler()
	
	// Test cases
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   HealthResponse
	}{
		{
			name:           "GET request returns ok",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   HealthResponse{Status: "ok"},
		},
		{
			name:           "POST request also works",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectedBody:   HealthResponse{Status: "ok"},
		},
		{
			name:           "PUT request works",
			method:         http.MethodPut,
			expectedStatus: http.StatusOK,
			expectedBody:   HealthResponse{Status: "ok"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new gin context for each test
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			// Create request
			req := httptest.NewRequest(tt.method, "/health", nil)
			c.Request = req
			
			// Call the handler
			handler.Health(c)
			
			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			// Parse response body
			var response HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			// Check response
			assert.Equal(t, tt.expectedBody, response)
		})
	}
}

func TestHealthHandler_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewHealthHandler()
	
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
	
	// Call handler
	handler.Health(c)
	
	// Check response format
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	
	// Check exact JSON format
	expectedJSON := `{"status":"ok"}`
	assert.JSONEq(t, expectedJSON, w.Body.String())
}

func TestHealthHandler_Concurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewHealthHandler()
	
	// Test concurrent requests
	concurrentRequests := 10
	done := make(chan bool, concurrentRequests)
	
	for i := 0; i < concurrentRequests; i++ {
		go func() {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
			
			handler.Health(c)
			
			assert.Equal(t, http.StatusOK, w.Code)
			
			var response HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "ok", response.Status)
			
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < concurrentRequests; i++ {
		<-done
	}
}

func TestHealthHandler_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a full gin router
	router := gin.New()
	handler := NewHealthHandler()
	
	// Register the route
	router.GET("/health", handler.Health)
	
	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Make actual HTTP request
	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
}

// Benchmark the health endpoint
func BenchmarkHealthHandler_Health(b *testing.B) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
		
		handler.Health(c)
	}
}

// Example of how to test with middleware
func TestHealthHandler_WithMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create router with middleware
	router := gin.New()
	
	// Add a test middleware that adds a header
	router.Use(func(c *gin.Context) {
		c.Header("X-Test-Middleware", "applied")
		c.Next()
	})
	
	handler := NewHealthHandler()
	router.GET("/health", handler.Health)
	
	// Test request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	
	router.ServeHTTP(w, req)
	
	// Check middleware was applied
	assert.Equal(t, "applied", w.Header().Get("X-Test-Middleware"))
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
}