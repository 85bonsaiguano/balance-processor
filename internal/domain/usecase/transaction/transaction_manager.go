package transaction

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
)

// Constants for retry logic
const (
	maxRetries = 5     // Increased from 3 to 5
	baseBackoff = 5 * time.Millisecond
)

// TransactionManager manages the processing of transactions
// This is a usecase (no interface as per requirements)
type TransactionManager struct {
	unitOfWork   persistence.UnitOfWork
	userLockRepo persistence.UserLockRepository
	timeProvider coreport.TimeProvider
	logger       coreport.Logger
	lockTimeout  time.Duration
	shutdown     bool
}

// NewTransactionManager creates a new TransactionManager
func NewTransactionManager(
	unitOfWork persistence.UnitOfWork,
	userLockRepo persistence.UserLockRepository,
	timeProvider coreport.TimeProvider,
	logger coreport.Logger,
) *TransactionManager {
	return &TransactionManager{
		unitOfWork:   unitOfWork,
		userLockRepo: userLockRepo,
		timeProvider: timeProvider,
		logger:       logger,
		lockTimeout:  5 * time.Second, // Default lock timeout
		shutdown:     false,
	}
}

// WithLockTimeout configures the lock timeout duration
func (m *TransactionManager) WithLockTimeout(timeout time.Duration) *TransactionManager {
	m.lockTimeout = timeout
	return m
}

// ProcessTransaction processes a transaction for a user
// This method is safe to be called concurrently from different instances
// as it uses database locks and transactions to ensure consistency
func (m *TransactionManager) ProcessTransaction(
	ctx context.Context,
	userID uint64,
	transactionID string,
	sourceType string,
	state string,
	amount string,
) (*entity.Transaction, error) {
	// Check if we're shutting down
	if m.shutdown {
		return nil, fmt.Errorf("transaction manager is shutting down")
	}

	// Step 1: Check for idempotency first before acquiring any locks
	txn, err := m.checkIdempotency(ctx, transactionID)
	if err == nil {
		// Transaction exists, return it (idempotent response)
		return txn, nil
	} else if err != errs.ErrTransactionNotFound {
		// Some other error occurred
		return nil, err
	}

	// Implement retry logic for potential concurrency issues
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Log retry attempt
			m.logger.Info("Retrying transaction processing", map[string]any{
				"transactionID": transactionID,
				"attempt":       attempt + 1,
				"maxAttempts":   maxRetries,
				"error":         lastErr.Error(),
			})
			
			// Apply exponential backoff
			backoffTime := baseBackoff * time.Duration(1<<uint(attempt))
			time.Sleep(backoffTime)
		}

		// Try to process the transaction
		txn, err = m.tryProcessTransaction(ctx, userID, transactionID, sourceType, state, amount)
		if err == nil {
			// Success
			return txn, nil
		}

		// Check if the error is retryable
		if isRetryableError(err) {
			lastErr = err
			continue
		}

		// Non-retryable error, return immediately
		return nil, err
	}

	// All retries failed
	m.logger.Error("Failed to process transaction after retries", map[string]any{
		"transactionID": transactionID,
		"attempts":      maxRetries,
		"error":         lastErr.Error(),
	})
	return nil, lastErr
}

// isRetryableError checks if an error can be retried
func isRetryableError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "deadlock") ||
		strings.Contains(errStr, "serialization") ||
		strings.Contains(errStr, "lock timeout") ||
		strings.Contains(errStr, "could not serialize access") ||
		strings.Contains(errStr, "concurrent update") ||
		strings.Contains(errStr, "retry transaction") ||
		strings.Contains(errStr, "serialization failure") ||
		strings.Contains(errStr, "transaction rollback") ||
		strings.Contains(errStr, "current transaction is aborted") ||
		strings.Contains(errStr, "40001") // PostgreSQL serialization failure code
}

// tryProcessTransaction attempts to process a transaction with proper locking
// This separates the retry logic from the transaction processing
func (m *TransactionManager) tryProcessTransaction(
	ctx context.Context,
	userID uint64,
	transactionID string,
	sourceType string,
	state string,
	amount string,
) (*entity.Transaction, error) {
	// Step 2: Acquire lock on user using database row lock
	// This ensures no other instance can process transactions for this user concurrently
	err := m.userLockRepo.AcquireLock(ctx, userID, m.lockTimeout)
	if err != nil {
		if err == errs.ErrUserLocked {
			return nil, fmt.Errorf("user %d is locked by another process: %w", userID, err)
		}
		return nil, fmt.Errorf("failed to acquire lock for user %d: %w", userID, err)
	}

	// Step 3: Begin a database transaction
	dbCtx, err := m.unitOfWork.Begin(ctx)
	if err != nil {
		// Release the lock if we couldn't start a transaction
		_ = m.userLockRepo.ReleaseLock(ctx, userID)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure we always end the database transaction and release the lock
	defer func() {
		// Rollback only has an effect if the transaction hasn't been committed
		_ = m.unitOfWork.Rollback(dbCtx)
		// Always release the lock
		_ = m.userLockRepo.ReleaseLock(ctx, userID)
	}()

	// Try to process the transaction
	result, err := m.executeTransaction(dbCtx, userID, transactionID, sourceType, state, amount)
	if err != nil {
		return nil, err
	}

	// Commit the database transaction
	if err := m.unitOfWork.Commit(dbCtx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// checkIdempotency checks if the transaction already exists
// This is separate so we don't have to acquire a lock for duplicate transactions
func (m *TransactionManager) checkIdempotency(ctx context.Context, transactionID string) (*entity.Transaction, error) {
	txnRepo := m.unitOfWork.GetTransactionRepository(ctx)
	return txnRepo.GetByTransactionID(ctx, transactionID)
}

// executeTransaction performs the actual transaction processing
func (m *TransactionManager) executeTransaction(
	ctx context.Context,
	userID uint64,
	transactionID string,
	sourceType string,
	state string,
	amount string,
) (*entity.Transaction, error) {
	// Get the repositories
	userRepo := m.unitOfWork.GetUserRepository(ctx)
	txnRepo := m.unitOfWork.GetTransactionRepository(ctx)

	// Check for idempotency again within the transaction (double-check)
	exists, err := txnRepo.TransactionExists(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if transaction exists: %w", err)
	}
	if exists {
		// Transaction already exists, return it
		return txnRepo.GetByTransactionID(ctx, transactionID)
	}

	// Create the transaction entity
	txn, err := entity.NewTransaction(
		userID,
		transactionID,
		sourceType,
		state,
		amount,
		m.timeProvider,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Get the user
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Process the transaction based on its state
	switch txn.State {
	case entity.StateWin:
		// Win transaction increases balance
		user.ApplyWinTransaction(txn.AmountInCents, m.timeProvider)

	case entity.StateLose:
		// Lose transaction decreases balance
		if err := user.ApplyLoseTransaction(txn.AmountInCents, m.timeProvider); err != nil {
			// Mark the transaction as failed and save it
			txn.MarkAsFailed(m.timeProvider, "Insufficient balance")
			if saveErr := txnRepo.Create(ctx, txn); saveErr != nil {
				m.logger.Error("Failed to save failed transaction", map[string]any{
					"error":         saveErr,
					"transactionID": transactionID,
				})
			}
			return txn, err
		}

	default:
		return nil, fmt.Errorf("unsupported transaction state: %s", txn.State)
	}

	// Update the transaction with the result
	txn.MarkAsProcessed(m.timeProvider, user.Balance())

	// Save the transaction
	if err := txnRepo.Create(ctx, txn); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	// Update the user
	if err := userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return txn, nil
}

// Shutdown gracefully shuts down the TransactionManager
func (m *TransactionManager) Shutdown() {
	m.logger.Info("Shutting down TransactionManager", nil)
	m.shutdown = true

	// Add any additional cleanup here if needed in the future
	// For example, waiting for pending transactions, closing connections, etc.
}
