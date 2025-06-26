package migration

import (
	"context"
	"errors"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/model"
	"gorm.io/gorm"
)

const (
	// CurrentSchemaVersion represents the current database schema version
	CurrentSchemaVersion = "1.0.1"
)

// MigrationManager manages database migrations
type MigrationManager struct {
	db               *gorm.DB
	logger           coreport.Logger
	timeProvider     coreport.TimeProvider
	advancedIndexMgr *AdvancedIndexManager
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *gorm.DB, logger coreport.Logger) *MigrationManager {
	return &MigrationManager{
		db:               db,
		logger:           logger,
		advancedIndexMgr: NewAdvancedIndexManager(db, logger),
	}
}

// NewMigrationManagerWithTimeProvider creates a new migration manager with time provider
func NewMigrationManagerWithTimeProvider(db *gorm.DB, logger coreport.Logger, timeProvider coreport.TimeProvider) *MigrationManager {
	return &MigrationManager{
		db:               db,
		logger:           logger,
		timeProvider:     timeProvider,
		advancedIndexMgr: NewAdvancedIndexManager(db, logger),
	}
}

// MigrateAll performs all migrations
func (m *MigrationManager) MigrateAll() error {
	m.logger.Info("Starting database migrations", map[string]any{
		"target_version": CurrentSchemaVersion,
	})

	// Create migration version table first
	if err := m.db.AutoMigrate(&model.MigrationVersion{}); err != nil {
		m.logger.Error("Failed to create migration version table", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Check current version
	currentVersion, err := m.GetCurrentVersion(context.Background())
	if err != nil {
		m.logger.Error("Failed to check current schema version", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	if currentVersion == CurrentSchemaVersion {
		m.logger.Info("Database already at target version, skipping migration", map[string]any{
			"version": currentVersion,
		})
		return nil
	}

	m.logger.Info("Current database version", map[string]any{
		"version": currentVersion,
	})

	// Auto-migrate models
	if err := m.autoMigrateModels(); err != nil {
		m.logger.Error("Failed to auto-migrate models", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Run custom migrations based on version
	if err := m.runVersionedMigrations(currentVersion); err != nil {
		m.logger.Error("Failed to run versioned migrations", map[string]any{
			"error":           err.Error(),
			"current_version": currentVersion,
			"target_version":  CurrentSchemaVersion,
		})
		return err
	}

	// Create basic indexes
	if err := m.createIndexes(); err != nil {
		m.logger.Error("Failed to create indexes", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Create advanced PostgreSQL indexes for better performance
	if err := m.createAdvancedIndexes(); err != nil {
		m.logger.Error("Failed to create advanced indexes", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Apply performance tweaks
	if err := m.applyPerformanceTweaks(); err != nil {
		m.logger.Error("Failed to apply performance tweaks", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	// Update migration version
	if err := m.setVersion(context.Background(), CurrentSchemaVersion, "Full schema migration"); err != nil {
		m.logger.Error("Failed to update schema version", map[string]any{
			"error":   err.Error(),
			"version": CurrentSchemaVersion,
		})
		return err
	}

	m.logger.Info("Database migrations completed successfully", map[string]any{
		"version": CurrentSchemaVersion,
	})
	return nil
}

// GetCurrentVersion gets the current migration version
func (m *MigrationManager) GetCurrentVersion(ctx context.Context) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var version model.MigrationVersion
	result := m.db.WithContext(ctx).Order("applied_at desc").First(&version)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", nil // No version found
		}
		return "", result.Error
	}

	return version.Version, nil
}

// setVersion records a new migration version
func (m *MigrationManager) setVersion(ctx context.Context, version string, details string) error {
	var appliedAt time.Time
	if m.timeProvider != nil {
		appliedAt = m.timeProvider.Now()
	} else {
		appliedAt = time.Now()
	}

	migrationVersion := model.MigrationVersion{
		Version:   version,
		AppliedAt: appliedAt,
		Details:   details,
	}

	result := m.db.WithContext(ctx).Create(&migrationVersion)
	return result.Error
}

// autoMigrateModels auto-migrates database models
func (m *MigrationManager) autoMigrateModels() error {
	m.logger.Info("Auto-migrating database models", nil)

	// Auto-migrate all models
	return m.db.AutoMigrate(
		&model.User{},
		&model.UserLock{},
		&model.Transaction{},
	)
}

// runVersionedMigrations runs migrations specific to version transitions
func (m *MigrationManager) runVersionedMigrations(currentVersion string) error {
	m.logger.Info("Running versioned migrations", map[string]any{
		"from": currentVersion,
		"to":   CurrentSchemaVersion,
	})

	// If starting from scratch
	if currentVersion == "" {
		return m.runBaseMigrations()
	}

	// Apply migrations based on current version
	switch currentVersion {
	case "0.9.0":
		if err := m.migrateFrom0_9_0To1_0_0(); err != nil {
			return err
		}
		fallthrough
	case "0.9.5":
		if err := m.migrateFrom0_9_5To1_0_0(); err != nil {
			return err
		}
		fallthrough
	case "1.0.0":
		if err := m.migrateFrom1_0_0To1_0_1(); err != nil {
			return err
		}
	}

	return nil
}

// runBaseMigrations runs the base migrations for a new database
func (m *MigrationManager) runBaseMigrations() error {
	m.logger.Info("Running base migrations", nil)

	// Add any SQL that needs to run for fresh migrations
	// Example: setting up sequences, special constraints, etc.

	return nil
}

// migrateFrom0_9_0To1_0_0 migrates from version 0.9.0 to 1.0.0
func (m *MigrationManager) migrateFrom0_9_0To1_0_0() error {
	m.logger.Info("Migrating from v0.9.0 to v1.0.0", nil)

	// Add migration steps specific to this version transition
	// Example: Adding columns, changing types, etc.

	return nil
}

// migrateFrom0_9_5To1_0_0 migrates from version 0.9.5 to 1.0.0
func (m *MigrationManager) migrateFrom0_9_5To1_0_0() error {
	m.logger.Info("Migrating from v0.9.5 to v1.0.0", nil)

	// Add migration steps specific to this version transition

	return nil
}

// migrateFrom1_0_0To1_0_1 migrates from version 1.0.0 to 1.0.1
func (m *MigrationManager) migrateFrom1_0_0To1_0_1() error {
	m.logger.Info("Migrating from v1.0.0 to v1.0.1", nil)

	// Add timestamps to user_locks table
	migration := NewAddTimestampsToUserLocks(m.db, m.logger)
	return migration.Run(context.Background())
}

// createIndexes creates basic database indexes
func (m *MigrationManager) createIndexes() error {
	m.logger.Info("Creating database indexes", nil)

	// Create unique index for transaction ID to prevent duplicate transactions
	if err := m.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_transaction_id_unique ON transactions (transaction_id)").Error; err != nil {
		return err
	}

	// Create user ID index for transactions
	if err := m.db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions (user_id)").Error; err != nil {
		return err
	}

	// Create index for user locks
	if err := m.db.Exec("CREATE INDEX IF NOT EXISTS idx_user_locks_expires_at ON user_locks (expires_at)").Error; err != nil {
		return err
	}

	return nil
}

// createAdvancedIndexes creates advanced PostgreSQL indexes
func (m *MigrationManager) createAdvancedIndexes() error {
	return m.advancedIndexMgr.CreateAdvancedIndexes()
}

// applyPerformanceTweaks applies PostgreSQL performance tweaks
func (m *MigrationManager) applyPerformanceTweaks() error {
	return m.advancedIndexMgr.CreatePerformanceTweaks()
}
