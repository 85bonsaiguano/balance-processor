package error

import (
	"errors"
	"fmt"
)

// Base error types
var (
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrInvalidAmount        = errors.New("invalid amount format")
	ErrInvalidUserID        = errors.New("user ID must be positive")
	ErrNegativeAmount       = errors.New("amount cannot be negative")
	ErrInvalidTransactionID = errors.New("transaction ID cannot be empty")
	ErrInvalidSourceType    = errors.New("invalid source type")
	ErrInvalidState         = errors.New("invalid transaction state")
	ErrDuplicateTransaction = errors.New("transaction with this ID already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrInternalServer       = errors.New("internal server error")
	ErrUserLocked           = errors.New("user is locked by another operation")
	ErrDatabaseConnection   = errors.New("database connection error")
	ErrDuplicateUser        = errors.New("user already exists")
	ErrConstraintViolation  = errors.New("database constraint violation")
	ErrNotFound             = errors.New("resource not found")
)

// ErrorCode returns standardized error codes for known errors
func ErrorCode(err error) int {
	switch {
	case errors.Is(err, ErrInsufficientBalance):
		return 4001
	case errors.Is(err, ErrInvalidAmount):
		return 4002
	case errors.Is(err, ErrInvalidUserID):
		return 4003
	case errors.Is(err, ErrDuplicateTransaction):
		return 4004
	case errors.Is(err, ErrUserNotFound):
		return 4040
	case errors.Is(err, ErrUserLocked):
		return 4230
	case errors.Is(err, ErrConstraintViolation):
		return 4005
	default:
		return 5000
	}
}

// BalanceError represents an error related to balance operations
type BalanceError struct {
	UserID         uint64
	Amount         string
	CurrentBalance string
	Err            error
}

func (e *BalanceError) Error() string {
	return fmt.Sprintf("balance operation failed for user %d (current balance: %s, amount: %s): %v",
		e.UserID, e.CurrentBalance, e.Amount, e.Err)
}

func (e *BalanceError) Unwrap() error {
	return e.Err
}

// TransactionError represents an error related to transaction processing
type TransactionError struct {
	TransactionID string
	UserID        uint64
	SourceType    string
	State         string
	Amount        string
	Reason        string
	Err           error
}

func (e *TransactionError) Error() string {
	return fmt.Sprintf("transaction error for ID %s (user: %d, amount: %s): %s - %v",
		e.TransactionID, e.UserID, e.Amount, e.Reason, e.Err)
}

func (e *TransactionError) Unwrap() error {
	return e.Err
}

// InsufficientBalanceError provides detailed error information for insufficient balance
type InsufficientBalanceError struct {
	UserID      uint64
	Amount      string
	CurrBalance string
}

// Error implements the error interface
func (e *InsufficientBalanceError) Error() string {
	return fmt.Sprintf("insufficient balance for user %d: required %s, available %s",
		e.UserID, e.Amount, e.CurrBalance)
}

// Is checks if the target error is an ErrInsufficientBalance
func (e *InsufficientBalanceError) Is(target error) bool {
	return target == ErrInsufficientBalance
}

// NewInsufficientBalanceError creates a new detailed insufficient balance error
func NewInsufficientBalanceError(userID uint64, amount, currentBalance string) error {
	return &InsufficientBalanceError{
		UserID:      userID,
		Amount:      amount,
		CurrBalance: currentBalance,
	}
}

// NewTransactionError creates a detailed transaction error
func NewTransactionError(transactionID string, userID uint64, sourceType, state, amount, reason string, err error) error {
	return &TransactionError{
		TransactionID: transactionID,
		UserID:        userID,
		SourceType:    sourceType,
		State:         state,
		Amount:        amount,
		Reason:        reason,
		Err:           err,
	}
}

// DuplicateTransactionError provides detailed information about duplicate transaction attempts
type DuplicateTransactionError struct {
	TransactionID string
	UserID        uint64
	SourceType    string
}

// Error implements the error interface
func (e *DuplicateTransactionError) Error() string {
	return fmt.Sprintf("duplicate transaction detected: transactionID=%s for user %d from source %s",
		e.TransactionID, e.UserID, e.SourceType)
}

// Is checks if the target error is an ErrDuplicateTransaction
func (e *DuplicateTransactionError) Is(target error) bool {
	return target == ErrDuplicateTransaction
}

// NewDuplicateTransactionError creates a new detailed duplicate transaction error
func NewDuplicateTransactionError(transactionID string, userID uint64, sourceType string) error {
	return &DuplicateTransactionError{
		TransactionID: transactionID,
		UserID:        userID,
		SourceType:    sourceType,
	}
}

// IsDuplicateTransactionError checks if the error is a duplicate transaction error
func IsDuplicateTransactionError(err error) bool {
	return errors.Is(err, ErrDuplicateTransaction)
}

// IsInsufficientBalanceError checks if the error is related to insufficient balance
func IsInsufficientBalanceError(err error) bool {
	return errors.Is(err, ErrInsufficientBalance)
}
