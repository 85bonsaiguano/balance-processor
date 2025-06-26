package database

import (
	"errors"
	"fmt"
	"strings"

	domainErr "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	"gorm.io/gorm"
)

// EntityType represents the type of entity for errors mapping
type EntityType string

const (
	// EntityTypeUser represents the user entity
	EntityTypeUser EntityType = "user"
	// EntityTypeTransaction represents the transaction entity
	EntityTypeTransaction EntityType = "transaction"
	// EntityTypeUserLock represents the user lock entity
	EntityTypeUserLock EntityType = "user_lock"
)

// ErrorMapper maps database errors to domain errors
type ErrorMapper struct{}

// NewErrorMapper creates a new ErrorMapper
func NewErrorMapper() *ErrorMapper {
	return &ErrorMapper{}
}

// MapError maps a database error to a domain error
func (m *ErrorMapper) MapError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check for common GORM errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domainErr.ErrUserNotFound
	}

	// Check for PostgreSQL specific errors
	errMsg := strings.ToLower(err.Error())

	switch {
	// Transaction and locking errors
	case strings.Contains(errMsg, "deadlock") ||
		strings.Contains(errMsg, "serialization") ||
		strings.Contains(errMsg, "lock timeout"):
		return domainErr.ErrUserLocked

	// Duplicate key errors
	case strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "unique constraint"):
		if strings.Contains(errMsg, "transaction") {
			return domainErr.ErrDuplicateTransaction
		}
		return domainErr.ErrDuplicateUser

	// Constraint violations
	case strings.Contains(errMsg, "check constraint") ||
		strings.Contains(errMsg, "foreign key constraint"):
		return domainErr.ErrConstraintViolation

	// Connection issues
	case strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no connection") ||
		strings.Contains(errMsg, "connection reset"):
		return domainErr.ErrDatabaseConnection

	// Timeout errors
	case strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded"):
		return fmt.Errorf("%w: %s operation timed out", domainErr.ErrDatabaseConnection, operation)

	// Default error
	default:
		return domainErr.ErrInternalServer
	}
}

// MapEntityNotFoundError maps database errors to specific entity not found errors
func (m *ErrorMapper) MapEntityNotFoundError(err error, entityType EntityType) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		switch entityType {
		case EntityTypeUser:
			return domainErr.ErrUserNotFound
		case EntityTypeTransaction:
			return domainErr.ErrTransactionNotFound
		default:
			return domainErr.ErrNotFound
		}
	}

	return m.MapError(err, string(entityType))
}

// MapUserNotFoundError maps database errors to user not found errors
func (m *ErrorMapper) MapUserNotFoundError(err error) error {
	return m.MapEntityNotFoundError(err, EntityTypeUser)
}

// MapTransactionNotFoundError maps database errors to transaction not found errors
func (m *ErrorMapper) MapTransactionNotFoundError(err error) error {
	return m.MapEntityNotFoundError(err, EntityTypeTransaction)
}
