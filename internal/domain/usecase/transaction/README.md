# Transaction Package

## Overview

The Transaction package is a critical component of the Balance Processor system, providing robust functionality for handling financial transactions. It implements the domain logic for processing transactions that affect user balances, ensuring data consistency, idempotency, and sequential processing.

## Components

### Core Components

1. **Service (`process_transaction.go`)**
   - Implements the `TransactionUseCase` interface
   - Manages the lifecycle of transactions from validation to completion
   - Integrates with user services, locking mechanisms, and database transactions
   - Ensures atomicity of operations

2. **TransactionManager (`transaction_manager.go`)**
   - Provides sequential processing of transactions per user
   - Implements a queue-based system to guarantee transaction ordering
   - Prevents race conditions in balance modifications
   - Supports graceful shutdown

3. **Validation (`validation.go`)**
   - Validates transaction requests before processing
   - Ensures required fields are present and correctly formatted
   - Checks for valid transaction states and amount formats

4. **Idempotency (`idempotency.go`)**
   - Prevents duplicate transaction processing
   - Checks if a transaction has already been processed

## Transaction Processing Workflow

1. **Request Validation**
   - Validates transaction format, required fields, and data types
   - Checks if the user exists before enqueuing

2. **Queuing**
   - Transactions are queued per user to ensure sequential processing
   - Each user has a dedicated worker goroutine to process their transactions

3. **Processing**
   - Locks the user account to prevent concurrent modifications
   - Checks for duplicate transactions (idempotency)
   - Begins a database transaction with SERIALIZABLE isolation level
   - Creates a transaction record with initial pending status
   - Updates user balance based on transaction type (win/lose)
   - Finalizes the transaction status
   - Commits or rolls back the database transaction

4. **Result Handling**
   - Returns transaction result to the caller
   - Provides detailed error information if processing fails

## Error Handling

The package implements comprehensive error handling for various scenarios:
- Validation errors (invalid transaction ID, state, or amount)
- Duplicate transactions
- User not found
- Locking failures
- Database transaction failures
- Concurrent modification conflicts

## Thread Safety

The transaction system is designed to be thread-safe through several mechanisms:
- User-specific transaction queues for strict ordering
- Database-level locks on user records
- Atomic operations within database transactions
- SERIALIZABLE isolation level for transaction consistency

## Dependencies

The package relies on the following interfaces:
- `UnitOfWork`: For transaction management
- `UserUseCase`: For user operations
- `UserLockRepository`: For account locking
- `TimeProvider`: For consistent timestamps
- `Logger`: For logging and monitoring

## Usage

The transaction service is initialized with all required dependencies and can be used to process financial transactions in a consistent and reliable manner:

```go
// Create transaction service
txnService := transaction.NewTransactionService(
    unitOfWork,
    userUseCase,
    userLockRepository,
    timeProvider,
    logger,
    lockTimeout,
)

// Process a transaction
result, err := txnService.ProcessTransaction(
    ctx,
    userId,
    usecase.TransactionRequest{
        TransactionID: "tx-123",
        SourceType:    "game",
        State:         "win",
        Amount:        "100.00",
    },
)
```

## Concurrency Model

The package implements a queue-based concurrency model that:
1. Ensures transactions for each user are processed sequentially
2. Allows transactions for different users to be processed concurrently
3. Provides backpressure handling with buffered channels
4. Supports graceful shutdown of all processing goroutines 