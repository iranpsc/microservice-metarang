# Dynasty Service

The Dynasty Service manages family dynasties, member relationships, join requests, permissions, and prizes within the MetaRGB microservices architecture.

## Overview

This service implements the core dynasty functionality including:
- Dynasty creation and management
- Family member relationships and join requests
- Children permission controls
- Dynasty prize system with financial integration
- User search for family member discovery

## Architecture

### Technology Stack
- **Language**: Go 1.24+
- **Database**: MySQL
- **gRPC**: Inter-service communication
- **Protocol Buffers**: Service definitions in `shared/proto/dynasty.proto`

### Service Dependencies
- **Auth Service**: User authentication and validation
- **Features Service**: Feature ownership and properties
- **Commercial Service**: Wallet updates and financial transactions
- **Notification Service**: Sending notifications to users

## API Endpoints

All endpoints are exposed via the `grpc-gateway` service as HTTP REST endpoints.

### Dynasty Management
- `GET /api/dynasty` - Get user's dynasty or available features/intro prizes
- `POST /api/dynasty/create/{feature}` - Create dynasty with feature
- `POST /api/dynasty/{dynasty}/update/{feature}` - Update dynasty feature

### Family Management
- `GET /api/dynasty/{dynasty}/family/{family}` - List family members
- `POST /api/dynasty/add/member` - Send join request
- `POST /api/dynasty/search` - Search users for family member discovery

### Join Requests
- `GET /api/dynasty/requests/sent` - List sent join requests
- `GET /api/dynasty/requests/sent/{joinRequest}` - View sent request
- `DELETE /api/dynasty/requests/sent/{joinRequest}` - Delete sent request
- `GET /api/dynasty/requests/recieved` - List received join requests
- `GET /api/dynasty/requests/recieved/{joinRequest}` - View received request
- `POST /api/dynasty/requests/recieved/{joinRequest}` - Accept join request
- `DELETE /api/dynasty/requests/recieved/{joinRequest}` - Reject join request

### Children Permissions
- `POST /api/dynasty/children/{user}` - Update child permission
- `POST /api/dynasty/add/member/get/permissions` - Get default permissions

### Dynasty Prizes
- `GET /api/dynasty/prizes` - List unclaimed prizes
- `GET /api/dynasty/prizes/{recievedPrize}` - View prize details
- `POST /api/dynasty/prizes/{recievedPrize}` - Claim prize

## Database Schema

The service uses the following main tables:
- `dynasties` - Dynasty records
- `families` - Family records linked to dynasties
- `family_members` - Family member relationships
- `join_requests` - Join request lifecycle
- `children_permissions` - Permission controls for minors
- `dynasty_permissions` - Default dynasty permissions
- `dynasty_prizes` - Prize definitions
- `received_prizes` - Awarded prizes awaiting redemption
- `dynasty_messages` - Message templates for notifications

See `scripts/dynasty_schema.sql` for the complete schema.

## Business Logic

### Dynasty Creation
- Users can create one dynasty per account
- Dynasty is tied to a residential feature (karbari='m')
- Creating a dynasty automatically creates a family with the user as owner

### Feature Updates
- Users can change their dynasty's feature
- Changes within 30 days trigger debt creation
- Previous feature is locked for one month to prevent rapid changes

### Join Request Flow
1. User sends join request with relationship type
2. Receiver views pending requests
3. Receiver accepts (adds to family) or rejects
4. Accepting awards prizes and syncs permissions for minors

### Children Permissions
- Parents can control permissions for underage family members
- Default permissions are set when minors join families
- Permission codes: BFR, SF, W, JU, DM, PIUP, PITC, PIC, ESOO, COTB

### Prize System
- Prizes are awarded when family members join
- Prizes include PSC, satisfaction, and variable increases
- Claiming a prize updates wallet and deletes the received prize record

## Configuration

Environment variables (see `config.env.sample`):
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `GRPC_PORT` - gRPC server port (default: 50056)
- `AUTH_SERVICE_ADDR` - Auth service address
- `COMMERCIAL_SERVICE_ADDR` - Commercial service address
- `FEATURES_SERVICE_ADDR` - Features service address
- `NOTIFICATION_SERVICE_ADDR` - Notification service address

## Development

### Building
```bash
cd services/dynasty-service
go build -o bin/dynasty-service ./cmd/server
```

### Running
```bash
./bin/dynasty-service
```

### Testing
```bash
# Unit tests
go test ./internal/...

# Integration tests
go test ./tests/dynasty-service/...
```

### Database Migrations
Run the schema script:
```bash
mysql -u user -p database < scripts/dynasty_schema.sql
```

## Monitoring & Metrics

### Key Metrics to Monitor

#### Golden Signals
- **Request Rate**: `dynasty_requests_total{method, endpoint, status}`
- **Error Rate**: `dynasty_errors_total{type, endpoint}`
- **Latency**: `dynasty_request_duration_seconds{endpoint, quantile}`
- **Saturation**: Resource usage (CPU, memory, connections)

#### Business Metrics
- `dynasty_created_total` - Total dynasties created
- `dynasty_feature_updates_total` - Feature change count
- `join_requests_sent_total{relationship}` - Join requests by relationship
- `join_requests_accepted_total` - Accepted requests
- `join_requests_rejected_total` - Rejected requests
- `prizes_awarded_total{member_type}` - Prizes awarded by member type
- `prizes_claimed_total` - Prizes claimed
- `child_permissions_updated_total` - Permission updates

#### Database Metrics
- `db_connections_active` - Active database connections
- `db_query_duration_seconds{query_type}` - Query performance
- `db_query_errors_total` - Database errors

#### Dependency Health
- `grpc_client_requests_total{service, method}` - gRPC call counts
- `grpc_client_errors_total{service, method}` - gRPC errors
- `grpc_client_duration_seconds{service, method}` - gRPC latency

### Logging

The service logs at different levels:
- **ERROR**: Service errors, gRPC failures, database errors
- **WARN**: Policy violations, validation failures
- **INFO**: Business events (dynasty created, requests sent, prizes awarded)
- **DEBUG**: Detailed request/response logging (development only)

### Log Format
Structured JSON logging with fields:
- `timestamp` - Log timestamp
- `level` - Log level
- `service` - Service name
- `method` - HTTP/gRPC method
- `endpoint` - Request endpoint
- `user_id` - User ID (if applicable)
- `message` - Log message
- `error` - Error details (if applicable)

### Health Checks

The service exposes a health check endpoint:
- gRPC: `grpc.health.v1.Health/Check`

Health check verifies:
- Database connectivity
- gRPC client connections to dependencies
- Service readiness

## Deployment

### Docker
```bash
docker build -t dynasty-service:latest -f Dockerfile .
docker run -p 50056:50056 --env-file config.env dynasty-service:latest
```

### Kubernetes
See `k8s/dynasty-service/` for Kubernetes deployment manifests.

### Environment Variables
All configuration is provided via environment variables or `config.env` file.

## Troubleshooting

### Common Issues

#### Service won't start
- Check database connectivity
- Verify environment variables are set
- Check gRPC port availability

#### gRPC client errors
- Verify dependency services are running
- Check service addresses in configuration
- Review network connectivity

#### Database errors
- Verify database schema is up to date
- Check connection pool settings
- Monitor connection pool exhaustion

### Debug Mode
Set `LOG_LEVEL=debug` for detailed logging.

## API Documentation

Full API documentation is available in:
- `api-docs/dynasty-service/dynasty_api.md`
- `api-docs/dynasty-service/dynasty_join_requests_api.md`
- `api-docs/dynasty-service/dynasty_children_permissions_api.md`
- `api-docs/dynasty-service/dynasty_prize_api.md`

## License

Proprietary - MetaRGB

