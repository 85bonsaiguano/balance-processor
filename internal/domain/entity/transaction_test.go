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
		assert.Equal(t, int64(10000), tx.AmountInCents)
		assert.Equal(t, fixedTime, tx.CreatedAt)
		assert.Nil(t, tx.ProcessedAt)
		assert.Equal(t, StatusPending, tx.Status)
		assert.Equal(t, "", tx.ErrorMessage)
	})

	t.Run("With custom status option", func(t *testing.T) {
		tx, err := NewTransaction(
			1,                  // userID
			"tx123",            // transactionID
			string(SourceGame), // sourceType
			string(StateWin),   // state
			"100.00",           // amount
			mockTime,
			WithCustomStatus(StatusCompleted),
		)

		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, tx.Status)
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
	assert.Equal(t, int64(0), tx.ResultBalanceInCents)
	assert.Equal(t, StatusPending, tx.Status)

	// Mark as processed
	mockProcessTime := coremocks.NewMockTimeProvider(t)
	mockProcessTime.EXPECT().Now().Return(processTime).Once()
	tx.MarkAsProcessed(mockProcessTime, 20000) // 200.00 in cents

	// Check updated state
	require.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, processTime, *tx.ProcessedAt)
	assert.Equal(t, int64(20000), tx.ResultBalanceInCents)
	assert.Equal(t, "200.00", tx.GetResultBalance())
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
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(fixedTime).Maybe()

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

func TestTransactionGetters(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(fixedTime).Maybe()

	t.Run("GetAmount and GetResultBalance", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx1", string(SourceGame), string(StateWin), "123.45", mockTime)

		assert.Equal(t, "123.45", tx.GetAmount())

		mockProcessTime := coremocks.NewMockTimeProvider(t)
		mockProcessTime.EXPECT().Now().Return(fixedTime).Once()
		tx.MarkAsProcessed(mockProcessTime, 67890) // 678.90 in cents

		assert.Equal(t, "678.90", tx.GetResultBalance())
	})
}

func TestTransactionStatusChecks(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(fixedTime).Maybe()

	t.Run("IsPending", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx1", string(SourceGame), string(StateWin), "100.00", mockTime)
		assert.True(t, tx.IsPending())
		assert.False(t, tx.IsAlreadyProcessed())
		assert.False(t, tx.IsFailed())
	})

	t.Run("IsAlreadyProcessed with completed", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx2", string(SourceGame), string(StateWin), "100.00", mockTime)

		mockProcessTime := coremocks.NewMockTimeProvider(t)
		mockProcessTime.EXPECT().Now().Return(fixedTime).Once()
		tx.MarkAsProcessed(mockProcessTime, 20000)

		assert.True(t, tx.IsAlreadyProcessed())
		assert.False(t, tx.IsPending())
		assert.False(t, tx.IsFailed())
	})

	t.Run("IsFailed", func(t *testing.T) {
		tx, _ := NewTransaction(1, "tx3", string(SourceGame), string(StateWin), "100.00", mockTime)

		mockFailTime := coremocks.NewMockTimeProvider(t)
		mockFailTime.EXPECT().Now().Return(fixedTime).Once()
		tx.MarkAsFailed(mockFailTime, "Error occurred")

		assert.True(t, tx.IsFailed())
		assert.True(t, tx.IsAlreadyProcessed())
		assert.False(t, tx.IsPending())
	})
}

func TestTransactionClone(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	processTime := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)
	newTime := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)

	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(initialTime).Once()

	tx, err := NewTransaction(1, "tx-clone", string(SourceGame), string(StateWin), "50.75", mockTime)
	require.NoError(t, err)

	mockProcessTime := coremocks.NewMockTimeProvider(t)
	mockProcessTime.EXPECT().Now().Return(processTime).Once()
	tx.MarkAsProcessed(mockProcessTime, 12345)

	// Create clone
	clone := tx.Clone()

	// Verify clone has identical values
	assert.Equal(t, tx.ID, clone.ID)
	assert.Equal(t, tx.UserID, clone.UserID)
	assert.Equal(t, tx.TransactionID, clone.TransactionID)
	assert.Equal(t, tx.SourceType, clone.SourceType)
	assert.Equal(t, tx.State, clone.State)
	assert.Equal(t, tx.AmountInCents, clone.AmountInCents)
	assert.Equal(t, tx.CreatedAt, clone.CreatedAt)
	assert.Equal(t, tx.ResultBalanceInCents, clone.ResultBalanceInCents)
	assert.Equal(t, tx.Status, clone.Status)
	assert.Equal(t, tx.ErrorMessage, clone.ErrorMessage)

	// ProcessedAt should be equal but a different pointer
	require.NotNil(t, clone.ProcessedAt)
	assert.Equal(t, *tx.ProcessedAt, *clone.ProcessedAt)
	assert.NotSame(t, tx.ProcessedAt, clone.ProcessedAt)

	// Modifying original should not affect clone
	mockNewTime := coremocks.NewMockTimeProvider(t)
	mockNewTime.EXPECT().Now().Return(newTime).Once()
	tx.MarkAsFailed(mockNewTime, "New error")

	assert.Equal(t, StatusFailed, tx.Status)
	assert.Equal(t, "New error", tx.ErrorMessage)
	assert.Equal(t, newTime, *tx.ProcessedAt)

	// Clone should be unchanged
	assert.Equal(t, StatusCompleted, clone.Status)
	assert.Equal(t, "", clone.ErrorMessage)
	assert.Equal(t, processTime, *clone.ProcessedAt)
}

func TestEnumValidation(t *testing.T) {
	t.Run("IsValidSourceType", func(t *testing.T) {
		validSources := []string{
			string(SourceGame),
			string(SourceServer),
			string(SourcePayment),
		}

		for _, source := range validSources {
			t.Run(source, func(t *testing.T) {
				assert.True(t, IsValidSourceType(source))
			})
		}

		invalidSources := []string{
			"",
			"invalid",
			"GAME", // Was case-sensitive in old implementation, now case-insensitive
			"game-type",
		}

		for _, source := range invalidSources {
			t.Run(source, func(t *testing.T) {
				// The updated implementation handles some cases differently
				if source == "GAME" {
					// Case-insensitive now
					assert.True(t, IsValidSourceType(source))
				} else {
					assert.False(t, IsValidSourceType(source))
				}
			})
		}
	})

	t.Run("IsValidState", func(t *testing.T) {
		validStates := []string{
			string(StateWin),
			string(StateLose),
		}

		for _, state := range validStates {
			t.Run(state, func(t *testing.T) {
				assert.True(t, IsValidState(state))
			})
		}

		invalidStates := []string{
			"",
			"invalid",
			"WIN", // Was case-sensitive in old implementation, now case-insensitive
			"draw",
		}

		for _, state := range invalidStates {
			t.Run(state, func(t *testing.T) {
				// The updated implementation handles some cases differently
				if state == "WIN" {
					// Case-insensitive now
					assert.True(t, IsValidState(state))
				} else {
					assert.False(t, IsValidState(state))
				}
			})
		}
	})
}

func TestBalanceEffect(t *testing.T) {
	t.Run("StateWin returns EffectIncrease", func(t *testing.T) {
		assert.Equal(t, EffectIncrease, StateWin.GetBalanceEffect())
	})

	t.Run("StateLose returns EffectDecrease", func(t *testing.T) {
		assert.Equal(t, EffectDecrease, StateLose.GetBalanceEffect())
	})

	t.Run("BalanceEffect.IsValid", func(t *testing.T) {
		assert.True(t, EffectIncrease.IsValid())
		assert.True(t, EffectDecrease.IsValid())
		assert.False(t, BalanceEffect("invalid").IsValid())
	})
}

func TestEnumRegistry(t *testing.T) {
	t.Run("Registry contains all transaction states", func(t *testing.T) {
		values := StateWin.Values()
		assert.Len(t, values, 2)
		assert.Contains(t, values, StateWin)
		assert.Contains(t, values, StateLose)
	})

	t.Run("Registry contains all source types", func(t *testing.T) {
		values := SourceGame.Values()
		assert.Len(t, values, 3)
		assert.Contains(t, values, SourceGame)
		assert.Contains(t, values, SourceServer)
		assert.Contains(t, values, SourcePayment)
	})

	t.Run("Registry contains all statuses", func(t *testing.T) {
		values := StatusPending.Values()
		assert.Len(t, values, 3)
		assert.Contains(t, values, StatusPending)
		assert.Contains(t, values, StatusCompleted)
		assert.Contains(t, values, StatusFailed)
	})

	t.Run("Can register new values", func(t *testing.T) {
		// Create a new state
		newState := TransactionState("draw")

		// Check it's not already valid
		assert.False(t, newState.IsValid())

		// Register it
		RegisterTransactionState(newState)

		// Now it should be valid
		assert.True(t, newState.IsValid())

		// And ParseTransactionState should work with it
		parsed, err := ParseTransactionState("draw")
		assert.NoError(t, err)
		assert.Equal(t, newState, parsed)
	})
}

func TestWithCustomStatus(t *testing.T) {
	t.Run("Valid status", func(t *testing.T) {
		option := WithCustomStatus(StatusCompleted)
		tx := &Transaction{Status: StatusPending}

		err := option(tx)
		assert.NoError(t, err)
		assert.Equal(t, StatusCompleted, tx.Status)
	})

	t.Run("Invalid status", func(t *testing.T) {
		option := WithCustomStatus(TransactionStatus("invalid"))
		tx := &Transaction{Status: StatusPending}

		err := option(tx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, errs.ErrInvalidState)
		assert.Equal(t, StatusPending, tx.Status) // Should not change
	})
}
