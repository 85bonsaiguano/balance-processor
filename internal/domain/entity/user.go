package entity

import (
	"time"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// User represents a user entity with a balance
type User struct {
	ID               uint64    // Unique identifier for the user
	balance          int64     // Balance stored in cents to avoid floating point precision issues (private)
	CreatedAt        time.Time // When the user was created
	UpdatedAt        time.Time // When the user was last updated
	TransactionCount uint64    // Count of transactions processed for this user
}

// NewUser creates a new user with the given ID and initial balance
func NewUser(id uint64, initialBalance string, timeProvider coreport.TimeProvider) (*User, error) {
	if id == 0 {
		return nil, errs.ErrInvalidUserID
	}

	balanceInCents, err := ValidateAndConvertAmount(initialBalance)
	if err != nil {
		return nil, err
	}

	now := timeProvider.Now()
	return &User{
		ID:               id,
		balance:          balanceInCents,
		CreatedAt:        now,
		UpdatedAt:        now,
		TransactionCount: 0,
	}, nil
}

// Balance returns the current balance in cents (for internal use)
func (u *User) Balance() int64 {
	return u.balance
}

// GetBalance returns the balance as a string with 2 decimal places
func (u *User) GetBalance() string {
	// Convert balance to string format
	balanceStr := AmountInCentsToString(u.balance)

	// We can safely ignore error here because AmountInCentsToString always produces
	// a valid string with exactly 2 decimal places
	formattedBalance, _ := EnsureTwoDecimalPlaces(balanceStr)
	return formattedBalance
}

// SetBalance updates the balance directly (for internal use, like repositories)
func (u *User) SetBalance(balanceInCents int64, timeProvider coreport.TimeProvider) {
	u.balance = balanceInCents
	u.UpdatedAt = timeProvider.Now()
}

// IncrementTransactionCount increases the transaction count by 1
func (u *User) IncrementTransactionCount() {
	u.TransactionCount++
}

// CanDeduct checks if the user has enough balance for a deduction
func (u *User) CanDeduct(amount string) (bool, error) {
	amountInCents, err := ValidateAndConvertAmount(amount)
	if err != nil {
		return false, err
	}

	return u.balance >= amountInCents, nil
}

// ApplyWinTransaction adds the amount to the balance
func (u *User) ApplyWinTransaction(amountInCents int64, timeProvider coreport.TimeProvider) {
	u.balance += amountInCents
	u.UpdatedAt = timeProvider.Now()
	u.IncrementTransactionCount()
}

// ApplyLoseTransaction subtracts the amount from balance if sufficient balance exists
// Returns error if insufficient balance
func (u *User) ApplyLoseTransaction(amountInCents int64, timeProvider coreport.TimeProvider) error {
	if u.balance < amountInCents {
		return errs.ErrInsufficientBalance
	}

	u.balance -= amountInCents
	u.UpdatedAt = timeProvider.Now()
	u.IncrementTransactionCount()
	return nil
}
