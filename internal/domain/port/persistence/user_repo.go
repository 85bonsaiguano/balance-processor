package persistence

import (
	"context"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
)

// UserRepository defines essential methods to interact with user data
// This simplified interface focuses on core operations needed for the Entain task
type UserRepository interface {
	// GetByID retrieves a user by ID
	// Used for the GET /user/{userId}/balance endpoint
	//
	// Possible errors:
	// - ErrUserNotFound: If user with specified ID doesn't exist
	// - ErrDatabaseConnection: If database connection fails
	GetByID(ctx context.Context, id uint64) (*entity.User, error)

	// Create creates a new user
	// Used for initializing default users (1, 2, 3)
	//
	// Possible errors:
	// - ErrDuplicateUser: If user with same ID already exists
	// - ErrInvalidUserData: If user data is invalid (e.g., negative balance)
	// - ErrDatabaseConnection: If database connection fails
	Create(ctx context.Context, user *entity.User) error

	// Update updates user information
	// Core method for modifying user data
	//
	// Possible errors:
	// - ErrUserNotFound: If user doesn't exist
	// - ErrInvalidUserData: If updated user data is invalid
	// - ErrDatabaseConnection: If database connection fails
	Update(ctx context.Context, user *entity.User) error

	// ProcessTransaction updates user balance atomically
	// Returns the updated user on success or error on failure
	// This is the primary method for transaction processing (POST /user/{userId}/transaction)
	//
	// Possible errors:
	// - ErrUserNotFound: If user doesn't exist
	// - ErrInsufficientBalance: If balance would become negative (for deductions)
	// - ErrUserLocked: If user is locked by another operation
	// - ErrDatabaseConnection: If database connection fails
	ProcessTransaction(ctx context.Context, userID uint64, balanceChange int64) (*entity.User, error)
}
