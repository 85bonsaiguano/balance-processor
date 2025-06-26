package logger

import (
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// NoopLogger implements the Logger interface but doesn't do anything
// Useful for testing or when logging is disabled
type NoopLogger struct {
	level core.LogLevel
}

// NewNoopLogger creates a new no-op logger
func NewNoopLogger() core.Logger {
	return &NoopLogger{
		level: core.LogLevelInfo,
	}
}

// SetLevel sets the minimum log level to output
func (l *NoopLogger) SetLevel(level core.LogLevel) {
	l.level = level
}

// GetLevel gets the current log level
func (l *NoopLogger) GetLevel() core.LogLevel {
	return l.level
}

// Debug logs debug messages
func (l *NoopLogger) Debug(message string, fields map[string]any) {
	// Do nothing
}

// Info logs informational messages
func (l *NoopLogger) Info(message string, fields map[string]any) {
	// Do nothing
}

// Warn logs warning messages
func (l *NoopLogger) Warn(message string, fields map[string]any) {
	// Do nothing
}

// Error logs errors messages
func (l *NoopLogger) Error(message string, fields map[string]any) {
	// Do nothing
}

// Flush ensures all buffered logs are written to their destination
func (l *NoopLogger) Flush() error {
	// No-op implementation doesn't need to flush anything
	return nil
}
