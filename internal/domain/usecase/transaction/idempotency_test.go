package transaction

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mockcore "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	mockpersistence "github.com/amirhossein-jamali/balance-processor/mocks/port/persistence"
	mockusecase "github.com/amirhossein-jamali/balance-processor/mocks/port/usecase"
)

func TestService_IsDuplicateTransaction(t *testing.T) {
	// Define test parameters
	testCases := []struct {
		name           string
		transactionID  string
		mockSetup      func(mockUow *mockpersistence.MockUnitOfWork, mockTxnRepo *mockpersistence.MockTransactionRepository)
		expectedResult bool
		expectedError  error
	}{
		{
			name:          "transaction exists",
			transactionID: "existing-transaction-id",
			mockSetup: func(mockUow *mockpersistence.MockUnitOfWork, mockTxnRepo *mockpersistence.MockTransactionRepository) {
				mockUow.EXPECT().GetTransactionRepository(mock.Anything).Return(mockTxnRepo)
				mockTxnRepo.EXPECT().TransactionExists(mock.Anything, "existing-transaction-id").Return(true, nil)
			},
			expectedResult: true,
			expectedError:  nil,
		},
		{
			name:          "transaction does not exist",
			transactionID: "new-transaction-id",
			mockSetup: func(mockUow *mockpersistence.MockUnitOfWork, mockTxnRepo *mockpersistence.MockTransactionRepository) {
				mockUow.EXPECT().GetTransactionRepository(mock.Anything).Return(mockTxnRepo)
				mockTxnRepo.EXPECT().TransactionExists(mock.Anything, "new-transaction-id").Return(false, nil)
			},
			expectedResult: false,
			expectedError:  nil,
		},
		{
			name:          "database error",
			transactionID: "any-transaction-id",
			mockSetup: func(mockUow *mockpersistence.MockUnitOfWork, mockTxnRepo *mockpersistence.MockTransactionRepository) {
				mockUow.EXPECT().GetTransactionRepository(mock.Anything).Return(mockTxnRepo)
				mockTxnRepo.EXPECT().TransactionExists(mock.Anything, "any-transaction-id").Return(false, errors.New("database connection error"))
			},
			expectedResult: false,
			expectedError:  errors.New("database connection error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockUow := new(mockpersistence.MockUnitOfWork)
			mockUserUseCase := new(mockusecase.MockUserUseCase)
			mockUserLockRepo := new(mockpersistence.MockUserLockRepository)
			mockTimeProvider := new(mockcore.MockTimeProvider)
			mockLogger := new(mockcore.MockLogger)
			mockTxnRepo := new(mockpersistence.MockTransactionRepository)

			// Setup logger mock expectations
			mockLogger.On("Info", "Transaction processor initialized", mock.Anything).Return()

			// Apply mock configurations for each test case
			tc.mockSetup(mockUow, mockTxnRepo)

			// Initialize the service to be tested
			service := NewTransactionService(
				mockUow,
				mockUserUseCase,
				mockUserLockRepo,
				mockTimeProvider,
				mockLogger,
				0, // lockTimeout value doesn't matter for this test
			).(*Service)

			// Call the method being tested
			result, err := service.IsDuplicateTransaction(context.Background(), tc.transactionID)

			// Check results
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedResult, result)

			// Ensure all mock expectations were met
			mockUow.AssertExpectations(t)
			mockTxnRepo.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// This additional test ensures that when an error occurs in TransactionExists method call,
// the boolean return value is false
func TestService_IsDuplicateTransaction_ErrorHandling(t *testing.T) {
	// Setup mocks
	mockUow := new(mockpersistence.MockUnitOfWork)
	mockUserUseCase := new(mockusecase.MockUserUseCase)
	mockUserLockRepo := new(mockpersistence.MockUserLockRepository)
	mockTimeProvider := new(mockcore.MockTimeProvider)
	mockLogger := new(mockcore.MockLogger)
	mockTxnRepo := new(mockpersistence.MockTransactionRepository)

	// Setup logger mock expectations
	mockLogger.On("Info", "Transaction processor initialized", mock.Anything).Return()

	// Define mock behaviors
	mockUow.On("GetTransactionRepository", mock.Anything).Return(mockTxnRepo)
	mockTxnRepo.On("TransactionExists", mock.Anything, "error-transaction-id").Return(false, errors.New("unexpected database error"))

	// Initialize the service
	service := NewTransactionService(
		mockUow,
		mockUserUseCase,
		mockUserLockRepo,
		mockTimeProvider,
		mockLogger,
		0,
	).(*Service)

	// Call the method being tested
	result, err := service.IsDuplicateTransaction(context.Background(), "error-transaction-id")

	// Check results
	assert.Error(t, err)
	assert.Equal(t, "unexpected database error", err.Error())
	assert.False(t, result, "When an error occurs, the method should return false")

	// Ensure all mock expectations were met
	mockUow.AssertExpectations(t)
	mockTxnRepo.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}
