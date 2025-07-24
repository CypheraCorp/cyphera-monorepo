package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within rate limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 20) // 10 requests per second, burst of 20
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make 10 requests (should all succeed)
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.1")
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
			assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
		}
	})

	t.Run("blocks requests exceeding rate limit", func(t *testing.T) {
		rl := NewRateLimiter(1, 2) // 1 request per second, burst of 2
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make 3 rapid requests
		var lastCode int
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.2")
			router.ServeHTTP(w, req)
			lastCode = w.Code
		}

		// The third request should be rate limited
		assert.Equal(t, http.StatusTooManyRequests, lastCode)
	})

	t.Run("different clients have separate limits", func(t *testing.T) {
		rl := NewRateLimiter(1, 1) // 1 request per second, burst of 1
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Client 1 makes a request
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Forwarded-For", "192.168.1.3")
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Client 2 makes a request (should succeed)
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Forwarded-For", "192.168.1.4")
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		// Client 1 makes another request (should be rate limited)
		w3 := httptest.NewRecorder()
		req3, _ := http.NewRequest("GET", "/test", nil)
		req3.Header.Set("X-Forwarded-For", "192.168.1.3")
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	})

	t.Run("API key based rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(1, 1)
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First request with API key
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-API-Key", "test-key-123")
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request with same API key (should be rate limited)
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-API-Key", "test-key-123")
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)

		// Wait a bit to allow rate limit to replenish
		time.Sleep(1100 * time.Millisecond)

		// Request with different API key (should succeed)
		w3 := httptest.NewRecorder()
		req3, _ := http.NewRequest("GET", "/test", nil)
		req3.Header.Set("X-API-Key", "test-key-456")
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusOK, w3.Code)
	})

	t.Run("health endpoints bypass rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(1, 1) // Very strict limit
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make many requests to health endpoint
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/health", nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("concurrent requests handling", func(t *testing.T) {
		rl := NewRateLimiter(10, 20) // 10 requests per second, burst of 20
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		var wg sync.WaitGroup
		successCount := 0
		rateLimitedCount := 0
		var mu sync.Mutex

		// Make 50 concurrent requests from same client
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-For", "192.168.1.100")
				router.ServeHTTP(w, req)
				
				mu.Lock()
				if w.Code == http.StatusOK {
					successCount++
				} else if w.Code == http.StatusTooManyRequests {
					rateLimitedCount++
				}
				mu.Unlock()
			}()
		}

		wg.Wait()
		
		// With burst of 20, we should have at least 20 successful and some rate limited
		assert.GreaterOrEqual(t, successCount, 20)
		assert.Greater(t, rateLimitedCount, 0)
		assert.Equal(t, 50, successCount+rateLimitedCount)
	})

	t.Run("rate limit headers are set correctly", func(t *testing.T) {
		rl := NewRateLimiter(10, 20)
		router := gin.New()
		router.Use(rl.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("cleanup removes old limiters", func(t *testing.T) {
		// Create a rate limiter with very short cleanup interval for testing
		rl := &RateLimiter{
			rate:            10,
			burst:           20,
			cleanupInterval: 100 * time.Millisecond,
		}
		go rl.cleanup()

		// Add a limiter
		limiter := rl.getLimiter("test-client")
		assert.NotNil(t, limiter)

		// Verify it exists
		_, exists := rl.limiters.Load("test-client")
		assert.True(t, exists)

		// Wait for cleanup to potentially run (but entry should still exist as it's recent)
		time.Sleep(200 * time.Millisecond)
		_, exists = rl.limiters.Load("test-client")
		assert.True(t, exists)

		// Now simulate old entry by manually setting old timestamp
		rl.limiters.Store("old-client", &limiterEntry{
			limiter:    limiter,
			lastAccess: time.Now().Add(-15 * time.Minute),
		})

		// Wait for cleanup to run
		time.Sleep(200 * time.Millisecond)

		// Old entry should be removed
		_, exists = rl.limiters.Load("old-client")
		assert.False(t, exists)

		// Recent entry should still exist
		_, exists = rl.limiters.Load("test-client")
		assert.True(t, exists)
	})
}

func TestRateLimiterMiddlewareWithConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("custom rate configuration", func(t *testing.T) {
		rl := NewRateLimiter(10, 20) // Default rate limiter
		router := gin.New()
		
		// Apply custom rate limiting to specific endpoint
		router.GET("/strict", rl.MiddlewareWithConfig(1, 2), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make 3 requests rapidly
		var lastCode int
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/strict", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.10")
			router.ServeHTTP(w, req)
			lastCode = w.Code
		}

		// Third request should be rate limited with the strict configuration
		assert.Equal(t, http.StatusTooManyRequests, lastCode)
	})
}