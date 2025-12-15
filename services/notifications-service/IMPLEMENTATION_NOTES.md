# Notification Service Implementation Notes

## Overview
The notification service has been implemented according to the API documentation in `api-docs/notification-service/notifications_api.md`.

## Completed Implementation

### 1. Proto File Updates
- Added `GetNotification` RPC method to `NotificationService`
- Added `GetNotificationRequest` message
- Added `unread_only` field to `GetNotificationsRequest` to filter unread notifications

### 2. Repository Layer
- Updated `ListNotifications` to support `UnreadOnly` filter
- Added `GetNotificationByID` method to retrieve a single notification
- All database queries properly handle unread filtering with `read_at IS NULL`

### 3. Service Layer
- Added `GetNotificationByID` method to `NotificationService` interface
- Implemented error handling for not found cases using `ErrNotificationNotFound`

### 4. Handler Layer
- Updated `GetNotifications` to support `unread_only` parameter
- Added `GetNotification` handler method
- Updated `convertNotification` to format dates/times in Jalali format:
  - `created_at`: Jalali date and time as "Y/m/d H:m:s"
  - `read_at`: RFC3339 format for read notifications, empty string for unread

### 5. gRPC Gateway Handler
- Created `notification_handler.go` with REST endpoint handlers:
  - `GET /api/notifications` - Returns unread notifications (defaults to unread_only=true)
  - `GET /api/notifications/{id}` - Returns single notification
  - `POST /api/notifications/read/{id}` - Marks notification as read
  - `POST /api/notifications/read/all` - Marks all notifications as read
- Response transformation matches API docs format:
  - Separate `date` (Jalali Y/m/d) and `time` (H:m:s) fields
  - `data` object with required fields (related-to, sender-name, sender-image, message)
  - Proper `read_at` handling (null for unread)

### 6. Kong Configuration
- Updated to route through `grpc-gateway-service` instead of direct gRPC
- Added separate routes for:
  - List notifications (GET /api/notifications)
  - Get notification (GET /api/notifications/{id})
  - Mark as read (POST /api/notifications/read/{id})
  - Mark all as read (POST /api/notifications/read/all)
- All routes configured with JWT authentication and rate limiting

### 7. Tests
- Added repository tests:
  - `TestListNotifications_UnreadOnly` - Tests unread filtering
  - `TestGetNotificationByID` - Tests single notification retrieval with various scenarios
- All repository tests pass

## Required Next Steps

### 1. Regenerate Proto Files
After updating the proto file, you need to regenerate the Go code:

```bash
# Install protoc-gen-go and protoc-gen-go-grpc if not already installed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Regenerate notifications proto
make gen-notifications

# Or regenerate all protos
make proto
```

### 2. Verify Compilation
After regenerating proto files, verify everything compiles:

```bash
cd services/notifications-service
go build ./...

cd ../grpc-gateway
go build ./...
```

### 3. Run Tests
```bash
cd services/notifications-service
go test ./... -v
```

### 4. Update gRPC Gateway Config
The gRPC gateway configuration in `services/grpc-gateway/internal/config/config.go` has been updated to include `NotificationServiceAddr`. Ensure the environment variable is set or uses the default value `notifications-service:50058`.

### 5. Database Schema
Ensure the `notifications` table exists with the following structure (matching Laravel's schema):
- `id` (VARCHAR/CHAR) - UUID primary key
- `type` (VARCHAR)
- `notifiable_type` (VARCHAR) - e.g., "App\\User"
- `notifiable_id` (BIGINT UNSIGNED)
- `data` (JSON/TEXT) - JSON encoded notification data
- `read_at` (TIMESTAMP NULL)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

## API Compatibility

The implementation follows the API documentation exactly:
- **GET /api/notifications**: Returns unread notifications only (as per API docs)
- **GET /api/notifications/{id}**: Returns single notification with proper 404 handling
- **POST /api/notifications/read/{id}**: Returns 204 No Content on success
- **POST /api/notifications/read/all**: Returns 204 No Content on success

Response format matches API docs:
- `id`: UUID string
- `data`: Object with related-to, sender-name, sender-image, message
- `read_at`: null for unread, RFC3339 timestamp for read
- `date`: Jalali date format (Y/m/d)
- `time`: Time format (H:m:s)

## Notes

1. **Jalali Date Formatting**: Uses `metargb/shared/pkg/helpers` for Jalali date conversion
2. **Unread Filtering**: By default, `GET /api/notifications` filters unread only (as per API docs)
3. **Error Handling**: Proper gRPC status codes are returned (NotFound, InvalidArgument, Internal)
4. **Authentication**: All endpoints require JWT authentication via Kong
5. **Rate Limiting**: Configured at 100 requests/minute, 2000 requests/hour per Kong config
