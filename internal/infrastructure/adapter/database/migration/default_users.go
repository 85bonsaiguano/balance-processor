package migration

import (
	"context"

	userUseCase "github.com/amirhossein-jamali/balance-processor/internal/domain/usecase/user"
)

// Default user IDs and balances
var defaultUsers = map[uint64]string{
	1: "100.00",
	2: "200.00",
	3: "300.00",
}

// CreateDefaultUsers creates the default users with predefined balances
func CreateDefaultUsers(ctx context.Context, userService *userUseCase.UserUseCase) error {
	for userID, balance := range defaultUsers {
		// Check if user exists
		exists, err := userService.UserExists(ctx, userID)
		if err != nil {
			return err
		}

		if !exists {
			// Create user if it doesn't exist
			if err := userService.CreateUser(ctx, userID, balance); err != nil {
				return err
			}
		}
	}

	return nil
}
