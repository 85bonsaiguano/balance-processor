package transaction

import (
	"strings"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
)

// ValidateTransactionRequest validates an incoming transaction request
func (s *Service) ValidateTransactionRequest(req usecase.TransactionRequest) error {
	// Check required fields
	if req.TransactionID == "" {
		return errs.ErrInvalidTransactionID
	}

	// Validate state (win/lose)
	if req.State != "win" && req.State != "lose" {
		return errs.ErrInvalidState
	}

	// Validate amount
	if strings.TrimSpace(req.Amount) == "" {
		return errs.ErrInvalidAmount
	}

	// Check decimal places
	parts := strings.Split(req.Amount, ".")
	if len(parts) > 2 || (len(parts) == 2 && len(parts[1]) > 2) {
		return errs.ErrInvalidAmount
	}

	// Further delegation to entity validation is done by the transaction creation

	return nil
}
