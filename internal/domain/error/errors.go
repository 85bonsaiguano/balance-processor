package error

import (
	"errors"
	"fmt"
)

// Error codes for standardized API responses
const (
	// 4xxx - Client errors
	CodeInsufficientBalance  = 4001
	CodeInvalidAmount        = 4002
	CodeInvalidUserID        = 4003
	CodeDuplicateTransaction = 4004
	CodeConstraintViolation  = 4005
	CodeAmountOverflow       = 4006
	CodeUserNotFound         = 4040
	CodeUserLocked           = 4230

	// 5xxx - Server errors
	CodeInternalServer = 5000
)

// Base error types
var (
	// ErrInsufficientBalance is returned when a user has insufficient funds for a transaction
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInvalidAmount is returned when the transaction amount format is invalid
	ErrInvalidAmount = errors.New("invalid amount format")

	// ErrInvalidUserID is returned when the user ID is not a positive integer
	ErrInvalidUserID = errors.New("user ID must be positive")

	// ErrNegativeAmount is returned when the transaction amount is negative
	ErrNegativeAmount = errors.New("amount cannot be negative")

	// ErrNegativeBalance is returned when an operation would result in negative balance
	ErrNegativeBalance = errors.New("balance cannot be negative")

	// ErrAmountOverflow is returned when the amount is too large and would cause overflow
	ErrAmountOverflow = errors.New("amount is too large and would cause overflow")

	// ErrInvalidTransactionID is returned when the transaction ID is empty or invalid
	ErrInvalidTransactionID = errors.New("transaction ID cannot be empty")

	// ErrInvalidSourceType is returned when the source type is not one of the allowed values
	ErrInvalidSourceType = errors.New("invalid source type")

	// ErrInvalidState is returned when the transaction state is not one of the allowed values
	ErrInvalidState = errors.New("invalid transaction state")

	// ErrDuplicateTransaction is returned when a transaction with the same ID already exists
	ErrDuplicateTransaction = errors.New("transaction with this ID already exists")

	// ErrUserNotFound is returned when the requested user doesn't exist
	ErrUserNotFound = errors.New("user not found")

	// ErrTransactionNotFound is returned when the requested transaction doesn't exist
	ErrTransactionNotFound = errors.New("transaction not found")

	// ErrInvalidRequest is returned when the request format is invalid
	ErrInvalidRequest = errors.New("invalid request")

	// ErrInternalServer is returned for unexpected server-side errors
	ErrInternalServer = errors.New("internal server error")

	// ErrUserLocked is returned when a user is locked by another operation
	ErrUserLocked = errors.New("user is locked by another operation")

	// ErrDatabaseConnection is returned when there's a problem connecting to the database
	ErrDatabaseConnection = errors.New("database connection error")

	// ErrDuplicateUser is returned when trying to create a user that already exists
	ErrDuplicateUser = errors.New("user already exists")

	// ErrConstraintViolation is returned when a database constraint is violated
	ErrConstraintViolation = errors.New("database constraint violation")

	// ErrNotFound is returned when a generic resource is not found
	ErrNotFound = errors.New("resource not found")
)

// ErrorCode returns standardized error codes for known errors
func ErrorCode(err error) int {
	switch {
	case errors.Is(err, ErrInsufficientBalance):
		return CodeInsufficientBalance
	case errors.Is(err, ErrInvalidAmount):
		return CodeInvalidAmount
	case errors.Is(err, ErrInvalidUserID):
		return CodeInvalidUserID
	case errors.Is(err, ErrDuplicateTransaction):
		return CodeDuplicateTransaction
	case errors.Is(err, ErrAmountOverflow):
		return CodeAmountOverflow
	case errors.Is(err, ErrUserNotFound):
		return CodeUserNotFound
	case errors.Is(err, ErrUserLocked):
		return CodeUserLocked
	case errors.Is(err, ErrConstraintViolation):
		return CodeConstraintViolation
	default:
		return CodeInternalServer
	}
}

// BalanceError represents an error related to balance operations
type BalanceError struct {
	UserID         uint64
	Amount         string
	CurrentBalance string
	Err            error
}

// Error implements the error interface for BalanceError
func (e *BalanceError) Error() string {
	return fmt.Sprintf("balance operation failed for user %d (current balance: %s, amount: %s): %v",
		e.UserID, e.CurrentBalance, e.Amount, e.Err)
}

// Unwrap returns the underlying error
func (e *BalanceError) Unwrap() error {
	return e.Err
}

// LogFields returns a map of fields for structured logging
func (e *BalanceError) LogFields() map[string]interface{} {
	return map[string]interface{}{
		"error_type":      "balance_error",
		"user_id":         e.UserID,
		"amount":          e.Amount,
		"current_balance": e.CurrentBalance,
		"error":           e.Err.Error(),
		"error_code":      ErrorCode(e.Err),
	}
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

// Error implements the error interface for TransactionError
func (e *TransactionError) Error() string {
	return fmt.Sprintf("transaction error for ID %s (user: %d, amount: %s): %s - %v",
		e.TransactionID, e.UserID, e.Amount, e.Reason, e.Err)
}

// Unwrap returns the underlying error
func (e *TransactionError) Unwrap() error {
	return e.Err
}

// LogFields returns a map of fields for structured logging
func (e *TransactionError) LogFields() map[string]any {
	return map[string]any{
		"error_type":     "transaction_error",
		"transaction_id": e.TransactionID,
		"user_id":        e.UserID,
		"source_type":    e.SourceType,
		"state":          e.State,
		"amount":         e.Amount,
		"reason":         e.Reason,
		"error":          e.Err.Error(),
		"error_code":     ErrorCode(e.Err),
	}
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

// LogFields returns a map of fields for structured logging
func (e *InsufficientBalanceError) LogFields() map[string]interface{} {
	return map[string]interface{}{
		"error_type":      "insufficient_balance",
		"user_id":         e.UserID,
		"amount":          e.Amount,
		"current_balance": e.CurrBalance,
		"error_code":      CodeInsufficientBalance,
	}
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

// LogFields returns a map of fields for structured logging
func (e *DuplicateTransactionError) LogFields() map[string]interface{} {
	return map[string]interface{}{
		"error_type":     "duplicate_transaction",
		"transaction_id": e.TransactionID,
		"user_id":        e.UserID,
		"source_type":    e.SourceType,
		"error_code":     CodeDuplicateTransaction,
	}
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

// IsUserNotFoundError checks if the error is a user not found error
func IsUserNotFoundError(err error) bool {
	return errors.Is(err, ErrUserNotFound)
}

// IsNotFoundError checks if the error is any "not found" type of error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrTransactionNotFound)
}

// IsUserLockedError checks if the error is related to a locked user
func IsUserLockedError(err error) bool {
	return errors.Is(err, ErrUserLocked)
}
