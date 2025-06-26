package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coremocks "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	persistencemocks "github.com/amirhossein-jamali/balance-processor/mocks/port/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("Successful user creation", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Setup expectations
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(123)).Return(nil, errs.ErrUserNotFound).Once()
		mockTime.EXPECT().Now().Return(fixedTime).Maybe()
		mockRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
			return user.ID == 123 && user.GetBalance() == "100.00"
		})).Return(nil).Once()

		mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Once()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		user, err := userUseCase.CreateUser(ctx, 123, "100.00")

		// Assertions
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, uint64(123), user.ID)
		assert.Equal(t, "100.00", user.GetBalance())
	})

	t.Run("Invalid user ID", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		user, err := userUseCase.CreateUser(ctx, 0, "100.00")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, errs.ErrInvalidUserID, err)
	})

	t.Run("Invalid balance format", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		user, err := userUseCase.CreateUser(ctx, 123, "invalid")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("User already exists", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Setup expectations
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(123)).Return(&entity.User{}, nil).Once()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		user, err := userUseCase.CreateUser(ctx, 123, "100.00")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, errs.ErrDuplicateUser, err)
	})

	t.Run("Error checking if user exists", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Setup expectations
		databaseError := errors.New("database connection error")
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(123)).Return(nil, databaseError).Once()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		user, err := userUseCase.CreateUser(ctx, 123, "100.00")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, databaseError, err)
	})

	t.Run("Error creating user in repository", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Setup expectations
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(123)).Return(nil, errs.ErrUserNotFound).Once()
		mockTime.EXPECT().Now().Return(fixedTime).Maybe()

		databaseError := errors.New("database insert error")
		mockRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
			return user.ID == 123
		})).Return(databaseError).Once()

		mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Once()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		user, err := userUseCase.CreateUser(ctx, 123, "100.00")

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, databaseError, err)
	})
}

func TestCreateDefaultUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully create all default users", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Each user will be looked up twice - once in UserExists and once in CreateUser
		// First call for each user ID is in the main loop in CreateDefaultUsers -> UserExists
		for i := 1; i <= 5; i++ {
			mockRepo.EXPECT().GetByID(mock.Anything, uint64(i)).Return(nil, errs.ErrUserNotFound).Once()
		}

		// Second call for each user ID is in CreateUser
		for i := 1; i <= 5; i++ {
			mockRepo.EXPECT().GetByID(mock.Anything, uint64(i)).Return(nil, errs.ErrUserNotFound).Once()
		}

		// Setup expectations for creating each user
		mockRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
			return user.ID >= 1 && user.ID <= 5
		})).Return(nil).Times(5)

		mockTime.EXPECT().Now().Return(time.Now()).Maybe()
		mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		err := userUseCase.CreateDefaultUsers(ctx)

		// Assertions
		assert.NoError(t, err)
	})

	t.Run("Skip users that already exist", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Setup expectations - users 1, 3 and 5 exist, 2 and 4 don't
		// These are the first calls from UserExists in CreateDefaultUsers
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(1)).Return(&entity.User{}, nil).Once()
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(2)).Return(nil, errs.ErrUserNotFound).Once()
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(3)).Return(&entity.User{}, nil).Once()
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(4)).Return(nil, errs.ErrUserNotFound).Once()
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(5)).Return(&entity.User{}, nil).Once()

		// For users 2 and 4, they don't exist, so will be created, which means GetByID
		// is called again inside CreateUser
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(2)).Return(nil, errs.ErrUserNotFound).Once()
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(4)).Return(nil, errs.ErrUserNotFound).Once()

		// Setup expectations for creating users 2 and 4
		mockRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
			return user.ID == 2 || user.ID == 4
		})).Return(nil).Times(2)

		mockTime.EXPECT().Now().Return(time.Now()).Maybe()
		mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		err := userUseCase.CreateDefaultUsers(ctx)

		// Assertions
		assert.NoError(t, err)
	})

	t.Run("Error checking if user exists", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// Setup expectations - error on first user check
		databaseError := errors.New("database error")
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(1)).Return(nil, databaseError).Once()

		mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		err := userUseCase.CreateDefaultUsers(ctx)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, databaseError, err)
	})

	t.Run("Error creating a user", func(t *testing.T) {
		// Setup mocks
		mockRepo := persistencemocks.NewMockUserRepository(t)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockLogger := coremocks.NewMockLogger(t)

		// First call in UserExists
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(1)).Return(nil, errs.ErrUserNotFound).Once()
		// Second call inside CreateUser
		mockRepo.EXPECT().GetByID(mock.Anything, uint64(1)).Return(nil, errs.ErrUserNotFound).Once()

		// Setup expectations for creating first user - with error
		databaseError := errors.New("database insert error")
		mockRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
			return user.ID == 1
		})).Return(databaseError).Once()

		mockTime.EXPECT().Now().Return(time.Now()).Maybe()
		mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
		mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

		// Create use case instance
		userUseCase := NewUserUseCase(mockRepo, mockTime, mockLogger)

		// Execute
		err := userUseCase.CreateDefaultUsers(ctx)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, databaseError, err)
	})
}
