# MetaRGB Microservices

This directory contains the microservices implementation for the MetaRGB platform migration from Laravel monolith to Golang/gRPC microservices.

## Architecture

### Services
- **auth-service**: Authentication, User Management, KYC (Port 50051)
- **commercial-service**: Wallet, Transactions, Payments (Port 50052)
- **features-service**: Features (Lands), Marketplace (Port 50053)
- **levels-service**: User Progression, Activities, Challenges (Port 50054)
- **dynasty-service**: Dynasty Management, Family Members (Port 50055)
- **support-service**: Tickets, Reports (Port 50056)
- **training-service**: Video Tutorials, Comments (Port 50057)
- **notifications-service**: Multi-channel Notifications (Port 50058)
- **calendar-service**: Events Management (Port 50059)
- **file-storage-service**: File Upload/Management (Port 50060)

### Shared Components
- **shared/proto**: Protocol Buffer definitions
- **shared/pkg**: Shared Go packages (db, auth, logger, metrics, helpers)

### Infrastructure
- **Database**: MySQL 8 (shared instance)
- **Gateway**: Kong (HTTP/REST → gRPC translation)
- **Service Mesh**: Istio (mTLS, traffic management)
- **Monitoring**: Prometheus + Grafana
- **Caching**: Redis

## Project Structure

```
metargb-microservices/
├── services/
│   ├── auth-service/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── handler/        # gRPC handlers
│   │   │   ├── service/        # Business logic
│   │   │   ├── repository/     # Data access
│   │   │   └── models/         # Domain models
│   │   └── go.mod
│   ├── commercial-service/
│   │   └── ...
│   └── .../
├── shared/
│   ├── proto/                  # .proto files
│   ├── pb/                     # Generated Go code
│   └── pkg/
│       ├── auth/               # gRPC auth interceptor
│       ├── db/                 # Database utilities
│       ├── helpers/            # Helper functions
│       ├── logger/             # Structured logging
│       └── metrics/            # Prometheus metrics
├── k8s/                        # Kubernetes manifests
├── scripts/                    # Database schema and utilities
└── Makefile
```

## Getting Started

### Prerequisites
- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- MySQL 8
- Docker & Kubernetes (for deployment)

### Generate Proto Files
```bash
make proto
```

### Run Auth Service (Development)
```bash
cd services/auth-service
cp config.env.sample .env
# Edit .env with your database credentials
go run cmd/server/main.go
```

### Run Commercial Service (Development)
```bash
cd services/commercial-service
cp config.env.sample .env
go run cmd/server/main.go
```

## Development Workflow

1. **Define API**: Update `.proto` files in `shared/proto/`
2. **Generate Code**: Run `make proto`
3. **Implement Service**: Create/update handlers, services, repositories
4. **Test**: Write unit and integration tests
5. **Build**: `docker build -t service-name .`
6. **Deploy**: Apply Kubernetes manifests

## Database Schema

The schema is shared across all services and maintained in `scripts/schema.sql`. Each service connects to the same MySQL instance but with table-level permissions.

**Important Schema Notes**:
- `transactions.id`: VARCHAR (format: TR-xxxxx)
- `feature_properties.id`: VARCHAR (with prefix/postfix)
- Wallet balances: DECIMAL(20,10) for high precision
- Soft deletes: Check `deleted_at` column
- Polymorphic relations: Use `{model}_type` and `{model}_id`

## API Compatibility

**CRITICAL**: All microservices MUST maintain 100% API compatibility with the Laravel monolith:
- Exact JSON field names and types
- Exact HTTP status codes
- Exact validation error formats
- Exact date/time formats (Jalali calendar)
- Exact URL structures

Golden JSON tests verify byte-for-byte compatibility.

## Configuration

Each service uses environment variables for configuration:

```bash
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=metargb_service
DB_PASSWORD=secret
DB_DATABASE=metargb_db

# gRPC
GRPC_PORT=50051

# Service-specific (e.g., OAuth for auth-service)
OAUTH_SERVER_URL=https://oauth.example.com
OAUTH_CLIENT_ID=...
OAUTH_CLIENT_SECRET=...
```

## Testing

> **Note:** All inter-service APIs use gRPC over HTTP/2. Hitting the Kong routes
> (for example `http://localhost:8000/api/auth`) with a plain REST client will
> now return `415 Unsupported Media Type`. Use a gRPC-capable client such as
> the gRPC tab in Postman or [grpcurl](https://github.com/fullstorydev/grpcurl):
>
> ```bash
> grpcurl -plaintext \
>   -import-path shared/proto -proto auth.proto \
>   -d '{"back_url":"https://example.com"}' \
>   localhost:8000 auth.AuthService/Register
> ```

### Unit Tests
```bash
cd services/auth-service
go test ./...
```

### Integration Tests
```bash
# Start all services
# Run integration test suite
go test ./tests/integration/...
```

### Golden JSON Tests
```bash
# Compare microservice responses with Laravel outputs
go test ./tests/golden/...
```

## Deployment

### Docker Build
```bash
docker build -t metargb/auth-service:latest -f services/auth-service/Dockerfile .
```

### Kubernetes Deploy
```bash
kubectl apply -f k8s/auth-service/
```

### Kong Gateway
```bash
# Configure routes
kubectl apply -f k8s/kong/routes.yaml
```

## Monitoring

- **Metrics**: Exposed on `:9090/metrics` (Prometheus format)
- **Health**: `:50051` gRPC health check
- **Logs**: JSON structured logs to stdout

## Contributing

1. Follow the inspection-before-implementation rule
2. Maintain API compatibility
3. Write tests
4. Update documentation
5. Run linters before commit

## License

Proprietary - MetaRGB Platform
