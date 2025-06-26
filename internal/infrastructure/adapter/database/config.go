package database

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents database configuration
type Config struct {
	Driver          string        `mapstructure:"db_driver"`
	Host            string        `mapstructure:"db_host"`
	Port            int           `mapstructure:"db_port"`
	Username        string        `mapstructure:"db_username"`
	Password        string        `mapstructure:"db_password"`
	Database        string        `mapstructure:"db_name"`
	SSLMode         string        `mapstructure:"db_ssl_mode"`
	MaxOpenConns    int           `mapstructure:"db_max_open_conns"`
	MaxIdleConns    int           `mapstructure:"db_max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"db_conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"db_conn_max_idle_time"`
	MigrationPath   string        `mapstructure:"db_migration_path"`
	QueryTimeout    time.Duration `mapstructure:"db_query_timeout"`
	LogLevel        string        `mapstructure:"db_log_level"`
	RetryAttempts   int           `mapstructure:"db_retry_attempts"`
	RetryDelay      int           `mapstructure:"db_retry_delay"`
}

// DefaultConfig returns a Config with default values
// No sensitive information is hardcoded - all must come from environment variables
func DefaultConfig() *Config {
	config := &Config{
		Driver:          configEnvOrDefault("BP_DB_DRIVER", "postgres"),
		Host:            configEnv("BP_DB_HOST"),
		Port:            configEnvAsInt("BP_DB_PORT", 5432),
		Username:        configEnv("BP_DB_USERNAME"),
		Password:        configEnv("BP_DB_PASSWORD"),
		Database:        configEnv("BP_DB_NAME"),
		SSLMode:         configEnvOrDefault("BP_DB_SSL_MODE", "disable"),
		MaxOpenConns:    configEnvAsInt("BP_DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    configEnvAsInt("BP_DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: time.Duration(configEnvAsInt("BP_DB_CONN_MAX_LIFETIME_MINUTES", 5)) * time.Minute,
		ConnMaxIdleTime: time.Duration(configEnvAsInt("BP_DB_CONN_MAX_IDLE_TIME_MINUTES", 5)) * time.Minute,
		MigrationPath:   configEnvOrDefault("BP_DB_MIGRATION_PATH", "migrations"),
		QueryTimeout:    time.Duration(configEnvAsInt("BP_DB_QUERY_TIMEOUT_SECONDS", 10)) * time.Second,
		LogLevel:        configEnvOrDefault("BP_LOGGER_LEVEL", "info"),
		RetryAttempts:   configEnvAsInt("BP_DB_RETRY_ATTEMPTS", 3),
		RetryDelay:      configEnvAsInt("BP_DB_RETRY_DELAY_SECONDS", 5),
	}

	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Host == "" {
		return errors.New("database host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Port)
	}
	if c.Username == "" {
		return errors.New("database username is required")
	}
	if c.Password == "" {
		return errors.New("database password is required")
	}
	if c.Database == "" {
		return errors.New("database name is required")
	}

	validDrivers := map[string]bool{
		"postgres": true,
		"mysql":    true,
		"sqlite":   true,
	}
	if !validDrivers[c.Driver] {
		return fmt.Errorf("unsupported database driver: %s", c.Driver)
	}

	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
		"prefer":      true,
	}
	if !validSSLModes[c.SSLMode] {
		return fmt.Errorf("invalid SSL mode: %s", c.SSLMode)
	}

	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max open connections must be positive, got: %d", c.MaxOpenConns)
	}
	if c.MaxIdleConns <= 0 {
		return fmt.Errorf("max idle connections must be positive, got: %d", c.MaxIdleConns)
	}
	if c.QueryTimeout <= 0 {
		return errors.New("query timeout must be positive")
	}
	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts must be non-negative, got: %d", c.RetryAttempts)
	}
	if c.RetryDelay < 0 {
		return fmt.Errorf("retry delay must be non-negative, got: %d", c.RetryDelay)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// DSN returns the database connection string
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode,
	)
}

// WithMaxOpenConnections returns a copy of the config with updated max open connections
func (c *Config) WithMaxOpenConnections(max int) *Config {
	newConfig := *c
	newConfig.MaxOpenConns = max
	return &newConfig
}

// WithMaxIdleConnections returns a copy of the config with updated max idle connections
func (c *Config) WithMaxIdleConnections(max int) *Config {
	newConfig := *c
	newConfig.MaxIdleConns = max
	return &newConfig
}

// WithQueryTimeout returns a copy of the config with updated query timeout
func (c *Config) WithQueryTimeout(timeout time.Duration) *Config {
	newConfig := *c
	newConfig.QueryTimeout = timeout
	return &newConfig
}

// Helper functions for environment variables

// configEnv gets a value from environment variables with no default
// Returns an empty string if not found
func configEnv(key string) string {
	return os.Getenv(key)
}

// configEnvOrDefault gets a value from environment variables with a default value
func configEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// configEnvAsInt gets an integer value from environment variables with a default
func configEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
