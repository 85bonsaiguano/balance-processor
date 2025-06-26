package database

import (
	"context"
	"fmt"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/domain/port/persistence"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/database/migration"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Manager manages database connections
type Manager struct {
	config            *Config
	db                *gorm.DB
	logger            coreport.Logger
	errorMapper       *ErrorMapper
	migrationMgr      *migration.MigrationManager
	connectionMonitor *ConnectionPoolMonitor
	timeProvider      coreport.TimeProvider
}

// NewManager creates a new database manager
func NewManager(config *Config, logger coreport.Logger, timeProvider coreport.TimeProvider) *Manager {
	return &Manager{
		config:       config,
		logger:       logger,
		errorMapper:  NewErrorMapper(),
		timeProvider: timeProvider,
	}
}

// Connect establishes a database connection with optimized settings
func (m *Manager) Connect() (*gorm.DB, error) {
	m.logger.Info("Connecting to database", map[string]any{
		"driver": m.config.Driver,
		"host":   m.config.Host,
		"port":   m.config.Port,
		"name":   m.config.Database,
	})

	var err error
	var gormDB *gorm.DB

	// Setup retry mechanism for initial connection
	for attempt := 0; attempt < m.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			m.logger.Warn("Retrying database connection", map[string]any{
				"attempt": attempt + 1,
				"of":      m.config.RetryAttempts,
				"delay":   fmt.Sprintf("%d", m.config.RetryDelay) + "s",
			})
			time.Sleep(time.Duration(m.config.RetryDelay) * time.Second)
		}

		// Get DSN for the selected driver
		dsn := m.getDSN()

		// Connect with gorm
		switch m.config.Driver {
		case "postgres":
			gormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: NewGormDatabaseLogger(m.logger),
				NowFunc: func() time.Time {
					return m.timeProvider.Now()
				},
				PrepareStmt: true, // Prepare statements for better performance
			})
		default:
			return nil, fmt.Errorf("unsupported database driver: %s", m.config.Driver)
		}

		if err == nil {
			break
		}

		m.logger.Error("Failed to connect to database", map[string]any{
			"error":   err.Error(),
			"attempt": attempt + 1,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", m.config.RetryAttempts, err)
	}

	// Configure connection pooling and query timeout
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(m.config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(m.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(m.config.ConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(m.config.ConnMaxIdleTime) * time.Minute)

	// Update the database with additional options
	gormDB = gormDB.WithContext(context.Background())

	// Set transaction timeout middleware
	gormDB = gormDB.Session(&gorm.Session{
		PrepareStmt: true,
	})

	m.logger.Info("Successfully connected to database", map[string]any{
		"driver":          m.config.Driver,
		"host":            m.config.Host,
		"port":            m.config.Port,
		"name":            m.config.Database,
		"max_open_conns":  m.config.MaxOpenConns,
		"max_idle_conns":  m.config.MaxIdleConns,
		"query_timeout_s": m.config.QueryTimeout,
	})

	// Register cleanup on application shutdown
	m.db = gormDB
	m.connectionMonitor = NewConnectionPoolMonitor(m, m.logger)

	// Start connection pool monitoring
	err = m.connectionMonitor.Start(30 * time.Second)
	if err != nil {
		m.logger.Warn("Failed to start connection pool monitoring", map[string]any{"error": err.Error()})
	}

	return m.db, nil
}

// DB returns the GORM database instance
func (m *Manager) DB() *gorm.DB {
	return m.db
}

// Close closes the database connection
func (m *Manager) Close() error {
	m.logger.Info("Closing database connection", nil)

	// Stop connection pool monitoring
	if m.connectionMonitor != nil {
		m.connectionMonitor.Stop()
	}

	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	return sqlDB.Close()
}

// WithTimeout returns a context with timeout for database operations
func (m *Manager) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, m.config.QueryTimeout)
}

// CreateUnitOfWork creates a new UnitOfWork instance
func (m *Manager) CreateUnitOfWork() persistence.UnitOfWork {
	return NewUnitOfWork(m.db, m.logger, m.timeProvider)
}

// GetErrorMapper returns the error mapper
func (m *Manager) GetErrorMapper() *ErrorMapper {
	return m.errorMapper
}

// MigrationManager returns the migration manager
func (m *Manager) MigrationManager() *migration.MigrationManager {
	return m.migrationMgr
}

// getDSN returns the DSN for the database connection
func (m *Manager) getDSN() string {
	// Use the DSN method from the config
	return m.config.DSN()
}
