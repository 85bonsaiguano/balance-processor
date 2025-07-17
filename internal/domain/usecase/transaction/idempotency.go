package transaction

import (
	"context"
	"fmt"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
)

// IdempotencyHandler provides idempotency checking for transactions
type IdempotencyHandler struct {
	transactionRepo persistence.TransactionRepository
}

// NewIdempotencyHandler creates a new IdempotencyHandler
func NewIdempotencyHandler(transactionRepo persistence.TransactionRepository) *IdempotencyHandler {
	return &IdempotencyHandler{
		transactionRepo: transactionRepo,
	}
}

// CheckIdempotency checks if a transaction with the given ID already exists
// Returns the transaction, a boolean indicating if it was found, and any error
func (h *IdempotencyHandler) CheckIdempotency(
	ctx context.Context,
	transactionID string,
) (*entity.Transaction, bool, error) {
	// Check if the transaction already exists
	exists, err := h.transactionRepo.TransactionExists(ctx, transactionID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check if transaction exists: %w", err)
	}

	if !exists {
		// Transaction doesn't exist, so we're good to proceed
		return nil, false, nil
	}

	// Transaction exists, retrieve it to return the idempotent response
	txn, err := h.transactionRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		if err == errs.ErrTransactionNotFound {
			// This is an edge case - the transaction existed when we checked but was deleted
			// before we could retrieve it. Treat it as non-existent.
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("failed to retrieve existing transaction: %w", err)
	}

	// Return the existing transaction
	return txn, true, nil
}
