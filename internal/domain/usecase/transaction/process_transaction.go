package transaction

import (
	"context"
	"fmt"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
)

// TransactionProcessor provides the main entry point for processing transactions
// It delegates to specialized components for validation, idempotency and actual processing
type TransactionProcessor struct {
	transactionManager *TransactionManager
	validator          *TransactionValidator
	idempotencyHandler *IdempotencyHandler
}

// NewTransactionProcessor creates a new TransactionProcessor
func NewTransactionProcessor(
	transactionManager *TransactionManager,
	validator *TransactionValidator,
	idempotencyHandler *IdempotencyHandler,
) *TransactionProcessor {
	return &TransactionProcessor{
		transactionManager: transactionManager,
		validator:          validator,
		idempotencyHandler: idempotencyHandler,
	}
}

// ProcessTransactionRequest represents the input for processing a transaction
type ProcessTransactionRequest struct {
	UserID        uint64
	TransactionID string
	SourceType    string
	State         string
	Amount        string
}

// Process handles the processing of a transaction
// This method orchestrates the entire process:
// 1. Validates the transaction input
// 2. Checks for idempotency
// 3. Processes the transaction through the transaction manager
func (p *TransactionProcessor) Process(
	ctx context.Context,
	req ProcessTransactionRequest,
) (*entity.Transaction, error) {
	// Step 1: Validate the request
	if err := p.validator.ValidateTransaction(req.UserID, req.TransactionID, req.SourceType, req.State, req.Amount); err != nil {
		return nil, fmt.Errorf("invalid transaction: %w", err)
	}

	// Step 2: Check for idempotency
	// Note: We also check idempotency in the transaction manager, but doing an initial check here
	// allows us to return quickly without acquiring database locks for duplicate requests
	txn, found, err := p.idempotencyHandler.CheckIdempotency(ctx, req.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	if found {
		return txn, nil
	}

	// Step 3: Process the transaction
	return p.transactionManager.ProcessTransaction(
		ctx,
		req.UserID,
		req.TransactionID,
		req.SourceType,
		req.State,
		req.Amount,
	)
}
