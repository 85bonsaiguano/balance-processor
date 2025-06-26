package database

import (
	"fmt"
	"os"
	"testing"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/model"
	timeprovider "github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/time"
	"gorm.io/gorm"
)

// TestDBManager provides utilities for testing with a database
type TestDBManager struct {
	Manager      *Manager
	Config       *Config
	Logger       coreport.Logger
	TimeProvider coreport.TimeProvider
}

// NewTestDBManager creates a new test database manager
func NewTestDBManager(t *testing.T, logger coreport.Logger) *TestDBManager {
	t.Helper()

	// Create time provider
	timeProvider := timeprovider.NewRealTimeProvider()

	// Get test database configuration from environment or use defaults
	config := &Config{
		Driver:          "postgres",
		Host:            getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            getEnvIntOrDefault("TEST_DB_PORT", 5432),
		Username:        getEnvOrDefault("TEST_DB_USERNAME", "postgres"),
		Password:        getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		Database:        getEnvOrDefault("TEST_DB_DATABASE", "balance_processor_test"),
		SSLMode:         getEnvOrDefault("TEST_DB_SSL_MODE", "disable"),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		QueryTimeout:    5 * time.Second,
		LogLevel:        "silent", // Silent logging in tests by default
		RetryAttempts:   1,        // One attempt for tests to fail fast
		RetryDelay:      1,        // 1 second delay
	}

	manager := NewManager(config, logger, timeProvider)

	return &TestDBManager{
		Manager:      manager,
		Config:       config,
		Logger:       logger,
		TimeProvider: timeProvider,
	}
}

// Connect connects to the test database
func (m *TestDBManager) Connect(t *testing.T) error {
	t.Helper()

	// Connect to database and get the DB instance
	db, err := m.Manager.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
		return err
	}

	// Store the DB instance for later use
	m.Manager.db = db

	return nil
}

// Close closes the test database connection
func (m *TestDBManager) Close(t *testing.T) {
	t.Helper()

	if err := m.Manager.Close(); err != nil {
		t.Logf("Warning: Failed to close test database connection: %v", err)
	}
}

// SetupTestDB sets up the test database with required tables
func (m *TestDBManager) SetupTestDB(t *testing.T) {
	t.Helper()

	db := m.Manager.DB()

	// Drop all tables to ensure clean state
	if err := dropAllTables(db); err != nil {
		t.Fatalf("Failed to drop tables: %v", err)
	}

	// Create tables
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserLock{},
		&model.Transaction{},
	); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Create basic indexes
	if err := createTestIndexes(db); err != nil {
		t.Fatalf("Failed to create indexes: %v", err)
	}
}

// dropAllTables drops all tables in the test database
func dropAllTables(db *gorm.DB) error {
	return db.Exec(`
		DO $$ DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`).Error
}

// createTestIndexes creates basic indexes for testing
func createTestIndexes(db *gorm.DB) error {
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_transaction_id ON transactions (transaction_id)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions (user_id)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_user_locks_expires_at ON user_locks (expires_at)").Error; err != nil {
		return err
	}

	return nil
}

// TruncateAllTables truncates all tables in the test database
func (m *TestDBManager) TruncateAllTables(t *testing.T) {
	t.Helper()

	db := m.Manager.DB()

	if err := db.Exec(`
		DO $$ DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`).Error; err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}
}

// CreateTestUser creates a test user with the specified ID and balance
func (m *TestDBManager) CreateTestUser(t *testing.T, id uint64, balance int64) {
	t.Helper()

	db := m.Manager.DB()

	user := model.User{
		ID:               id,
		Balance:          balance,
		TransactionCount: 0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}

// Helper functions to get environment variables or defaults
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
