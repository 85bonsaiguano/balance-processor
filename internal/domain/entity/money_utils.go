package entity

import (
	"fmt"
	"strconv"
	"strings"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
)

// MoneyUtils contains utility functions for handling monetary values

// MaxDecimalPlaces defines the maximum number of decimal places allowed for money amounts
const MaxDecimalPlaces = 2

// ValidateAndConvertAmount validates and formats a string amount
// Uses a string-based approach to handle decimal places:
// - If no decimal point: adds ".00" and removes the point to get an integer
// - If one digit after decimal: adds a "0" and removes the point
// - If two digits after decimal: just removes the point
// Returns the amount as int64 and error if the validation fails
func ValidateAndConvertAmount(amount string) (int64, error) {
	// Trim whitespace and check for empty string
	amount = strings.TrimSpace(amount)
	if len(amount) == 0 {
		return 0, fmt.Errorf("%w: empty value", errs.ErrInvalidAmount)
	}

	// Check for negative values
	if strings.HasPrefix(amount, "-") {
		return 0, errs.ErrNegativeAmount
	}

	// Process based on presence of decimal point
	parts := strings.Split(amount, ".")

	if len(parts) > 2 {
		// Multiple decimal points
		return 0, fmt.Errorf("%w: invalid number format", errs.ErrInvalidAmount)
	}

	var integerValue string

	if len(parts) == 1 {
		// No decimal point - add ".00"
		integerValue = parts[0] + "00"
	} else {
		// Has decimal point
		switch len(parts[1]) {
		case 0:
			// Like "10." - add "00"
			integerValue = parts[0] + "00"
		case 1:
			// One digit after decimal - add one zero
			integerValue = parts[0] + parts[1] + "0"
		case 2:
			// Two digits after decimal - use as is
			integerValue = parts[0] + parts[1]
		default:
			// More than 2 digits - error
			return 0, fmt.Errorf("%w: maximum %d decimal places allowed", errs.ErrInvalidAmount, MaxDecimalPlaces)
		}
	}

	// Convert to integer
	value, err := strconv.ParseInt(integerValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errs.ErrInvalidAmount, err.Error())
	}

	return value, nil
}

// AmountInCentsToString converts integer amount to a decimal string
// For example:
// - 1015 becomes "10.15"
// - 1000 becomes "10.00"
func AmountInCentsToString(amountInCents int64) string {
	isNegative := amountInCents < 0
	if isNegative {
		amountInCents = -amountInCents
	}

	amountStr := fmt.Sprintf("%d", amountInCents)

	// Ensure minimum length
	for len(amountStr) < 3 {
		amountStr = "0" + amountStr
	}

	// Extract decimal parts
	decimalPos := len(amountStr) - 2
	wholePart := amountStr[:decimalPos]
	decimalPart := amountStr[decimalPos:]

	// Handle zero whole part
	if wholePart == "" {
		wholePart = "0"
	}

	// Format with sign
	if isNegative {
		return "-" + wholePart + "." + decimalPart
	}
	return wholePart + "." + decimalPart
}

// EnsureTwoDecimalPlaces ensures a string representation of money has exactly 2 decimal places
// It handles strings with different decimal formats and standardizes them for consistency
// Example: "10.1" becomes "10.10", "10" becomes "10.00", "10.156" becomes "10.16" (truncated)
func EnsureTwoDecimalPlaces(amount string) string {
	// Handle empty strings
	if len(strings.TrimSpace(amount)) == 0 {
		return "0.00"
	}

	// Manual handling to avoid floating-point precision issues
	parts := strings.Split(amount, ".")

	// No decimal point
	if len(parts) == 1 {
		return parts[0] + ".00"
	}

	// Has decimal point
	wholePart := parts[0]
	decimalPart := parts[1]

	// Handle different decimal part lengths
	switch len(decimalPart) {
	case 0:
		return wholePart + ".00"
	case 1:
		return wholePart + "." + decimalPart + "0"
	case 2:
		// Already has 2 decimal places, return as is
		return wholePart + "." + decimalPart
	default:
		// More than 2 digits, truncate to 2 digits
		// This is a simple truncation approach - preserves exact values rather than rounding
		return wholePart + "." + decimalPart[:2]
	}
}
