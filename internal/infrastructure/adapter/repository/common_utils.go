package repository

import (
	"strings"
)

// ErrorType represents the type of database error that occurred
type ErrorType string

const (
	DuplicateKeyError ErrorType = "duplicate_key"
	TransientError    ErrorType = "transient"
	LockError         ErrorType = "lock"
	ConnectionError   ErrorType = "connection"
	ConstraintError   ErrorType = "constraint"
)

// ErrorClassifier provides methods to classify database errors
type ErrorClassifier struct{}

// NewErrorClassifier creates a new ErrorClassifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

// Classify returns the type of error
func (c *ErrorClassifier) Classify(err error) ErrorType {
	if err == nil {
		return ""
	}

	if c.IsDuplicateKeyError(err) {
		return DuplicateKeyError
	}
	if c.IsLockError(err) {
		return LockError
	}
	if c.IsTransientError(err) {
		return TransientError
	}
	if c.IsConnectionError(err) {
		return ConnectionError
	}
	if c.IsConstraintError(err) {
		return ConstraintError
	}

	return ""
}

// IsDuplicateKeyError checks if the error is a duplicate key error
func (c *ErrorClassifier) IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "UNIQUE constraint") ||
		strings.Contains(err.Error(), "Duplicate entry")
}

// IsTransientError checks if an error is transient and can be retried
func (c *ErrorClassifier) IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "EOF") ||
		strings.Contains(err.Error(), "server closed") ||
		strings.Contains(err.Error(), "broken pipe")
}

// IsLockError checks if the error is due to locking
func (c *ErrorClassifier) IsLockError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "deadlock") ||
		strings.Contains(err.Error(), "lock wait timeout") ||
		strings.Contains(err.Error(), "could not serialize access") ||
		strings.Contains(err.Error(), "serialization failure")
}

// IsConnectionError checks if the error is related to database connectivity
func (c *ErrorClassifier) IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "dial") ||
		strings.Contains(err.Error(), "network") ||
		c.IsTransientError(err)
}

// IsConstraintError checks if the error is related to constraint violations
func (c *ErrorClassifier) IsConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "constraint") ||
		strings.Contains(err.Error(), "violates") ||
		strings.Contains(err.Error(), "foreign key") ||
		strings.Contains(err.Error(), "not null") ||
		c.IsDuplicateKeyError(err)
}
