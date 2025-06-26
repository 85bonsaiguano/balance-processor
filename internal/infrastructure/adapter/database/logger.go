package database

import (
	"context"
	"strings"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"gorm.io/gorm/logger"
)

// DatabaseLogger is a custom GORM logger that uses our core logger
type DatabaseLogger struct {
	coreLogger    coreport.Logger
	logLevel      logger.LogLevel
	slowThreshold time.Duration
	timeProvider  coreport.TimeProvider
}

// NewDatabaseLogger creates a new database logger
func NewDatabaseLogger(coreLogger coreport.Logger, level string) logger.Interface {
	var logLevel logger.LogLevel
	switch strings.ToLower(level) {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Info
	}

	return &DatabaseLogger{
		coreLogger:    coreLogger,
		logLevel:      logLevel,
		slowThreshold: time.Second, // Default threshold for slow queries
	}
}

// NewDatabaseLoggerWithTimeProvider creates a new database logger with a time provider
func NewDatabaseLoggerWithTimeProvider(coreLogger coreport.Logger, timeProvider coreport.TimeProvider, level string) logger.Interface {
	var logLevel logger.LogLevel
	switch strings.ToLower(level) {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Info
	}

	return &DatabaseLogger{
		coreLogger:    coreLogger,
		logLevel:      logLevel,
		slowThreshold: time.Duration(200 * coreport.Millisecond),
		timeProvider:  timeProvider,
	}
}

// NewGormDatabaseLogger creates a new GORM database logger with the core logger
func NewGormDatabaseLogger(coreLogger coreport.Logger) logger.Interface {
	return &DatabaseLogger{
		coreLogger:    coreLogger,
		logLevel:      logger.Info,
		slowThreshold: time.Second, // Default threshold for slow queries
	}
}

// LogMode sets the log level for the logger
func (l *DatabaseLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// WithSlowThreshold returns a new logger with updated slow threshold
func (l *DatabaseLogger) WithSlowThreshold(threshold time.Duration) logger.Interface {
	newLogger := *l
	newLogger.slowThreshold = threshold
	return &newLogger
}

// Info logs info messages
func (l *DatabaseLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		l.coreLogger.Info(msg, map[string]any{"source": "database"})
	}
}

// Warn logs warn messages
func (l *DatabaseLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.coreLogger.Warn(msg, map[string]any{"source": "database"})
	}
}

// Error logs error messages
func (l *DatabaseLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		l.coreLogger.Error(msg, map[string]any{"source": "database"})
	}
}

// Trace logs SQL operations
func (l *DatabaseLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	// Get elapsed time
	var elapsed time.Duration
	var elapsedStr string

	if l.timeProvider != nil {
		elapsedDomain := l.timeProvider.Since(begin)
		elapsed = elapsedDomain.Std()
		elapsedStr = elapsed.String()
	} else {
		elapsed = time.Since(begin)
		elapsedStr = elapsed.String()
	}

	sql, rows := fc()

	// Create detailed log fields
	fields := map[string]any{
		"elapsed": elapsedStr,
		"rows":    rows,
		"sql":     sql,
		"source":  "database",
	}

	// Extract query type for better categorization
	queryType := extractQueryType(sql)
	if queryType != "" {
		fields["type"] = queryType
	}

	// Extract table name if possible
	tableName := extractTableName(sql)
	if tableName != "" {
		fields["table"] = tableName
	}

	// Add context info if available
	if traceID := extractTraceIDFromContext(ctx); traceID != "" {
		fields["trace_id"] = traceID
	}

	// Add error information if present
	if err != nil {
		fields["error"] = err.Error()
	}

	// Log based on error and elapsed time
	switch {
	case err != nil && l.logLevel >= logger.Error:
		l.coreLogger.Error("SQL Error", fields)
	case elapsed > l.slowThreshold && l.slowThreshold > 0:
		l.coreLogger.Warn("Slow SQL Query", fields)
	case l.logLevel >= logger.Info:
		l.coreLogger.Debug("SQL Query", fields) // Using debug level for regular SQL queries to reduce noise
	}
}

// extractQueryType determines the type of SQL query (SELECT, INSERT, UPDATE, DELETE)
func extractQueryType(sql string) string {
	sqlUpper := strings.ToUpper(strings.TrimSpace(sql))

	if strings.HasPrefix(sqlUpper, "SELECT") {
		return "SELECT"
	} else if strings.HasPrefix(sqlUpper, "INSERT") {
		return "INSERT"
	} else if strings.HasPrefix(sqlUpper, "UPDATE") {
		return "UPDATE"
	} else if strings.HasPrefix(sqlUpper, "DELETE") {
		return "DELETE"
	}
	return ""
}

// extractTableName attempts to extract the table name from the SQL query
func extractTableName(sql string) string {
	// This is a very simplified extraction and won't work for all queries
	// A more robust implementation would use SQL parsing
	sqlUpper := strings.ToUpper(strings.TrimSpace(sql))

	// Try to find table name based on common patterns
	var fromIndex int
	if strings.Contains(sqlUpper, " FROM ") {
		fromIndex = strings.Index(sqlUpper, " FROM ") + 6
	} else if strings.Contains(sqlUpper, " INTO ") {
		fromIndex = strings.Index(sqlUpper, " INTO ") + 6
	} else if strings.Contains(sqlUpper, "UPDATE ") {
		fromIndex = strings.Index(sqlUpper, "UPDATE ") + 7
	} else {
		return ""
	}

	// Extract the string after FROM/INTO/UPDATE until the next space or end
	remainder := sqlUpper[fromIndex:]
	spaceIndex := strings.Index(remainder, " ")

	if spaceIndex == -1 {
		return remainder
	}

	return remainder[:spaceIndex]
}

// extractTraceIDFromContext attempts to extract a trace ID from context if available
func extractTraceIDFromContext(ctx context.Context) string {
	// Implementation depends on how trace IDs are stored in your context
	// This is a placeholder implementation
	if ctx == nil {
		return ""
	}

	// Example: if you use a specific key for trace ID
	// if id, ok := ctx.Value("trace_id").(string); ok {
	//     return id
	// }

	return ""
}
