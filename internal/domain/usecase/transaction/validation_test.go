package transaction

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/amirhossein-jamali/balance-processor/internal/domain/entity"
	domainerrs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/usecase"
	mcore "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
)

// TestValidateTransactionRequestComprehensive is a comprehensive test for the ValidateTransactionRequest function
func TestValidateTransactionRequestComprehensive(t *testing.T) {
	// Define test cases table
	tests := []struct {
		name          string
		transactionID string
		state         string
		amount        string
		sourceType    entity.SourceType
		expectedError error
		errorMessage  string
	}{
		// Valid cases
		{
			name:          "Valid Win Transaction",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "100.50",
			sourceType:    entity.SourceGame,
			expectedError: nil,
		},
		{
			name:          "Valid Lose Transaction",
			transactionID: "tx-789012",
			state:         "lose",
			amount:        "75.25",
			sourceType:    entity.SourcePayment,
			expectedError: nil,
		},
		{
			name:          "Valid Integer Amount",
			transactionID: "tx-345678",
			state:         "win",
			amount:        "500",
			sourceType:    entity.SourceServer,
			expectedError: nil,
		},
		{
			name:          "Valid Single Decimal Place",
			transactionID: "tx-901234",
			state:         "lose",
			amount:        "25.5",
			sourceType:    entity.SourceGame,
			expectedError: nil,
		},

		// Invalid transaction ID
		{
			name:          "Empty Transaction ID",
			transactionID: "",
			state:         "win",
			amount:        "50.00",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidTransactionID,
			errorMessage:  "transaction ID cannot be empty",
		},

		// Invalid state
		{
			name:          "Invalid State - Not Win Or Lose",
			transactionID: "tx-123456",
			state:         "draw",
			amount:        "100.00",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidState,
			errorMessage:  "invalid transaction state",
		},
		{
			name:          "Empty State",
			transactionID: "tx-123456",
			state:         "",
			amount:        "100.00",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidState,
			errorMessage:  "invalid transaction state",
		},

		// Invalid amount
		{
			name:          "Empty Amount",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidAmount,
			errorMessage:  "invalid amount format",
		},
		{
			name:          "Whitespace Amount",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "   ",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidAmount,
			errorMessage:  "invalid amount format",
		},
		{
			name:          "Too Many Decimal Places",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "100.567",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidAmount,
			errorMessage:  "invalid amount format",
		},
		{
			name:          "Multiple Decimal Points",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "100.5.6",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidAmount,
			errorMessage:  "invalid amount format",
		},
	}

	// Run each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockLogger := new(mcore.MockLogger)
			service := &Service{
				logger: mockLogger,
			}

			// Create the request
			req := usecase.TransactionRequest{
				TransactionID: tt.transactionID,
				State:         tt.state,
				Amount:        tt.amount,
				SourceType:    tt.sourceType,
			}

			// Call the validation function
			err := service.ValidateTransactionRequest(req)

			// Assertions
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateTransactionRequestEdgeCases tests edge cases for the validation function
func TestValidateTransactionRequestEdgeCases(t *testing.T) {
	// Define test cases table for edge cases
	tests := []struct {
		name          string
		transactionID string
		state         string
		amount        string
		sourceType    entity.SourceType
		expectedError error
	}{
		{
			name:          "Amount With Leading/Trailing Spaces",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "  50.25  ",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidAmount,
		},
		{
			name:          "Amount With Zero Decimals",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "100.00",
			sourceType:    entity.SourceGame,
			expectedError: nil,
		},
		{
			name:          "Very Large Amount",
			transactionID: "tx-123456",
			state:         "win",
			amount:        "99999999999.99",
			sourceType:    entity.SourceGame,
			expectedError: nil, // Only validating format, not value range
		},
		{
			name:          "Case Sensitivity - WIN",
			transactionID: "tx-123456",
			state:         "WIN",
			amount:        "100.00",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidState, // Case sensitive comparison
		},
		{
			name:          "Case Sensitivity - LOSE",
			transactionID: "tx-123456",
			state:         "LOSE",
			amount:        "100.00",
			sourceType:    entity.SourceGame,
			expectedError: domainerrs.ErrInvalidState, // Case sensitive comparison
		},
	}

	// Run each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockLogger := new(mcore.MockLogger)
			service := &Service{
				logger: mockLogger,
			}

			// Create the request
			req := usecase.TransactionRequest{
				TransactionID: tt.transactionID,
				State:         tt.state,
				Amount:        tt.amount,
				SourceType:    tt.sourceType,
			}

			// Call the validation function
			err := service.ValidateTransactionRequest(req)

			// Assertions
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
