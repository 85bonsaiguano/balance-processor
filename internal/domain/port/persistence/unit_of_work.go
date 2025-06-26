package persistence

import (
	"context"
)

// UnitOfWork defines an interface for coordinating transaction operations
// across multiple repositories to maintain data consistency
type UnitOfWork interface {
	// Begin starts a new transaction and returns a transactional context
	Begin(ctx context.Context) (context.Context, error)

	// Commit commits the transaction in the given context
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction in the given context
	Rollback(ctx context.Context) error

	// GetUserRepository returns a user repository bound to the current transaction
	GetUserRepository(ctx context.Context) UserRepository

	// GetTransactionRepository returns a transaction repository bound to the current transaction
	GetTransactionRepository(ctx context.Context) TransactionRepository
}
