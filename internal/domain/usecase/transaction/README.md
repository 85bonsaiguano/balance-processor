# Transaction Processing System for Multi-Instance Deployment

## Architecture Overview

This transaction processing system is designed to handle financial transactions in a distributed environment with multiple instances running concurrently. The architecture ensures atomicity, consistency, isolation, and durability (ACID properties) by leveraging database capabilities rather than in-memory state or queues.

## Key Components

1. **TransactionProcessor**: The main entry point that orchestrates the entire process:
   - Validates transaction requests
   - Checks for idempotency
   - Processes transactions via the TransactionManager

2. **TransactionManager**: Handles the core transaction processing logic:
   - Acquires user locks via database
   - Manages database transactions
   - Processes balance changes
   - Ensures atomicity and consistency

3. **IdempotencyHandler**: Prevents duplicate transaction processing:
   - Checks if a transaction ID already exists
   - Returns existing transactions for duplicate requests

4. **TransactionValidator**: Validates transaction input parameters:
   - Ensures all required fields are present
   - Validates data formats and ranges

## Concurrency and Scalability Approach

### Database-Level Guarantees

The system relies on database mechanisms to ensure consistent transaction processing across multiple instances:

1. **Row-Level Locks**: The `UserLockRepository` acquires exclusive locks on user records to prevent concurrent transactions for the same user.

2. **Database Transactions**: Every operation uses database transactions with SERIALIZABLE isolation level to ensure atomicity and prevent race conditions.

3. **Idempotency Checks**: Multiple layers of idempotency checking prevent duplicate transaction processing.

### Stateless Implementation

The implementation is completely stateless, with no dependency on:
- In-memory state
- Local caches
- Message queues
- Instance-specific storage

This allows horizontal scaling by simply adding more instances without configuration changes.

## Deployment in Multi-Instance Environment

### Docker Deployment

For Docker Compose deployment:

```yaml
version: '3'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: password
      POSTGRES_DB: balance_processor
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  balance-processor-1:
    build: .
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://app:password@postgres:5432/balance_processor
      LOG_LEVEL: info
    ports:
      - "8081:8080"

  balance-processor-2:
    build: .
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://app:password@postgres:5432/balance_processor
      LOG_LEVEL: info
    ports:
      - "8082:8080"

  balance-processor-3:
    build: .
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://app:password@postgres:5432/balance_processor
      LOG_LEVEL: info
    ports:
      - "8083:8080"

  load-balancer:
    image: nginx:latest
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - balance-processor-1
      - balance-processor-2
      - balance-processor-3

volumes:
  postgres_data:
```

### Kubernetes Deployment

For Kubernetes deployment, use a StatefulSet for the database and a Deployment for the balance processor:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: balance-processor
spec:
  replicas: 3  # Can be scaled as needed
  selector:
    matchLabels:
      app: balance-processor
  template:
    metadata:
      labels:
        app: balance-processor
    spec:
      containers:
      - name: balance-processor
        image: balance-processor:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
        - name: LOG_LEVEL
          value: "info"
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
          requests:
            cpu: "0.5"
            memory: "256Mi"
```

### Database Considerations

For production deployment, ensure that:

1. The PostgreSQL database is configured with appropriate settings:
   - Connection pool size that can handle peak load
   - SERIALIZABLE transaction isolation level support
   - Properly sized hardware for high transaction throughput

2. Consider using a managed database service with automatic scaling and high availability.

3. Enable database monitoring to track lock contention and transaction performance.

## Sequence Diagram

```
┌─────────┐           ┌──────────────────┐        ┌────────────────┐        ┌───────┐
│  Client │           │ Transaction API  │        │ Database       │        │ User  │
└────┬────┘           └────────┬─────────┘        └───────┬────────┘        └───┬───┘
     │                         │                          │                     │
     │ POST /transaction       │                          │                     │
     │────────────────────────>│                          │                     │
     │                         │                          │                     │
     │                         │ Check idempotency        │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │                     │
     │                         │ Acquire lock on user     │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │ Lock acquired       │
     │                         │                          │ ───────────────────>│
     │                         │                          │                     │
     │                         │ Begin transaction        │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │                     │
     │                         │ Update user balance      │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │                     │
     │                         │ Store transaction        │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │                     │
     │                         │ Commit transaction       │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │                     │
     │                         │ Release lock             │                     │
     │                         │─────────────────────────>│                     │
     │                         │                          │ Lock released       │
     │                         │                          │ ───────────────────>│
     │                         │                          │                     │
     │ 200 OK (Transaction)    │                          │                     │
     │<────────────────────────│                          │                     │
     │                         │                          │                     │
```

## Performance and Optimizations

1. **Early Idempotency Check**: Checks for duplicate transactions before acquiring locks to reduce database contention.

2. **Short-Lived Locks**: User locks are held only for the duration of the transaction processing.

3. **Optimistic Locking**: The system could be extended with optimistic locking for even better concurrency in high-volume scenarios.

4. **Connection Pooling**: Use database connection pools to minimize the overhead of creating new connections.

5. **Database Indexes**: Ensure proper indexes on transaction ID, user ID, and other frequently queried fields.

## Conclusion

This architecture enables a truly stateless, horizontally scalable transaction processing system that can run across multiple instances while maintaining data consistency and integrity. By leveraging database mechanisms for concurrency control instead of application-level coordination, the system avoids the complexity and potential bottlenecks of distributed synchronization mechanisms. 