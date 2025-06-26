package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	"github.com/amirhossein-jamali/balance-processor/mocks/port/persistence"
)

func TestUserUseCase_ModifyBalance(t *testing.T) {
	// Define fixed time for consistent testing
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Common test variables
	ctx := context.Background()
	userID := uint64(123)
	validAmount := "50.00"
	transactionID := "tx-123456"
	sourceType := "game"

	t.Run("should add to balance on win transaction", func(t *testing.T) {
		// Arrange
		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure time provider mock
		mockTimeProvider.On("Now").Return(fixedTime)

		// Create test user with initial balance
		user := &entity.User{
			ID: userID,
		}
		user.SetBalance(10000, mockTimeProvider) // 100.00 in cents

		// Configure mock expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockLogger.On("Info", "User balance modified", mock.AnythingOfType("map[string]interface {}")).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, validAmount, true, transactionID, sourceType)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, "150.00", updatedUser.GetBalance()) // 100.00 + 50.00 = 150.00
		assert.Equal(t, fixedTime, txTime)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockTimeProvider.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("should subtract from balance on lose transaction", func(t *testing.T) {
		// Arrange
		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure time provider mock
		mockTimeProvider.On("Now").Return(fixedTime)

		// Create test user with initial balance
		user := &entity.User{
			ID: userID,
		}
		user.SetBalance(10000, mockTimeProvider) // 100.00 in cents

		// Configure mock expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
		mockLogger.On("Info", "User balance modified", mock.AnythingOfType("map[string]interface {}")).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, validAmount, false, transactionID, sourceType)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, "50.00", updatedUser.GetBalance()) // 100.00 - 50.00 = 50.00
		assert.Equal(t, fixedTime, txTime)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockTimeProvider.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("should return error with invalid user ID", func(t *testing.T) {
		// Arrange
		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, 0, validAmount, true, transactionID, sourceType)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.True(t, txTime.IsZero())
		assert.ErrorIs(t, err, errs.ErrInvalidUserID)

		// Verify no repository calls were made
		mockUserRepo.AssertNotCalled(t, "GetByID")
		mockUserRepo.AssertNotCalled(t, "Update")
	})

	t.Run("should return error with invalid amount format", func(t *testing.T) {
		// Arrange
		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, "invalid-amount", true, transactionID, sourceType)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.True(t, txTime.IsZero())

		// Verify no repository calls were made
		mockUserRepo.AssertNotCalled(t, "GetByID")
		mockUserRepo.AssertNotCalled(t, "Update")
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		// Arrange
		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure mock expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(nil, errs.ErrUserNotFound)
		mockLogger.On("Warn", "Attempt to modify balance of non-existent user", mock.AnythingOfType("map[string]interface {}")).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, validAmount, true, transactionID, sourceType)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.True(t, txTime.IsZero())
		assert.ErrorIs(t, err, errs.ErrUserNotFound)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("should return error on database failure when getting user", func(t *testing.T) {
		// Arrange
		dbError := errors.New("database connection error")

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure mock expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(nil, dbError)
		mockLogger.On("Error", "Failed to get user", mock.AnythingOfType("map[string]interface {}")).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, validAmount, true, transactionID, sourceType)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.True(t, txTime.IsZero())
		assert.Equal(t, dbError, err)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("should return error on insufficient balance for lose transaction", func(t *testing.T) {
		// Arrange
		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure time provider mock
		mockTimeProvider.On("Now").Return(fixedTime)

		// Create test user with insufficient balance
		user := &entity.User{
			ID: userID,
		}
		user.SetBalance(2000, mockTimeProvider) // 20.00 in cents (less than 50.00)

		// Configure mock expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, validAmount, false, transactionID, sourceType)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.True(t, txTime.IsZero())
		// Check that it's an insufficient balance error
		var insufficientErr *errs.InsufficientBalanceError
		assert.True(t, errors.As(err, &insufficientErr))

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockTimeProvider.AssertExpectations(t)
	})

	t.Run("should return error on database failure when updating user", func(t *testing.T) {
		// Arrange
		dbError := errors.New("database update error")

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure time provider mock
		mockTimeProvider.On("Now").Return(fixedTime)

		// Create test user
		user := &entity.User{
			ID: userID,
		}
		user.SetBalance(10000, mockTimeProvider) // 100.00 in cents

		// Configure mock expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(dbError)
		mockLogger.On("Error", "Failed to update user balance", mock.AnythingOfType("map[string]interface {}")).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		updatedUser, txTime, err := useCase.ModifyBalance(ctx, userID, validAmount, true, transactionID, sourceType)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.True(t, txTime.IsZero())
		assert.Equal(t, dbError, err)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockTimeProvider.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}
