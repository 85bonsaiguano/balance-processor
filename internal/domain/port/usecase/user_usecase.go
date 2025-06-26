package usecase

import (
	"context"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
)

// UserActivityMetrics tracks key user activity times and metrics
type UserActivityMetrics struct {
	LastActivityTime    time.Time
	AccountCreatedAt    time.Time
	LastBalanceUpdateAt time.Time
	SessionDuration     time.Duration
	TotalTransactions   int
}

// UserBalanceResponse represents the standardized balance response
type UserBalanceResponse struct {
	UserID  uint64 `json:"userId"`
	Balance string `json:"balance"` // Formatted with 2 decimal places
}

// UserUseCase defines methods for user-related business operations
type UserUseCase interface {
	// GetFormattedUserBalance retrieves user balance with properly formatted response
	// This is the main method used by the GET /user/{userId}/balance endpoint
	GetFormattedUserBalance(ctx context.Context, userID uint64) (*UserBalanceResponse, error)

	// CreateUser creates a new user with the given ID and initial balance
	CreateUser(ctx context.Context, id uint64, initialBalance string) (*entity.User, error)

	// CreateDefaultUsers creates predefined users with IDs 1, 2, 3 as required by the task
	CreateDefaultUsers(ctx context.Context) error

	// UserExists checks if a user exists with the given ID
	UserExists(ctx context.Context, userID uint64) (bool, error)

	// ModifyBalance handles both addition and deduction with unified interface
	// This is the core method used by the POST /user/{userId}/transaction endpoint
	// Parameters:
	// - userID: ID of the user
	// - amount: String amount to modify (e.g. "10.50")
	// - isWin: true for win/add, false for lose/deduct
	// - transactionID: Unique transaction identifier for idempotency
	// - sourceType: Source of the transaction (game, server, payment)
	// Returns: Updated user, transaction time, error if any
	ModifyBalance(ctx context.Context, userID uint64, amount string, isWin bool, transactionID string, sourceType string) (*entity.User, time.Time, error)
}
