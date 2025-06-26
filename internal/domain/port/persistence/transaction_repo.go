package persistence

import (
	"context"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
)

// TransactionRepository defines essential methods to interact with transaction data
type TransactionRepository interface {
	// Create saves a new transaction
	// Primary method for storing transaction records during transaction processing
	//
	// Possible errors:
	// - ErrDuplicateTransaction: If transaction with the same ID already exists
	// - ErrInvalidTransaction: If transaction data is invalid
	// - ErrUserNotFound: If referenced user does not exist
	// - ErrDatabaseConnection: If database connection fails
	Create(ctx context.Context, transaction *entity.Transaction) error

	// Update updates an existing transaction by transaction ID
	// Used to update transaction status and other fields
	//
	// Possible errors:
	// - ErrTransactionNotFound: If transaction with the given ID doesn't exist
	// - ErrDatabaseConnection: If database connection fails
	Update(ctx context.Context, transaction *entity.Transaction) error

	// GetByTransactionID retrieves a transaction by its external transaction ID
	//
	// Possible errors:
	// - ErrTransactionNotFound: If transaction with the given ID doesn't exist
	// - ErrDatabaseConnection: If database connection fails
	GetByTransactionID(ctx context.Context, transactionID string) (*entity.Transaction, error)

	// TransactionExists checks if a transaction with the given ID already exists
	// Used for idempotency checking
	//
	// Possible errors:
	// - ErrDatabaseConnection: If database connection fails
	TransactionExists(ctx context.Context, transactionID string) (bool, error)
}
