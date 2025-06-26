FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o balance-processor ./cmd/api

# Use a minimal alpine image for the runtime container
FROM alpine:3.19

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/balance-processor .
# Copy configuration files
COPY --from=builder /app/configs /app/configs

# Create a non-root user to run the application
RUN adduser -D -g '' appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose the port the application runs on
EXPOSE 8080

# Run the application
CMD ["./balance-processor"] 