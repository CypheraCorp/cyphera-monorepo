package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// bodyLogWriter is a wrapper around gin.ResponseWriter that captures the response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// EnhancedLoggingMiddleware provides detailed request/response logging in development mode
func EnhancedLoggingMiddleware(isDevelopment bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if not in development mode
		if !isDevelopment {
			c.Next()
			return
		}

		// Start timer
		startTime := time.Now()

		// Get correlation ID
		correlationID := GetCorrelationID(c)

		// Create logger with correlation ID (skip if logger not initialized)
		if logger.Log == nil {
			c.Next()
			return
		}
		log := logger.Log.With(zap.String("correlation_id", correlationID))

		// Read and log request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Log request headers (excluding sensitive ones)
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			// Skip sensitive headers
			if key == "Authorization" || key == "X-Api-Key" || key == "Cookie" {
				headers[key] = "[REDACTED]"
			} else {
				headers[key] = values[0]
			}
		}

		// Parse request body for logging (only for JSON content)
		var requestJSON interface{}
		if c.GetHeader("Content-Type") == "application/json" && len(requestBody) > 0 {
			json.Unmarshal(requestBody, &requestJSON)
		}

		// Log detailed request
		log.Info("Detailed request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Any("headers", headers),
			zap.Any("body", requestJSON),
			zap.Int("body_size", len(requestBody)),
		)

		// Wrap response writer to capture response body
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(startTime)

		// Parse response body for logging (only for JSON content)
		var responseJSON interface{}
		responseBody := blw.body.Bytes()
		contentType := c.Writer.Header().Get("Content-Type")
		if strings.HasPrefix(contentType, "application/json") && len(responseBody) > 0 {
			if err := json.Unmarshal(responseBody, &responseJSON); err != nil {
				log.Debug("Failed to parse response JSON", zap.Error(err))
				responseJSON = string(responseBody)
			}
		}

		// Response headers
		responseHeaders := make(map[string]string)
		for key, values := range c.Writer.Header() {
			responseHeaders[key] = values[0]
		}

		// Log detailed response
		log.Info("Detailed response",
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.Any("headers", responseHeaders),
			zap.Any("body", responseJSON),
			zap.Int("body_size", len(responseBody)),
			zap.Int("errors_count", len(c.Errors)),
		)

		// Log any errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				log.Error("Request error",
					zap.Error(err.Err),
					zap.Uint64("type", uint64(err.Type)),
					zap.Any("meta", err.Meta),
				)
			}
		}
	}
}

// RequestLoggingMiddleware provides basic request logging for production
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(startTime)

		// Get correlation ID
		correlationID := GetCorrelationID(c)

		// Log basic request info (only if logger is initialized)
		if logger.Log != nil {
			logger.Log.Info("Request completed",
				zap.String("correlation_id", correlationID),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("duration", duration),
				zap.String("client_ip", c.ClientIP()),
				zap.Int("body_size", c.Writer.Size()),
			)
		}
	}
}
