package user

import (
	"context"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
)

// CreateUser creates a new user with the given ID and initial balance
func (u *UserUseCase) CreateUser(ctx context.Context, id uint64, initialBalance string) (*entity.User, error) {
	// Validate userID
	if id == 0 {
		return nil, errs.ErrInvalidUserID
	}

	// Validate initial balance
	if _, err := entity.ValidateAndConvertAmount(initialBalance); err != nil {
		return nil, err
	}

	// Check if user already exists
	exists, err := u.UserExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.ErrDuplicateUser
	}

	// Create new user entity
	user, err := entity.NewUser(id, initialBalance, u.timeProvider)
	if err != nil {
		return nil, err
	}

	// Save the user to the database
	if err := u.userRepo.Create(ctx, user); err != nil {
		u.logger.Error("Failed to create user", map[string]any{
			"userId": id,
			"error":  err.Error(),
		})
		return nil, err
	}

	u.logger.Info("User created", map[string]any{
		"userId":         id,
		"initialBalance": initialBalance,
	})

	return user, nil
}

// CreateDefaultUsers creates predefined users with IDs 1, 2, 3, ... as required by the task
func (u *UserUseCase) CreateDefaultUsers(ctx context.Context) error {
	defaultUsers := []struct {
		id      uint64
		balance string
	}{
		{id: 1, balance: "100.00"},
		{id: 2, balance: "200.00"},
		{id: 3, balance: "300.00"},
		{id: 4, balance: "400.00"},
		{id: 5, balance: "500.00"},
	}

	for _, defaultUser := range defaultUsers {
		// Check if user already exists
		exists, err := u.UserExists(ctx, defaultUser.id)
		if err != nil {
			return err
		}

		// Skip if user already exists
		if exists {
			u.logger.Info("Default user already exists", map[string]any{
				"userId": defaultUser.id,
			})
			continue
		}

		// Create the user
		_, err = u.CreateUser(ctx, defaultUser.id, defaultUser.balance)
		if err != nil {
			return err
		}
	}

	u.logger.Info("Default users created or verified", nil)
	return nil
}
