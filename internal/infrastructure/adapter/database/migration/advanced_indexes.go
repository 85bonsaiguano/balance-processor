package migration

import (
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"gorm.io/gorm"
)

// AdvancedIndexManager manages PostgreSQL-specific advanced indexes
type AdvancedIndexManager struct {
	db     *gorm.DB
	logger coreport.Logger
}

// NewAdvancedIndexManager creates a new advanced index manager
func NewAdvancedIndexManager(db *gorm.DB, logger coreport.Logger) *AdvancedIndexManager {
	return &AdvancedIndexManager{
		db:     db,
		logger: logger,
	}
}

// CreateAdvancedIndexes creates advanced PostgreSQL indexes for better performance
func (m *AdvancedIndexManager) CreateAdvancedIndexes() error {
	m.logger.Info("Creating advanced PostgreSQL indexes", nil)

	// Create unique index on transaction_id for fast idempotency checks
	if err := m.db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_transaction_id 
		ON transactions (transaction_id)
	`).Error; err != nil {
		m.logger.Error("Failed to create unique index on transaction_id", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create index on user_locks to improve locking performance
	if err := m.db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_locks_user_id 
		ON user_locks (user_id)
	`).Error; err != nil {
		m.logger.Error("Failed to create unique index on user_locks.user_id", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create index on user_locks expiration time for cleanup
	if err := m.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_locks_expires_at 
		ON user_locks (expires_at)
	`).Error; err != nil {
		m.logger.Error("Failed to create index on user_locks.expires_at", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create composite index for user_id and state to speed up filtered queries
	if err := m.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_transactions_user_state 
		ON transactions (user_id, state)
	`).Error; err != nil {
		m.logger.Error("Failed to create user_state composite index", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create partial index for successful transactions
	if err := m.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_transactions_successful 
		ON transactions (user_id, created_at) 
		WHERE status = 'success'
	`).Error; err != nil {
		m.logger.Error("Failed to create successful transactions partial index", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create BRIN index for created_at (more efficient for temporal data)
	if err := m.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_transactions_created_at_brin 
		ON transactions USING BRIN (created_at)
		WITH (pages_per_range = 32)
	`).Error; err != nil {
		m.logger.Error("Failed to create BRIN index on created_at", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create index on transaction source_type
	if err := m.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_transactions_source_type
		ON transactions (source_type)
	`).Error; err != nil {
		m.logger.Error("Failed to create index on source_type", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	m.logger.Info("Advanced PostgreSQL indexes created successfully", nil)
	return nil
}

// CreatePerformanceTweaks applies PostgreSQL performance tweaks
func (m *AdvancedIndexManager) CreatePerformanceTweaks() error {
	m.logger.Info("Applying PostgreSQL performance tweaks", nil)

	// Set fillfactor for transaction table to reduce page splits
	if err := m.db.Exec(`
		ALTER TABLE transactions SET (fillfactor = 90)
	`).Error; err != nil {
		m.logger.Warn("Failed to set fillfactor for transactions table", map[string]any{
			"error": err.Error(),
		})
		// Don't return error as this is not critical
	}

	// Set statistics target for better query planning
	if err := m.db.Exec(`
		ALTER TABLE transactions ALTER COLUMN user_id SET STATISTICS 1000
	`).Error; err != nil {
		m.logger.Warn("Failed to set statistics target for user_id", map[string]any{
			"error": err.Error(),
		})
		// Don't return error as this is not critical
	}

	m.logger.Info("PostgreSQL performance tweaks applied successfully", nil)
	return nil
}
