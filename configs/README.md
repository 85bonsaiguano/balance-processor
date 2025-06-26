# Configuration

This directory contains configuration files for different environments:

- `development.yaml`: Configuration for development environment
- `production.yaml`: Configuration for production environment
- `test.yaml`: Configuration for test environment

## Environment Variables

The application now prioritizes environment variables over configuration files for sensitive information. Create a `.env` file at the root of the project or in this directory to set sensitive values.

### Sample .env file

```
# Environment Configuration
BP_ENV=development  # Options: development, production, test

# Server Configuration
BP_SERVER_HOST=0.0.0.0
BP_SERVER_PORT=8080

# Database Configuration - SENSITIVE VALUES
BP_DB_HOST=localhost
BP_DB_PORT=5432
BP_DB_USERNAME=postgres
BP_DB_PASSWORD=your_secure_password_here
BP_DB_NAME=balance_processor_dev
BP_DB_SSL_MODE=disable  # Options: disable, require, verify-ca, verify-full

# Logger Configuration
BP_LOGGER_LEVEL=debug  # Options: debug, info, warn, error

# Transaction Configuration
BP_TRANSACTION_CONCURRENCY_LEVEL=50
BP_TRANSACTION_LOCK_TIMEOUT_MS=10000
```

## Configuration Loading Priority

1. Environment variables (highest priority)
2. `.env` file values
3. Configuration YAML file values
4. Default values (lowest priority)

## Configuration Structure

### Server Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  readTimeout: 10    # seconds
  writeTimeout: 15   # seconds
  idleTimeout: 120   # seconds
  readHeaderTimeout: 5  # seconds
  shutdownTimeout: 10  # seconds
```

### Database Configuration

```yaml
database:
  driver: "postgres"
  host: ""         # Set via BP_DB_HOST
  port: ""         # Set via BP_DB_PORT
  username: ""     # Set via BP_DB_USERNAME
  password: ""     # Set via BP_DB_PASSWORD
  database: ""     # Set via BP_DB_NAME
  sslMode: "disable"
  maxOpenConns: 50
  maxIdleConns: 10
  connMaxLifetime: 30   # minutes
  connMaxIdleTime: 5    # minutes
  queryTimeout: 120     # seconds
  retryAttempts: 5
  retryDelay: 5         # seconds
```

### Logger Configuration

```yaml
logger:
  level: "debug"
  format: "json"
  output: "stdout"
  timeFormat: "2006-01-02T15:04:05.000Z07:00"
  callerInfo: true
```

### Transaction Configuration

```yaml
transaction:
  concurrencyLevel: 50
  lockTimeoutMs: 10000
  maxRetries: 3
  userBalanceDecimalPlaces: 2
```

## Configuration Structure

The configuration files are structured with the following main sections:

### Server Configuration
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  readTimeout: 5    # seconds
  writeTimeout: 10  # seconds
  idleTimeout: 120  # seconds
```

### Database Configuration
```yaml
database:
  driver: "postgres"
  host: "postgres"  # Use container name for Docker
  port: "5432"
  username: ""      # Set via BP_DB_USERNAME
  password: ""      # Set via BP_DB_PASSWORD
  database: ""      # Set via BP_DB_NAME
  sslMode: "disable"
  maxOpenConns: 25
  maxIdleConns: 5
  connMaxLifetime: 30  # minutes
```

### Logger Configuration
```yaml
logger:
  level: "debug"  # debug, info, warn, error
  format: "json"  # json, console
  output: "stdout"
  timeFormat: "2006-01-02T15:04:05.000Z07:00"
  callerInfo: true
```

### Transaction Configuration
```yaml
transaction:
  concurrencyLevel: 50     # Number of concurrent transaction processors
  lockTimeoutMs: 5000      # Lock timeout in milliseconds
  maxRetries: 3            # Maximum number of retries for failed transactions
  userBalanceDecimalPlaces: 2  # Decimal places for user balance
```

## Environment Variables

The configuration values can be overridden by environment variables. The environment variables are prefixed with `BP_` and follow the structure of the configuration file. For example:

- `BP_SERVER_PORT` overrides `server.port`
- `BP_DATABASE_HOST` overrides `database.host`

Sensitive values like database credentials should be set via environment variables:

- `BP_DB_HOST` - Database host
- `BP_DB_PORT` - Database port
- `BP_DB_USERNAME` - Database username
- `BP_DB_PASSWORD` - Database password
- `BP_DB_NAME` - Database name

## Selecting Environment

The environment is selected by the `BP_ENV` environment variable. The possible values are:

- `development` (default)
- `production`
- `test`

Example:
```
BP_ENV=production go run main.go
``` 