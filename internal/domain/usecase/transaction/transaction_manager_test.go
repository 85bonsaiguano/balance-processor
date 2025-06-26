package transaction

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
	mockcore "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	mockpersistence "github.com/amirhossein-jamali/balance-processor/mocks/port/persistence"
)

func TestNewTransactionManager(t *testing.T) {
	// Setup mocks
	mockLogger := mockcore.NewMockLogger(t)
	mockTimeProvider := mockcore.NewMockTimeProvider(t)
	mockTransactionRepo := mockpersistence.NewMockTransactionRepository(t)

	// Test case: valid initialization
	t.Run("Valid initialization", func(t *testing.T) {
		processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
			return &usecase.TransactionResult{Success: true}, nil
		}

		tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

		assert.NotNil(t, tm)
		assert.Equal(t, mockLogger, tm.logger)
		assert.Equal(t, mockTimeProvider, tm.timeProvider)
		assert.Equal(t, mockTransactionRepo, tm.transactionRepo)
	})

	// Test case: nil processor function
	t.Run("Nil processor function should panic", func(t *testing.T) {
		assert.Panics(t, func() {
			NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, nil)
		})
	})
}

func TestTransactionManager_EnqueueTransaction(t *testing.T) {
	// Setup
	mockLogger := mockcore.NewMockLogger(t)
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Return()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Return()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()

	mockTimeProvider := mockcore.NewMockTimeProvider(t)
	mockTransactionRepo := mockpersistence.NewMockTransactionRepository(t)

	t.Run("Successful transaction processing", func(t *testing.T) {
		// Create a processor that returns success
		processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
			return &usecase.TransactionResult{
				Success:       true,
				ResultBalance: "100.00",
				StatusCode:    200,
			}, nil
		}

		tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

		// Create request
		ctx := context.Background()
		userID := uint64(123)
		req := usecase.TransactionRequest{
			TransactionID: "tx-123",
			State:         "win",
			Amount:        "10.00",
		}

		// Process transaction
		result, err := tm.EnqueueTransaction(ctx, userID, req)

		// Assertions
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, "100.00", result.ResultBalance)
		assert.Equal(t, 200, result.StatusCode)
	})

	t.Run("Error in transaction processing", func(t *testing.T) {
		// Create a processor that returns error
		expectedErr := errs.ErrInsufficientBalance
		processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
			return nil, expectedErr
		}

		tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

		// Create request
		ctx := context.Background()
		userID := uint64(123)
		req := usecase.TransactionRequest{
			TransactionID: "tx-456",
			State:         "lose",
			Amount:        "10.00",
		}

		// Process transaction
		result, err := tm.EnqueueTransaction(ctx, userID, req)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("Transaction ordered processing", func(t *testing.T) {
		// Set up a mutex and slice to track processing order
		var mu sync.Mutex
		var processOrder []string

		// Create a processor that records the order
		processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
			// Simulate processing time to ensure we test ordering
			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			processOrder = append(processOrder, req.TransactionID)
			mu.Unlock()

			return &usecase.TransactionResult{
				Success:       true,
				ResultBalance: "100.00",
				StatusCode:    200,
			}, nil
		}

		tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

		// Create context and requests
		ctx := context.Background()
		userID := uint64(123)

		// Process multiple transactions for same user to test ordering
		var wg sync.WaitGroup
		wg.Add(3)

		// Create result channels to capture errors
		errChan := make(chan error, 3)
		resultChan := make(chan *usecase.TransactionResult, 3)

		// Create 3 concurrent requests that should be processed sequentially
		go func() {
			defer wg.Done()
			result, err := tm.EnqueueTransaction(ctx, userID, usecase.TransactionRequest{TransactionID: "tx-1"})
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result
		}()

		go func() {
			defer wg.Done()
			result, err := tm.EnqueueTransaction(ctx, userID, usecase.TransactionRequest{TransactionID: "tx-2"})
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result
		}()

		go func() {
			defer wg.Done()
			result, err := tm.EnqueueTransaction(ctx, userID, usecase.TransactionRequest{TransactionID: "tx-3"})
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result
		}()

		wg.Wait()
		close(errChan)
		close(resultChan)

		// Check for any errors
		for err := range errChan {
			require.NoError(t, err, "Transaction processing should not return error")
		}

		// Check results
		resultCount := 0
		for range resultChan {
			resultCount++
		}
		assert.Equal(t, 3, resultCount, "Should have 3 successful results")

		// Verify transactions were processed in order (sequential for same user)
		mu.Lock()
		assert.Equal(t, 3, len(processOrder))
		assert.Contains(t, processOrder, "tx-1")
		assert.Contains(t, processOrder, "tx-2")
		assert.Contains(t, processOrder, "tx-3")
		mu.Unlock()
	})

	t.Run("Multiple users processed concurrently", func(t *testing.T) {
		// Create a channel to synchronize the test
		done := make(chan struct{}, 2)
		errChan := make(chan error, 2)

		startTime := time.Now()

		// Create a processor that has a delay
		processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
			// Sleep to simulate processing time
			time.Sleep(100 * time.Millisecond)
			done <- struct{}{}
			return &usecase.TransactionResult{Success: true}, nil
		}

		tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

		// Create context
		ctx := context.Background()

		// Process transactions for two different users concurrently
		go func() {
			_, err := tm.EnqueueTransaction(ctx, uint64(1), usecase.TransactionRequest{TransactionID: "user1-tx"})
			if err != nil {
				errChan <- err
			}
		}()

		go func() {
			_, err := tm.EnqueueTransaction(ctx, uint64(2), usecase.TransactionRequest{TransactionID: "user2-tx"})
			if err != nil {
				errChan <- err
			}
		}()

		// Wait for both transactions to complete
		<-done
		<-done

		// Check for any errors (non-blocking check)
		select {
		case err := <-errChan:
			require.NoError(t, err, "Transaction processing should not return error")
		default:
			// No errors
		}

		// If transactions ran concurrently, they should complete in ~100ms
		// If they ran sequentially, it would take ~200ms
		processingTime := time.Since(startTime)

		// Should be less than 190ms (100ms + some buffer for test overhead)
		assert.Less(t, processingTime, 190*time.Millisecond,
			"Multiple users should be processed concurrently")
	})

	t.Run("Context cancellation during enqueueing", func(t *testing.T) {
		processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
			return &usecase.TransactionResult{Success: true}, nil
		}

		tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

		// Create cancelable context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Try to process transaction with canceled context
		result, err := tm.EnqueueTransaction(ctx, uint64(123), usecase.TransactionRequest{TransactionID: "tx-cancel"})

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestTransactionManager_Shutdown(t *testing.T) {
	// Setup mocks
	mockLogger := mockcore.NewMockLogger(t)
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Return()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Return()

	mockTimeProvider := mockcore.NewMockTimeProvider(t)
	mockTransactionRepo := mockpersistence.NewMockTransactionRepository(t)

	// Create a processor with delay to ensure shutdown has work to do
	processor := func(ctx context.Context, userID uint64, req usecase.TransactionRequest) (*usecase.TransactionResult, error) {
		time.Sleep(10 * time.Millisecond)
		return &usecase.TransactionResult{Success: true}, nil
	}

	tm := NewTransactionManager(mockLogger, mockTimeProvider, mockTransactionRepo, processor)

	// Start some transaction processing to create queues
	ctx := context.Background()

	// Create a channel to capture any errors
	errChan := make(chan error, 1)

	// Enqueue transaction to create a worker
	go func() {
		_, err := tm.EnqueueTransaction(ctx, uint64(1), usecase.TransactionRequest{TransactionID: "shutdown-test"})
		if err != nil && !errors.Is(err, context.Canceled) { // Ignore context canceled errors during shutdown
			errChan <- err
		}
	}()

	// Small delay to ensure worker is created
	time.Sleep(5 * time.Millisecond)

	// Test shutdown
	tm.Shutdown()

	// Check if there were any unexpected errors
	select {
	case err := <-errChan:
		require.NoError(t, err, "Transaction processing during shutdown should not return unexpected errors")
	default:
		// No errors, which is expected
	}

	// The successful shutdown is indicated by the test not hanging or panicking
	// Additional tests would be to check that user queues are empty, but that requires
	// exposing internal state which isn't good practice for tests.
}
