# Structured Logging Guide

This guide explains how to use the enhanced structured logging system in the Cyphera API.

## Overview

The logging system provides:
- **Structured logging** with consistent field formats
- **Component-based organization** for easy filtering
- **Performance monitoring** with built-in timing
- **Log filtering and aggregation** for analysis
- **Context propagation** for request tracing

## Quick Start

### Basic Usage

```go
import "github.com/cyphera/cyphera-api/libs/go/logger"

// Create a component-specific logger
apiLogger := logger.NewStructuredLogger(logger.ComponentAPI)

// Log with context
apiLogger.WithUserID("user-123").
    WithWorkspaceID("workspace-456").
    Info("Processing subscription request")
```

### Configuration

Set the log level via environment variable:
```bash
export LOG_LEVEL=debug  # Options: debug, info, warn, error, fatal
```

## Components

The system organizes logs by component for easy filtering:

- `ComponentAPI` - HTTP API handlers
- `ComponentDB` - Database operations  
- `ComponentAuth` - Authentication/authorization
- `ComponentSubscription` - Subscription processing
- `ComponentPayment` - Payment operations
- `ComponentCircle` - Circle API integration
- `ComponentWebhook` - Webhook processing
- `ComponentMiddleware` - HTTP middleware
- `ComponentServer` - Server lifecycle
- `ComponentWorker` - Background workers

## Structured Context

### Standard Fields

All logs can include these standard context fields:

```go
logger := logger.NewStructuredLogger(logger.ComponentAPI).
    WithUserID("user-123").               // User performing the action
    WithWorkspaceID("workspace-456").     // Workspace context
    WithCorrelationID("req-789").         // Request correlation ID
    WithOperation("create_subscription")  // Operation being performed
```

### Custom Fields

Add custom fields for specific context:

```go
logger.WithFields(map[string]interface{}{
    "subscription_id": "sub-123",
    "payment_method": "card",
    "amount": 1000,
}).Info("Payment processed")
```

## Specialized Logging Methods

### HTTP Request Logging

```go
middlewareLogger := logger.NewStructuredLogger(logger.ComponentMiddleware)

middlewareLogger.LogHTTPRequest(
    "POST",                    // Method
    "/api/subscriptions",      // Path
    201,                       // Status code
    time.Millisecond*150,      // Duration
)
```

### Database Query Logging

```go
dbLogger := logger.NewStructuredLogger(logger.ComponentDB)

dbLogger.LogDatabaseQuery(
    "SELECT * FROM subscriptions WHERE status = $1",  // Query
    time.Millisecond*500,                              // Duration
    25,                                                // Rows affected
)
```

### Performance Timing

```go
// Automatic timing with LogOperation
err := logger.LogOperation("process_subscriptions", func() error {
    // Do work here
    return nil
})

// Manual timing with Timer
timer := logger.NewTimer("complex_operation")
// Do work...
timer.StopWithResult(success, err)
```

### Business Event Logging

```go
subLogger := logger.NewStructuredLogger(logger.ComponentSubscription)

// Subscription events
subLogger.LogSubscriptionEvent(
    "sub-123",
    "payment_processed",
    map[string]interface{}{
        "amount": 1000,
        "currency": "USD",
        "payment_method": "card",
    },
)

// Payment events
subLogger.LogPaymentEvent("pay-456", "completed", 1000, "USD")

// Webhook events
webhookLogger.LogWebhookEvent("stripe", "payment.completed", true, time.Millisecond*50)

// Authentication events
authLogger.LogAuthEvent("login", "user-123", true, "valid_credentials")
```

## Log Filtering

### Runtime Filtering

Create filtered loggers that only emit logs matching criteria:

```go
// Only log errors from subscription component
filter := logger.LogFilter{
    MinLevel:   logger.ErrorLevel,
    Components: []logger.LogComponent{logger.ComponentSubscription},
}

filteredLogger := logger.NewFilteredLogger(logger.ComponentSubscription, filter)
```

### Filter Options

```go
filter := logger.LogFilter{
    MinLevel:          logger.InfoLevel,                    // Minimum log level
    Components:        []logger.LogComponent{...},          // Include only these components
    ExcludeComponents: []logger.LogComponent{...},          // Exclude these components
    Operations:        []string{"create", "update"},       // Include operations containing these strings
    ExcludeOperations: []string{"health", "ping"},         // Exclude operations containing these strings
    UserIDs:           []string{"user-123"},               // Only logs for these users
    WorkspaceIDs:      []string{"workspace-456"},          // Only logs for these workspaces
    TimeRange: logger.TimeRange{                           // Time-based filtering
        Start: time.Now().Add(-time.Hour),
        End:   time.Now(),
    },
}
```

### Preset Filters

Use common filter presets:

```go
// Get preset filter
filter, exists := logger.GetFilterPreset("errors_only")
if exists {
    filteredLogger := logger.NewFilteredLogger(logger.ComponentAPI, filter)
}
```

Available presets:
- `errors_only` - Only error-level logs
- `subscription_debug` - Debug logs for subscription/payment components
- `api_requests` - API and middleware logs
- `database_operations` - Database operation logs
- `webhook_processing` - Webhook-related logs
- `performance` - Performance-related logs

## Log Aggregation and Analysis

### Create Aggregator

```go
aggregator := logger.NewLogAggregator()

// Add log entries (in practice, these would come from log files/streams)
aggregator.AddEntry(logger.LogEntry{
    Timestamp: time.Now(),
    Level:     "error",
    Message:   "Payment failed",
    Component: "payment",
    // ... other fields
})
```

### Error Analysis

```go
timeRange := logger.TimeRange{
    Start: time.Now().Add(-24 * time.Hour),
    End:   time.Now(),
}

errorSummary := aggregator.GetErrorSummary(timeRange)

fmt.Printf("Total errors: %d\n", errorSummary.TotalErrors)
fmt.Printf("Errors by component: %+v\n", errorSummary.ErrorsByComponent)

// Get top errors
for _, error := range errorSummary.TopErrors {
    fmt.Printf("Error: %s (count: %d)\n", error.Message, error.Count)
}
```

### Performance Analysis

```go
perfSummary := aggregator.GetPerformanceSummary(timeRange)

fmt.Printf("Average latency: %v\n", perfSummary.AverageLatency)
fmt.Printf("95th percentile: %v\n", perfSummary.P95Latency)

// Get slowest operations
for _, op := range perfSummary.SlowestOperations {
    fmt.Printf("Operation: %s, Avg Duration: %v\n", op.Operation, op.AvgDuration)
}
```

### Activity Analysis

```go
activitySummary := aggregator.GetActivitySummary(timeRange)

fmt.Printf("Total requests: %d\n", activitySummary.TotalRequests)
fmt.Printf("Requests by endpoint: %+v\n", activitySummary.RequestsByEndpoint)
```

### Custom Queries

```go
// Find slow queries
slowQueries := aggregator.GetSlowQueries(time.Second*5, timeRange)

// Get errors for specific user
userErrors := aggregator.GetErrorsForUser("user-123", timeRange)

// Get all logs for a correlation ID
correlatedLogs := aggregator.GetCorrelatedLogs("req-789")

// Search logs with regex
results, err := aggregator.SearchLogs("payment.*failed", timeRange, 100)
```

## Integration Patterns

### HTTP Middleware

```go
func LoggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        correlationID := c.GetHeader("X-Correlation-ID")
        
        logger := logger.NewStructuredLogger(logger.ComponentMiddleware).
            WithCorrelationID(correlationID).
            WithFields(map[string]interface{}{
                "method": c.Request.Method,
                "path":   c.Request.URL.Path,
            })
        
        logger.Info("Request started")
        
        c.Next()
        
        duration := time.Since(start)
        logger.LogHTTPRequest(c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
    }
}
```

### Handler Pattern

```go
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
    // Extract context
    userID := c.GetString("user_id")
    workspaceID := c.GetString("workspace_id")
    correlationID := c.GetString("correlation_id")
    
    // Create logger with context
    logger := logger.NewStructuredLogger(logger.ComponentSubscription).
        WithUserID(userID).
        WithWorkspaceID(workspaceID).
        WithCorrelationID(correlationID).
        WithOperation("create_subscription")
    
    logger.Info("Creating subscription")
    
    // Use logger throughout the handler
    logger.WithField("subscription_type", "monthly").Debug("Validating parameters")
    
    // Track performance
    timer := logger.NewTimer("database_operations")
    // ... do database work ...
    timer.Stop()
    
    logger.WithField("subscription_id", "sub-123").Info("Subscription created successfully")
}
```

### Background Worker Pattern

```go
func ProcessDueSubscriptions() {
    logger := logger.NewStructuredLogger(logger.ComponentWorker).
        WithOperation("process_due_subscriptions")
    
    logger.Info("Starting subscription processing")
    
    err := logger.LogOperation("batch_processing", func() error {
        // Process subscriptions
        return nil
    })
    
    if err != nil {
        logger.Error("Batch processing failed", err)
    } else {
        logger.Info("Batch processing completed successfully")
    }
}
```

## Best Practices

### 1. Use Component-Specific Loggers

```go
// Good - component-specific logger
apiLogger := logger.NewStructuredLogger(logger.ComponentAPI)

// Avoid - generic logging
logger.Info("Something happened")
```

### 2. Include Context Early

```go
// Good - set context once, use throughout
requestLogger := logger.NewStructuredLogger(logger.ComponentAPI).
    WithUserID(userID).
    WithWorkspaceID(workspaceID)

requestLogger.Info("Processing request")
requestLogger.Debug("Validating parameters")

// Avoid - repeating context
logger.WithUserID(userID).Info("Processing request")
logger.WithUserID(userID).Debug("Validating parameters")  // Repeated context
```

### 3. Use Appropriate Log Levels

```go
// Debug - Detailed information for debugging
logger.Debug("Validating subscription parameters")

// Info - General operational information
logger.Info("Subscription created successfully")

// Warn - Something unexpected but not an error
logger.Warn("Retrying failed payment")

// Error - Error conditions that need attention
logger.Error("Payment processing failed", err)

// Fatal - Critical errors that cause shutdown
logger.Fatal("Database connection failed", err)
```

### 4. Include Relevant Fields

```go
// Good - relevant context
logger.WithFields(map[string]interface{}{
    "subscription_id": "sub-123",
    "amount": 1000,
    "currency": "USD",
}).Info("Payment processed")

// Avoid - too much noise
logger.WithFields(map[string]interface{}{
    "random_field": "unnecessary_value",
    "internal_detail": "not_helpful_for_operations",
}).Info("Payment processed")
```

### 5. Use Structured Events for Business Logic

```go
// Good - structured business event
logger.LogSubscriptionEvent("sub-123", "payment_failed", map[string]interface{}{
    "reason": "insufficient_funds",
    "retry_count": 3,
})

// Avoid - unstructured message
logger.Error("Payment failed for sub-123 due to insufficient funds (retry 3)", nil)
```

## Configuration Examples

### Development Environment

```bash
export LOG_LEVEL=debug
export STAGE=dev
```

### Production Environment

```bash
export LOG_LEVEL=info
export STAGE=prod
```

### Custom Configuration

```go
config := logger.LoggerConfig{
    Level:       "debug",
    Stage:       "local",
    EnableJSON:  true,   // Force JSON format
    EnableColor: false,  // Disable colors
}

logger.InitLoggerWithConfig(config)
```

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Error Rates**: Track error logs by component
2. **Performance**: Monitor operation durations and slow queries
3. **Activity**: Track request volumes and patterns
4. **Security**: Monitor authentication failures and suspicious activity

### Sample Queries for Log Analysis Tools

For structured JSON logs, use these patterns:

```json
// Find all errors in the last hour
{
  "query": {
    "bool": {
      "must": [
        {"term": {"level": "error"}},
        {"range": {"timestamp": {"gte": "now-1h"}}}
      ]
    }
  }
}

// Find slow operations (>5 seconds)
{
  "query": {
    "bool": {
      "must": [
        {"range": {"duration": {"gte": 5000000000}}}
      ]
    }
  }
}

// Find subscription-related errors
{
  "query": {
    "bool": {
      "must": [
        {"term": {"level": "error"}},
        {"term": {"component": "subscription"}}
      ]
    }
  }
}
```

## Migration from Legacy Logging

### Before (Legacy)

```go
import "log"

log.Printf("User %s created subscription %s", userID, subscriptionID)
```

### After (Structured)

```go
logger := logger.NewStructuredLogger(logger.ComponentSubscription).
    WithUserID(userID).
    WithField("subscription_id", subscriptionID)

logger.Info("Subscription created")
```

### Gradual Migration Strategy

1. **Phase 1**: Initialize structured logger alongside existing logger
2. **Phase 2**: Replace critical paths (errors, business events)
3. **Phase 3**: Replace remaining log statements
4. **Phase 4**: Remove legacy logging imports

This approach ensures no logs are lost during migration.