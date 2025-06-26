package transaction

import (
	"context"
	"sync"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
)

// TransactionManager provides sequential processing of transactions per user
type TransactionManager struct {
	logger          coreport.Logger
	timeProvider    coreport.TimeProvider
	transactionRepo persistence.TransactionRepository

	// User-based transaction queues for strict ordering
	userQueues     sync.Map // map[uint64]chan *transactionRequest
	queueWaitGroup sync.WaitGroup

	// Function to process transactions
	processor TransactionProcessorFunc
}

// TransactionProcessorFunc is the function signature for processing transactions
type TransactionProcessorFunc func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error)

// transactionRequest represents a queued transaction request
type transactionRequest struct {
	ctx        context.Context
	userID     uint64
	req        usecase.TransactionRequest
	resultChan chan *transactionResult
}

// transactionResult represents the result of a processed transaction
type transactionResult struct {
	result *usecase.TransactionResult
	err    error
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(
	logger coreport.Logger,
	timeProvider coreport.TimeProvider,
	transactionRepo persistence.TransactionRepository,
	processor TransactionProcessorFunc,
) *TransactionManager {
	if processor == nil {
		panic("Transaction processor function cannot be nil")
	}

	return &TransactionManager{
		logger:          logger,
		timeProvider:    timeProvider,
		transactionRepo: transactionRepo,
		userQueues:      sync.Map{},
		processor:       processor,
	}
}

// EnqueueTransaction adds a transaction to the user's queue for sequential processing
// Returns a channel that will receive the result when the transaction is processed
func (m *TransactionManager) EnqueueTransaction(
	ctx context.Context,
	userID uint64,
	req usecase.TransactionRequest,
) (*usecase.TransactionResult, error) {
	m.logger.Debug("Enqueuing transaction for sequential processing", map[string]any{
		"user_id":        userID,
		"transaction_id": req.TransactionID,
	})

	// Create a channel for the result
	resultChan := make(chan *transactionResult, 1)

	// Get or create queue for this user
	var queue chan *transactionRequest
	queueIface, loaded := m.userQueues.LoadOrStore(userID, make(chan *transactionRequest, 100))
	if queueCh, ok := queueIface.(chan *transactionRequest); ok {
		queue = queueCh
	} else {
		m.logger.Error("Failed to type assert queue channel", nil)
		return nil, errs.ErrInternalServer
	}

	// Start worker if this is a new queue
	if !loaded {
		m.logger.Info("Starting new transaction queue worker for user", map[string]any{
			"user_id": userID,
		})
		m.queueWaitGroup.Add(1)
		go m.processUserTransactions(userID, queue)
	}

	// Create transaction request
	txnReq := &transactionRequest{
		ctx:        ctx,
		userID:     userID,
		req:        req,
		resultChan: resultChan,
	}

	// Send request to queue
	select {
	case queue <- txnReq:
		m.logger.Debug("Transaction enqueued successfully", map[string]any{
			"user_id":        userID,
			"transaction_id": req.TransactionID,
		})
	case <-ctx.Done():
		m.logger.Warn("Context canceled while enqueueing transaction", map[string]any{
			"user_id":        userID,
			"transaction_id": req.TransactionID,
			"error":          ctx.Err().Error(),
		})
		return nil, ctx.Err()
	}

	// Wait for result
	select {
	case result := <-resultChan:
		return result.result, result.err
	case <-ctx.Done():
		m.logger.Warn("Context canceled while waiting for transaction result", map[string]any{
			"user_id":        userID,
			"transaction_id": req.TransactionID,
			"error":          ctx.Err().Error(),
		})
		return nil, ctx.Err()
	}
}

// processUserTransactions handles the worker goroutine for a user's transaction queue
func (m *TransactionManager) processUserTransactions(userID uint64, queue chan *transactionRequest) {
	defer m.queueWaitGroup.Done()

	m.logger.Info("Transaction queue worker started", map[string]any{
		"user_id": userID,
	})

	// Process transactions sequentially
	for txnReq := range queue {
		m.logger.Debug("Processing queued transaction", map[string]any{
			"user_id":        userID,
			"transaction_id": txnReq.req.TransactionID,
		})

		// Process the transaction using the injected processor function
		result, err := m.processor(txnReq.ctx, userID, txnReq.req)

		// Send result back
		txnReq.resultChan <- &transactionResult{
			result: result,
			err:    err,
		}
		close(txnReq.resultChan)
	}

	m.logger.Info("Transaction queue worker stopped", map[string]any{
		"user_id": userID,
	})
}

// Shutdown stops all worker goroutines cleanly
func (m *TransactionManager) Shutdown() {
	m.logger.Info("Shutting down transaction manager", nil)

	// Close all queues to stop workers
	m.userQueues.Range(func(userID, queueIface interface{}) bool {
		if queue, ok := queueIface.(chan *transactionRequest); ok {
			m.logger.Debug("Closing transaction queue for user", map[string]any{
				"user_id": userID,
			})
			close(queue)
		}
		return true
	})

	// Wait for all workers to finish
	m.queueWaitGroup.Wait()
	m.logger.Info("Transaction manager shut down successfully", nil)
}
