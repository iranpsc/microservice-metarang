# MetaRGB Shared Libraries

This directory contains shared Go packages and Protocol Buffer definitions used across all MetaRGB microservices.

## Structure

```
shared/
├── pkg/              # Go packages
│   ├── db/          # Database utilities
│   ├── auth/        # Authentication middleware
│   ├── logger/      # Logging utilities
│   ├── metrics/     # Prometheus metrics
│   └── helpers/     # Helper functions
└── proto/           # Protocol Buffer definitions
    ├── common.proto
    ├── auth.proto
    ├── commercial.proto
    ├── features.proto
    └── notifications.proto
```

## Packages

### db/
Database connection management, schema validation, and soft-delete query helpers.

- `connection.go`: MySQL connection pool with retry logic
- `schema_guard.go`: Validates database schema matches expectations
- `soft_delete.go`: Query builder for soft-delete aware queries

### auth/
gRPC authentication and authorization interceptors.

- `interceptor.go`: Token validation and user context injection

### logger/
Structured logging with request ID propagation.

- `logger.go`: JSON logger with gRPC interceptors

### metrics/
Prometheus metrics collection.

- `metrics.go`: Request counters, histograms, and gauges

### helpers/
Various utility functions.

- `jalali.go`: Jalali (Persian) calendar formatting
- `numbers.go`: Number formatting and Persian number normalization
- `id_generator.go`: ID generation for VARCHAR primary keys
- `validation.go`: Custom validators for Persian data

## Proto Definitions

Protocol Buffer definitions for gRPC services. Generate Go code with:

```bash
cd shared/proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    *.proto
```

## Usage

Import in your service:

```go
import (
    "github.com/metargb/shared/pkg/db"
    "github.com/metargb/shared/pkg/logger"
    "github.com/metargb/shared/proto/auth"
)
```

## Development

Run tests:
```bash
go test ./...
```

Update dependencies:
```bash
go mod tidy
```

