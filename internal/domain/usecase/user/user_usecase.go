package user

import (
	"context"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
)

// Default user IDs and balances
var defaultUsers = map[uint64]string{
	1: "100.00",
	2: "200.00",
	3: "300.00",
}

// UserUseCase handles user-related business logic
type UserUseCase struct {
	userRepo     persistence.UserRepository
	timeProvider coreport.TimeProvider
	logger       coreport.Logger
}

// NewUserUseCase creates a new UserUseCase
func NewUserUseCase(
	userRepo persistence.UserRepository,
	timeProvider coreport.TimeProvider,
	logger coreport.Logger,
) *UserUseCase {
	return &UserUseCase{
		userRepo:     userRepo,
		timeProvider: timeProvider,
		logger:       logger,
	}
}

// GetUserBalance returns a user's balance
func (u *UserUseCase) GetUserBalance(ctx context.Context, userID uint64) (string, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.GetBalance(), nil
}

// UserExists checks if a user with the given ID exists
func (u *UserUseCase) UserExists(ctx context.Context, userID uint64) (bool, error) {
	_, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == errs.ErrUserNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateUser creates a new user with the given ID and initial balance
func (u *UserUseCase) CreateUser(ctx context.Context, userID uint64, initialBalance string) error {
	// Check if user already exists
	exists, err := u.UserExists(ctx, userID)
	if err != nil {
		return err
	}
	if exists {
		return errs.ErrDuplicateUser
	}

	// Create new user entity
	user, err := entity.NewUser(userID, initialBalance, u.timeProvider)
	if err != nil {
		return err
	}

	// Save user to repository
	return u.userRepo.Create(ctx, user)
}

// CreateDefaultUsers creates the default users with predefined balances
func (u *UserUseCase) CreateDefaultUsers(ctx context.Context) error {
	for userID, balance := range defaultUsers {
		// Check if user exists
		exists, err := u.UserExists(ctx, userID)
		if err != nil {
			return err
		}

		if !exists {
			// Create user if it doesn't exist
			if err := u.CreateUser(ctx, userID, balance); err != nil {
				if err != errs.ErrDuplicateUser {
					return err
				}
			}
		}
	}

	return nil
}

// GetBalanceResponse represents a user balance response
type GetBalanceResponse struct {
	UserID  uint64
	Balance string
}

// GetBalance returns a user's balance as a response object
func (u *UserUseCase) GetBalance(ctx context.Context, userID uint64) (*GetBalanceResponse, error) {
	balance, err := u.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &GetBalanceResponse{
		UserID:  userID,
		Balance: balance,
	}, nil
}
