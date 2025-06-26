# Load Testing Scripts

## Overview

This directory contains scripts for performance and load testing of the Balance Processor API. These scripts are designed to validate the application's performance under various load conditions and verify its ability to maintain data consistency during high concurrency scenarios.

## Available Scripts

### load-test-with-delay.go

A configurable load testing script that simulates concurrent user transactions with controlled request rates to measure the performance and reliability of the Balance Processor API.

#### Features

- **Configurable Concurrency**: Set the number of parallel workers sending requests
- **User Distribution**: Distribute load across multiple user IDs
- **Transaction Scenarios**: Various transaction types (win/lose with different amounts) 
- **Request Rate Control**: Configurable delay between requests to prevent rate limiting
- **Detailed Metrics**: Comprehensive performance statistics including:
  - Success/failure rates
  - Response time distribution (min, max, average)
  - Throughput measurements
  - Per-user and per-scenario statistics
  - Error categorization

#### Usage

```bash
go run script/load-test-with-delay.go [options]
```

#### Options

| Flag      | Description                                       | Default             |
|-----------|---------------------------------------------------|---------------------|
| `-c`      | Number of concurrent goroutines                   | 5                   |
| `-n`      | Total number of requests to make                  | 100                 |
| `-u`      | Comma-separated list of user IDs for testing      | "1,2,3"             |
| `-url`    | Base URL for the API                              | "http://localhost:8080" |
| `-source` | Source-Type header (game, server, or payment)     | "game"              |
| `-delay`  | Delay between requests in milliseconds            | 100                 |

#### Example Commands

Basic test with default settings:
```bash
go run script/load-test-with-delay.go
```

High concurrency test:
```bash
go run script/load-test-with-delay.go -c 20 -n 5000
```

Test against production environment with specific users:
```bash
go run script/load-test-with-delay.go -url https://api.example.com -u 10,11,12 -n 1000
```

Stress test with minimal delay:
```bash
go run script/load-test-with-delay.go -c 50 -n 10000 -delay 10
```

#### Output

The script provides real-time progress updates and comprehensive result statistics after completion, including:

- Total transactions processed
- Success/failure rates
- Response time statistics (min, max, average, percentiles)
- Transactions per second (TPS)
- Distribution of requests across users and scenarios
- Categorized error counts

## Adding New Scripts

When adding new testing scripts to this directory:

1. Use descriptive filenames that indicate the script's purpose
2. Follow Go best practices for code organization
3. Include command-line flags for configuration
4. Provide detailed output and statistics
5. Update this README with usage instructions 