# Balance Processor Main Application

## Overview

The `main.go` file serves as the entry point for the Balance Processor application. It handles the initialization and orchestration of all application components, following clean architecture principles to ensure proper separation of concerns.

## Architecture Principles

### Clean Architecture Implementation

The application follows the principles of Clean Architecture (also known as Hexagonal Architecture), which separates the codebase into distinct layers:

- **Domain Layer**: Contains business logic and entities
- **Use Case Layer**: Implements application-specific business rules
- **Infrastructure Layer**: Handles external concerns like databases, HTTP servers, etc.

This separation makes the codebase more maintainable, testable, and allows business logic to remain independent of external frameworks.

### Dependency Injection

- Manual dependency injection is used throughout the codebase
- Initialization code remains explicit and easy to understand
- Avoids hidden magic and unnecessary complexity
- Makes testing easier by allowing dependencies to be mocked

### Configuration Management

1. **Validation**
   - All required settings are properly configured before startup
   - Environment-specific validation rules are applied
   - Configuration errors are reported clearly with detailed error messages
   - No hardcoded values are used within the application code

2. **Environment-Specific Configuration**
   - **Development**: Uses reasonable defaults with ability to override
   - **Production**: Requires explicit configuration of critical settings
   - **Test**: Uses test-specific settings optimized for automated testing

3. **Security-First Approach**
   - Database connections use proper SSL settings in production
   - Timeout values are validated to prevent DoS vulnerabilities
   - Security warnings for potentially insecure configuration choices

### Application Lifecycle

1. **Startup**
   - Configuration loading and validation
   - Component initialization in proper order
   - Database migration execution
   - Default data creation
   - HTTP server initialization

2. **Graceful Shutdown**
   - In-flight requests complete processing
   - Transaction Manager is properly shut down
   - Resources are properly released
   - No data loss occurs during shutdown

## Key Components

### Configuration Loading and Validation

```go
// Load configuration
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Validate essential configuration
if err := validateConfig(cfg); err != nil {
    log.Fatalf("Configuration validation failed: %v", err)
}
```

Configuration is loaded from external sources and validated before proceeding to ensure:
- All required fields are present
- Environment variables are properly set in production
- Security settings are appropriate for the environment
- No critical settings are left at default values

### Database Connection

```go
// Setup database configuration
dbConfig := &database.Config{
    Driver:          "postgres",
    Host:            cfg.Database.Host,
    Port:            database.ParsePort(cfg.Database.Port),
    Username:        cfg.Database.Username,
    Password:        cfg.Database.Password,
    Database:        cfg.Database.Database,
    SSLMode:         cfg.Database.SSLMode,
    MaxOpenConns:    cfg.Database.MaxOpenConns,
    MaxIdleConns:    cfg.Database.MaxIdleConns,
    ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
    ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
    QueryTimeout:    cfg.Database.QueryTimeout,
    LogLevel:        cfg.Logger.Level,
    RetryAttempts:   3,
    RetryDelay:      5,
}
```

Features:
- No hardcoded connection values
- Connection pooling for performance optimization
- Configurable timeouts and retry strategies
- Proper resource cleanup with defer
- Clear error handling

### Infrastructure Services

1. **Time Provider**
   ```go
   // Initialize time provider
   tp := timeProvider.NewRealTimeProvider()
   ```
   - Enables deterministic testing by allowing time to be mocked
   - Ensures consistent time handling throughout the application
   - Supports scenarios like time zone handling if needed in the future

2. **Logger**
   ```go
   // Create logger
   appLogger := logger.NewZapLogger(cfg.Environment == "production")
   ```
   - Uses structured logging format
   - Configurable based on environment
   - Consistent log format across the application

### Data Access Layer

1. **Repositories**
   ```go
   // Initialize repositories
   userRepo := repository.NewUserRepository(dbManager.DB(), tp, appLogger)
   userLockRepo := repository.NewUserLockRepository(dbManager.DB(), tp, appLogger)
   // transactionRepo is used inside the UnitOfWork
   _ = repository.NewTransactionRepository(dbManager.DB(), appLogger)
   ```
   - Encapsulate data access patterns
   - Handle database mapping logic
   - Provide error handling specific to data persistence

2. **Unit of Work**
   ```go
   // Unit of work (transaction manager)
   uow := database.NewUnitOfWork(dbManager.DB(), appLogger, tp)
   ```
   - Ensures atomicity of operations
   - Provides transaction management
   - Maintains data consistency across related operations

3. **Database Migrations**
   ```go
   // Run migrations
   migrationMgr := migration.NewMigrationManagerWithTimeProvider(dbManager.DB(), appLogger, tp)
   err = migrationMgr.MigrateAll()
   ```
   - Ensures schema is always up to date
   - Tracks schema version changes
   - Allows for smooth upgrades and rollbacks

### Business Logic

1. **Use Cases**
   ```go
   userUseCaseImpl := userUseCase.NewUserUseCase(userRepo, tp, appLogger)

   lockTimeout := time.Duration(cfg.Transaction.LockTimeoutMs) * time.Millisecond

   transactionUseCaseImpl := transactionUseCase.NewTransactionService(
       uow,
       userUseCaseImpl,
       userLockRepo,
       tp,
       appLogger,
       lockTimeout,
   )
   ```
   - Independent of delivery mechanisms (HTTP, gRPC, etc.)
   - Implement core application functionality
   - Configure operational parameters from configuration

2. **Default Data**
   ```go
   // Create default users
   err = migration.CreateDefaultUsers(context.Background(), userUseCaseImpl)
   ```
   - Ensures necessary initial data is available
   - Supports first-time use and testing

### HTTP Server

1. **API Handlers**
   ```go
   // Initialize API handlers
   userHandler := handler.NewUserHandler(userUseCaseImpl, appLogger)
   transactionHandler := handler.NewTransactionHandler(transactionUseCaseImpl, userUseCaseImpl, appLogger)
   ```
   - Transform HTTP requests into domain calls
   - Handle HTTP-specific concerns (status codes, headers)
   - Delegate business logic to use cases

2. **Routing**
   ```go
   // Initialize Gin router
   router := gin.New()

   // Setup middlewares
   routes.SetupMiddlewares(router, appLogger)

   // Setup routes
   routes.SetupRoutes(router, transactionHandler, userHandler)
   ```
   - Centralized routing logic
   - Consistent middleware application
   - Separation of route definition from handler implementation

3. **Server Configuration**
   ```go
   // Create HTTP server with configurable timeout values
   server := &http.Server{
       Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
       Handler:           router,
       ReadTimeout:       cfg.Server.ReadTimeout,
       WriteTimeout:      cfg.Server.WriteTimeout,
       ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
       IdleTimeout:       cfg.Server.IdleTimeout,
   }
   ```
   - Configurable timeouts from configuration
   - No hardcoded port values
   - Integration with the router
   - Support for proper server lifecycle management

### Graceful Shutdown

```go
// Wait for interrupt signal to gracefully shut down the server
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

appLogger.Info("Shutting down server...", nil)

// Create a deadline to wait for
ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
defer cancel()

// Shutdown TransactionManager cleanly
if txService, ok := transactionUseCaseImpl.(*transactionUseCase.Service); ok {
    if txManager := txService.GetManager(); txManager != nil {
        appLogger.Info("Shutting down transaction manager...", nil)
        txManager.Shutdown()
    }
}

// Shutdown the server
if err := server.Shutdown(ctx); err != nil {
    appLogger.Error("Server forced to shutdown", map[string]any{
        "error": err.Error(),
    })
}
```

Features:
- Catches termination signals (SIGINT, SIGTERM)
- Allows in-flight requests to complete
- Properly shuts down the Transaction Manager
- Ensures all resources are properly released
- Logs the shutdown progress

## Usage

The main application is designed to be run as a standalone service:

```bash
# Running in development mode
go run cmd/api/main.go

# Running with a specific config file
go run cmd/api/main.go -config=configs/development.yaml

# Building and running in production
go build -o balance-processor cmd/api/main.go
./balance-processor -config=configs/production.yaml
```

## Dependencies

- **Gin**: HTTP web framework
- **Zap**: Structured logging
- **PostgreSQL**: Database for persistence
- **Viper**: Configuration management
- **Clean Architecture**: Architectural pattern

## Conclusion

The `main.go` file orchestrates the bootstrapping of the entire Balance Processor application, following best practices for Go applications with a focus on clean architecture, proper resource management, robust configuration validation, and security-first design. The graceful shutdown mechanism ensures that all components, including the Transaction Manager, are properly closed to maintain data integrity.