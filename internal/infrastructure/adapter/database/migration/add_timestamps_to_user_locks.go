package migration

import (
	"context"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"gorm.io/gorm"
)

// AddTimestampsToUserLocks is a migration to add created_at and updated_at columns to user_locks table
type AddTimestampsToUserLocks struct {
	db     *gorm.DB
	logger coreport.Logger
}

// NewAddTimestampsToUserLocks creates a new migration instance
func NewAddTimestampsToUserLocks(db *gorm.DB, logger coreport.Logger) *AddTimestampsToUserLocks {
	return &AddTimestampsToUserLocks{
		db:     db,
		logger: logger,
	}
}

// Run executes the migration
func (m *AddTimestampsToUserLocks) Run(ctx context.Context) error {
	m.logger.Info("Adding timestamp columns to user_locks table", nil)

	// Check if columns already exist
	var hasCreatedAt, hasUpdatedAt bool
	if err := m.checkColumnExists(&hasCreatedAt, &hasUpdatedAt); err != nil {
		return err
	}

	// Add created_at column if it doesn't exist
	if !hasCreatedAt {
		if err := m.db.Exec(`ALTER TABLE user_locks ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`).Error; err != nil {
			m.logger.Error("Failed to add created_at column", map[string]any{"error": err.Error()})
			return err
		}
	}

	// Add updated_at column if it doesn't exist
	if !hasUpdatedAt {
		if err := m.db.Exec(`ALTER TABLE user_locks ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`).Error; err != nil {
			m.logger.Error("Failed to add updated_at column", map[string]any{"error": err.Error()})
			return err
		}
	}

	m.logger.Info("Successfully added timestamp columns to user_locks table", nil)
	return nil
}

// checkColumnExists checks if the columns already exist in the table
func (m *AddTimestampsToUserLocks) checkColumnExists(hasCreatedAt, hasUpdatedAt *bool) error {
	// For PostgreSQL
	var columns []struct {
		ColumnName string `gorm:"column:column_name"`
	}

	err := m.db.Raw(`
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = 'user_locks' AND column_name IN ('created_at', 'updated_at')
	`).Scan(&columns).Error

	if err != nil {
		m.logger.Error("Failed to check columns existence", map[string]any{"error": err.Error()})
		return err
	}

	for _, column := range columns {
		if column.ColumnName == "created_at" {
			*hasCreatedAt = true
		}
		if column.ColumnName == "updated_at" {
			*hasUpdatedAt = true
		}
	}

	return nil
}
