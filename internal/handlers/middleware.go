package handlers

import (
	"bytes"
	"cyphera-api/internal/logger"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// RequestLog represents a structured log entry for an HTTP request
type RequestLog struct {
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Query      string    `json:"query"`
	UserAgent  string    `json:"user_agent"`
	ClientIP   string    `json:"client_ip"`
	RequestID  string    `json:"request_id"`
	AccountID  string    `json:"account_id"`
	SupabaseID string    `json:"supabase_id,omitempty"`
	Body       string    `json:"body"`
	Timestamp  time.Time `json:"timestamp"`
}

// SupabaseClaims represents the expected structure of the Supabase JWT claims
type SupabaseClaims struct {
	Sub         string `json:"sub"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Aud         string `json:"aud"`
	Exp         int64  `json:"exp"`
	SupabaseRef string `json:"reference_id"`
	jwt.RegisteredClaims
}

// ValidateSupabaseToken validates the Supabase JWT token
func ValidateSupabaseToken(tokenString string) (*SupabaseClaims, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &SupabaseClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Log.Debug("Token validation failed",
				zap.String("reason", "unexpected signing method"),
				zap.String("algorithm", token.Header["alg"].(string)),
			)
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get JWT secret from environment
		jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
		if jwtSecret == "" {
			logger.Log.Debug("Token validation failed",
				zap.String("reason", "SUPABASE_JWT_SECRET not set"),
			)
			return nil, fmt.Errorf("SUPABASE_JWT_SECRET not set")
		}

		return []byte(jwtSecret), nil
	})

	if err != nil {
		logger.Log.Debug("Token parsing failed",
			zap.Error(err),
			zap.String("token_prefix", tokenString[:10]+"..."), // Safely log token prefix
		)
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*SupabaseClaims); ok && token.Valid {
		// Check if token is expired
		if time.Unix(claims.Exp, 0).Before(time.Now()) {
			logger.Log.Debug("Token validation failed",
				zap.String("reason", "token expired"),
				zap.Time("expiration", time.Unix(claims.Exp, 0)),
				zap.Time("current_time", time.Now()),
			)
			return nil, fmt.Errorf("token is expired")
		}
		return claims, nil
	}

	logger.Log.Debug("Token validation failed",
		zap.String("reason", "invalid token claims"),
	)
	return nil, fmt.Errorf("invalid token claims")
}

// shouldSkipLogging determines if request logging should be skipped for a given path
func shouldSkipLogging(path string) bool {
	// Skip logging for health check endpoints
	if path == "/healthz" || path == "/readyz" {
		return true
	}
	return false
}

// getRequestBody safely reads and returns the request body
func getRequestBody(c *gin.Context) ([]byte, error) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		// Restore the request body for subsequent middleware/handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return bodyBytes, nil
}

// AuthMiddleware handles authentication for both Supabase JWT and API keys
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check for API key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			logger.Log.Debug("Using API key authentication")
			c.Next()
			return
		}

		// Check for Supabase JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Log.Debug("Authentication failed",
				zap.String("reason", "No authentication header provided"),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication provided"})
			c.Abort()
			return
		}

		// Validate Supabase token
		claims, err := ValidateSupabaseToken(authHeader)
		if err != nil {
			logger.Log.Debug("Supabase token validation failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
				zap.String("auth_header", strings.Replace(authHeader, "Bearer ", "***.", 1)), // Safely log partial token
			)
			logger.Log.Error("Failed to validate Supabase token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		logger.Log.Debug("Supabase authentication successful",
			zap.String("user_id", claims.Sub),
			zap.String("email", claims.Email),
			zap.String("role", claims.Role),
			zap.String("path", c.Request.URL.Path),
		)

		// Set Supabase user information in context
		c.Set("supabase_id", claims.Sub)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// LogRequest is a middleware that logs the request body
func LogRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		if shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get request body
		bodyBytes, err := getRequestBody(c)
		if err != nil {
			logger.Log.Debug("Request body reading failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			logger.Log.Error("Failed to read request body", zap.Error(err))
			c.Next()
			return
		}

		// Create request log entry
		requestLog := RequestLog{
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Query:      c.Request.URL.RawQuery,
			UserAgent:  c.Request.UserAgent(),
			ClientIP:   c.ClientIP(),
			RequestID:  c.GetString("request_id"),
			AccountID:  c.GetString("account_id"),
			SupabaseID: c.GetString("supabase_id"), // Add Supabase ID to logs
			Body:       string(bodyBytes),
			Timestamp:  time.Now().UTC(),
		}

		// Log the request
		logger.Log.Debug("Request received",
			zap.String("method", requestLog.Method),
			zap.String("path", requestLog.Path),
			zap.String("query", requestLog.Query),
			zap.String("user_agent", requestLog.UserAgent),
			zap.String("client_ip", requestLog.ClientIP),
			zap.String("request_id", requestLog.RequestID),
			zap.String("account_id", requestLog.AccountID),
			zap.String("supabase_id", requestLog.SupabaseID),
			zap.String("body", requestLog.Body),
			zap.Time("timestamp", requestLog.Timestamp),
		)

		c.Next()
	}
}
