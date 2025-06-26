package config

import "time"

// Config holds all configuration for the application
type Config struct {
	Environment string           `mapstructure:"environment"`
	Server      ServerConfig     `mapstructure:"server"`
	Database    DatabaseConfig   `mapstructure:"database"`
	Logger      LoggerConfig     `mapstructure:"logger"`
	Transaction TransactionConfig `mapstructure:"transaction"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host             string        `mapstructure:"host"`
	Port             int           `mapstructure:"port"`
	ReadTimeout      time.Duration `mapstructure:"readTimeout"`  // seconds
	WriteTimeout     time.Duration `mapstructure:"writeTimeout"` // seconds
	IdleTimeout      time.Duration `mapstructure:"idleTimeout"`  // seconds
	ReadHeaderTimeout time.Duration `mapstructure:"readHeaderTimeout"` // seconds
	ShutdownTimeout  time.Duration `mapstructure:"shutdownTimeout"` // seconds
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"sslMode"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"` // minutes
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"` // minutes
	QueryTimeout    time.Duration `mapstructure:"queryTimeout"`    // seconds
	RetryAttempts   int           `mapstructure:"retryAttempts"`
	RetryDelay      time.Duration `mapstructure:"retryDelay"` // seconds
}

// LoggerConfig contains logger settings
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	TimeFormat string `mapstructure:"timeFormat"`
	CallerInfo bool   `mapstructure:"callerInfo"`
}

// TransactionConfig contains transaction processing settings
type TransactionConfig struct {
	ConcurrencyLevel         int   `mapstructure:"concurrencyLevel"`
	LockTimeoutMs            int64 `mapstructure:"lockTimeoutMs"`
	MaxRetries               int   `mapstructure:"maxRetries"`
	UserBalanceDecimalPlaces int   `mapstructure:"userBalanceDecimalPlaces"`
}
