package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Environment constants
const (
	Development = "development"
	Production  = "production"
	Test        = "test"
)

// ConfigPaths defines the paths to look for config files
var ConfigPaths = []string{
	"./configs",
	"../configs",
	"../../configs",
}

// DotEnvPaths defines the paths to look for .env files
var DotEnvPaths = []string{
	".env",
	"./.env",
	"../.env",
	"../../.env",
	"./configs/.env",
	"../configs/.env",
	"../../configs/.env",
}

// LoadConfig loads configuration from file based on the environment
func LoadConfig() (*Config, error) {
	// Load environment variables from .env file first
	if err := loadDotEnvFile(); err != nil {
		// Don't return error, just log it or continue
		fmt.Println("Warning: Could not load .env file:", err)
	}

	// Get environment
	env := getEnvironment()

	v := viper.New()
	v.SetConfigName(env)
	v.SetConfigType("yaml")

	// Add config paths
	for _, path := range ConfigPaths {
		v.AddConfigPath(path)
	}

	// Set default values for non-critical settings
	setDefaults(v)

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Set environment variables to override config
	v.SetEnvPrefix("BP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Process environment variable overrides for sensitive values
	processEnvOverrides(v)

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Set the environment in the config
	config.Environment = env

	// Convert time.Duration fields from their raw values
	processDurations(&config)

	return &config, nil
}

// loadDotEnvFile attempts to load environment variables from .env files
func loadDotEnvFile() error {
	var lastError error
	
	for _, path := range DotEnvPaths {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err == nil {
				return nil // Successfully loaded .env file
			} else {
				lastError = err
			}
		}
	}
	
	// Return the last error encountered if no .env file was successfully loaded
	if lastError != nil {
		return fmt.Errorf("could not load any .env file: %w", lastError)
	}
	
	return fmt.Errorf("no .env file found in search paths")
}

// setDefaults sets default values for non-critical configuration
func setDefaults(v *viper.Viper) {
	// Non-critical server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)  // Default port but can be overridden
	v.SetDefault("server.readTimeout", 15)       // seconds
	v.SetDefault("server.writeTimeout", 15)      // seconds
	v.SetDefault("server.idleTimeout", 60)       // seconds
	v.SetDefault("server.readHeaderTimeout", 10) // seconds
	v.SetDefault("server.shutdownTimeout", 10)   // seconds

	// Database defaults for non-sensitive settings
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.sslMode", "disable")
	v.SetDefault("database.maxOpenConns", 100)   // Increased for better TPS
	v.SetDefault("database.maxIdleConns", 50)    // Increased for better TPS
	v.SetDefault("database.connMaxLifetime", 30) // minutes - Increased for stability
	v.SetDefault("database.connMaxIdleTime", 15) // minutes - Increased for better reuse
	v.SetDefault("database.queryTimeout", 5)     // seconds - Decreased for faster responses
	v.SetDefault("database.retryAttempts", 3)    // Optimized for performance
	v.SetDefault("database.retryDelay", 1)       // seconds - Decreased for faster recovery

	// Logger defaults
	v.SetDefault("logger.level", "info")        // Changed to info for better performance
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.output", "stdout")
	v.SetDefault("logger.callerInfo", true)

	// Transaction defaults - Optimized for high TPS
	v.SetDefault("transaction.concurrencyLevel", 200)  // Increased for better parallelism
	v.SetDefault("transaction.lockTimeoutMs", 5000)    // Optimized lock timeout
	v.SetDefault("transaction.maxRetries", 3)
	v.SetDefault("transaction.userBalanceDecimalPlaces", 2)
}

// getEnvironment determines the environment to use based on BP_ENV environment variable
func getEnvironment() string {
	env := os.Getenv("BP_ENV")
	if env == "" {
		// Default to development if not specified
		env = Development
	}
	return strings.ToLower(env)
}

// processEnvOverrides ensures environment variables override config values
// This function prioritizes environment variables over configuration file values
func processEnvOverrides(v *viper.Viper) {
	// Database sensitive information
	if dbHost := os.Getenv("BP_DB_HOST"); dbHost != "" {
		v.Set("database.host", dbHost)
	}
	if dbPort := os.Getenv("BP_DB_PORT"); dbPort != "" {
		v.Set("database.port", dbPort)
	}
	if dbUser := os.Getenv("BP_DB_USERNAME"); dbUser != "" {
		v.Set("database.username", dbUser)
	}
	if dbPass := os.Getenv("BP_DB_PASSWORD"); dbPass != "" {
		v.Set("database.password", dbPass)
	}
	if dbName := os.Getenv("BP_DB_NAME"); dbName != "" {
		v.Set("database.database", dbName)
	}
	if sslMode := os.Getenv("BP_DB_SSL_MODE"); sslMode != "" {
		v.Set("database.sslMode", sslMode)
	}
	
	// Database performance settings
	if maxOpenConns := getEnvInt("BP_DB_MAX_OPEN_CONNS", 0); maxOpenConns > 0 {
		v.Set("database.maxOpenConns", maxOpenConns)
	}
	if maxIdleConns := getEnvInt("BP_DB_MAX_IDLE_CONNS", 0); maxIdleConns > 0 {
		v.Set("database.maxIdleConns", maxIdleConns)
	}
	if connMaxLifetime := getEnvInt("BP_DB_CONN_MAX_LIFETIME_MINUTES", 0); connMaxLifetime > 0 {
		v.Set("database.connMaxLifetime", connMaxLifetime)
	}
	if connMaxIdleTime := getEnvInt("BP_DB_CONN_MAX_IDLE_TIME_MINUTES", 0); connMaxIdleTime > 0 {
		v.Set("database.connMaxIdleTime", connMaxIdleTime)
	}
	if queryTimeout := getEnvInt("BP_DB_QUERY_TIMEOUT_SECONDS", 0); queryTimeout > 0 {
		v.Set("database.queryTimeout", queryTimeout)
	}
	if retryAttempts := getEnvInt("BP_DB_RETRY_ATTEMPTS", 0); retryAttempts >= 0 {
		v.Set("database.retryAttempts", retryAttempts)
	}
	if retryDelay := getEnvInt("BP_DB_RETRY_DELAY_SECONDS", 0); retryDelay >= 0 {
		v.Set("database.retryDelay", retryDelay)
	}

	// Server settings
	if serverHost := os.Getenv("BP_SERVER_HOST"); serverHost != "" {
		v.Set("server.host", serverHost)
	}
	if serverPort := os.Getenv("BP_SERVER_PORT"); serverPort != "" {
		v.Set("server.port", serverPort)
	}
	
	// Logger settings
	if logLevel := os.Getenv("BP_LOGGER_LEVEL"); logLevel != "" {
		v.Set("logger.level", logLevel)
	}

	// Transaction settings
	if concurrencyLevel := getEnvInt("BP_TRANSACTION_CONCURRENCY_LEVEL", 0); concurrencyLevel > 0 {
		v.Set("transaction.concurrencyLevel", concurrencyLevel)
	}
	if lockTimeout := getEnvInt("BP_TRANSACTION_LOCK_TIMEOUT_MS", 0); lockTimeout > 0 {
		v.Set("transaction.lockTimeoutMs", lockTimeout)
	}
	if maxRetries := getEnvInt("BP_TRANSACTION_MAX_RETRIES", 0); maxRetries >= 0 {
		v.Set("transaction.maxRetries", maxRetries) 
	}
}

// Helper function to get environment variable as int
func getEnvInt(name string, defaultVal int) int {
	valStr := os.Getenv(name)
	if valStr == "" {
		return defaultVal
	}
	
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// processDurations converts time.Duration fields from their raw values to actual durations
func processDurations(config *Config) {
	// Convert seconds to time.Duration
	config.Server.ReadTimeout = time.Duration(config.Server.ReadTimeout) * time.Second
	config.Server.WriteTimeout = time.Duration(config.Server.WriteTimeout) * time.Second
	config.Server.IdleTimeout = time.Duration(config.Server.IdleTimeout) * time.Second
	config.Server.ReadHeaderTimeout = time.Duration(config.Server.ReadHeaderTimeout) * time.Second
	config.Server.ShutdownTimeout = time.Duration(config.Server.ShutdownTimeout) * time.Second

	// Convert minutes to time.Duration
	config.Database.ConnMaxLifetime = time.Duration(config.Database.ConnMaxLifetime) * time.Minute
	config.Database.ConnMaxIdleTime = time.Duration(config.Database.ConnMaxIdleTime) * time.Minute

	// Convert seconds to time.Duration
	config.Database.QueryTimeout = time.Duration(config.Database.QueryTimeout) * time.Second
	config.Database.RetryDelay = time.Duration(config.Database.RetryDelay) * time.Second
}
