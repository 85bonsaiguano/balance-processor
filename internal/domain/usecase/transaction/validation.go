package transaction

import (
	"fmt"
	"strings"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
)

// TransactionValidator provides validation for transaction requests
type TransactionValidator struct{}

// NewTransactionValidator creates a new TransactionValidator
func NewTransactionValidator() *TransactionValidator {
	return &TransactionValidator{}
}

// ValidateTransaction validates all transaction fields
func (v *TransactionValidator) ValidateTransaction(
	userID uint64,
	transactionID string,
	sourceType string,
	state string,
	amount string,
) error {
	// Validate User ID
	if userID == 0 {
		return errs.ErrInvalidUserID
	}

	// Validate Transaction ID
	if err := v.validateTransactionID(transactionID); err != nil {
		return err
	}

	// Validate Source Type
	if err := v.validateSourceType(sourceType); err != nil {
		return err
	}

	// Validate State
	if err := v.validateState(state); err != nil {
		return err
	}

	// Validate Amount
	if err := v.validateAmount(amount); err != nil {
		return err
	}

	return nil
}

// validateTransactionID checks if the transaction ID is valid
func (v *TransactionValidator) validateTransactionID(transactionID string) error {
	if transactionID == "" {
		return errs.ErrInvalidTransactionID
	}

	// Additional validation rules could be added here
	// For example, checking length, format, etc.

	return nil
}

// validateSourceType checks if the source type is valid
func (v *TransactionValidator) validateSourceType(sourceType string) error {
	if sourceType == "" {
		return errs.ErrInvalidSourceType
	}

	// Check if the source type is a valid enum value
	if !entity.IsValidSourceType(sourceType) {
		return fmt.Errorf("%w: invalid source type %s", errs.ErrInvalidSourceType, sourceType)
	}

	return nil
}

// validateState checks if the state is valid
func (v *TransactionValidator) validateState(state string) error {
	if state == "" {
		return errs.ErrInvalidState
	}

	// Check if the state is a valid enum value
	if !entity.IsValidState(state) {
		return fmt.Errorf("%w: invalid state %s", errs.ErrInvalidState, state)
	}

	return nil
}

// validateAmount checks if the amount is valid
func (v *TransactionValidator) validateAmount(amount string) error {
	if amount == "" {
		return errs.ErrInvalidAmount
	}

	// Check if the amount is a valid number with at most 2 decimal places
	trimmed := strings.TrimSpace(amount)
	_, err := entity.ValidateAndConvertAmount(trimmed)
	if err != nil {
		return fmt.Errorf("%w: %s", errs.ErrInvalidAmount, err.Error())
	}

	return nil
}
