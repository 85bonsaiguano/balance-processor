package entity

import (
	"fmt"
	"strconv"
	"strings"

	errs "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
)

// Package entity contains core domain entities and value objects for the balance processor.
//
// CONCURRENCY MODEL:
// The entities in this package follow an immutability pattern where appropriate to support
// concurrent access patterns. Key points to understand:
//
// 1. User Entity:
//    - Methods like ApplyWinTransaction and ApplyLoseTransaction create new User instances
//      and don't modify the original, making them safe for concurrent reads.
//    - However, SetBalance directly modifies the User and is NOT thread-safe. Callers must
//      ensure proper synchronization when using this method.
//
// 2. Transaction Entity:
//    - Creation methods (NewTransaction) are thread-safe as they create new instances.
//    - Status mutation methods (MarkAsProcessed, MarkAsFailed) modify the Transaction
//      directly and are NOT thread-safe. Proper synchronization must be handled by callers.
//
// 3. Money Utilities:
//    - All money utility functions are pure functions with no side effects and are thread-safe.
//
// For high-volume transaction processing, a synchronization strategy at the repository
// or service layer is necessary. This often includes database transactions, optimistic locking,
// or other concurrency control mechanisms beyond the scope of this domain package.

// MaxDecimalPlaces defines the maximum number of decimal places allowed for money amounts
const MaxDecimalPlaces = 2

// ValidateAndConvertAmount validates and formats a string amount to cents (int64).
// This function handles a variety of input formats and ensures precise money handling.
//
// Input requirements:
//   - Must be a non-negative number
//   - Maximum 2 decimal places allowed
//   - Must not exceed maximum int64 value when converted to cents
//
// Side effects: None - this is a pure function.
// Thread safety: This function is thread-safe as it only performs calculations.
//
// Returns the amount as int64 cents and error if the validation fails.
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

	// Overflow check using mathematical approach
	// Maximum value for int64 is 9,223,372,036,854,775,807
	// For money values with 2 decimal places, the maximum safe value would be:
	// 92,233,720,368,547,758.07

	// First check: simple length-based pre-check
	if len(integerValue) > 19 {
		return 0, errs.ErrAmountOverflow
	}

	// Second check: for values with length 19, need precise comparison
	if len(integerValue) == 19 {
		// Define max int64 value as a string
		maxInt64AsString := "9223372036854775807"

		// Direct string comparison for potential overflow
		if integerValue > maxInt64AsString {
			return 0, errs.ErrAmountOverflow
		}
	}

	// Convert to integer
	value, err := strconv.ParseInt(integerValue, 10, 64)
	if err != nil {
		// Handle potential overflow errors from ParseInt
		if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
			return 0, errs.ErrAmountOverflow
		}
		return 0, fmt.Errorf("%w: %s", errs.ErrInvalidAmount, err.Error())
	}

	return value, nil
}

// AmountInCentsToString converts integer amount to a decimal string
// For example:
// - 1015 becomes "10.15"
// - 1000 becomes "10.00"
//
// Side effects: None - this is a pure function.
// Thread safety: This function is thread-safe as it only performs calculations.
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

// EnsureTwoDecimalPlaces validates and standardizes a string representation of money to have exactly 2 decimal places
// It handles strings with different decimal formats and returns an error if more than 2 decimal places are provided
//
// Side effects: None - this is a pure function.
// Thread safety: This function is thread-safe as it only performs string operations.
//
// Returns the formatted string with exactly 2 decimal places or an error if invalid format is detected
func EnsureTwoDecimalPlaces(amount string) (string, error) {
	// Handle empty strings
	if len(strings.TrimSpace(amount)) == 0 {
		return "0.00", nil
	}

	// Manual handling to avoid floating-point precision issues
	parts := strings.Split(amount, ".")

	// No decimal point
	if len(parts) == 1 {
		return parts[0] + ".00", nil
	}

	// Has decimal point
	wholePart := parts[0]
	decimalPart := parts[1]

	// Handle different decimal part lengths
	switch {
	case len(decimalPart) == 0:
		return wholePart + ".00", nil
	case len(decimalPart) == 1:
		return wholePart + "." + decimalPart + "0", nil
	case len(decimalPart) == 2:
		// Already has 2 decimal places, return as is
		return wholePart + "." + decimalPart, nil
	default:
		// More than 2 digits, return error
		return "", fmt.Errorf("%w: maximum %d decimal places allowed", errs.ErrInvalidAmount, MaxDecimalPlaces)
	}
}

// ValidateDecimalPlaces checks if a string amount has more than MaxDecimalPlaces decimal places
// Returns true if the number of decimal places is valid (≤ MaxDecimalPlaces), false otherwise
//
// Side effects: None - this is a pure function.
// Thread safety: This function is thread-safe as it only performs string operations.
func ValidateDecimalPlaces(amount string) bool {
	parts := strings.Split(amount, ".")
	if len(parts) < 2 {
		// No decimal point
		return true
	}

	// Check decimal part length
	return len(parts[1]) <= MaxDecimalPlaces
}
