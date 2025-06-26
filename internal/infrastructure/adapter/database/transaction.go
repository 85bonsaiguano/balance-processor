package database

import (
	"context"
	"fmt"
	"strings"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/repository"
	"gorm.io/gorm"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys
const txKey contextKey = "tx"

// UnitOfWork implements the unit of work pattern for database transactions
type UnitOfWork struct {
	db           *gorm.DB
	logger       coreport.Logger
	timeProvider coreport.TimeProvider
}

// NewUnitOfWork creates a new UnitOfWork instance
func NewUnitOfWork(db *gorm.DB, logger coreport.Logger, timeProvider coreport.TimeProvider) persistence.UnitOfWork {
	return &UnitOfWork{
		db:           db,
		logger:       logger,
		timeProvider: timeProvider,
	}
}

// Begin starts a new database transaction
func (u *UnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	u.logger.Debug("Beginning database transaction with SERIALIZABLE isolation", nil)

	// Start a transaction
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		u.logger.Error("Failed to begin transaction", map[string]any{"error": tx.Error.Error()})
		return ctx, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Set transaction isolation level explicitly to SERIALIZABLE
	if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE").Error; err != nil {
		tx.Rollback()
		u.logger.Error("Failed to set transaction isolation level", map[string]any{"error": err.Error()})
		return ctx, fmt.Errorf("failed to set transaction isolation level: %w", err)
	}

	// Store transaction in context
	return context.WithValue(ctx, txKey, tx), nil
}

// Commit commits the current transaction
func (u *UnitOfWork) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if !ok || tx == nil {
		return fmt.Errorf("no transaction found in context")
	}

	u.logger.Debug("Committing database transaction", nil)
	if err := tx.Commit().Error; err != nil {
		u.logger.Error("Failed to commit transaction", map[string]any{"error": err.Error()})
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the current transaction with improved error handling
func (u *UnitOfWork) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if !ok || tx == nil {
		return fmt.Errorf("no transaction found in context")
	}

	u.logger.Debug("Rolling back database transaction", nil)

	// Execute rollback and capture error
	err := tx.Rollback().Error

	// If the error indicates the transaction was already committed or rolled back,
	// log it as a warning but don't return an error
	if err != nil && strings.Contains(err.Error(), "already been committed or rolled back") {
		u.logger.Warn("Transaction has already been committed or rolled back", map[string]any{
			"error": err.Error(),
		})
		return nil
	}

	// For other errors, log and return
	if err != nil {
		u.logger.Error("Failed to rollback transaction", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// GetUserRepository returns a user repository in the current transaction
func (u *UnitOfWork) GetUserRepository(ctx context.Context) persistence.UserRepository {
	db := u.getDbFromContext(ctx)
	return repository.NewUserRepository(db, u.timeProvider, u.logger)
}

// GetTransactionRepository returns a transaction repository in the current transaction
func (u *UnitOfWork) GetTransactionRepository(ctx context.Context) persistence.TransactionRepository {
	db := u.getDbFromContext(ctx)
	return repository.NewTransactionRepository(db, u.logger)
}

// getDbFromContext retrieves the database instance from context
func (u *UnitOfWork) getDbFromContext(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if ok && tx != nil {
		return tx
	}
	return u.db.WithContext(ctx)
}
