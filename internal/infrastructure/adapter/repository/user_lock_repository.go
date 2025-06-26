package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/model"
	"gorm.io/gorm"
)

// UserLockRepository implements user locking functionality using GORM
type UserLockRepository struct {
	db              *gorm.DB
	timeProvider    coreport.TimeProvider
	logger          coreport.Logger
	errorClassifier *ErrorClassifier
}

// NewUserLockRepository creates a new UserLockRepository instance
func NewUserLockRepository(db *gorm.DB, timeProvider coreport.TimeProvider, logger coreport.Logger) *UserLockRepository {
	return &UserLockRepository{
		db:              db,
		timeProvider:    timeProvider,
		logger:          logger,
		errorClassifier: NewErrorClassifier(),
	}
}

// AcquireLock attempts to acquire a lock on the user for transaction processing
// Streamlined version with simplified error handling for better performance
func (r *UserLockRepository) AcquireLock(ctx context.Context, userID uint64, duration time.Duration) error {
	r.logger.Debug("Attempting to acquire lock", map[string]any{
		"user_id":  userID,
		"duration": duration.String(),
	})

	now := r.timeProvider.Now()
	expiresAt := now.Add(duration)

	// Use SQL directly for better performance with upsert logic
	// This performs an insert or update in a single operation
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_locks (user_id, locked_at, expires_at, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE 
		SET locked_at = EXCLUDED.locked_at, 
		    expires_at = EXCLUDED.expires_at, 
		    updated_at = EXCLUDED.updated_at
		WHERE user_locks.expires_at <= ?`,
		userID, now, expiresAt, now, now, // INSERT values
		now, // WHERE condition for the ON CONFLICT clause
	).Error

	if err != nil {
		// Check if this is a unique constraint violation that wasn't caught by the ON CONFLICT clause
		// This indicates the lock exists and hasn't expired
		if r.errorClassifier.IsDuplicateKeyError(err) {
			r.logger.Warn("User is already locked", map[string]any{
				"user_id": userID,
			})
			return errs.ErrUserLocked
		}

		// For context errors, return a more specific error
		if isContextError(err) {
			r.logger.Warn("Context timeout acquiring lock", map[string]any{
				"user_id": userID,
				"error":   err.Error(),
			})
			return fmt.Errorf("lock acquisition timeout: %w", err)
		}

		// For other database errors
		r.logger.Error("Database error acquiring lock", map[string]any{
			"user_id": userID,
			"error":   err.Error(),
		})
		return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, err.Error())
	}

	// If the row was affected, we got the lock
	r.logger.Info("Lock acquired successfully", map[string]any{
		"user_id":    userID,
		"locked_at":  now,
		"expires_at": expiresAt,
	})
	return nil
}

// isContextError checks if an error is related to context timeout or cancellation
func isContextError(err error) bool {
	if err == nil {
		return false
	}

	// Check standard context errors
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// Check error string for common timeout patterns
	errStr := err.Error()
	return strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "context canceled") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "timeout")
}

// We're not using this function, but keeping it commented for future reference
// handleDatabaseError standardizes database error handling
// func (r *UserLockRepository) handleDatabaseError(operation string, err error, userID uint64) error {
// 	r.logger.Error(fmt.Sprintf("Database error when %s", operation), map[string]any{
// 		"user_id": userID,
// 		"error":   err.Error(),
// 	})
// 	return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, err.Error())
// }

// ReleaseLock releases a previously acquired lock with simplified approach
func (r *UserLockRepository) ReleaseLock(ctx context.Context, userID uint64) error {
	r.logger.Debug("Releasing lock", map[string]any{
		"user_id": userID,
	})

	// First check if the lock exists - this allows us to distinguish between
	// "lock wasn't there" vs "delete failed" cases
	var lock model.UserLock
	findResult := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&lock)

	// If lock doesn't exist (already released or expired)
	if errors.Is(findResult.Error, gorm.ErrRecordNotFound) {
		r.logger.Debug("No lock found to release - may have already expired", map[string]any{
			"user_id": userID,
		})
		return nil
	}

	// If there was another error finding the lock
	if findResult.Error != nil && !isContextError(findResult.Error) {
		r.logger.Error("Error checking lock status", map[string]any{
			"user_id": userID,
			"error":   findResult.Error.Error(),
		})
		return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, findResult.Error.Error())
	}

	// Delete the lock if it exists
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.UserLock{})

	// If there's an error but it's a context error, don't treat it as critical
	// The lock will expire automatically after its timeout
	if result.Error != nil && isContextError(result.Error) {
		r.logger.Warn("Context timeout when releasing lock, lock will expire automatically", map[string]any{
			"user_id": userID,
			"error":   result.Error.Error(),
		})
		return nil
	}

	// For other errors
	if result.Error != nil {
		r.logger.Error("Failed to release lock", map[string]any{
			"user_id": userID,
			"error":   result.Error.Error(),
		})
		return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, result.Error.Error())
	}

	// Success log only if a lock was actually deleted
	if result.RowsAffected > 0 {
		r.logger.Info("Lock released successfully", map[string]any{
			"user_id": userID,
		})
	}

	return nil
}

// CleanupExpiredLocks removes all expired locks from the database
func (r *UserLockRepository) CleanupExpiredLocks(ctx context.Context) error {
	now := r.timeProvider.Now()

	r.logger.Debug("Cleaning up expired locks", map[string]any{
		"current_time": now,
	})

	result := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&model.UserLock{})

	if result.Error != nil {
		r.logger.Error("Failed to clean up expired locks", map[string]any{
			"error": result.Error.Error(),
		})
		return fmt.Errorf("%w: %s", errs.ErrDatabaseConnection, result.Error.Error())
	}

	r.logger.Info("Expired locks cleanup completed", map[string]any{
		"locks_removed": result.RowsAffected,
	})
	return nil
}
