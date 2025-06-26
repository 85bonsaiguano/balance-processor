package migration

import (
	"context"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
)

// CreateDefaultUsers creates the default users as specified in the user use case
func CreateDefaultUsers(ctx context.Context, userUseCase usecase.UserUseCase) error {
	// Use the domain usecase method to create default users with predefined balances
	return userUseCase.CreateDefaultUsers(ctx)
}
