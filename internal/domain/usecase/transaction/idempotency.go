package transaction

import (
	"context"
)

// IsDuplicateTransaction checks if a transaction with the given ID already exists
func (s *Service) IsDuplicateTransaction(ctx context.Context, transactionID string) (bool, error) {
	// Get regular transaction repository (not bound to a transaction)
	txnRepo := s.uow.GetTransactionRepository(ctx)

	// Delegate to repository method
	return txnRepo.TransactionExists(ctx, transactionID)
}
