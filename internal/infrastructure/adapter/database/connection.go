package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connection holds database connection and configuration
type Connection struct {
	DB     *gorm.DB
	Config *Config
}

// NewConnection establishes a new database connection with the given configuration
func NewConnection(config *Config) (*Connection, error) {
	// Validate configuration before connecting
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Configure GORM logger based on config
	logLevel := logger.Info
	switch config.LogLevel {
	case "debug":
		logLevel = logger.Info
	case "info":
		logLevel = logger.Info
	case "warn":
		logLevel = logger.Warn
	case "error":
		logLevel = logger.Error
	}

	// Create GORM configuration
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(config.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		DB:     db,
		Config: config,
	}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.Close()
}

// WithContext returns a GORM DB instance with timeout context
func (c *Connection) WithContext() *gorm.DB {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.QueryTimeout)
	// Use defer to ensure cancel is called when this function returns
	// This prevents context leak
	defer cancel()
	return c.DB.WithContext(ctx)
}
