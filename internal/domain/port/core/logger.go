package core

// LogLevel represents logging severity levels
type LogLevel int

const (
	// LogLevelDebug for detailed debug information
	LogLevelDebug LogLevel = iota
	// LogLevelInfo for general operational information
	LogLevelInfo
	// LogLevelWarn for warnings
	LogLevelWarn
	// LogLevelError for errors information
	LogLevelError
)

// Logger defines logging operations
type Logger interface {
	// SetLevel sets the minimum log level to output
	SetLevel(level LogLevel)
	// GetLevel gets the current log level
	GetLevel() LogLevel
	// Debug logs debug messages
	Debug(message string, fields map[string]any)
	// Info logs informational messages
	Info(message string, fields map[string]any)
	// Warn logs warning messages
	Warn(message string, fields map[string]any)
	// Error logs errors messages
	Error(message string, fields map[string]any)
	// Flush ensures all buffered logs are written to their destination
	Flush() error
}
