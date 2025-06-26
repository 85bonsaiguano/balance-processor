package database

import (
	"fmt"

	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/config"
	"github.com/spf13/viper"
)

// LoadFromViper loads database configuration from viper
// This function should be called after Viper is initialized with configuration sources
func LoadFromViper(v *viper.Viper) (*Config, error) {
	// Start with environment-based defaults
	config := DefaultConfig()

	// Only use viper values as fallback if environment variables aren't set
	if config.Host == "" {
		config.Host = v.GetString("database.host")
	}
	if config.Port == 0 {
		config.Port = ParsePort(v.GetString("database.port"))
	}
	if config.Username == "" {
		config.Username = v.GetString("database.username")
	}
	if config.Password == "" {
		config.Password = v.GetString("database.password")
	}
	if config.Database == "" {
		config.Database = v.GetString("database.database")
	}
	
	// Non-sensitive settings can still be overridden by viper if present
	if v.IsSet("database.sslMode") {
		config.SSLMode = v.GetString("database.sslMode")
	}
	if v.IsSet("database.maxOpenConns") {
		config.MaxOpenConns = v.GetInt("database.maxOpenConns")
	}
	if v.IsSet("database.maxIdleConns") {
		config.MaxIdleConns = v.GetInt("database.maxIdleConns")
	}
	if v.IsSet("database.connMaxLifetime") {
		config.ConnMaxLifetime = v.GetDuration("database.connMaxLifetime")
	}
	if v.IsSet("database.queryTimeout") {
		config.QueryTimeout = v.GetDuration("database.queryTimeout")
	}
	if v.IsSet("database.retryAttempts") {
		config.RetryAttempts = v.GetInt("database.retryAttempts")
	}
	if v.IsSet("database.retryDelay") {
		config.RetryDelay = v.GetInt("database.retryDelay")
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// CreateConfigFromViperConfig adapts the global configuration to database configuration
func CreateConfigFromViperConfig(conf *config.Config) *Config {
	// Start with environment-based defaults
	dbConf := DefaultConfig()

	// Only use conf values if environment variables aren't set
	if dbConf.Host == "" {
		dbConf.Host = conf.Database.Host
	}
	if dbConf.Port == 0 {
		dbConf.Port = ParsePort(conf.Database.Port)
	}
	if dbConf.Username == "" {
		dbConf.Username = conf.Database.Username
	}
	if dbConf.Password == "" {
		dbConf.Password = conf.Database.Password
	}
	if dbConf.Database == "" {
		dbConf.Database = conf.Database.Database
	}

	// For non-sensitive values, we can override from the config
	if conf.Database.SSLMode != "" {
		dbConf.SSLMode = conf.Database.SSLMode
	}
	if conf.Database.MaxOpenConns > 0 {
		dbConf.MaxOpenConns = conf.Database.MaxOpenConns
	}
	if conf.Database.MaxIdleConns > 0 {
		dbConf.MaxIdleConns = conf.Database.MaxIdleConns
	}
	if conf.Database.ConnMaxLifetime > 0 {
		dbConf.ConnMaxLifetime = conf.Database.ConnMaxLifetime
	}
	if conf.Database.ConnMaxIdleTime > 0 {
		dbConf.ConnMaxIdleTime = conf.Database.ConnMaxIdleTime
	}
	if conf.Database.QueryTimeout > 0 {
		dbConf.QueryTimeout = conf.Database.QueryTimeout
	}
	if conf.Database.RetryAttempts >= 0 {
		dbConf.RetryAttempts = conf.Database.RetryAttempts
	}
	if conf.Database.RetryDelay > 0 {
		dbConf.RetryDelay = int(conf.Database.RetryDelay.Seconds())
	}
	if conf.Logger.Level != "" {
		dbConf.LogLevel = conf.Logger.Level
	}

	return dbConf
}

// ParsePort converts a port string to an int
func ParsePort(port string) int {
	var p int
	_, err := fmt.Sscanf(port, "%d", &p)
	if err != nil || p <= 0 || p > 65535 {
		return 0 // Return 0 to signal not set instead of defaulting
	}
	return p
}
