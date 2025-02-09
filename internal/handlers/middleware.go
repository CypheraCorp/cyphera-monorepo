package handlers

import (
	"bytes"
	"cyphera-api/internal/logger"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLog represents a structured log entry for an HTTP request
type RequestLog struct {
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Query     string    `json:"query"`
	UserAgent string    `json:"user_agent"`
	ClientIP  string    `json:"client_ip"`
	RequestID string    `json:"request_id"`
	AccountID string    `json:"account_id"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
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

// LogRequestBody is a middleware that logs the request body
func LogRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		if shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get request body
		bodyBytes, err := getRequestBody(c)
		if err != nil {
			logger.Log.Error("Failed to read request body", zap.Error(err))
			c.Next()
			return
		}

		// Create request log entry
		requestLog := RequestLog{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			UserAgent: c.Request.UserAgent(),
			ClientIP:  c.ClientIP(),
			RequestID: c.GetString("request_id"),
			AccountID: c.GetString("account_id"),
			Body:      string(bodyBytes),
			Timestamp: time.Now().UTC(),
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
			zap.String("body", requestLog.Body),
			zap.Time("timestamp", requestLog.Timestamp),
		)

		c.Next()
	}
}
