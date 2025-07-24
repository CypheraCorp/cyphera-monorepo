package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter holds the configuration for rate limiting
type RateLimiter struct {
	// limiters stores rate limiters per IP/API key
	limiters sync.Map
	// rate is the number of requests per second allowed
	rate int
	// burst is the maximum burst size
	burst int
	// cleanupInterval is how often to clean up old limiters
	cleanupInterval time.Duration
}

// limiterEntry holds a rate limiter and its last access time
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst
func NewRateLimiter(requestsPerSecond, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:            requestsPerSecond,
		burst:           burst,
		cleanupInterval: 5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old limiters that haven't been accessed recently
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		rl.limiters.Range(func(key, value interface{}) bool {
			if entry, ok := value.(*limiterEntry); ok {
				// Remove limiters that haven't been accessed in 10 minutes
				if now.Sub(entry.lastAccess) > 10*time.Minute {
					rl.limiters.Delete(key)
				}
			}
			return true
		})
	}
}

// getLimiter returns the rate limiter for a specific key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	// Try to get existing limiter
	if val, ok := rl.limiters.Load(key); ok {
		entry := val.(*limiterEntry)
		entry.lastAccess = time.Now()
		return entry.limiter
	}

	// Create new limiter
	limiter := rate.NewLimiter(rate.Limit(rl.rate), rl.burst)
	entry := &limiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	
	// Store it (handle race condition where another goroutine created it)
	actual, _ := rl.limiters.LoadOrStore(key, entry)
	return actual.(*limiterEntry).limiter
}

// getClientIdentifier returns a unique identifier for the client
func getClientIdentifier(c *gin.Context) string {
	// First check for API key
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		// Use first 8 characters of API key as identifier
		if len(apiKey) >= 8 {
			return fmt.Sprintf("api:%s", apiKey[:8])
		}
		return fmt.Sprintf("api:%s", apiKey)
	}

	// Then check for authenticated user
	if userID, exists := c.Get("userID"); exists {
		return fmt.Sprintf("user:%v", userID)
	}

	// Fall back to IP address
	// In test environment, check X-Forwarded-For first
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		return fmt.Sprintf("ip:%s", forwardedFor)
	}
	
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "unknown"
	}
	return fmt.Sprintf("ip:%s", clientIP)
}

// Middleware returns a Gin middleware handler for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health check endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}

		// Get client identifier
		clientID := getClientIdentifier(c)
		
		// Get the limiter for this client
		limiter := rl.getLimiter(clientID)

		// Check if request is allowed
		if !limiter.Allow() {
			if logger.Log != nil {
				logger.Log.Warn("Rate limit exceeded",
					zap.String("client_id", clientID),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("client_ip", c.ClientIP()),
				)
			}

			// Add rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			c.Header("Retry-After", "1")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
				"retry_after": 1,
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.Burst()-int(limiter.Tokens())))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

		c.Next()
	}
}

// MiddlewareWithConfig returns a Gin middleware with custom configuration per endpoint
func (rl *RateLimiter) MiddlewareWithConfig(customRate, customBurst int) gin.HandlerFunc {
	// Create a new rate limiter with custom settings
	customRL := &RateLimiter{
		rate:            customRate,
		burst:           customBurst,
		cleanupInterval: rl.cleanupInterval,
	}

	// Start cleanup for custom limiter
	go customRL.cleanup()

	return customRL.Middleware()
}

// Global rate limiters with different configurations
var (
	// DefaultRateLimiter for general API endpoints (100 requests per second, burst of 200)
	DefaultRateLimiter = NewRateLimiter(100, 200)
	
	// StrictRateLimiter for sensitive endpoints like auth (10 requests per second, burst of 20)
	StrictRateLimiter = NewRateLimiter(10, 20)
	
	// RelaxedRateLimiter for read-heavy endpoints (500 requests per second, burst of 1000)
	RelaxedRateLimiter = NewRateLimiter(500, 1000)
)