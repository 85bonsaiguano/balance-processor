package usecase

import (
	"context"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
)

// TransactionResult contains info about a processed transaction
type TransactionResult struct {
	Success       bool
	ResultBalance string
	ErrorMessage  string
	StatusCode    int // HTTP status code
}

// TransactionRequest represents an incoming transaction request
type TransactionRequest struct {
	State         string `json:"state"`
	Amount        string `json:"amount"`
	TransactionID string `json:"transactionId"`
	SourceType    entity.SourceType
}

// TransactionUseCase defines methods for transaction-related business operations
type TransactionUseCase interface {
	// ProcessTransaction processes a transaction and updates user balance
	// Returns detailed information about the transaction result
	ProcessTransaction(ctx context.Context, userID uint64, req TransactionRequest) (*TransactionResult, error)

	// IsDuplicateTransaction checks if a transaction with the given ID was already processed
	IsDuplicateTransaction(ctx context.Context, transactionID string) (bool, error)

	// ValidateTransactionRequest validates an incoming transaction request
	ValidateTransactionRequest(req TransactionRequest) error
}
