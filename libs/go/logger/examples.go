package logger

import (
	"context"
	"time"
)

// LoggingExamples demonstrates how to use the structured logging system
// This file serves as both documentation and examples for developers

// Example 1: Basic structured logging with components
func ExampleBasicLogging() {
	// Create a logger for the API component
	apiLogger := NewStructuredLogger(ComponentAPI)
	
	// Log a simple info message
	apiLogger.Info("Server starting up")
	
	// Log with additional context
	apiLogger.WithWorkspaceID("workspace-123").
		WithUserID("user-456").
		Info("User authenticated")
	
	// Log an error with context
	apiLogger.WithOperation("create_subscription").
		Error("Failed to create subscription", nil)
}

// Example 2: HTTP request logging
func ExampleHTTPRequestLogging() {
	middlewareLogger := NewStructuredLogger(ComponentMiddleware)
	
	// Log HTTP request processing
	middlewareLogger.LogHTTPRequest(
		"POST",
		"/api/subscriptions",
		201,
		time.Millisecond*150,
	)
	
	// Log with correlation ID for tracking
	middlewareLogger.WithCorrelationID("req-789").
		WithFields(map[string]interface{}{
			"user_agent": "cyphera-web/1.0",
			"ip_address": "192.168.1.100",
		}).
		Info("Request processed successfully")
}

// Example 3: Database operation logging
func ExampleDatabaseLogging() {
	dbLogger := NewStructuredLogger(ComponentDB)
	
	// Log slow query
	dbLogger.LogDatabaseQuery(
		"SELECT * FROM subscriptions WHERE status = $1",
		time.Millisecond*500,
		25,
	)
	
	// Log transaction
	dbLogger.WithOperation("create_customer_with_wallet").
		WithDuration(time.Millisecond*200).
		Info("Transaction completed successfully")
}

// Example 4: Subscription processing logging
func ExampleSubscriptionLogging() {
	subLogger := NewStructuredLogger(ComponentSubscription)
	
	// Log subscription event
	subLogger.LogSubscriptionEvent(
		"sub-123",
		"payment_processed",
		map[string]interface{}{
			"amount":        1000,
			"currency":      "USD",
			"payment_method": "card",
		},
	)
	
	// Log processing with timing
	timer := subLogger.NewTimer("process_due_subscriptions")
	// ... do work ...
	timer.StopWithResult(true, nil)
}

// Example 5: Error handling and logging
func ExampleErrorLogging() {
	paymentLogger := NewStructuredLogger(ComponentPayment)
	
	// Log different types of errors
	paymentLogger.WithField("payment_id", "pay-456").
		Error("Payment failed due to insufficient funds", nil)
	
	paymentLogger.WithWorkspaceID("workspace-123").
		WithField("retry_count", 3).
		Error("Max retries exceeded for payment processing", nil)
}

// Example 6: Performance monitoring
func ExamplePerformanceLogging() {
	workerLogger := NewStructuredLogger(ComponentWorker)
	
	// Track operation performance
	err := workerLogger.LogOperation("batch_process_subscriptions", func() error {
		// Simulate work
		time.Sleep(time.Millisecond * 100)
		return nil
	})
	
	if err != nil {
		workerLogger.Error("Batch processing failed", err)
	}
}

// Example 7: Webhook processing
func ExampleWebhookLogging() {
	webhookLogger := NewStructuredLogger(ComponentWebhook)
	
	// Log webhook processing
	webhookLogger.LogWebhookEvent(
		"stripe",
		"payment_intent.succeeded",
		true,
		time.Millisecond*50,
	)
	
	// Log webhook retry
	webhookLogger.WithFields(map[string]interface{}{
		"webhook_id":    "wh-789",
		"attempt":       2,
		"max_attempts":  3,
	}).Warn("Webhook delivery retry")
}

// Example 8: Authentication logging
func ExampleAuthLogging() {
	authLogger := NewStructuredLogger(ComponentAuth)
	
	// Log successful authentication
	authLogger.LogAuthEvent(
		"login",
		"user-123",
		true,
		"valid_credentials",
	)
	
	// Log failed authentication
	authLogger.LogAuthEvent(
		"login",
		"user-456",
		false,
		"invalid_password",
	)
}

// Example 9: Using filtered logging
func ExampleFilteredLogging() {
	// Create a filter that only logs errors from subscription component
	filter := LogFilter{
		MinLevel:   ErrorLevel,
		Components: []LogComponent{ComponentSubscription},
	}
	
	filteredLogger := NewFilteredLogger(ComponentSubscription, filter)
	
	// This will be logged (error level)
	filteredLogger.Error("Subscription processing failed", nil)
	
	// This will be filtered out (info level)
	filteredLogger.Info("Subscription created")
}

// Example 10: Context-aware logging
func ExampleContextLogging() {
	// In a real handler, you'd extract context from the request
	logContext := LogContext{
		UserID:        "user-123",
		WorkspaceID:   "workspace-456",
		CorrelationID: "req-789",
		Component:     ComponentAPI,
		Operation:     "create_subscription",
	}
	
	apiLogger := NewStructuredLogger(ComponentAPI).WithContext(logContext)
	
	// All logs will now include the context automatically
	apiLogger.Info("Starting subscription creation")
	apiLogger.WithField("subscription_type", "monthly").Info("Validating subscription parameters")
	apiLogger.Info("Subscription created successfully")
}

// Helper functions for common logging patterns

// LogRequestStart logs the beginning of a request with standard fields
func LogRequestStart(logger *StructuredLogger, method, path, userID, workspaceID string) *StructuredLogger {
	return logger.WithUserID(userID).
		WithWorkspaceID(workspaceID).
		WithFields(map[string]interface{}{
			"http_method": method,
			"http_path":   path,
		})
}

// LogRequestEnd logs the completion of a request with timing
func LogRequestEnd(logger *StructuredLogger, statusCode int, duration time.Duration) {
	logger.WithFields(map[string]interface{}{
		"http_status":   statusCode,
		"response_time": duration,
	}).WithDuration(duration).Info("Request completed")
}

// LogDatabaseOperation logs database operations with standard fields
func LogDatabaseOperation(logger *StructuredLogger, operation, table string, recordsAffected int) {
	logger.WithOperation(operation).
		WithFields(map[string]interface{}{
			"table":            table,
			"records_affected": recordsAffected,
		}).Debug("Database operation completed")
}

// LogExternalAPICall logs calls to external APIs
func LogExternalAPICall(logger *StructuredLogger, service, endpoint string, duration time.Duration, success bool) {
	logger.WithFields(map[string]interface{}{
		"external_service": service,
		"endpoint":         endpoint,
		"success":          success,
	}).WithDuration(duration).Info("External API call completed")
}

// LogBusinessEvent logs important business events
func LogBusinessEvent(logger *StructuredLogger, eventType string, entityID string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"event_type": eventType,
		"entity_id":  entityID,
	}
	
	// Merge metadata
	for k, v := range metadata {
		fields[k] = v
	}
	
	logger.WithFields(fields).Info("Business event occurred")
}

// Performance logging helpers

// LogSlowOperation logs operations that exceed a threshold
func LogSlowOperation(logger *StructuredLogger, operation string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		logger.WithOperation(operation).
			WithDuration(duration).
			WithField("threshold", threshold).
			Warn("Slow operation detected")
	}
}

// LogResourceUsage logs resource usage metrics
func LogResourceUsage(logger *StructuredLogger, cpu float64, memory int64, goroutines int) {
	logger.WithFields(map[string]interface{}{
		"cpu_percent":      cpu,
		"memory_bytes":     memory,
		"goroutine_count":  goroutines,
	}).Debug("Resource usage metrics")
}

// Security logging helpers

// LogSecurityEvent logs security-related events
func LogSecurityEvent(logger *StructuredLogger, eventType, userID, ipAddress string, success bool, reason string) {
	logger.WithFields(map[string]interface{}{
		"security_event": eventType,
		"user_id":        userID,
		"ip_address":     ipAddress,
		"success":        success,
		"reason":         reason,
	}).Info("Security event occurred")
}

// LogSuspiciousActivity logs potentially suspicious activities
func LogSuspiciousActivity(logger *StructuredLogger, activity, userID, details string) {
	logger.WithFields(map[string]interface{}{
		"suspicious_activity": activity,
		"user_id":             userID,
		"details":             details,
		"requires_review":     true,
	}).Warn("Suspicious activity detected")
}

// Example usage in a typical handler function
func ExampleHandlerLogging(ctx context.Context, userID, workspaceID string) {
	// Create component logger
	logger := NewStructuredLogger(ComponentAPI)
	
	// Add request context
	reqLogger := logger.WithUserID(userID).
		WithWorkspaceID(workspaceID).
		WithCorrelationID("req-123")
	
	// Log request start
	reqLogger.Info("Processing subscription creation request")
	
	// Track the operation
	timer := reqLogger.NewTimer("create_subscription")
	
	// Simulate work with database logging
	dbLogger := NewStructuredLogger(ComponentDB).
		WithUserID(userID).
		WithWorkspaceID(workspaceID)
	
	dbLogger.Debug("Starting database transaction")
	
	// ... do work ...
	
	// Complete timing
	timer.StopWithResult(true, nil)
	
	// Log final result
	reqLogger.WithField("subscription_id", "sub-123").
		Info("Subscription created successfully")
}