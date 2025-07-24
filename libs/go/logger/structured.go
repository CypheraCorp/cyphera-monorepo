package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents different logging levels
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// LogComponent represents different system components for filtering
type LogComponent string

const (
	ComponentAPI          LogComponent = "api"
	ComponentDB           LogComponent = "database"
	ComponentAuth         LogComponent = "auth"
	ComponentSubscription LogComponent = "subscription"
	ComponentPayment      LogComponent = "payment"
	ComponentCircle       LogComponent = "circle"
	ComponentWebhook      LogComponent = "webhook"
	ComponentMiddleware   LogComponent = "middleware"
	ComponentServer       LogComponent = "server"
	ComponentWorker       LogComponent = "worker"
)

// LogContext holds structured context information for logs
type LogContext struct {
	UserID        string
	WorkspaceID   string
	CorrelationID string
	RequestID     string
	Component     LogComponent
	Operation     string
	Duration      time.Duration
	Fields        map[string]interface{}
}

// StructuredLogger provides enhanced logging with structured context
type StructuredLogger struct {
	logger    *zap.Logger
	component LogComponent
	context   LogContext
}

// NewStructuredLogger creates a new structured logger for a specific component
func NewStructuredLogger(component LogComponent) *StructuredLogger {
	return &StructuredLogger{
		logger:    Log,
		component: component,
		context:   LogContext{Component: component, Fields: make(map[string]interface{})},
	}
}

// WithContext adds context information to the logger
func (sl *StructuredLogger) WithContext(ctx LogContext) *StructuredLogger {
	newLogger := &StructuredLogger{
		logger:    sl.logger,
		component: sl.component,
		context:   ctx,
	}
	
	// Ensure component is set
	newLogger.context.Component = sl.component
	
	// Initialize fields map if nil
	if newLogger.context.Fields == nil {
		newLogger.context.Fields = make(map[string]interface{})
	}
	
	return newLogger
}

// WithField adds a field to the log context
func (sl *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	newLogger := sl.clone()
	newLogger.context.Fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the log context
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := sl.clone()
	for k, v := range fields {
		newLogger.context.Fields[k] = v
	}
	return newLogger
}

// WithUserID adds user ID to the log context
func (sl *StructuredLogger) WithUserID(userID string) *StructuredLogger {
	newLogger := sl.clone()
	newLogger.context.UserID = userID
	return newLogger
}

// WithWorkspaceID adds workspace ID to the log context
func (sl *StructuredLogger) WithWorkspaceID(workspaceID string) *StructuredLogger {
	newLogger := sl.clone()
	newLogger.context.WorkspaceID = workspaceID
	return newLogger
}

// WithCorrelationID adds correlation ID to the log context
func (sl *StructuredLogger) WithCorrelationID(correlationID string) *StructuredLogger {
	newLogger := sl.clone()
	newLogger.context.CorrelationID = correlationID
	return newLogger
}

// WithOperation adds operation name to the log context
func (sl *StructuredLogger) WithOperation(operation string) *StructuredLogger {
	newLogger := sl.clone()
	newLogger.context.Operation = operation
	return newLogger
}

// WithDuration adds duration to the log context
func (sl *StructuredLogger) WithDuration(duration time.Duration) *StructuredLogger {
	newLogger := sl.clone()
	newLogger.context.Duration = duration
	return newLogger
}

// clone creates a copy of the structured logger
func (sl *StructuredLogger) clone() *StructuredLogger {
	newFields := make(map[string]interface{})
	for k, v := range sl.context.Fields {
		newFields[k] = v
	}
	
	return &StructuredLogger{
		logger:    sl.logger,
		component: sl.component,
		context: LogContext{
			UserID:        sl.context.UserID,
			WorkspaceID:   sl.context.WorkspaceID,
			CorrelationID: sl.context.CorrelationID,
			RequestID:     sl.context.RequestID,
			Component:     sl.context.Component,
			Operation:     sl.context.Operation,
			Duration:      sl.context.Duration,
			Fields:        newFields,
		},
	}
}

// buildFields creates zap fields from the log context
func (sl *StructuredLogger) buildFields() []zapcore.Field {
	fields := make([]zapcore.Field, 0)
	
	// Add standard context fields
	if sl.context.Component != "" {
		fields = append(fields, zap.String("component", string(sl.context.Component)))
	}
	if sl.context.UserID != "" {
		fields = append(fields, zap.String("user_id", sl.context.UserID))
	}
	if sl.context.WorkspaceID != "" {
		fields = append(fields, zap.String("workspace_id", sl.context.WorkspaceID))
	}
	if sl.context.CorrelationID != "" {
		fields = append(fields, zap.String("correlation_id", sl.context.CorrelationID))
	}
	if sl.context.RequestID != "" {
		fields = append(fields, zap.String("request_id", sl.context.RequestID))
	}
	if sl.context.Operation != "" {
		fields = append(fields, zap.String("operation", sl.context.Operation))
	}
	if sl.context.Duration > 0 {
		fields = append(fields, zap.Duration("duration", sl.context.Duration))
	}
	
	// Add custom fields
	for key, value := range sl.context.Fields {
		fields = append(fields, zap.Any(key, value))
	}
	
	return fields
}

// Debug logs a debug message with structured context
func (sl *StructuredLogger) Debug(msg string) {
	sl.logger.Debug(msg, sl.buildFields()...)
}

// Info logs an info message with structured context
func (sl *StructuredLogger) Info(msg string) {
	sl.logger.Info(msg, sl.buildFields()...)
}

// Warn logs a warning message with structured context
func (sl *StructuredLogger) Warn(msg string) {
	sl.logger.Warn(msg, sl.buildFields()...)
}

// Error logs an error message with structured context
func (sl *StructuredLogger) Error(msg string, err error) {
	fields := sl.buildFields()
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	sl.logger.Error(msg, fields...)
}

// Fatal logs a fatal message with structured context and exits
func (sl *StructuredLogger) Fatal(msg string, err error) {
	fields := sl.buildFields()
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	sl.logger.Fatal(msg, fields...)
}

// LogOperation logs the start and end of an operation with timing
func (sl *StructuredLogger) LogOperation(operation string, fn func() error) error {
	start := time.Now()
	opLogger := sl.WithOperation(operation)
	
	opLogger.Info("Operation started")
	
	err := fn()
	duration := time.Since(start)
	
	finalLogger := opLogger.WithDuration(duration)
	
	if err != nil {
		finalLogger.Error("Operation failed", err)
	} else {
		finalLogger.Info("Operation completed")
	}
	
	return err
}

// LogHTTPRequest logs HTTP request details
func (sl *StructuredLogger) LogHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	sl.WithFields(map[string]interface{}{
		"http_method":     method,
		"http_path":       path,
		"http_status":     statusCode,
		"response_time":   duration,
	}).WithDuration(duration).Info("HTTP request processed")
}

// LogDatabaseQuery logs database query details
func (sl *StructuredLogger) LogDatabaseQuery(query string, duration time.Duration, rowsAffected int64) {
	sl.WithFields(map[string]interface{}{
		"query":          query,
		"rows_affected":  rowsAffected,
		"query_duration": duration,
	}).WithDuration(duration).Debug("Database query executed")
}

// LogSubscriptionEvent logs subscription-related events
func (sl *StructuredLogger) LogSubscriptionEvent(subscriptionID, eventType string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"subscription_id": subscriptionID,
		"event_type":      eventType,
	}
	
	// Add metadata fields
	for k, v := range metadata {
		fields[k] = v
	}
	
	sl.WithFields(fields).Info("Subscription event occurred")
}

// LogPaymentEvent logs payment-related events
func (sl *StructuredLogger) LogPaymentEvent(paymentID, status string, amount int64, currency string) {
	sl.WithFields(map[string]interface{}{
		"payment_id": paymentID,
		"status":     status,
		"amount":     amount,
		"currency":   currency,
	}).Info("Payment event occurred")
}

// LogWebhookEvent logs webhook-related events
func (sl *StructuredLogger) LogWebhookEvent(provider, eventType string, processed bool, processingTime time.Duration) {
	sl.WithFields(map[string]interface{}{
		"webhook_provider":     provider,
		"webhook_event_type":   eventType,
		"processed":            processed,
		"processing_duration":  processingTime,
	}).WithDuration(processingTime).Info("Webhook event processed")
}

// LogAuthEvent logs authentication-related events
func (sl *StructuredLogger) LogAuthEvent(action, userID string, success bool, reason string) {
	sl.WithFields(map[string]interface{}{
		"auth_action":  action,
		"user_id":      userID,
		"success":      success,
		"reason":       reason,
	}).Info("Authentication event occurred")
}

// NewLogContextFromGinContext extracts log context from a Gin context
func NewLogContextFromGinContext(c interface{}) LogContext {
	// This would be implemented to extract context from Gin request
	// For now, return empty context - could be enhanced with actual Gin integration
	return LogContext{
		Fields: make(map[string]interface{}),
	}
}

// Performance logging helpers

// Timer helps measure operation duration
type Timer struct {
	start  time.Time
	logger *StructuredLogger
	name   string
}

// NewTimer creates a new timer for measuring operation duration
func (sl *StructuredLogger) NewTimer(operationName string) *Timer {
	return &Timer{
		start:  time.Now(),
		logger: sl,
		name:   operationName,
	}
}

// Stop stops the timer and logs the duration
func (t *Timer) Stop() {
	duration := time.Since(t.start)
	t.logger.WithOperation(t.name).WithDuration(duration).Debug("Operation timing")
}

// StopWithResult stops the timer and logs the result
func (t *Timer) StopWithResult(success bool, err error) {
	duration := time.Since(t.start)
	logger := t.logger.WithOperation(t.name).WithDuration(duration).WithField("success", success)
	
	if success {
		logger.Info(fmt.Sprintf("%s completed successfully", t.name))
	} else {
		logger.Error(fmt.Sprintf("%s failed", t.name), err)
	}
}