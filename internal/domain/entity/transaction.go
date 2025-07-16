package entity

import (
	"fmt"
	"time"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	tport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// TransactionState represents the state of a transaction
type TransactionState string

// SourceType represents the source of a transaction
type SourceType string

// Transaction states
const (
	StateWin  TransactionState = "win"
	StateLose TransactionState = "lose"
)

// Source types
const (
	SourceGame    SourceType = "game"
	SourceServer  SourceType = "server"
	SourcePayment SourceType = "payment"
)

// TransactionStatus defines possible status values for a transaction
type TransactionStatus string

// TransactionStatus constants
const (
	StatusPending   TransactionStatus = "pending"
	StatusCompleted TransactionStatus = "completed"
	StatusFailed    TransactionStatus = "failed"
)

// TransactionResponse represents the simplified API response for a transaction
type TransactionResponse struct {
	TransactionID string `json:"transactionId"`
	UserID        uint64 `json:"userId"`
	Success       bool   `json:"success"`
	ResultBalance string `json:"resultBalance,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

// Transaction represents a financial transaction that affects a user's balance
type Transaction struct {
	ID            uint64            // Unique identifier for the transaction
	UserID        uint64            // ID of the user this transaction belongs to
	TransactionID string            // Unique external transaction identifier
	SourceType    SourceType        // Source of the transaction
	State         TransactionState  // State of the transaction (win/lose)
	Amount        string            // Amount as a string with 2 decimal places
	AmountInCents int64             // Amount converted to cents for precise calculations
	CreatedAt     time.Time         // When the transaction was created
	ProcessedAt   *time.Time        // When the transaction was processed (nullable)
	ResultBalance string            // Balance after this transaction was processed
	Status        TransactionStatus // Status of the transaction
	ErrorMessage  string            // Error message if transaction failed
}

// NewTransaction creates a new transaction with basic validation
func NewTransaction(
	userID uint64,
	transactionID string,
	sourceType string,
	state string,
	amount string,
	timeProvider tport.TimeProvider,
) (*Transaction, error) {
	// Basic validations
	if userID == 0 {
		return nil, errs.ErrInvalidUserID
	}
	if transactionID == "" {
		return nil, errs.ErrInvalidTransactionID
	}

	// Validate source type and state
	if !isValidSourceType(sourceType) {
		return nil, fmt.Errorf("%w: %s", errs.ErrInvalidSourceType, sourceType)
	}
	if !isValidState(state) {
		return nil, fmt.Errorf("%w: %s", errs.ErrInvalidState, state)
	}

	// Validate amount
	amountInCents, err := ValidateAndConvertAmount(amount)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		UserID:        userID,
		TransactionID: transactionID,
		SourceType:    SourceType(sourceType),
		State:         TransactionState(state),
		Amount:        amount,
		AmountInCents: amountInCents,
		CreatedAt:     timeProvider.Now(),
		Status:        StatusPending,
	}, nil
}

// MarkAsProcessed marks the transaction as processed and completed
func (t *Transaction) MarkAsProcessed(timeProvider tport.TimeProvider, resultBalance string) {
	now := timeProvider.Now()
	t.ProcessedAt = &now

	// Format the result balance, ignoring potential error since we only care about the formatted string
	// If there's an error, the input value wasn't properly formatted, so we'll use it as is
	formattedBalance, err := EnsureTwoDecimalPlaces(resultBalance)
	if err == nil {
		t.ResultBalance = formattedBalance
	} else {
		// Fall back to the original value if we can't format it properly
		t.ResultBalance = resultBalance
	}

	t.Status = StatusCompleted
}

// MarkAsFailed marks the transaction as failed
func (t *Transaction) MarkAsFailed(timeProvider tport.TimeProvider, errorMessage string) {
	now := timeProvider.Now()
	t.ProcessedAt = &now
	t.Status = StatusFailed
	t.ErrorMessage = errorMessage
}

// IsCredit returns true if this transaction should increase the user's balance
func (t *Transaction) IsCredit() bool {
	return t.State == StateWin
}

// IsDebit returns true if this transaction should decrease the user's balance
func (t *Transaction) IsDebit() bool {
	return t.State == StateLose
}

// ToResponse converts the transaction to a response object for API
func (t *Transaction) ToResponse() TransactionResponse {
	return TransactionResponse{
		TransactionID: t.TransactionID,
		UserID:        t.UserID,
		Success:       t.Status == StatusCompleted,
		ResultBalance: t.ResultBalance,
		ErrorMessage:  t.ErrorMessage,
	}
}

// Helper functions

// isValidSourceType validates if the source type is allowed
func isValidSourceType(sourceType string) bool {
	return sourceType == string(SourceGame) ||
		sourceType == string(SourceServer) ||
		sourceType == string(SourcePayment)
}

// isValidState validates if the state is allowed
func isValidState(state string) bool {
	return state == string(StateWin) || state == string(StateLose)
}
