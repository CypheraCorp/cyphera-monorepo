package logger

import (
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogFilter provides filtering capabilities for logs
type LogFilter struct {
	MinLevel         LogLevel
	Components       []LogComponent
	ExcludeComponents []LogComponent
	Operations       []string
	ExcludeOperations []string
	UserIDs          []string
	WorkspaceIDs     []string
	TimeRange        TimeRange
}

// TimeRange represents a time range for log filtering
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// LogEntry represents a structured log entry for filtering/querying
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Component     string                 `json:"component,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	WorkspaceID   string                 `json:"workspace_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Operation     string                 `json:"operation,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

// FilteredLogger wraps the structured logger with filtering capabilities
type FilteredLogger struct {
	*StructuredLogger
	filter LogFilter
}

// NewFilteredLogger creates a new filtered logger
func NewFilteredLogger(component LogComponent, filter LogFilter) *FilteredLogger {
	return &FilteredLogger{
		StructuredLogger: NewStructuredLogger(component),
		filter:          filter,
	}
}

// shouldLog determines if a log entry should be written based on the filter
func (fl *FilteredLogger) shouldLog(level LogLevel, ctx LogContext) bool {
	// Check minimum level
	if !fl.isLevelEnabled(level) {
		return false
	}
	
	// Check component filters
	if !fl.isComponentAllowed(ctx.Component) {
		return false
	}
	
	// Check operation filters
	if !fl.isOperationAllowed(ctx.Operation) {
		return false
	}
	
	// Check user ID filters
	if !fl.isUserAllowed(ctx.UserID) {
		return false
	}
	
	// Check workspace ID filters
	if !fl.isWorkspaceAllowed(ctx.WorkspaceID) {
		return false
	}
	
	return true
}

// isLevelEnabled checks if the log level meets the minimum requirement
func (fl *FilteredLogger) isLevelEnabled(level LogLevel) bool {
	levelOrder := map[LogLevel]int{
		DebugLevel: 0,
		InfoLevel:  1,
		WarnLevel:  2,
		ErrorLevel: 3,
		FatalLevel: 4,
	}
	
	currentLevel, exists := levelOrder[level]
	if !exists {
		return true // Allow unknown levels
	}
	
	minLevel, exists := levelOrder[fl.filter.MinLevel]
	if !exists {
		return true // No minimum level set
	}
	
	return currentLevel >= minLevel
}

// isComponentAllowed checks if the component is allowed
func (fl *FilteredLogger) isComponentAllowed(component LogComponent) bool {
	// Check exclusions first
	for _, excluded := range fl.filter.ExcludeComponents {
		if component == excluded {
			return false
		}
	}
	
	// If no inclusions specified, allow all (except excluded)
	if len(fl.filter.Components) == 0 {
		return true
	}
	
	// Check inclusions
	for _, allowed := range fl.filter.Components {
		if component == allowed {
			return true
		}
	}
	
	return false
}

// isOperationAllowed checks if the operation is allowed
func (fl *FilteredLogger) isOperationAllowed(operation string) bool {
	if operation == "" && len(fl.filter.Operations) == 0 && len(fl.filter.ExcludeOperations) == 0 {
		return true
	}
	
	// Check exclusions first
	for _, excluded := range fl.filter.ExcludeOperations {
		if strings.Contains(operation, excluded) {
			return false
		}
	}
	
	// If no inclusions specified, allow all (except excluded)
	if len(fl.filter.Operations) == 0 {
		return true
	}
	
	// Check inclusions
	for _, allowed := range fl.filter.Operations {
		if strings.Contains(operation, allowed) {
			return true
		}
	}
	
	return false
}

// isUserAllowed checks if the user ID is allowed
func (fl *FilteredLogger) isUserAllowed(userID string) bool {
	if len(fl.filter.UserIDs) == 0 {
		return true
	}
	
	for _, allowed := range fl.filter.UserIDs {
		if userID == allowed {
			return true
		}
	}
	
	return false
}

// isWorkspaceAllowed checks if the workspace ID is allowed
func (fl *FilteredLogger) isWorkspaceAllowed(workspaceID string) bool {
	if len(fl.filter.WorkspaceIDs) == 0 {
		return true
	}
	
	for _, allowed := range fl.filter.WorkspaceIDs {
		if workspaceID == allowed {
			return true
		}
	}
	
	return false
}

// Override logging methods to apply filtering

// Debug logs a debug message if it passes the filter
func (fl *FilteredLogger) Debug(msg string) {
	if fl.shouldLog(DebugLevel, fl.context) {
		fl.StructuredLogger.Debug(msg)
	}
}

// Info logs an info message if it passes the filter
func (fl *FilteredLogger) Info(msg string) {
	if fl.shouldLog(InfoLevel, fl.context) {
		fl.StructuredLogger.Info(msg)
	}
}

// Warn logs a warning message if it passes the filter
func (fl *FilteredLogger) Warn(msg string) {
	if fl.shouldLog(WarnLevel, fl.context) {
		fl.StructuredLogger.Warn(msg)
	}
}

// Error logs an error message if it passes the filter
func (fl *FilteredLogger) Error(msg string, err error) {
	if fl.shouldLog(ErrorLevel, fl.context) {
		fl.StructuredLogger.Error(msg, err)
	}
}

// Fatal logs a fatal message if it passes the filter
func (fl *FilteredLogger) Fatal(msg string, err error) {
	if fl.shouldLog(FatalLevel, fl.context) {
		fl.StructuredLogger.Fatal(msg, err)
	}
}

// LogLevel Configuration

// SetGlobalLogLevel sets the global minimum log level
func SetGlobalLogLevel(level LogLevel) {
	var zapLevel zapcore.Level
	
	switch level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case FatalLevel:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	
	// Reconfigure the global logger with new level
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	
	logger, err := config.Build()
	if err != nil {
		// Fallback to existing logger if rebuild fails
		return
	}
	
	Log = logger
}

// GetLogLevel returns the current log level as a string
func GetLogLevel() string {
	return Log.Level().String()
}

// LogLevelFromString converts a string to LogLevel
func LogLevelFromString(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// Dynamic Filter Configuration

// FilterPresets contains common filter configurations
var FilterPresets = map[string]LogFilter{
	"errors_only": {
		MinLevel: ErrorLevel,
	},
	"subscription_debug": {
		MinLevel:   DebugLevel,
		Components: []LogComponent{ComponentSubscription, ComponentPayment},
	},
	"api_requests": {
		MinLevel:   InfoLevel,
		Components: []LogComponent{ComponentAPI, ComponentMiddleware},
		Operations: []string{"http", "api"},
	},
	"database_operations": {
		MinLevel:   DebugLevel,
		Components: []LogComponent{ComponentDB},
		Operations: []string{"query", "transaction"},
	},
	"webhook_processing": {
		MinLevel:   InfoLevel,
		Components: []LogComponent{ComponentWebhook},
	},
	"performance": {
		MinLevel:   InfoLevel,
		Operations: []string{"slow", "performance", "duration"},
	},
}

// GetFilterPreset returns a predefined filter configuration
func GetFilterPreset(name string) (LogFilter, bool) {
	filter, exists := FilterPresets[name]
	return filter, exists
}

// CombineFilters combines multiple filters using AND logic
func CombineFilters(filters ...LogFilter) LogFilter {
	if len(filters) == 0 {
		return LogFilter{}
	}
	
	if len(filters) == 1 {
		return filters[0]
	}
	
	combined := filters[0]
	
	for _, filter := range filters[1:] {
		// Use the highest minimum level
		if filter.MinLevel != "" {
			if combined.MinLevel == "" {
				combined.MinLevel = filter.MinLevel
			} else {
				levelOrder := map[LogLevel]int{
					DebugLevel: 0,
					InfoLevel:  1,
					WarnLevel:  2,
					ErrorLevel: 3,
					FatalLevel: 4,
				}
				
				if levelOrder[filter.MinLevel] > levelOrder[combined.MinLevel] {
					combined.MinLevel = filter.MinLevel
				}
			}
		}
		
		// Combine component filters (intersection)
		if len(filter.Components) > 0 {
			if len(combined.Components) == 0 {
				combined.Components = filter.Components
			} else {
				// Find intersection
				intersection := make([]LogComponent, 0)
				for _, c1 := range combined.Components {
					for _, c2 := range filter.Components {
						if c1 == c2 {
							intersection = append(intersection, c1)
							break
						}
					}
				}
				combined.Components = intersection
			}
		}
		
		// Combine exclusions (union)
		combined.ExcludeComponents = append(combined.ExcludeComponents, filter.ExcludeComponents...)
		combined.ExcludeOperations = append(combined.ExcludeOperations, filter.ExcludeOperations...)
	}
	
	return combined
}