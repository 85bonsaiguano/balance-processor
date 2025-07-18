# Usecase Layer Design

This document explains the design decisions and architecture of the usecase layer in the Balance Processor service.

## Key Design Decisions

### No Interfaces for Usecases

The usecase layer is implemented without interfaces for the usecase components themselves. This is by design, as usecases represent the highest level of business logic in the application and are not meant to be abstracted away or replaced with alternative implementations.

Benefits of this approach:
1. **Simplified Architecture**: Removes an unnecessary layer of indirection
2. **Clear Dependencies**: Makes the flow of dependencies more explicit
3. **Better Testing**: Components can be tested directly without mocking usecases

### Interface for Infrastructure Components

While usecases themselves don't have interfaces, they depend on interfaces for:
- Repositories
- Logging
- Time providers
- Database transactions

This allows for easy mocking and testing of these components while keeping the usecase implementation concrete.

## Multi-Instance Transaction Processing

The transaction processing system is designed to work across multiple instances without shared memory or message queues:

### How It Works

1. **Database-level Concurrency Control**:
   - Row-level locking on user records
   - SERIALIZABLE isolation level for transactions
   - Optimistic locking where appropriate

2. **Idempotency**:
   - Transaction IDs ensure each transaction is processed only once
   - Multiple layers of idempotency checking (before and during DB transaction)

3. **Stateless Design**:
   - No reliance on local memory
   - No need for sticky sessions
   - No cross-instance coordination required

### Components

1. **TransactionProcessor**: Main entry point that orchestrates transaction processing
2. **TransactionManager**: Handles core business logic, DB transactions, and user locking
3. **IdempotencyHandler**: Ensures transactions are processed exactly once
4. **TransactionValidator**: Validates transaction input data

## Deployment Considerations

The stateless design allows for horizontal scaling across multiple containers or pods:

1. **Load Balancing**: Any request can go to any instance
2. **No Shared State**: No need for shared caches or distributed locks
3. **Database Scaling**: The main scaling consideration is database capacity and connection pooling

## Testing Approach

1. **Unit Tests**: Test individual components with mocked dependencies
2. **Integration Tests**: Test with a real database to verify concurrency behavior
3. **Load Tests**: Verify behavior under high concurrency with multiple instances 