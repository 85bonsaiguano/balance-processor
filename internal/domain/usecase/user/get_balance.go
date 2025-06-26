package user

import (
	"context"
	"errors"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
)

// UserUseCase implements the user business logic
type UserUseCase struct {
	userRepo     persistence.UserRepository
	timeProvider coreport.TimeProvider
	logger       coreport.Logger
}

// NewUserUseCase creates a new user use case instance
func NewUserUseCase(
	userRepo persistence.UserRepository,
	timeProvider coreport.TimeProvider,
	logger coreport.Logger,
) usecase.UserUseCase {
	return &UserUseCase{
		userRepo:     userRepo,
		timeProvider: timeProvider,
		logger:       logger,
	}
}

// GetFormattedUserBalance retrieves a user's balance and returns it in the standardized format
func (u *UserUseCase) GetFormattedUserBalance(ctx context.Context, userID uint64) (*usecase.UserBalanceResponse, error) {
	// Validate userID
	if userID == 0 {
		return nil, errs.ErrInvalidUserID
	}

	// Get the user from the repository
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		u.logger.Error("Failed to get user", map[string]any{
			"userId": userID,
			"error":  err.Error(),
		})
		return nil, err
	}

	// Format the response with properly formatted balance
	response := &usecase.UserBalanceResponse{
		UserID:  user.ID,
		Balance: user.GetBalance(), // Uses the entity's formatting logic
	}

	u.logger.Info("User balance retrieved", map[string]any{
		"userId":  userID,
		"balance": response.Balance,
	})

	return response, nil
}

// UserExists checks if a user exists with the given ID
func (u *UserUseCase) UserExists(ctx context.Context, userID uint64) (bool, error) {
	// Validate userID
	if userID == 0 {
		return false, errs.ErrInvalidUserID
	}

	// Try to get the user
	_, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		// If it's a not found error, return false without an error
		if errors.Is(err, errs.ErrUserNotFound) {
			return false, nil
		}
		// Return any other error
		return false, err
	}

	// User was found
	return true, nil
}
