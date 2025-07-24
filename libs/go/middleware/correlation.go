package middleware

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	correlationIDKey    = "correlationID"
)

// CorrelationIDMiddleware ensures every request has a correlation ID
// for distributed tracing and debugging
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if correlation ID already exists in request header
		correlationID := c.GetHeader(CorrelationIDHeader)
		
		// Generate new ID if not provided
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		
		// Store in context for use in handlers
		c.Set(correlationIDKey, correlationID)
		
		// Add to response headers
		c.Header(CorrelationIDHeader, correlationID)
		
		// Add to logger context
		ctx := WithCorrelationID(c.Request.Context(), correlationID)
		c.Request = c.Request.WithContext(ctx)
		
		// Log request with correlation ID (only if logger is initialized)
		if logger.Log != nil {
			logger.Log.Info("Request received",
				zap.String("correlation_id", correlationID),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("client_ip", c.ClientIP()),
			)
		}
		
		c.Next()
	}
}

// GetCorrelationID retrieves the correlation ID from the Gin context
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(correlationIDKey); exists {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	return ""
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const correlationIDContextKey contextKey = "correlationID"

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDContextKey, correlationID)
}

// CorrelationIDFromContext retrieves correlation ID from context
func CorrelationIDFromContext(ctx context.Context) string {
	if id := ctx.Value(correlationIDContextKey); id != nil {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	return ""
}

// LogWithCorrelationID creates a logger with correlation ID field
func LogWithCorrelationID(ctx context.Context) *zap.Logger {
	if logger.Log == nil {
		return nil
	}
	
	correlationID := CorrelationIDFromContext(ctx)
	if correlationID != "" {
		return logger.Log.With(zap.String("correlation_id", correlationID))
	}
	return logger.Log
}