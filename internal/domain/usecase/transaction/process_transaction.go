package transaction

import (
	"context"
	"net/http"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
)

// Service implements the transaction use case
type Service struct {
	uow            persistence.UnitOfWork
	userUseCase    usecase.UserUseCase
	userLockRepo   persistence.UserLockRepository
	timeProvider   coreport.TimeProvider
	logger         coreport.Logger
	lockTimeout    time.Duration
	txManager      *TransactionManager
	processorReady bool
}

// GetManager returns the transaction manager instance
func (s *Service) GetManager() *TransactionManager {
	return s.txManager
}

// NewTransactionService creates a new transaction service
func NewTransactionService(
	uow persistence.UnitOfWork,
	userUseCase usecase.UserUseCase,
	userLockRepo persistence.UserLockRepository,
	timeProvider coreport.TimeProvider,
	logger coreport.Logger,
	lockTimeout time.Duration,
) usecase.TransactionUseCase {
	// Create the service instance first
	svc := &Service{
		uow:          uow,
		userUseCase:  userUseCase,
		userLockRepo: userLockRepo,
		timeProvider: timeProvider,
		logger:       logger,
		lockTimeout:  lockTimeout,
	}

	// Create transaction manager with the service's processTransactionInternal method
	svc.txManager = NewTransactionManager(
		logger,
		timeProvider,
		uow.GetTransactionRepository(context.Background()),
		svc.processTransactionInternal, // Pass the processor function
	)

	svc.processorReady = true
	logger.Info("Transaction processor initialized", nil)

	return svc
}

// ProcessTransaction processes a transaction and updates the user's balance
// This is now a wrapper that delegates to the queue-based processor
func (s *Service) ProcessTransaction(
	ctx context.Context,
	userID uint64,
	req usecase.TransactionRequest,
) (*usecase.TransactionResult, error) {
	// Validate the request first
	if err := s.ValidateTransactionRequest(req); err != nil {
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: err.Error(),
			StatusCode:   http.StatusBadRequest,
		}, err
	}

	// Check if user exists before enqueuing
	userExists, err := s.userUseCase.UserExists(ctx, userID)
	if err != nil {
		s.logger.Error("Error checking user existence", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Failed to verify user",
			StatusCode:   http.StatusInternalServerError,
		}, err
	}

	if !userExists {
		s.logger.Warn("User not found for transaction", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
		})
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "User not found",
			StatusCode:   http.StatusNotFound,
		}, errs.ErrUserNotFound
	}

	// Enqueue the transaction for sequential processing
	s.logger.Info("Enqueuing transaction for sequential processing", map[string]any{
		"userId":        userID,
		"transactionId": req.TransactionID,
	})

	return s.txManager.EnqueueTransaction(ctx, userID, req)
}

// processTransactionInternal is the actual transaction processing implementation
// This is called by the queue worker to ensure sequential processing
func (s *Service) processTransactionInternal(
	ctx context.Context,
	userID uint64,
	req usecase.TransactionRequest,
) (*usecase.TransactionResult, error) {
	s.logger.Info("Processing transaction from queue", map[string]any{
		"userId":        userID,
		"transactionId": req.TransactionID,
	})

	// Check idempotency - if we've already processed this transaction
	isDuplicate, err := s.IsDuplicateTransaction(ctx, req.TransactionID)
	if err != nil {
		s.logger.Error("Failed to check for duplicate transaction", map[string]any{
			"transactionId": req.TransactionID,
			"userId":        userID,
			"error":         err.Error(),
		})
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Failed to verify transaction uniqueness",
			StatusCode:   http.StatusInternalServerError,
		}, err
	}

	if isDuplicate {
		s.logger.Warn("Duplicate transaction detected", map[string]any{
			"transactionId": req.TransactionID,
			"userId":        userID,
		})
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Transaction already processed",
			StatusCode:   http.StatusConflict,
		}, errs.NewDuplicateTransactionError(req.TransactionID, userID, string(req.SourceType))
	}

	// Try to acquire lock on user
	if err := s.userLockRepo.AcquireLock(ctx, userID, s.lockTimeout); err != nil {
		s.logger.Error("Failed to acquire user lock", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "User account is locked for processing",
			StatusCode:   http.StatusConflict,
		}, err
	}

	// Make sure we release the lock when done
	defer func() {
		if err := s.userLockRepo.ReleaseLock(ctx, userID); err != nil {
			s.logger.Error("Failed to release user lock", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         err.Error(),
			})
		}
	}()

	// Begin transaction with SERIALIZABLE isolation level
	txCtx, err := s.uow.Begin(ctx)
	if err != nil {
		s.logger.Error("Failed to begin database transaction", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Failed to start transaction processing",
			StatusCode:   http.StatusInternalServerError,
		}, err
	}

	// Create new transaction record with pending status
	txn, err := entity.NewTransaction(
		userID,
		req.TransactionID,
		string(req.SourceType),
		req.State,
		req.Amount,
		s.timeProvider,
	)
	if err != nil {
		s.logger.Error("Failed to create transaction", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction after creation error", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         rollbackErr.Error(),
			})
		}
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Invalid transaction data",
			StatusCode:   http.StatusBadRequest,
		}, err
	}

	// Save transaction record first (with pending status)
	txnRepo := s.uow.GetTransactionRepository(txCtx)
	if err := txnRepo.Create(txCtx, txn); err != nil {
		s.logger.Error("Failed to save transaction", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction after save error", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         rollbackErr.Error(),
			})
		}
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Failed to record transaction",
			StatusCode:   http.StatusInternalServerError,
		}, err
	}

	// Determine if this is a win or lose transaction
	isWin := txn.State == entity.StateWin

	// Apply the balance change
	user, transactionTime, err := s.userUseCase.ModifyBalance(
		txCtx,
		userID,
		req.Amount,
		isWin,
		req.TransactionID,
		string(req.SourceType),
	)

	if err != nil {
		s.logger.Error("Failed to modify user balance", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})

		// Mark transaction as failed
		errorMessage := err.Error()
		txn.MarkAsFailed(s.timeProvider, errorMessage)
		if updateErr := txnRepo.Update(txCtx, txn); updateErr != nil {
			s.logger.Error("Failed to update transaction status", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         updateErr.Error(),
			})
		}

		if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction after balance modification error", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         rollbackErr.Error(),
			})
		}

		// Handle specific error types
		statusCode := http.StatusInternalServerError
		if errs.IsInsufficientBalanceError(err) {
			statusCode = http.StatusBadRequest
		}

		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: errorMessage,
			StatusCode:   statusCode,
		}, err
	}

	// Transaction successful - mark as completed
	txn.MarkAsProcessed(s.timeProvider, user.GetBalance())
	if err := txnRepo.Update(txCtx, txn); err != nil {
		s.logger.Error("Failed to update transaction status", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
			s.logger.Error("Failed to rollback transaction after status update error", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         rollbackErr.Error(),
			})
		}
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Failed to update transaction status",
			StatusCode:   http.StatusInternalServerError,
		}, err
	}

	// Commit the transaction
	if err := s.uow.Commit(txCtx); err != nil {
		s.logger.Error("Failed to commit transaction", map[string]any{
			"userId":        userID,
			"transactionId": req.TransactionID,
			"error":         err.Error(),
		})
		if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
			s.logger.Error("Failed to rollback after commit error", map[string]any{
				"userId":        userID,
				"transactionId": req.TransactionID,
				"error":         rollbackErr.Error(),
			})
		}
		return &usecase.TransactionResult{
			Success:      false,
			ErrorMessage: "Failed to complete transaction",
			StatusCode:   http.StatusInternalServerError,
		}, err
	}

	// Log successful transaction
	s.logger.Info("Transaction processed successfully", map[string]any{
		"userId":        userID,
		"transactionId": req.TransactionID,
		"state":         req.State,
		"amount":        req.Amount,
		"sourceType":    req.SourceType,
		"resultBalance": user.GetBalance(),
		"timestamp":     transactionTime.Format(time.RFC3339),
	})

	// Return success result
	return &usecase.TransactionResult{
		Success:       true,
		ResultBalance: user.GetBalance(),
		StatusCode:    http.StatusOK,
	}, nil
}
