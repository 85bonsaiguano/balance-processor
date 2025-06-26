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

func TestUserUseCase_GetFormattedUserBalance(t *testing.T) {
	// Define fixed time for consistent testing
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("should return formatted balance for valid user", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(123)

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure mocks
		mockTimeProvider.On("Now").Return(fixedTime)

		// Create test user
		user := &entity.User{
			ID: userID, // Set the user ID to match
		}
		user.SetBalance(12345, mockTimeProvider) // 123.45 in cents

		// Setup expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockLogger.On("Info", "User balance retrieved", mock.Anything).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		response, err := useCase.GetFormattedUserBalance(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, userID, response.UserID)
		assert.Equal(t, "123.45", response.Balance)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("should return error with invalid user ID", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(0) // Invalid ID

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		response, err := useCase.GetFormattedUserBalance(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, errs.ErrInvalidUserID)

		// Verify no repository calls were made
		mockUserRepo.AssertNotCalled(t, "GetByID")
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(999) // Non-existent user

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Setup expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(nil, errs.ErrUserNotFound)
		mockLogger.On("Error", "Failed to get user", mock.Anything).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		response, err := useCase.GetFormattedUserBalance(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, errs.ErrUserNotFound)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(123)
		dbError := errors.New("database connection error")

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Setup expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(nil, dbError)
		mockLogger.On("Error", "Failed to get user", mock.Anything).Return()

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		response, err := useCase.GetFormattedUserBalance(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, dbError, err)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}

func TestUserUseCase_UserExists(t *testing.T) {
	// Define fixed time for consistent testing
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("should return true when user exists", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(123)

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Configure mocks
		mockTimeProvider.On("Now").Return(fixedTime)

		// Create test user
		user := &entity.User{}

		// Setup expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		exists, err := useCase.UserExists(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.True(t, exists)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("should return false when user does not exist", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(999) // Non-existent user

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Setup expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(nil, errs.ErrUserNotFound)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		exists, err := useCase.UserExists(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.False(t, exists)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("should return error with invalid user ID", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(0) // Invalid ID

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		exists, err := useCase.UserExists(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.False(t, exists)
		assert.ErrorIs(t, err, errs.ErrInvalidUserID)

		// Verify no repository calls were made
		mockUserRepo.AssertNotCalled(t, "GetByID")
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		userID := uint64(123)
		dbError := errors.New("database connection error")

		// Create mocks
		mockUserRepo := new(persistence.MockUserRepository)
		mockTimeProvider := new(core.MockTimeProvider)
		mockLogger := new(core.MockLogger)

		// Setup expectations
		mockUserRepo.On("GetByID", ctx, userID).Return(nil, dbError)

		// Create the use case with mocked dependencies
		useCase := NewUserUseCase(mockUserRepo, mockTimeProvider, mockLogger)

		// Act
		exists, err := useCase.UserExists(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Equal(t, dbError, err)

		// Verify mocks
		mockUserRepo.AssertExpectations(t)
	})
}
