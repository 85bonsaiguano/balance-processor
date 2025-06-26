package entity

import (
	"testing"
	"time"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coremocks "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(fixedTime).Maybe()

	t.Run("Valid user creation", func(t *testing.T) {
		user, err := NewUser(1, "100.00", mockTime)

		require.NoError(t, err)
		assert.Equal(t, uint64(1), user.ID)
		assert.Equal(t, int64(10000), user.Balance())
		assert.Equal(t, "100.00", user.GetBalance())
		assert.Equal(t, fixedTime, user.CreatedAt)
		assert.Equal(t, fixedTime, user.UpdatedAt)
		assert.Equal(t, uint64(0), user.TransactionCount)
	})

	t.Run("Zero ID should return error", func(t *testing.T) {
		user, err := NewUser(0, "100.00", mockTime)

		assert.Error(t, err)
		assert.Equal(t, errs.ErrInvalidUserID, err)
		assert.Nil(t, user)
	})

	t.Run("Invalid balance format", func(t *testing.T) {
		testCases := []string{
			"invalid",
			"",
			"100.123",
			"$100.00",
		}

		for _, tc := range testCases {
			t.Run(tc, func(t *testing.T) {
				user, err := NewUser(1, tc, mockTime)
				assert.Error(t, err)
				assert.Nil(t, user)
			})
		}
	})

	t.Run("Very large balance", func(t *testing.T) {
		user, err := NewUser(1, "9999999999.99", mockTime)

		require.NoError(t, err)
		assert.Equal(t, int64(999999999999), user.Balance())
		assert.Equal(t, "9999999999.99", user.GetBalance())
	})
}

func TestUserSetBalance(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	updateTime := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(initialTime).Once()

	user, _ := NewUser(1, "100.00", mockTime)

	mockTime.EXPECT().Now().Return(updateTime).Once()
	user.SetBalance(20000, mockTime)

	assert.Equal(t, int64(20000), user.Balance())
	assert.Equal(t, "200.00", user.GetBalance())
	assert.Equal(t, initialTime, user.CreatedAt)
	assert.Equal(t, updateTime, user.UpdatedAt)
}

func TestUserIncrementTransactionCount(t *testing.T) {
	nowTime := time.Now()
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(nowTime).Maybe()

	user, _ := NewUser(1, "100.00", mockTime)

	assert.Equal(t, uint64(0), user.TransactionCount)

	user.IncrementTransactionCount()
	assert.Equal(t, uint64(1), user.TransactionCount)

	user.IncrementTransactionCount()
	assert.Equal(t, uint64(2), user.TransactionCount)
}

func TestUserCanDeduct(t *testing.T) {
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(time.Now()).Maybe()

	user, _ := NewUser(1, "100.00", mockTime)

	t.Run("Valid deduction amount", func(t *testing.T) {
		canDeduct, err := user.CanDeduct("50.00")
		assert.NoError(t, err)
		assert.True(t, canDeduct)
	})

	t.Run("Exact amount", func(t *testing.T) {
		canDeduct, err := user.CanDeduct("100.00")
		assert.NoError(t, err)
		assert.True(t, canDeduct)
	})

	t.Run("Insufficient balance", func(t *testing.T) {
		canDeduct, err := user.CanDeduct("150.00")
		assert.NoError(t, err)
		assert.False(t, canDeduct)
	})

	t.Run("Invalid amount format", func(t *testing.T) {
		canDeduct, err := user.CanDeduct("invalid")
		assert.Error(t, err)
		assert.False(t, canDeduct)
	})
}

func TestApplyWinTransaction(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	updateTime := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(initialTime).Once()

	user, _ := NewUser(1, "100.00", mockTime)

	mockTime.EXPECT().Now().Return(updateTime).Once()
	user.ApplyWinTransaction(5000, mockTime)

	assert.Equal(t, int64(15000), user.Balance())
	assert.Equal(t, "150.00", user.GetBalance())
	assert.Equal(t, uint64(1), user.TransactionCount)
	assert.Equal(t, updateTime, user.UpdatedAt)

	// Test with zero amount
	mockTime.EXPECT().Now().Return(updateTime).Once()
	user.ApplyWinTransaction(0, mockTime)
	assert.Equal(t, int64(15000), user.Balance())
	assert.Equal(t, uint64(2), user.TransactionCount)

	// Test with large amount
	mockTime.EXPECT().Now().Return(updateTime).Once()
	user.ApplyWinTransaction(1000000, mockTime)
	assert.Equal(t, int64(1015000), user.Balance())
	assert.Equal(t, "10150.00", user.GetBalance())
	assert.Equal(t, uint64(3), user.TransactionCount)
}

func TestApplyLoseTransaction(t *testing.T) {
	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	updateTime := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(initialTime).Once()

	user, _ := NewUser(1, "100.00", mockTime)

	t.Run("Valid deduction", func(t *testing.T) {
		mockTimeLocal := coremocks.NewMockTimeProvider(t)
		mockTimeLocal.EXPECT().Now().Return(updateTime).Once()

		err := user.ApplyLoseTransaction(5000, mockTimeLocal)

		assert.NoError(t, err)
		assert.Equal(t, int64(5000), user.Balance())
		assert.Equal(t, "50.00", user.GetBalance())
		assert.Equal(t, uint64(1), user.TransactionCount)
		assert.Equal(t, updateTime, user.UpdatedAt)
	})

	t.Run("Exact balance deduction", func(t *testing.T) {
		mockTimeLocal := coremocks.NewMockTimeProvider(t)
		mockTimeLocal.EXPECT().Now().Return(updateTime).Once()

		err := user.ApplyLoseTransaction(5000, mockTimeLocal)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), user.Balance())
		assert.Equal(t, "0.00", user.GetBalance())
		assert.Equal(t, uint64(2), user.TransactionCount)
	})

	t.Run("Insufficient balance", func(t *testing.T) {
		err := user.ApplyLoseTransaction(1, mockTime)

		assert.Equal(t, errs.ErrInsufficientBalance, err)
		assert.Equal(t, int64(0), user.Balance())
		assert.Equal(t, uint64(2), user.TransactionCount)
	})

	t.Run("Zero amount deduction", func(t *testing.T) {
		mockTimeLocal := coremocks.NewMockTimeProvider(t)
		mockTimeLocal.EXPECT().Now().Return(initialTime).Once()

		user, _ := NewUser(2, "100.00", mockTimeLocal)

		mockTimeLocal.EXPECT().Now().Return(updateTime).Once()
		err := user.ApplyLoseTransaction(0, mockTimeLocal)

		assert.NoError(t, err)
		assert.Equal(t, int64(10000), user.Balance())
		assert.Equal(t, uint64(1), user.TransactionCount)
	})
}

func TestIntegrationOfTransactions(t *testing.T) {
	nowTime := time.Now()
	mockTime := coremocks.NewMockTimeProvider(t)
	mockTime.EXPECT().Now().Return(nowTime).Maybe()

	user, _ := NewUser(1, "100.00", mockTime)

	// Series of transactions
	user.ApplyWinTransaction(5000, mockTime)         // +50.00
	err := user.ApplyLoseTransaction(2000, mockTime) // -20.00
	require.NoError(t, err)
	user.ApplyWinTransaction(1000, mockTime)        // +10.00
	err = user.ApplyLoseTransaction(3000, mockTime) // -30.00
	require.NoError(t, err)

	// Final balance should be 100 + 50 - 20 + 10 - 30 = 110
	assert.Equal(t, int64(11000), user.Balance())
	assert.Equal(t, "110.00", user.GetBalance())
	assert.Equal(t, uint64(4), user.TransactionCount)

	// Check edge case with exact deduction
	err = user.ApplyLoseTransaction(11000, mockTime)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), user.Balance())

	// Now should fail
	err = user.ApplyLoseTransaction(1, mockTime)
	assert.Equal(t, errs.ErrInsufficientBalance, err)
}
