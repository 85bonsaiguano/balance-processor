package entity

import (
	"fmt"
	"strings"
	"time"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	tport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
)

// Transaction module implements domain logic for financial transactions in the balance processor.
// It provides:
//   - Type-safe enums for transaction states, source types, and statuses
//   - Generic implementation of enum handling for extensibility
//   - Complete transaction lifecycle management (creation, processing, completion)
//   - Built-in validation for all transaction properties
//
// The implementation uses a combination of:
//   - Generics for flexible enum handling
//   - Immutability where appropriate for data consistency
//   - Function options pattern for optional configuration
//   - Consistent error handling with domain-specific errors
//
// All monetary values are stored as integers (cents) to avoid floating-point precision issues.

// EnumConstraint is a constraint for enum types
type EnumConstraint interface {
	~string
	String() string
	IsValid() bool
}

// EnumRegistry provides a generic registry for enum types
type EnumRegistry[T EnumConstraint] struct {
	values  []T
	errType error // The error type to return for invalid values
}

// Register adds a new enum value to the registry if it doesn't already exist
func (e *EnumRegistry[T]) Register(value T) {
	for _, v := range e.values {
		if v == value {
			return
		}
	}
	e.values = append(e.values, value)
}

// Contains checks if a value exists in the registry
func (e *EnumRegistry[T]) Contains(value T) bool {
	for _, v := range e.values {
		if v == value {
			return true
		}
	}
	return false
}

// Parse converts a string to the enum type, with proper error handling
func (e *EnumRegistry[T]) Parse(str string) (T, error) {
	normalized := strings.ToLower(strings.TrimSpace(str))
	for _, v := range e.values {
		if strings.EqualFold(v.String(), normalized) {
			return v, nil
		}
	}
	var zero T
	return zero, fmt.Errorf("%w: %s", e.errType, str)
}

// Values returns a copy of the registered enum values
func (e *EnumRegistry[T]) Values() []T {
	result := make([]T, len(e.values))
	copy(result, e.values)
	return result
}

// NewEnumRegistry creates and returns a new enum registry with the provided values
func NewEnumRegistry[T EnumConstraint](errType error, values ...T) *EnumRegistry[T] {
	return &EnumRegistry[T]{
		values:  values,
		errType: errType,
	}
}

// BalanceEffect represents how a transaction affects the user's balance
type BalanceEffect string

const (
	// EffectIncrease represents a transaction that increases the user's balance (win)
	EffectIncrease BalanceEffect = "increase"
	// EffectDecrease represents a transaction that decreases the user's balance (lose)
	EffectDecrease BalanceEffect = "decrease"
)

// String returns the string representation of BalanceEffect
func (b BalanceEffect) String() string {
	return string(b)
}

// IsValid checks if the BalanceEffect is valid
func (b BalanceEffect) IsValid() bool {
	switch b {
	case EffectIncrease, EffectDecrease:
		return true
	default:
		return false
	}
}

// TransactionState represents the state of a transaction
type TransactionState string

// String methods to satisfy EnumConstraint
func (s TransactionState) String() string {
	return string(s)
}

// Transaction states
const (
	StateWin  TransactionState = "win"  // Win state increases the balance
	StateLose TransactionState = "lose" // Lose state decreases the balance
	// Future states can be added here
)

// GetBalanceEffect returns the corresponding BalanceEffect for this transaction state
func (s TransactionState) GetBalanceEffect() BalanceEffect {
	switch s {
	case StateWin:
		return EffectIncrease
	case StateLose:
		return EffectDecrease
	default:
		// This shouldn't happen if the state is valid
		return ""
	}
}

// Source types
type SourceType string

// String methods to satisfy EnumConstraint
func (s SourceType) String() string {
	return string(s)
}

const (
	SourceGame    SourceType = "game"
	SourceServer  SourceType = "server"
	SourcePayment SourceType = "payment"
	// Future sources can be added here
)

// TransactionStatus constants
type TransactionStatus string

// String methods to satisfy EnumConstraint
func (s TransactionStatus) String() string {
	return string(s)
}

const (
	StatusPending   TransactionStatus = "pending"
	StatusCompleted TransactionStatus = "completed"
	StatusFailed    TransactionStatus = "failed"
)

// Enum registries - using the improved registry system with proper error types
var (
	transactionStateRegistry = NewEnumRegistry(
		errs.ErrInvalidState,
		StateWin,
		StateLose,
	)

	sourceTypeRegistry = NewEnumRegistry(
		errs.ErrInvalidSourceType,
		SourceGame,
		SourceServer,
		SourcePayment,
	)

	transactionStatusRegistry = NewEnumRegistry(
		errs.ErrInvalidState,
		StatusPending,
		StatusCompleted,
		StatusFailed,
	)
)

// Methods for TransactionState
func (s TransactionState) IsValid() bool {
	return transactionStateRegistry.Contains(s)
}

func (s TransactionState) Values() []TransactionState {
	return transactionStateRegistry.Values()
}

func RegisterTransactionState(state TransactionState) {
	transactionStateRegistry.Register(state)
}

func ParseTransactionState(state string) (TransactionState, error) {
	return transactionStateRegistry.Parse(state)
}

// Methods for SourceType
func (s SourceType) IsValid() bool {
	return sourceTypeRegistry.Contains(s)
}

func (s SourceType) Values() []SourceType {
	return sourceTypeRegistry.Values()
}

func RegisterSourceType(source SourceType) {
	sourceTypeRegistry.Register(source)
}

func ParseSourceType(sourceType string) (SourceType, error) {
	return sourceTypeRegistry.Parse(sourceType)
}

// Methods for TransactionStatus
func (s TransactionStatus) IsValid() bool {
	return transactionStatusRegistry.Contains(s)
}

func (s TransactionStatus) Values() []TransactionStatus {
	return transactionStatusRegistry.Values()
}

func RegisterTransactionStatus(status TransactionStatus) {
	transactionStatusRegistry.Register(status)
}

func ParseTransactionStatus(status string) (TransactionStatus, error) {
	return transactionStatusRegistry.Parse(status)
}

// Transaction represents a financial transaction that affects a user's balance
// Thread safety: The Transaction struct is not thread-safe for concurrent modifications.
// Methods like MarkAsProcessed and MarkAsFailed modify the transaction state directly
// and require external synchronization when used in concurrent scenarios.
type Transaction struct {
	ID                   uint64            // Unique identifier for the transaction
	UserID               uint64            // ID of the user this transaction belongs to
	TransactionID        string            // Unique external transaction identifier
	SourceType           SourceType        // Source of the transaction
	State                TransactionState  // State of the transaction (win/lose)
	AmountInCents        int64             // Amount converted to cents for precise calculations
	CreatedAt            time.Time         // When the transaction was created
	ProcessedAt          *time.Time        // When the transaction was processed (nullable)
	ResultBalanceInCents int64             // Balance after this transaction was processed, in cents
	Status               TransactionStatus // Status of the transaction
	ErrorMessage         string            // Error message if transaction failed
}

// TransactionOption is a functional option for configuring a Transaction
type TransactionOption func(*Transaction) error

// WithCustomStatus sets a custom status for the transaction
func WithCustomStatus(status TransactionStatus) TransactionOption {
	return func(t *Transaction) error {
		if !status.IsValid() {
			return fmt.Errorf("%w: %s", errs.ErrInvalidState, status)
		}
		t.Status = status
		return nil
	}
}

// NewTransaction creates a new transaction with basic validation
func NewTransaction(
	userID uint64,
	transactionID string,
	sourceType string,
	state string,
	amount string,
	timeProvider tport.TimeProvider,
	opts ...TransactionOption,
) (*Transaction, error) {
	// Validate transaction ID
	if transactionID == "" {
		return nil, errs.ErrInvalidTransactionID
	}

	// Parse and validate source type
	parsedSourceType, err := ParseSourceType(sourceType)
	if err != nil {
		return nil, err
	}

	// Parse and validate transaction state
	parsedState, err := ParseTransactionState(state)
	if err != nil {
		return nil, err
	}

	// Parse and validate amount (ensuring it's a valid money format)
	amountInCents, err := ValidateAndConvertAmount(amount)
	if err != nil {
		return nil, err
	}

	// Create transaction with default values
	txn := &Transaction{
		UserID:        userID,
		TransactionID: transactionID,
		SourceType:    parsedSourceType,
		State:         parsedState,
		AmountInCents: amountInCents,
		CreatedAt:     timeProvider.Now(),
		Status:        StatusPending,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(txn); err != nil {
			return nil, err
		}
	}

	return txn, nil
}

// MarkAsProcessed updates the transaction status to completed with the resulting balance
func (t *Transaction) MarkAsProcessed(timeProvider tport.TimeProvider, resultBalanceInCents int64) {
	now := timeProvider.Now()
	t.ProcessedAt = &now
	t.Status = StatusCompleted
	t.ResultBalanceInCents = resultBalanceInCents
}

// MarkAsFailed updates the transaction status to failed with an error message
func (t *Transaction) MarkAsFailed(timeProvider tport.TimeProvider, errorMessage string) {
	now := timeProvider.Now()
	t.ProcessedAt = &now
	t.Status = StatusFailed
	t.ErrorMessage = errorMessage
}

// GetAmount returns the transaction amount as a formatted string
func (t *Transaction) GetAmount() string {
	return AmountInCentsToString(t.AmountInCents)
}

// GetResultBalance returns the result balance as a formatted string
func (t *Transaction) GetResultBalance() string {
	return AmountInCentsToString(t.ResultBalanceInCents)
}

// IsCredit checks if the transaction is a credit transaction (increases balance)
func (t *Transaction) IsCredit() bool {
	return t.State.GetBalanceEffect() == EffectIncrease
}

// IsDebit checks if the transaction is a debit transaction (decreases balance)
func (t *Transaction) IsDebit() bool {
	return t.State.GetBalanceEffect() == EffectDecrease
}

// IsAlreadyProcessed checks if the transaction has already been processed
func (t *Transaction) IsAlreadyProcessed() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed
}

// IsFailed checks if the transaction has failed
func (t *Transaction) IsFailed() bool {
	return t.Status == StatusFailed
}

// IsPending checks if the transaction is still pending
func (t *Transaction) IsPending() bool {
	return t.Status == StatusPending
}

// IsValidSourceType checks if a string is a valid source type
func IsValidSourceType(sourceType string) bool {
	_, err := ParseSourceType(sourceType)
	return err == nil
}

// IsValidState checks if a string is a valid transaction state
func IsValidState(state string) bool {
	_, err := ParseTransactionState(state)
	return err == nil
}

// Clone creates a deep copy of the transaction
func (t *Transaction) Clone() *Transaction {
	clone := *t // Create a copy of the transaction

	// Deep copy of pointers
	if t.ProcessedAt != nil {
		processedAt := *t.ProcessedAt
		clone.ProcessedAt = &processedAt
	}

	return &clone
}
