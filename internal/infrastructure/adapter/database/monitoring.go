package database

import (
	"context"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// QueryMetrics holds metrics about a database query
type QueryMetrics struct {
	Operation    string
	Duration     time.Duration
	RowsAffected int64
	RowsReturned int64
	Failed       bool
	ErrorMessage string
}

// MetricsCollector collects database operation metrics
type MetricsCollector struct {
	logger       coreport.Logger
	timeProvider coreport.TimeProvider
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger coreport.Logger, timeProvider coreport.TimeProvider) *MetricsCollector {
	return &MetricsCollector{
		logger:       logger,
		timeProvider: timeProvider,
	}
}

// MeasureQuery measures the execution time of a database query
func (c *MetricsCollector) MeasureQuery(ctx context.Context, operation string, fn func() (int64, error)) (*QueryMetrics, error) {
	start := c.timeProvider.Now()

	rowsAffected, err := fn()

	metrics := &QueryMetrics{
		Operation:    operation,
		Duration:     c.timeProvider.Now().Sub(start),
		RowsAffected: rowsAffected,
		Failed:       err != nil,
	}

	if err != nil {
		metrics.ErrorMessage = err.Error()
	}

	// Log slow queries
	if metrics.Duration > 100*time.Millisecond {
		c.logger.Warn("Slow database query detected", map[string]any{
			"operation":     operation,
			"duration_ms":   metrics.Duration.Milliseconds(),
			"rows_affected": rowsAffected,
			"failed":        metrics.Failed,
			"error_message": metrics.ErrorMessage,
		})
	}

	return metrics, err
}
