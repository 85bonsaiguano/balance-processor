# User Module

The user module handles all user-related business logic in the balance-processor application. It implements the core functionality for user management and balance operations.

## Structure Overview

This package implements the `UserUseCase` interface defined in the domain ports layer, providing the following user management capabilities:

- User creation
- User balance retrieval
- Balance modification (for win/lose transactions)

## Files and Responsibilities

### 1. `get_balance.go`

Contains the core `UserUseCase` structure definition and implements:

- `NewUserUseCase`: Constructor for the use case
- `GetFormattedUserBalance`: Retrieves a user's balance in standardized format
- `UserExists`: Utility method to check if a user exists by ID

### 2. `create_user.go`

Implements user creation functionality:

- `CreateUser`: Creates a single user with specified ID and initial balance
- `CreateDefaultUsers`: Creates predefined users (IDs 1-5) with predefined balances for testing/demo purposes

### 3. `modify_balance.go`

Implements balance modification functionality:

- `ModifyBalance`: Unified interface for both adding and deducting from user balance
  - Handles win transactions (adding to balance)
  - Handles lose transactions (deducting from balance)
  - Prevents deductions that would result in negative balance

## Test Files

Each implementation file has a corresponding test file:

- `get_balance_test.go`: Tests for balance retrieval functionality
- `create_user_test.go`: Tests for user creation functionality
- `modify_balance_test.go`: Tests for balance modification functionality

## Key Features

1. **Business Logic Isolation**: Implements pure business logic independent of infrastructure
2. **Domain-Driven Design**: Uses entities and value objects from the domain layer
3. **Error Handling**: Thorough validation and domain-specific error types
4. **Logging**: Comprehensive logging of operations for monitoring and debugging
5. **Time Management**: Uses dependency injection for time operations for testability

## Dependencies

- Entity layer for domain objects and business rules
- Port interfaces for dependency inversion
- Error definitions for domain-specific error handling

## Usage

The UserUseCase is typically instantiated in the application's dependency injection setup and used by API handlers or other services that need user-related functionality. 