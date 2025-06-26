package database

import (
	"context"
	"strings"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
	MaxInterval   time.Duration
	JitterFactor  float64 // Factor to add randomness to retry intervals (0.0-1.0)
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    5,
		RetryInterval: 100 * time.Millisecond,
		MaxInterval:   2 * time.Second,
		JitterFactor:  0.2, // 20% jitter to avoid thundering herd
	}
}

// RetryOnTransientError retries an operation when a transient error occurs
func RetryOnTransientError(
	ctx context.Context,
	config RetryConfig,
	operation func() error,
	errorMapper *ErrorMapper,
	logger coreport.Logger,
) error {
	var err error
	var attempt int

	for attempt = 0; attempt < config.MaxRetries; attempt++ {
		// Execute the operation
		err = operation()
		if err == nil {
			return nil
		}

		// Only retry on transient errors
		if !isTransientError(err) {
			return err
		}

		// Log the retry attempt
		backoff := calculateBackoffWithJitter(attempt, config)
		logger.Warn("Transient database error, retrying operation", map[string]any{
			"attempt":     attempt + 1,
			"max_retries": config.MaxRetries,
			"error":       err.Error(),
			"retry_after": backoff.String(),
		})

		// Apply backoff with exponential delay and jitter
		select {
		case <-time.After(backoff):
			// Continue with next retry
		case <-ctx.Done():
			// Context was canceled
			logger.Warn("Retry operation canceled by context", map[string]any{
				"attempts":    attempt + 1,
				"max_retries": config.MaxRetries,
				"error":       ctx.Err().Error(),
			})
			return ctx.Err()
		}
	}

	// All retries failed
	logger.Error("All retry attempts failed", map[string]any{
		"attempts":    attempt,
		"max_retries": config.MaxRetries,
		"error":       err.Error(),
	})

	return err
}

// calculateBackoffWithJitter computes the backoff duration with exponential increase and jitter
func calculateBackoffWithJitter(attempt int, config RetryConfig) time.Duration {
	// Calculate exponential backoff: baseInterval * 2^attempt
	backoff := config.RetryInterval * (1 << uint(attempt))

	// Cap at max interval
	if backoff > config.MaxInterval {
		backoff = config.MaxInterval
	}

	// Add jitter to avoid thundering herd problem
	if config.JitterFactor > 0 {
		jitter := time.Duration(float64(backoff) * config.JitterFactor * (float64(time.Now().UnixNano()%100) / 100.0))
		backoff = backoff + jitter
	}

	return backoff
}

// isTransientError checks if an error is transient and can be retried
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "deadlock") ||
		strings.Contains(errMsg, "serialization") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "too many connections") ||
		strings.Contains(errMsg, "server closed") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "write: broken pipe") ||
		strings.Contains(errMsg, "lock timeout") ||
		strings.Contains(errMsg, "duplicate key") || // For handling duplicate transactions
		strings.Contains(errMsg, "eof")
}
