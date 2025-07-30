package logger

import (
	"os"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is the global logger instance
	Log *zap.Logger
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level       string `json:"level"`
	Stage       string `json:"stage"`
	EnableJSON  bool   `json:"enable_json"`
	EnableColor bool   `json:"enable_color"`
}

// InitLogger initializes the logger with the appropriate configuration
// based on the provided stage.
func InitLogger(stage string) {
	config := LoggerConfig{
		Level:       getEnvWithDefault("LOG_LEVEL", "info"),
		Stage:       stage,
		EnableJSON:  stage == constants.ProdEnvironment,
		EnableColor: stage != constants.ProdEnvironment,
	}

	InitLoggerWithConfig(config)
}

// InitLoggerWithConfig initializes the logger with custom configuration
func InitLoggerWithConfig(config LoggerConfig) {
	var zapConfig zap.Config

	// Determine log level
	level := zapcore.InfoLevel
	switch strings.ToLower(config.Level) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case constants.ErrorLevel:
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	}

	if config.Stage == constants.ProdEnvironment || config.EnableJSON {
		// Production config - JSON structured logging
		zapConfig = zap.NewProductionConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(level)
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.MessageKey = "message"
		zapConfig.EncoderConfig.LevelKey = "level"
		zapConfig.EncoderConfig.CallerKey = "caller"
		zapConfig.EncoderConfig.StacktraceKey = "stacktrace"

		// Add custom fields for structured logging
		zapConfig.InitialFields = map[string]interface{}{
			"service": "cyphera-api",
			"stage":   config.Stage,
		}
	} else {
		// Development config - human-readable console logging
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(level)

		if config.EnableColor {
			zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}

		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// Enable caller information for debugging
	zapConfig.DisableCaller = false
	// Enable stacktraces for development and debug levels
	zapConfig.DisableStacktrace = config.Stage == constants.ProdEnvironment && level > zapcore.DebugLevel

	// Build the logger
	logger, err := zapConfig.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	// Set global logger
	Log = logger
}

// getEnvWithDefault returns environment variable value or default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Info logs a message at InfoLevel
func Info(msg string, fields ...zapcore.Field) {
	Log.Info(msg, fields...)
}

// Error logs a message at ErrorLevel
func Error(msg string, fields ...zapcore.Field) {
	Log.Error(msg, fields...)
}

// Debug logs a message at DebugLevel
func Debug(msg string, fields ...zapcore.Field) {
	Log.Debug(msg, fields...)
}

// Warn logs a message at WarnLevel
func Warn(msg string, fields ...zapcore.Field) {
	Log.Warn(msg, fields...)
}

// Fatal logs a message at FatalLevel
// and then calls os.Exit(1)
func Fatal(msg string, fields ...zapcore.Field) {
	Log.Fatal(msg, fields...)
}

// With creates a child logger and adds structured context to it
func With(fields ...zapcore.Field) *zap.Logger {
	return Log.With(fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return Log.Sync()
}
