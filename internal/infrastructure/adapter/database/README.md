# Database Package

This package provides optimized PostgreSQL database infrastructure for the Balance Processor service.

## Components

### Core Components

- **Config** - Database configuration with environment variable support
- **Manager** - Main entry point for database operations
- **ErrorMapper** - Maps database errors to domain errors
- **Logger** - Custom GORM logger that uses the application logger

### Transaction Management

- **UnitOfWork** - Implements the unit of work pattern for coordinated transaction operations
- **RetryOnTransientError** - Utility for retrying operations on transient errors with exponential backoff

### Migration Components

- **MigrationManager** - Manages database schema migrations
- **AdvancedIndexManager** - Creates and optimizes PostgreSQL-specific indexes

### Monitoring and Performance

- **ConnectionPoolMonitor** - Monitors the database connection pool
- **MetricsCollector** - Collects metrics about database operations

### Testing Utilities

- **TestDBManager** - Utilities for testing with a real PostgreSQL database

## PostgreSQL Optimizations

The database layer includes several PostgreSQL-specific optimizations:

1. **Advanced Indexes**:
   - Composite indexes for efficient filtering
   - Partial indexes for common query patterns
   - BRIN indexes for time-series data

2. **Connection Pool Management**:
   - Optimized connection pool settings
   - Connection pool monitoring
   - Automatic cleanup of idle connections

3. **Performance Tweaks**:
   - Table fillfactor settings to reduce page splits
   - Statistics targets for better query planning
   - Query timeout settings to prevent long-running queries

4. **Error Handling**:
   - Intelligent error classification
   - Automatic retry for transient errors
   - Detailed error logging

## Usage Example

```go
// Create database manager
config := &database.Config{
    Host:     "localhost",
    Port:     5432,
    Username: "postgres",
    Password: "postgres",
    Database: "balance_processor",
}
dbManager := database.NewManager(config, logger)

// Connect to database
if err := dbManager.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect to database: %v", err)
}
defer dbManager.Close()

// Run migrations
if err := dbManager.MigrationManager().MigrateAll(); err != nil {
    log.Fatalf("Failed to run migrations: %v", err)
}

// Create unit of work for transactional operations
unitOfWork := dbManager.CreateUnitOfWork(timeProvider)

// Use repositories in a transaction
txCtx, err := unitOfWork.Begin(ctx)
if err != nil {
    log.Fatalf("Failed to begin transaction: %v", err)
}

userRepo := unitOfWork.GetUserRepository(txCtx)
transactionRepo := unitOfWork.GetTransactionRepository(txCtx)

// Perform operations...

// Commit or rollback
if err := unitOfWork.Commit(txCtx); err != nil {
    unitOfWork.Rollback(txCtx)
    log.Fatalf("Failed to commit transaction: %v", err)
}
``` 