package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is the global logger instance
	Log *zap.Logger
)

// InitLogger initializes the logger with the appropriate configuration
// based on the environment (development or production)
func InitLogger() {
	// Get environment from GIN_MODE, default to "development"
	env := os.Getenv("GIN_MODE")
	if env == "" {
		env = "development"
	}

	var config zap.Config
	if env == "release" {
		// Production config
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		// Development config
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	// Set global logger
	Log = logger
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
