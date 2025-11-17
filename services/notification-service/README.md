# Notification Service

The Notification Service provides multi-channel delivery capabilities for MetaRGB. It is responsible for persisting in-app notifications and dispatching outbound messages via SMS and email providers.

## Responsibilities
- Persist user notifications for in-app consumption.
- Deliver SMS messages (transactional and OTP).
- Deliver email messages with plain-text and HTML support.
- Expose gRPC endpoints defined in `shared/proto/notifications.proto`.

## Project Layout
```
notification-service/
├── cmd/server            # Application entrypoint
├── internal/
│   ├── handler           # gRPC handlers (Notification, SMS, Email)
│   ├── models            # Domain models and payload DTOs
│   ├── repository        # Database persistence layer
│   └── service           # Business logic and provider abstractions
├── config.env.sample     # Example configuration
├── Dockerfile            # Multi-stage container build
└── go.mod                # Go module definition
```

## Getting Started
1. Copy environment configuration:
   ```bash
   cd services/notification-service
   cp config.env.sample .env
   ```
2. Update `.env` with your database credentials and provider secrets.
3. Run the service locally:
   ```bash
   go run ./cmd/server
   ```

## Environment Variables
Key variables consumed by the service:

- `GRPC_PORT`: gRPC listener port (default `50058`).
- `DB_*`: MySQL connection settings.
- `REDIS_*`: Optional Redis connection for rate limiting and delivery tracking.
- `SMS_*`: SMS provider configuration (Kavenegar by default).
- `SMTP_*`: SMTP server credentials for email delivery.

## Next Steps
- Implement the repository layer to match Laravel's notification persistence.
- Integrate SMS and Email providers under `internal/service`.
- Add unit tests for handlers and services.


