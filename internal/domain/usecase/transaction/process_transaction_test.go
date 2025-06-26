package transaction

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	domainerrs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	portuse "github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
	mcore "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	mpers "github.com/amirhossein-jamali/balance-processor/mocks/port/persistence"
	muse "github.com/amirhossein-jamali/balance-processor/mocks/port/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Key for transaction context
const txKey contextKey = "tx"

func TestProcessTransaction(t *testing.T) {
	// Setup common test fixtures
	ctx := context.Background()
	userID := uint64(123)
	transactionID := "tx-12345"
	amount := "10.50"
	now := time.Now()

	tests := []struct {
		name               string
		req                portuse.TransactionRequest
		setupMocks         func(*mpers.MockUnitOfWork, *muse.MockUserUseCase, *mpers.MockUserLockRepository, *mpers.MockTransactionRepository, *mcore.MockTimeProvider, *mcore.MockLogger)
		expectedSuccess    bool
		expectedStatusCode int
		expectedError      error
	}{
		{
			name: "Successful Win Transaction",
			req: portuse.TransactionRequest{
				TransactionID: transactionID,
				State:         "win",
				Amount:        amount,
				SourceType:    entity.SourceGame,
			},
			setupMocks: func(uow *mpers.MockUnitOfWork, userUseCase *muse.MockUserUseCase, userLockRepo *mpers.MockUserLockRepository, txRepo *mpers.MockTransactionRepository, timeProvider *mcore.MockTimeProvider, logger *mcore.MockLogger) {
				// Setup for constructor
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)

				// Setup for checking duplicate transaction
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(false, nil)

				// Setup for user lock
				userLockRepo.On("AcquireLock", mock.Anything, userID, mock.AnythingOfType("time.Duration")).Return(nil)
				userLockRepo.On("ReleaseLock", mock.Anything, userID).Return(nil)

				// Setup for database transaction
				txCtx := context.WithValue(ctx, txKey, "mockTransaction")
				uow.On("Begin", mock.Anything).Return(txCtx, nil)
				uow.On("Commit", txCtx).Return(nil)

				// Setup for user existence check and balance update
				// Create a mock time provider for the User constructor
				mockTimeForUser := new(mcore.MockTimeProvider)
				mockTimeForUser.On("Now").Return(now)

				// Create user properly using the constructor
				mockUser, _ := entity.NewUser(userID, "110.50", mockTimeForUser)
				userUseCase.On("UserExists", mock.Anything, userID).Return(true, nil)
				userUseCase.On("ModifyBalance", txCtx, userID, amount, true, transactionID, string(entity.SourceGame)).Return(mockUser, now, nil)

				// Setup for transaction creation
				timeProvider.On("Now").Return(now)

				// Setup transaction repository operations
				txRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Transaction")).Return(nil)
				txRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Transaction")).Return(nil)

				// Logger setup - just accept anything with Maybe() to allow zero or more calls
				logger.On("Info", mock.Anything, mock.Anything).Maybe()
				logger.On("Error", mock.Anything, mock.Anything).Maybe()
				logger.On("Warn", mock.Anything, mock.Anything).Maybe()
				logger.On("Debug", mock.Anything, mock.Anything).Maybe()
			},
			expectedSuccess:    true,
			expectedStatusCode: http.StatusOK,
			expectedError:      nil,
		},
		{
			name: "Duplicate Transaction",
			req: portuse.TransactionRequest{
				TransactionID: transactionID,
				State:         "win",
				Amount:        amount,
				SourceType:    entity.SourceGame,
			},
			setupMocks: func(uow *mpers.MockUnitOfWork, userUseCase *muse.MockUserUseCase, userLockRepo *mpers.MockUserLockRepository, txRepo *mpers.MockTransactionRepository, timeProvider *mcore.MockTimeProvider, logger *mcore.MockLogger) {
				// Setup for constructor
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)

				// Setup for checking duplicate transaction - return true to indicate duplicate
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(true, nil)

				// Setup user existence
				userUseCase.On("UserExists", mock.Anything, userID).Return(true, nil)

				// Logger setup
				logger.On("Info", mock.Anything, mock.Anything).Maybe()
				logger.On("Warn", mock.Anything, mock.Anything).Maybe()
				logger.On("Debug", mock.Anything, mock.Anything).Maybe()
				logger.On("Error", mock.Anything, mock.Anything).Maybe()
			},
			expectedSuccess:    false,
			expectedStatusCode: http.StatusConflict,
			expectedError:      domainerrs.NewDuplicateTransactionError(transactionID, userID, string(entity.SourceGame)),
		},
		{
			name: "User Not Found",
			req: portuse.TransactionRequest{
				TransactionID: transactionID,
				State:         "win",
				Amount:        amount,
				SourceType:    entity.SourceGame,
			},
			setupMocks: func(uow *mpers.MockUnitOfWork, userUseCase *muse.MockUserUseCase, userLockRepo *mpers.MockUserLockRepository, txRepo *mpers.MockTransactionRepository, timeProvider *mcore.MockTimeProvider, logger *mcore.MockLogger) {
				// Setup for constructor
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)

				// Setup user existence check to return false
				userUseCase.On("UserExists", mock.Anything, userID).Return(false, nil)

				// Logger setup
				logger.On("Info", mock.Anything, mock.Anything).Maybe()
				logger.On("Warn", mock.Anything, mock.Anything).Maybe()
				logger.On("Debug", mock.Anything, mock.Anything).Maybe()
			},
			expectedSuccess:    false,
			expectedStatusCode: http.StatusNotFound,
			expectedError:      domainerrs.ErrUserNotFound,
		},
		{
			name: "Lock Acquisition Failure",
			req: portuse.TransactionRequest{
				TransactionID: transactionID,
				State:         "win",
				Amount:        amount,
				SourceType:    entity.SourceGame,
			},
			setupMocks: func(uow *mpers.MockUnitOfWork, userUseCase *muse.MockUserUseCase, userLockRepo *mpers.MockUserLockRepository, txRepo *mpers.MockTransactionRepository, timeProvider *mcore.MockTimeProvider, logger *mcore.MockLogger) {
				// Setup for constructor
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)

				// Setup for checking duplicate transaction
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(false, nil)

				// Setup user existence
				userUseCase.On("UserExists", mock.Anything, userID).Return(true, nil)

				// Lock acquisition fails
				lockError := errors.New("failed to acquire lock")
				userLockRepo.On("AcquireLock", mock.Anything, userID, mock.AnythingOfType("time.Duration")).Return(lockError)

				// Logger setup
				logger.On("Info", mock.Anything, mock.Anything).Maybe()
				logger.On("Error", mock.Anything, mock.Anything).Maybe()
				logger.On("Warn", mock.Anything, mock.Anything).Maybe()
				logger.On("Debug", mock.Anything, mock.Anything).Maybe()
			},
			expectedSuccess:    false,
			expectedStatusCode: http.StatusConflict,
			expectedError:      errors.New("failed to acquire lock"),
		},
		{
			name: "Insufficient Balance Error",
			req: portuse.TransactionRequest{
				TransactionID: transactionID,
				State:         "lose",
				Amount:        amount,
				SourceType:    entity.SourceGame,
			},
			setupMocks: func(uow *mpers.MockUnitOfWork, userUseCase *muse.MockUserUseCase, userLockRepo *mpers.MockUserLockRepository, txRepo *mpers.MockTransactionRepository, timeProvider *mcore.MockTimeProvider, logger *mcore.MockLogger) {
				// Setup for constructor
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)

				// Setup for checking duplicate transaction
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(false, nil)

				// Setup for user lock
				userLockRepo.On("AcquireLock", mock.Anything, userID, mock.AnythingOfType("time.Duration")).Return(nil)
				userLockRepo.On("ReleaseLock", mock.Anything, userID).Return(nil)

				// Setup user existence
				userUseCase.On("UserExists", mock.Anything, userID).Return(true, nil)

				// Setup database transaction
				txCtx := context.WithValue(ctx, txKey, "mockTransaction")
				uow.On("Begin", mock.Anything).Return(txCtx, nil)
				uow.On("Rollback", txCtx).Return(nil)

				// Setup for transaction creation
				timeProvider.On("Now").Return(now)

				// Create transaction fails with insufficient balance
				insufficientBalanceError := domainerrs.NewInsufficientBalanceError(userID, "5.00", "10.50")
				userUseCase.On("ModifyBalance", txCtx, userID, amount, false, transactionID, string(entity.SourceGame)).Return(nil, time.Time{}, insufficientBalanceError)

				// Setup transaction repository operations
				txRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Transaction")).Return(nil)
				txRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Transaction")).Return(nil)

				// Logger setup
				logger.On("Info", mock.Anything, mock.Anything).Maybe()
				logger.On("Error", mock.Anything, mock.Anything).Maybe()
				logger.On("Warn", mock.Anything, mock.Anything).Maybe()
				logger.On("Debug", mock.Anything, mock.Anything).Maybe()
			},
			expectedSuccess:    false,
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      domainerrs.NewInsufficientBalanceError(userID, "5.00", "10.50"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUow := new(mpers.MockUnitOfWork)
			mockUserUseCase := new(muse.MockUserUseCase)
			mockUserLockRepo := new(mpers.MockUserLockRepository)
			mockTxRepo := new(mpers.MockTransactionRepository)
			mockTimeProvider := new(mcore.MockTimeProvider)
			mockLogger := new(mcore.MockLogger)

			// Configure mocks
			tt.setupMocks(mockUow, mockUserUseCase, mockUserLockRepo, mockTxRepo, mockTimeProvider, mockLogger)

			// Create service
			service := NewTransactionService(
				mockUow,
				mockUserUseCase,
				mockUserLockRepo,
				mockTimeProvider,
				mockLogger,
				5*time.Second, // lockTimeout
			)

			// Call the method
			result, err := service.ProcessTransaction(ctx, userID, tt.req)

			// Assert expected behavior
			if tt.expectedError != nil {
				assert.Error(t, err)
				// For some specific error types, check the error type
				if errors.Is(tt.expectedError, domainerrs.ErrUserNotFound) {
					assert.ErrorIs(t, err, domainerrs.ErrUserNotFound)
				} else if dupErr, ok := tt.expectedError.(*domainerrs.DuplicateTransactionError); ok {
					// For duplicate transaction errors, check if it's the right type
					assert.IsType(t, &domainerrs.DuplicateTransactionError{}, err)
					if actualDupErr, ok := err.(*domainerrs.DuplicateTransactionError); ok {
						assert.Equal(t, dupErr.TransactionID, actualDupErr.TransactionID)
						assert.Equal(t, dupErr.UserID, actualDupErr.UserID)
					}
				} else if _, ok := tt.expectedError.(*domainerrs.InsufficientBalanceError); ok {
					// For insufficient balance errors, check if it's the right type
					assert.True(t, domainerrs.IsInsufficientBalanceError(err))
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify result
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedSuccess, result.Success)
			assert.Equal(t, tt.expectedStatusCode, result.StatusCode)

			// Verify all mocks were called as expected
			mockUow.AssertExpectations(t)
			mockUserUseCase.AssertExpectations(t)
			mockUserLockRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
			mockTimeProvider.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestValidateTransactionRequest(t *testing.T) {
	tests := []struct {
		name          string
		req           portuse.TransactionRequest
		expectedError error
	}{
		{
			name: "Valid Win Request",
			req: portuse.TransactionRequest{
				TransactionID: "tx-123",
				State:         "win",
				Amount:        "10.50",
				SourceType:    entity.SourceGame,
			},
			expectedError: nil,
		},
		{
			name: "Valid Lose Request",
			req: portuse.TransactionRequest{
				TransactionID: "tx-456",
				State:         "lose",
				Amount:        "5.75",
				SourceType:    entity.SourceGame,
			},
			expectedError: nil,
		},
		{
			name: "Missing TransactionID",
			req: portuse.TransactionRequest{
				TransactionID: "",
				State:         "win",
				Amount:        "10.50",
				SourceType:    entity.SourceGame,
			},
			expectedError: domainerrs.ErrInvalidTransactionID,
		},
		{
			name: "Invalid State",
			req: portuse.TransactionRequest{
				TransactionID: "tx-123",
				State:         "invalid",
				Amount:        "10.50",
				SourceType:    entity.SourceGame,
			},
			expectedError: domainerrs.ErrInvalidState,
		},
		{
			name: "Empty Amount",
			req: portuse.TransactionRequest{
				TransactionID: "tx-123",
				State:         "win",
				Amount:        "",
				SourceType:    entity.SourceGame,
			},
			expectedError: domainerrs.ErrInvalidAmount,
		},
		{
			name: "Too Many Decimal Places",
			req: portuse.TransactionRequest{
				TransactionID: "tx-123",
				State:         "win",
				Amount:        "10.509",
				SourceType:    entity.SourceGame,
			},
			expectedError: domainerrs.ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal service just for validation
			mockLogger := new(mcore.MockLogger)
			service := &Service{
				logger: mockLogger,
			}

			// Call the method
			err := service.ValidateTransactionRequest(tt.req)

			// Assert expected behavior
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsDuplicateTransaction(t *testing.T) {
	// Setup
	ctx := context.Background()
	transactionID := "tx-12345"

	tests := []struct {
		name          string
		setupMocks    func(*mpers.MockUnitOfWork, *mpers.MockTransactionRepository)
		expected      bool
		expectedError error
	}{
		{
			name: "Transaction Exists",
			setupMocks: func(uow *mpers.MockUnitOfWork, txRepo *mpers.MockTransactionRepository) {
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(true, nil)
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)
			},
			expected:      true,
			expectedError: nil,
		},
		{
			name: "Transaction Does Not Exist",
			setupMocks: func(uow *mpers.MockUnitOfWork, txRepo *mpers.MockTransactionRepository) {
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(false, nil)
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)
			},
			expected:      false,
			expectedError: nil,
		},
		{
			name: "Repository Error",
			setupMocks: func(uow *mpers.MockUnitOfWork, txRepo *mpers.MockTransactionRepository) {
				repoErr := errors.New("database error")
				txRepo.On("TransactionExists", mock.Anything, transactionID).Return(false, repoErr)
				uow.On("GetTransactionRepository", mock.Anything).Return(txRepo)
			},
			expected:      false,
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUow := new(mpers.MockUnitOfWork)
			mockTxRepo := new(mpers.MockTransactionRepository)

			// Configure mocks
			tt.setupMocks(mockUow, mockTxRepo)

			// Create service
			mockLogger := new(mcore.MockLogger)
			service := &Service{
				uow:    mockUow,
				logger: mockLogger,
			}

			// Call the method
			result, err := service.IsDuplicateTransaction(ctx, transactionID)

			// Assert expected behavior
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)

			// Verify all mocks were called as expected
			mockUow.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
		})
	}
}
