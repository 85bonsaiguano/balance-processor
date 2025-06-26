package entity

import (
	"testing"
	"time"

	coremocks "github.com/amirhossein-jamali/balance-processor/mocks/port/core"
	"github.com/stretchr/testify/assert"
)

func TestUserToBalanceResponse(t *testing.T) {
	t.Run("Converts user to balance response", func(t *testing.T) {
		// Create a test user with a known balance
		fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		mockTime := coremocks.NewMockTimeProvider(t)
		mockTime.EXPECT().Now().Return(fixedTime).Once()

		user, err := NewUser(42, "123.45", mockTime)
		assert.NoError(t, err)

		// Convert to balance response
		response := UserToBalanceResponse(user)

		// Verify the conversion
		assert.Equal(t, uint64(42), response.UserID)
		assert.Equal(t, "123.45", response.Balance)
	})

	t.Run("Handles zero balance", func(t *testing.T) {
		nowTime := time.Now()
		mockTime := coremocks.NewMockTimeProvider(t)
		mockTime.EXPECT().Now().Return(nowTime).Once()

		user, err := NewUser(123, "0.00", mockTime)
		assert.NoError(t, err)

		response := UserToBalanceResponse(user)

		assert.Equal(t, uint64(123), response.UserID)
		assert.Equal(t, "0.00", response.Balance)
	})

	t.Run("Handles large balance values", func(t *testing.T) {
		nowTime := time.Now()
		mockTime := coremocks.NewMockTimeProvider(t)
		mockTime.EXPECT().Now().Return(nowTime).Once()

		user, err := NewUser(999, "9876543.21", mockTime)
		assert.NoError(t, err)

		response := UserToBalanceResponse(user)

		assert.Equal(t, uint64(999), response.UserID)
		assert.Equal(t, "9876543.21", response.Balance)
	})
}
