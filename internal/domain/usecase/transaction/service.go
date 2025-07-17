package transaction

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
)

// TransactionRequest represents a request to process a transaction
type TransactionRequest struct {
	TransactionID string
	SourceType    entity.SourceType
	State         string
	Amount        string
}

// TransactionResponse represents the response after processing a transaction
type TransactionResponse struct {
	Success       bool
	ResultBalance string
	ErrorMessage  string
	StatusCode    int
}

// Service is the main transaction service implementation that ties together
// all the components for transaction processing without using interfaces
type Service struct {
	manager            *TransactionManager
	processor          *TransactionProcessor
	validator          *TransactionValidator
	idempotencyHandler *IdempotencyHandler
	logger             coreport.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(
	uow persistence.UnitOfWork,
	userLockRepo persistence.UserLockRepository,
	timeProvider coreport.TimeProvider,
	logger coreport.Logger,
	lockTimeout time.Duration,
) *Service {
	// Create components
	manager := NewTransactionManager(uow, userLockRepo, timeProvider, logger)
	manager.WithLockTimeout(lockTimeout)

	validator := NewTransactionValidator()

	txnRepo := uow.GetTransactionRepository(context.Background())
	idempotencyHandler := NewIdempotencyHandler(txnRepo)

	processor := NewTransactionProcessor(manager, validator, idempotencyHandler)

	return &Service{
		manager:            manager,
		processor:          processor,
		validator:          validator,
		idempotencyHandler: idempotencyHandler,
		logger:             logger,
	}
}

// ProcessTransaction processes a transaction request and returns an appropriate response
func (s *Service) ProcessTransaction(
	ctx context.Context,
	userID uint64,
	req TransactionRequest,
) (*TransactionResponse, error) {
	// Create process request
	processReq := ProcessTransactionRequest{
		UserID:        userID,
		TransactionID: req.TransactionID,
		SourceType:    string(req.SourceType),
		State:         req.State,
		Amount:        req.Amount,
	}

	// Process the transaction
	txn, err := s.processor.Process(ctx, processReq)

	// Handle the response
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := err.Error()

		// Map known errors to appropriate status codes
		switch {
		case errs.IsUserNotFoundError(err):
			statusCode = http.StatusNotFound
			
		case errs.IsDuplicateTransactionError(err):
			statusCode = http.StatusConflict
			
		case errs.IsInsufficientBalanceError(err):
			statusCode = http.StatusBadRequest
			
		case errs.IsUserLockedError(err):
			statusCode = http.StatusConflict
			
		case errs.IsNotFoundError(err):
			statusCode = http.StatusNotFound
			
		// Identify database concurrency errors specifically
		case strings.Contains(strings.ToLower(err.Error()), "deadlock"):
			statusCode = http.StatusConflict
			errorMessage = "Transaction could not be processed due to concurrent operations. Please try again."
			
		case strings.Contains(strings.ToLower(err.Error()), "serialization"):
			statusCode = http.StatusConflict
			errorMessage = "Transaction could not be processed due to concurrent operations. Please try again."
			
		case strings.Contains(strings.ToLower(err.Error()), "lock timeout"):
			statusCode = http.StatusConflict
			errorMessage = "Transaction processing timed out due to lock contention. Please try again."
		}

		// Log the error with more detail for internal use
		s.logger.Error("Transaction processing failed", map[string]any{
			"error":         err.Error(),
			"status_code":   statusCode,
			"transaction_id": req.TransactionID,
			"user_id":       userID,
		})

		return &TransactionResponse{
			Success:      false,
			ErrorMessage: errorMessage,
			StatusCode:   statusCode,
		}, err
	}

	// Successful transaction
	return &TransactionResponse{
		Success:       true,
		ResultBalance: txn.GetResultBalance(),
		StatusCode:    http.StatusOK,
	}, nil
}

// GetManager returns the underlying transaction manager
// Used for graceful shutdown
func (s *Service) GetManager() *TransactionManager {
	return s.manager
}

// Shutdown performs any cleanup tasks needed
func (s *Service) Shutdown() {
	// Currently no cleanup needed, but this method can be extended
	// if we add background workers or connection pools
}
