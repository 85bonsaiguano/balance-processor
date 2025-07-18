package entity

import (
	"testing"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"github.com/stretchr/testify/assert"
)

func TestValidateAndConvertAmount(t *testing.T) {
	t.Run("Valid amounts", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected int64
		}{
			{"100.00", 10000},
			{"0.01", 1},
			{"0.10", 10},
			{"1", 100},
			{"1.5", 150},
			{"1234567.89", 123456789},
			{"0.00", 0},
			{"0", 0},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				cents, err := ValidateAndConvertAmount(tc.input)
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, cents)
			})
		}
	})

	t.Run("Invalid amounts", func(t *testing.T) {
		testCases := []struct {
			input       string
			errorType   error
			description string
		}{
			{"", errs.ErrInvalidAmount, "Empty string"},
			{"   ", errs.ErrInvalidAmount, "Whitespace only"},
			{"-1.00", errs.ErrNegativeAmount, "Negative amount"},
			{"1.234", errs.ErrInvalidAmount, "Too many decimal places"},
			{"abc", errs.ErrInvalidAmount, "Non-numeric"},
			{"1,000.00", errs.ErrInvalidAmount, "Comma as thousands separator"},
			{"1.00.00", errs.ErrInvalidAmount, "Multiple decimal points"},
			{"$100", errs.ErrInvalidAmount, "Currency symbol"},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				_, err := ValidateAndConvertAmount(tc.input)
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.errorType)
			})
		}
	})

	t.Run("Edge cases", func(t *testing.T) {
		// Very large valid number
		cents, err := ValidateAndConvertAmount("9999999999.99")
		assert.NoError(t, err)
		assert.Equal(t, int64(999999999999), cents)

		// Zero with decimal
		cents, err = ValidateAndConvertAmount("0.00")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), cents)
	})
}

func TestAmountInCentsToString(t *testing.T) {
	testCases := []struct {
		cents    int64
		expected string
	}{
		{10000, "100.00"},
		{1, "0.01"},
		{10, "0.10"},
		{100, "1.00"},
		{150, "1.50"},
		{123456789, "1234567.89"},
		{0, "0.00"},
		{-10000, "-100.00"},
		{-1, "-0.01"},
		{2147483647, "21474836.47"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := AmountInCentsToString(tc.cents)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test conversion round trip: string -> cents -> string
	testCases := []string{
		"0.00",
		"0.01",
		"1.00",
		"10.50",
		"1234.56",
		"9999999.99",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			cents, err := ValidateAndConvertAmount(tc)
			assert.NoError(t, err)

			result := AmountInCentsToString(cents)
			assert.Equal(t, tc, result)
		})
	}
}

func TestEnsureTwoDecimalPlaces(t *testing.T) {
	t.Run("Valid cases", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			// Standard cases
			{"100", "100.00"},
			{"100.0", "100.00"},
			{"100.1", "100.10"},
			{"100.12", "100.12"},

			// Edge cases
			{"0", "0.00"},
			{"", "0.00"},
			{"   ", "0.00"},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result, err := EnsureTwoDecimalPlaces(tc.input)
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result, "Input: %s, Expected: %s, Got: %s", tc.input, tc.expected, result)
			})
		}
	})

	t.Run("Invalid cases - Too many decimal places", func(t *testing.T) {
		testCases := []string{
			"100.123",
			"100.129",
			"531.959",
			"10.999",
		}

		for _, input := range testCases {
			t.Run(input, func(t *testing.T) {
				_, err := EnsureTwoDecimalPlaces(input)
				assert.Error(t, err)
				assert.ErrorIs(t, err, errs.ErrInvalidAmount)
			})
		}
	})
}
