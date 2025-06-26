package entity

import (
	"testing"
	"time"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coremocks "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(fixedTime).Maybe()

	t.Run("Valid transaction creation", func(t *testing.T) {
		tx, err := NewTransaction(
			1,                  // userID
			"tx123",            // transactionID
			string(SourceGame), // sourceType
			string(StateWin),   // state
			"100.00",           // amount
			mockTime,
		)

		require.NoError(t, err)
		assert.Equal(t, uint64(1), tx.UserID)
		assert.Equal(t, "tx123", tx.TransactionID)
		assert.Equal(t, SourceGame, tx.SourceType)
		assert.Equal(t, StateWin, tx.State)
		assert.Equal(t, "100.00", tx.Amount)
		assert.Equal(t, int64(10000), tx.AmountInCents)
		assert.Equal(t, fixedTime, tx.CreatedAt)
		assert.Nil(t, tx.ProcessedAt)
		assert.Equal(t, "", tx.ResultBalance)
		assert.Equal(t, StatusPending, tx.Status)
		assert.Equal(t, "", tx.ErrorMessage)
	})

	t.Run("Invalid userID", func(t *testing.T) {
		tx, err := NewTransaction(
			0, // invalid userID
			"tx123",
			string(SourceGame),
			string(StateWin),
			"100.00",
			mockTime,
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidUserID)
		assert.Nil(t, tx)
	})

	t.Run("Empty transactionID", func(t *testing.T) {
		tx, err := NewTransaction(
			1,
			"", // invalid transactionID
			string(SourceGame),
			string(StateWin),
			"100.00",
			mockTime,
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidTransactionID)
		assert.Nil(t, tx)
	})

	t.Run("Invalid sourceType", func(t *testing.T) {
		tx, err := NewTransaction(
			1,
			"tx123",
			"invalid-source", // invalid sourceType
			string(StateWin),
			"100.00",
			mockTime,
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidSourceType)
		assert.Nil(t, tx)
	})

	t.Run("Invalid state", func(t *testing.T) {
		tx, err := NewTransaction(
			1,
			"tx123",
			string(SourceGame),
			"invalid-state", // invalid state
			"100.00",
			mockTime,
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidState)
		assert.Nil(t, tx)
	})

	t.Run("Invalid amount", func(t *testing.T) {
		tx, err := NewTransaction(
			1,
			"tx123",
			string(SourceGame),
			string(StateWin),
			"invalid-amount", // invalid amount
			mockTime,
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidAmount)
		assert.Nil(t, tx)
	})

	t.Run("All valid source types", func(t *testing.T) {
		sourceTypes := []string{
			string(SourceGame),
			string(SourceServer),
			string(SourcePayment),
		}

		for _, sourceType := range sourceTypes {
			t.Run(sourceType, func(t *testing.T) {
				mockTimeLocal := coremocks.NewMockTimeProvider(t)
				mockTimeLocal.EXPECT().Now().Return(fixedTime).Once()

				tx, err := NewTransaction(
					1,
					"tx123",
					sourceType,
					string(StateWin),
					"100.00",
					mockTimeLocal,
				)

				require.NoError(t, err)
				assert.Equal(t, SourceType(sourceType), tx.SourceType)
			})
		}
	})

	t.Run("All valid states", func(t *testing.T) {
		states := []string{
			string(StateWin),
			string(StateLose),
		}

		for _, state := range states {
			t.Run(state, func(t *testing.T) {
				mockTimeLocal := coremocks.NewMockTimeProvider(t)
				mockTimeLocal.EXPECT().Now().Return(fixedTime).Once()

				tx, err := NewTransaction(
					1,
					"tx123",
					string(SourceGame),
					state,
					"100.00",
					mockTimeLocal,
				)

				require.NoError(t, err)
				assert.Equal(t, TransactionState(state), tx.State)
			})
		}
	})
}

func TestTransactionMarkAsProcessed(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	processTime := time.Date(2023, 1, 1, 12, 5, 0, 0, time.UTC)

	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(initialTime).Once()

	tx, err := NewTransaction(
		1,
		"tx123",
		string(SourceGame),
		string(StateWin),
		"100.00",
		mockTime,
	)
	require.NoError(t, err)

	// Initial state
	assert.Nil(t, tx.ProcessedAt)
	assert.Equal(t, "", tx.ResultBalance)
	assert.Equal(t, StatusPending, tx.Status)

	// Mark as processed
	mockProcessTime := coremocks.NewMockTimeProvider(t)
	mockProcessTime.EXPECT().Now().Return(processTime).Once()
	tx.MarkAsProcessed(mockProcessTime, "200.00")

	// Check updated state
	require.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, processTime, *tx.ProcessedAt)
	assert.Equal(t, "200.00", tx.ResultBalance)
	assert.Equal(t, StatusCompleted, tx.Status)
}

func TestTransactionMarkAsFailed(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	failTime := time.Date(2023, 1, 1, 12, 5, 0, 0, time.UTC)

	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(initialTime).Once()

	tx, err := NewTransaction(
		1,
		"tx123",
		string(SourceGame),
		string(StateLose),
		"100.00",
		mockTime,
	)
	require.NoError(t, err)

	// Initial state
	assert.Nil(t, tx.ProcessedAt)
	assert.Equal(t, "", tx.ErrorMessage)
	assert.Equal(t, StatusPending, tx.Status)

	// Mark as failed
	mockFailTime := coremocks.NewMockTimeProvider(t)
	mockFailTime.EXPECT().Now().Return(failTime).Once()

	errorMsg := "Insufficient balance"
	tx.MarkAsFailed(mockFailTime, errorMsg)

	// Check updated state
	require.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, failTime, *tx.ProcessedAt)
	assert.Equal(t, errorMsg, tx.ErrorMessage)
	assert.Equal(t, StatusFailed, tx.Status)
}

func TestTransactionIsCreditDebit(t *testing.T) {
	nowTime := time.Now()
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(nowTime).Maybe()

	t.Run("Win transaction is credit", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx1", string(SourceGame), string(StateWin), "100.00", mockTime)
		assert.True(t, tx.IsCredit())
		assert.False(t, tx.IsDebit())
	})

	t.Run("Lose transaction is debit", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx2", string(SourceGame), string(StateLose), "50.00", mockTime)
		assert.False(t, tx.IsCredit())
		assert.True(t, tx.IsDebit())
	})
}

func TestTransactionToResponse(t *testing.T) {
	nowTime := time.Now()
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(nowTime).Maybe()

	t.Run("Successful transaction response", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx1", string(SourceGame), string(StateWin), "100.00", mockTime)

		mockProcessTime := coremocks.NewMockTimeProvider(t)
		mockProcessTime.EXPECT().Now().Return(nowTime).Once()
		tx.MarkAsProcessed(mockProcessTime, "200.00")

		response := tx.ToResponse()
		assert.Equal(t, "tx1", response.TransactionID)
		assert.Equal(t, uint64(1), response.UserID)
		assert.True(t, response.Success)
		assert.Equal(t, "200.00", response.ResultBalance)
		assert.Equal(t, "", response.ErrorMessage)
	})

	t.Run("Failed transaction response", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx2", string(SourceGame), string(StateLose), "300.00", mockTime)

		mockFailTime := coremocks.NewMockTimeProvider(t)
		mockFailTime.EXPECT().Now().Return(nowTime).Once()
		tx.MarkAsFailed(mockFailTime, "Insufficient balance")

		response := tx.ToResponse()
		assert.Equal(t, "tx2", response.TransactionID)
		assert.Equal(t, uint64(1), response.UserID)
		assert.False(t, response.Success)
		assert.Equal(t, "", response.ResultBalance)
		assert.Equal(t, "Insufficient balance", response.ErrorMessage)
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isValidSourceType", func(t *testing.T) {
		validSources := []string{
			string(SourceGame),
			string(SourceServer),
			string(SourcePayment),
		}

		for _, source := range validSources {
			t.Run(source, func(t *testing.T) {
				assert.True(t, isValidSourceType(source))
			})
		}

		invalidSources := []string{
			"",
			"invalid",
			"GAME", // Case-sensitive
			"game-type",
		}

		for _, source := range invalidSources {
			t.Run(source, func(t *testing.T) {
				assert.False(t, isValidSourceType(source))
			})
		}
	})

	t.Run("isValidState", func(t *testing.T) {
		validStates := []string{
			string(StateWin),
			string(StateLose),
		}

		for _, state := range validStates {
			t.Run(state, func(t *testing.T) {
				assert.True(t, isValidState(state))
			})
		}

		invalidStates := []string{
			"",
			"invalid",
			"WIN", // Case-sensitive
			"draw",
		}

		for _, state := range invalidStates {
			t.Run(state, func(t *testing.T) {
				assert.False(t, isValidState(state))
			})
		}
	})
}
