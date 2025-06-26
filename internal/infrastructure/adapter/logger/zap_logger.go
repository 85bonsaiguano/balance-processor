package logger

import (
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger implements the Logger interface using Zap
type ZapLogger struct {
	logger *zap.Logger
	level  core.LogLevel
}

// NewZapLogger creates a new zap-based logger instance
func NewZapLogger(isProduction bool) core.Logger {
	// Configure zap logger
	var cfg zap.Config

	if isProduction {
		// In production, use a JSON encoder for structured logging
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		// In development, use a console encoder for easier reading
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set additional encoding options
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.MessageKey = "message"

	// Build the logger
	zapLogger, err := cfg.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return &ZapLogger{
		logger: zapLogger,
		level:  core.LogLevelInfo, // Default level
	}
}

// NewDefaultLogger creates a standard logger for the application
func NewDefaultLogger() core.Logger {
	return NewZapLogger(false) // Default to development mode
}

// SetLevel sets the minimum log level
func (l *ZapLogger) SetLevel(level core.LogLevel) {
	l.level = level

	// Convert to zap level
	var zapLevel zapcore.Level
	switch level {
	case core.LogLevelDebug:
		zapLevel = zap.DebugLevel
	case core.LogLevelInfo:
		zapLevel = zap.InfoLevel
	case core.LogLevelWarn:
		zapLevel = zap.WarnLevel
	case core.LogLevelError:
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	// Update the logger's level
	l.logger.Core().Enabled(zapLevel)
}

// GetLevel gets the current log level
func (l *ZapLogger) GetLevel() core.LogLevel {
	return l.level
}

// mapToZapFields converts a map of fields to zap fields
func mapToZapFields(fields map[string]any) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return zapFields
}

// Debug logs debug messages
func (l *ZapLogger) Debug(message string, fields map[string]any) {
	if l.level > core.LogLevelDebug {
		return
	}
	l.logger.Debug(message, mapToZapFields(fields)...)
}

// Info logs informational messages
func (l *ZapLogger) Info(message string, fields map[string]any) {
	if l.level > core.LogLevelInfo {
		return
	}
	l.logger.Info(message, mapToZapFields(fields)...)
}

// Warn logs warning messages
func (l *ZapLogger) Warn(message string, fields map[string]any) {
	if l.level > core.LogLevelWarn {
		return
	}
	l.logger.Warn(message, mapToZapFields(fields)...)
}

// Error logs error messages
func (l *ZapLogger) Error(message string, fields map[string]any) {
	if l.level > core.LogLevelError {
		return
	}
	l.logger.Error(message, mapToZapFields(fields)...)
}

// Flush ensures all buffered logs are written
func (l *ZapLogger) Flush() error {
	return l.logger.Sync()
}
