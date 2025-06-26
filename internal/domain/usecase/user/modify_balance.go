package user

import (
	"context"
	"errors"
	"time"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
)

// ModifyBalance handles both addition and deduction with unified interface
// This is used by the transaction processing flow to update user balance
func (u *UserUseCase) ModifyBalance(
	ctx context.Context,
	userID uint64,
	amount string,
	isWin bool,
	transactionID string,
	sourceType string,
) (*entity.User, time.Time, error) {
	// Validate userID
	if userID == 0 {
		return nil, time.Time{}, errs.ErrInvalidUserID
	}

	// Validate amount format
	amountInCents, err := entity.ValidateAndConvertAmount(amount)
	if err != nil {
		return nil, time.Time{}, err
	}

	// Get the user
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			u.logger.Warn("Attempt to modify balance of non-existent user", map[string]any{
				"userId":        userID,
				"transactionId": transactionID,
				"sourceType":    sourceType,
			})
		} else {
			u.logger.Error("Failed to get user", map[string]any{
				"userId":     userID,
				"error":      err.Error(),
				"sourceType": sourceType,
			})
		}
		return nil, time.Time{}, err
	}

	transactionTime := u.timeProvider.Now()

	// Apply the balance change based on transaction type
	if isWin {
		// Win transaction - add to balance
		user.ApplyWinTransaction(amountInCents, u.timeProvider)
	} else {
		// Lose transaction - deduct from balance
		// This will return an error if insufficient balance
		if err := user.ApplyLoseTransaction(amountInCents, u.timeProvider); err != nil {
			// Create a detailed error with balance info
			detailedErr := errs.NewInsufficientBalanceError(
				user.ID,
				amount,
				user.GetBalance(),
			)
			return nil, time.Time{}, detailedErr
		}
	}

	// Update the user in the database
	if err := u.userRepo.Update(ctx, user); err != nil {
		u.logger.Error("Failed to update user balance", map[string]any{
			"userId":        userID,
			"transactionId": transactionID,
			"error":         err.Error(),
			"sourceType":    sourceType,
		})
		return nil, time.Time{}, err
	}

	// Log the successful balance modification
	u.logger.Info("User balance modified", map[string]any{
		"userId":        userID,
		"transactionId": transactionID,
		"isWin":         isWin,
		"amount":        amount,
		"newBalance":    user.GetBalance(),
		"sourceType":    sourceType,
	})

	return user, transactionTime, nil
}
