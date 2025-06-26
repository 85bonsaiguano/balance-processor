# Balance Processor

A high-performance application for processing financial transactions for third-party providers.

## Overview

This service processes incoming transaction requests from game, server, and payment providers, updates user balances, and provides balance information. It is built with Go, PostgreSQL, and follows clean architecture principles for maximum maintainability, testability, and performance.

The application handles financial transactions with strict guarantees for:
- Idempotency (no duplicate processing)
- Atomicity (transactions either fully succeed or fully fail)
- Data consistency (no race conditions)
- Proper ordering of transactions per user

## Key Features

- Process win/lose transactions with idempotency checks
- Sequential per-user transaction processing with queue-based design
- Prevent negative balances with proper validation
- Thread-safe concurrent request handling
- High throughput (30+ transactions per second)
- RESTful API with comprehensive error handling
- PostgreSQL optimizations for performance
- Default users created automatically on startup
- Graceful shutdown with proper resource cleanup
- Comprehensive load testing capabilities

## Architecture

The application follows clean/hexagonal architecture with clear separation of concerns:

### Domain Layer
- Business entities and core business rules
- Entity objects with validation logic
- Domain-specific errors

### Use Case Layer
- Application-specific business logic
- Transaction processing workflow
- User management and balance operations
- Independent of external frameworks

### Infrastructure Layer
- Database access and optimization
- HTTP API implementation
- Configuration management
- Logging and monitoring

This separation ensures the business logic remains independent of external frameworks and facilitates testing.

## API Documentation

### Get User Balance

```
GET /user/{userId}/balance
```

**Response**:
```json
{
  "userId": 1,
  "balance": "100.25"
}
```

### Process Transaction

```
POST /user/{userId}/transaction
```

**Headers**:
```
Source-Type: game|server|payment
Content-Type: application/json
```

**Request Body**:
```json
{
  "state": "win|lose",
  "amount": "10.15",
  "transactionId": "unique-transaction-id"
}
```

**Response (success)**:
```json
{
  "transactionId": "unique-transaction-id",
  "userId": 1,
  "success": true,
  "resultBalance": "110.40"
}
```

## Running the Application

### Prerequisites

- Docker
- Docker Compose

### Using Docker Compose

1. Clone the repository:
```bash
git clone https://github.com/amirhossein-jamali/balance-processor.git
cd balance-processor
```

2. Start the application with Docker Compose:
```bash
docker-compose up -d
```

The application will be accessible at `http://localhost:8080`.

### Configuration

The application supports environment-specific configuration through YAML files and environment variables:

- `development.yaml`: For local development
- `production.yaml`: For production deployment
- `test.yaml`: For testing environments

Environment variables take precedence over configuration files for sensitive information. Create a `.env` file at the root of the project to set sensitive values.

Key configuration sections include:
- Server settings (port, timeouts)
- Database connection parameters
- Logger configuration
- Transaction processing settings (concurrency, timeouts)

For complete details, see the [configuration documentation](configs/README.md).

## Testing

### Manual Testing

Use curl or any API tool to test the endpoints:

1. Check balance for default user:
```bash
curl -X GET http://localhost:8080/user/1/balance
```

2. Process a win transaction:
```bash
curl -X POST http://localhost:8080/user/1/transaction \
  -H "Source-Type: game" \
  -H "Content-Type: application/json" \
  -d '{"state": "win", "amount": "10.15", "transactionId": "tx-12345"}'
```

3. Process a lose transaction:
```bash
curl -X POST http://localhost:8080/user/1/transaction \
  -H "Source-Type: game" \
  -H "Content-Type: application/json" \
  -d '{"state": "lose", "amount": "5.25", "transactionId": "tx-67890"}'
```

### Automated Testing

The project includes unit and integration tests. To run them:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

## Performance Testing

The service includes comprehensive load testing scripts for performance validation:

```bash
# Basic load test with default settings
go run script/load-test-with-delay.go

# High concurrency test (20 concurrent users, 5000 requests)
go run script/load-test-with-delay.go -c 20 -n 5000

# Test with specific users and minimal delay
go run script/load-test-with-delay.go -u 1,2,3 -delay 10
```

For more details on load testing options, see the [script documentation](script/README.md).

## Project Structure

- `cmd/api`: Application entry point and main initialization
- `configs`: Environment-specific configuration files
- `internal/domain`: Business entities and core business logic
  - `entity`: Domain models and validation
  - `error`: Domain-specific error definitions
  - `port`: Interface definitions for dependency inversion
  - `usecase`: Business logic implementation
- `internal/infrastructure`: External concerns implementation
  - `adapter`: Implementation of interfaces defined in domain
  - `config`: Configuration loading and validation
- `mocks`: Mock implementations for testing
- `script`: Load testing and utility scripts

## Transaction Processing Workflow

1. Request validation (format, required fields, data types)
2. Queuing transactions per user for sequential processing
3. Processing with user account locking and database transactions
4. Result handling with detailed error information

The service implements thread safety through:
- User-specific transaction queues for strict ordering
- Database-level locks on user records
- Atomic operations within database transactions
- SERIALIZABLE isolation level for transaction consistency

For more details on the transaction processing workflow, see the [transaction documentation](internal/domain/usecase/transaction/README.md). 