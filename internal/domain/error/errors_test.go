package error

import (
	"errors"
	"fmt"
	"testing"
)

func TestBaseErrorTypes(t *testing.T) {
	// Test to ensure all base error types are defined properly
	if ErrInsufficientBalance.Error() != "insufficient balance" {
		t.Errorf("ErrInsufficientBalance has unexpected message: %s", ErrInsufficientBalance.Error())
	}
	if ErrInvalidAmount.Error() != "invalid amount format" {
		t.Errorf("ErrInvalidAmount has unexpected message: %s", ErrInvalidAmount.Error())
	}
	// Add more assertions for other base error types as needed
}

func TestErrorCode(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected int
	}{
		{"InsufficientBalance", ErrInsufficientBalance, 4001},
		{"InvalidAmount", ErrInvalidAmount, 4002},
		{"InvalidUserID", ErrInvalidUserID, 4003},
		{"DuplicateTransaction", ErrDuplicateTransaction, 4004},
		{"UserNotFound", ErrUserNotFound, 4040},
		{"UserLocked", ErrUserLocked, 4230},
		{"ConstraintViolation", ErrConstraintViolation, 4005},
		{"UnknownError", errors.New("unknown error"), 5000},
		{"WrappedError", fmt.Errorf("wrapped: %w", ErrInvalidUserID), 4003},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := ErrorCode(tc.err)
			if code != tc.expected {
				t.Errorf("ErrorCode(%v) = %d, want %d", tc.err, code, tc.expected)
			}
		})
	}
}

func TestBalanceError(t *testing.T) {
	baseErr := ErrInsufficientBalance
	balanceErr := &BalanceError{
		UserID:         123,
		Amount:         "100.50",
		CurrentBalance: "50.25",
		Err:            baseErr,
	}

	// Test Error method
	expectedErrMsg := "balance operation failed for user 123 (current balance: 50.25, amount: 100.50): insufficient balance"
	if balanceErr.Error() != expectedErrMsg {
		t.Errorf("BalanceError.Error() = %s, want %s", balanceErr.Error(), expectedErrMsg)
	}

	// Test Unwrap method
	if !errors.Is(balanceErr, baseErr) {
		t.Errorf("errors.Is(balanceErr, baseErr) = false, want true")
	}
}

func TestTransactionError(t *testing.T) {
	baseErr := ErrInvalidAmount
	txError := &TransactionError{
		TransactionID: "tx123",
		UserID:        456,
		SourceType:    "api",
		State:         "pending",
		Amount:        "200.75",
		Reason:        "validation failed",
		Err:           baseErr,
	}

	// Test Error method
	expectedErrMsg := "transaction error for ID tx123 (user: 456, amount: 200.75): validation failed - invalid amount format"
	if txError.Error() != expectedErrMsg {
		t.Errorf("TransactionError.Error() = %s, want %s", txError.Error(), expectedErrMsg)
	}

	// Test Unwrap method
	if !errors.Is(txError, baseErr) {
		t.Errorf("errors.Is(txError, baseErr) = false, want true")
	}
}

func TestInsufficientBalanceError(t *testing.T) {
	err := NewInsufficientBalanceError(789, "300.00", "150.00")
	if err == nil {
		t.Fatal("NewInsufficientBalanceError returned nil")
	}

	// Test Error method
	expectedErrMsg := "insufficient balance for user 789: required 300.00, available 150.00"
	if err.Error() != expectedErrMsg {
		t.Errorf("InsufficientBalanceError.Error() = %s, want %s", err.Error(), expectedErrMsg)
	}

	// Test Is method through errors.Is
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("errors.Is(err, ErrInsufficientBalance) = false, want true")
	}

	// Test through helper function
	if !IsInsufficientBalanceError(err) {
		t.Errorf("IsInsufficientBalanceError(err) = false, want true")
	}
}

func TestDuplicateTransactionError(t *testing.T) {
	err := NewDuplicateTransactionError("tx456", 321, "mobile")
	if err == nil {
		t.Fatal("NewDuplicateTransactionError returned nil")
	}

	// Test Error method
	expectedErrMsg := "duplicate transaction detected: transactionID=tx456 for user 321 from source mobile"
	if err.Error() != expectedErrMsg {
		t.Errorf("DuplicateTransactionError.Error() = %s, want %s", err.Error(), expectedErrMsg)
	}

	// Test Is method through errors.Is
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Errorf("errors.Is(err, ErrDuplicateTransaction) = false, want true")
	}

	// Test through helper function
	if !IsDuplicateTransactionError(err) {
		t.Errorf("IsDuplicateTransactionError(err) = false, want true")
	}
}

func TestErrorHelperFunctions(t *testing.T) {
	// Test regular errors
	if IsInsufficientBalanceError(ErrInvalidUserID) {
		t.Errorf("IsInsufficientBalanceError(ErrInvalidUserID) = true, want false")
	}

	if IsDuplicateTransactionError(ErrInvalidAmount) {
		t.Errorf("IsDuplicateTransactionError(ErrInvalidAmount) = true, want false")
	}

	// Test wrapped errors
	wrappedInsufficientErr := fmt.Errorf("wrapped: %w", ErrInsufficientBalance)
	if !IsInsufficientBalanceError(wrappedInsufficientErr) {
		t.Errorf("IsInsufficientBalanceError(wrappedInsufficientErr) = false, want true")
	}

	wrappedDuplicateErr := fmt.Errorf("wrapped: %w", ErrDuplicateTransaction)
	if !IsDuplicateTransactionError(wrappedDuplicateErr) {
		t.Errorf("IsDuplicateTransactionError(wrappedDuplicateErr) = false, want true")
	}
}

func TestNewTransactionError(t *testing.T) {
	baseErr := ErrInvalidState
	txErr := NewTransactionError(
		"tx789",
		123,
		"web",
		"failed",
		"50.00",
		"invalid state transition",
		baseErr,
	)

	if txErr == nil {
		t.Fatal("NewTransactionError returned nil")
	}

	// Check if the error is correctly created
	var txErrCast *TransactionError
	if !errors.As(txErr, &txErrCast) {
		t.Fatalf("errors.As failed: not a *TransactionError")
	}

	if txErrCast.TransactionID != "tx789" {
		t.Errorf("TransactionID = %s, want tx789", txErrCast.TransactionID)
	}
	
	if txErrCast.UserID != 123 {
		t.Errorf("UserID = %d, want 123", txErrCast.UserID)
	}
	
	if txErrCast.SourceType != "web" {
		t.Errorf("SourceType = %s, want web", txErrCast.SourceType)
	}
	
	if txErrCast.State != "failed" {
		t.Errorf("State = %s, want failed", txErrCast.State)
	}
	
	if txErrCast.Amount != "50.00" {
		t.Errorf("Amount = %s, want 50.00", txErrCast.Amount)
	}
	
	if txErrCast.Reason != "invalid state transition" {
		t.Errorf("Reason = %s, want invalid state transition", txErrCast.Reason)
	}
	
	// Compare errors using errors.Is instead of direct equality
	if !errors.Is(txErrCast.Err, baseErr) {
		t.Errorf("errors.Is(txErrCast.Err, baseErr) = false, want true")
	}

	// Test unwrapping
	if !errors.Is(txErr, baseErr) {
		t.Errorf("errors.Is(txErr, baseErr) = false, want true")
	}
} 