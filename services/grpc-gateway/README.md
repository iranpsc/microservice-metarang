# gRPC Gateway Service

## Overview

The gRPC Gateway Service translates REST/JSON HTTP requests to gRPC/protobuf calls and vice versa. It acts as a bridge between REST clients and gRPC microservices.

## Architecture

```
Client → Kong (8000) → gRPC Gateway (8080) → gRPC Services
                ↓              ↓                    ↓
            REST/JSON    JSON↔Protobuf        gRPC/Protobuf
```

## Features

- REST to gRPC translation for all auth endpoints
- JSON to Protobuf message conversion
- Protobuf to JSON response conversion
- Proper error handling with HTTP status codes
- CORS support
- Health check endpoint

## Endpoints

### Authentication Endpoints

- `POST /api/auth/register` - User registration
- `GET /api/auth/redirect` - OAuth redirect
- `GET /api/auth/callback` - OAuth callback
- `POST /api/auth/me` - Get current user
- `POST /api/auth/logout` - User logout
- `POST /api/auth/validate` - Validate token
- `POST /api/auth/account-security/request` - Request account security OTP
- `POST /api/auth/account-security/verify` - Verify account security OTP

### User Endpoints

- `GET /api/user?user_id={id}` - Get user by ID
- `PUT /api/user/profile` - Update user profile

### KYC Endpoints

- `POST /api/kyc/submit` - Submit KYC information
- `GET /api/kyc/status?user_id={id}` - Get KYC status
- `POST /api/kyc/bank-account` - Verify bank account

## Configuration

Environment variables:

- `HTTP_PORT` - HTTP server port (default: 8080)
- `AUTH_SERVICE_ADDR` - Auth service gRPC address (default: auth-service:50051)

## Building

```bash
cd services/grpc-gateway
go mod download
go build -o grpc-gateway ./cmd/server
```

## Running

```bash
./grpc-gateway
```

Or with Docker:

```bash
docker-compose up grpc-gateway
```

## Testing

```bash
# Health check
curl http://localhost:8080/health

# Example: Get user info
curl -X POST http://localhost:8080/api/auth/me \
  -H "Content-Type: application/json" \
  -d '{"token": "your-token-here"}'
```

## Error Handling

The gateway translates gRPC error codes to HTTP status codes:

- `Unauthenticated` → 401 Unauthorized
- `NotFound` → 404 Not Found
- `InvalidArgument` → 400 Bad Request
- `PermissionDenied` → 403 Forbidden
- `AlreadyExists` → 409 Conflict
- `FailedPrecondition` → 412 Precondition Failed
- Others → 500 Internal Server Error

