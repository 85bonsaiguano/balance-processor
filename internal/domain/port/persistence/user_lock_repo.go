package persistence

import (
	"context"
	"time"
)

// UserLockRepository defines methods for managing user locks
// Simplified version with only essential locking functionality
type UserLockRepository interface {
	// AcquireLock attempts to acquire a lock on the user for transaction processing
	// The lock expires after the given duration
	//
	// Possible errors:
	// - ErrUserNotFound: If user with specified ID doesn't exist
	// - ErrUserLocked: If user is already locked by another process
	// - ErrDatabaseConnection: If database connection fails
	AcquireLock(ctx context.Context, userID uint64, duration time.Duration) error

	// ReleaseLock releases a previously acquired lock
	// This should be called after transaction processing completes
	//
	// Possible errors:
	// - ErrUserNotFound: If user with specified ID doesn't exist
	// - ErrDatabaseConnection: If database connection fails
	ReleaseLock(ctx context.Context, userID uint64) error
}
