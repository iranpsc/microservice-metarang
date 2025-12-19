# Training Service - Inner Service Communication Setup

## Overview

The training service has been updated to communicate with the auth-service via gRPC to retrieve user information instead of using direct database queries. This follows microservices best practices.

## Changes Made

### 1. Auth Client Created
- **File**: `internal/client/auth_client.go`
- **Purpose**: gRPC client wrapper for auth-service UserService
- **Methods**:
  - `GetUser(ctx, userID)` - Get user by ID
  - `GetUserByCode(ctx, code)` - Get user by code (uses ListUsers search)
  - `GetUserProfile(ctx, userID)` - Get user profile with images

### 2. UserRepository Updated
- **File**: `internal/repository/user_repository.go`
- **Changes**:
  - Now accepts optional `authClient` parameter
  - Uses auth-service gRPC client when available
  - Falls back to direct DB queries if auth client is nil or fails
  - Profile photos still retrieved from DB (auth-service User doesn't include them)

## Implementation Status: ✅ COMPLETE

The `cmd/server/main.go` file has been created and configured with all required setup.

### 1. Auth Client Initialization ✅

The main.go file initializes the auth client:
```go
authServiceAddr := getEnv("AUTH_SERVICE_ADDR", "auth-service:50051")
authClient, err := client.NewAuthClient(authServiceAddr)
if err != nil {
    log.Printf("Warning: Failed to connect to auth service - will use direct DB queries: %v", err)
    authClient = nil
} else {
    log.Printf("Successfully connected to auth service at %s", authServiceAddr)
    defer authClient.Close()
}
```

### 2. UserRepository Initialization ✅

The UserRepository is initialized with the auth client:
```go
userRepo := repository.NewUserRepository(db, authClient)
```

### 3. Environment Variable ✅

Added to `config.env.sample`:
```
AUTH_SERVICE_ADDR=auth-service:50051
```

## Current Implementation Details

### Fallback Behavior
- If auth client is `nil` or connection fails, the repository falls back to direct database queries
- This ensures the service continues to work even if auth-service is unavailable
- Profile photos are always retrieved from the database (not included in auth-service User response)

### Profile Photo Handling
- Auth-service `GetUser` doesn't include profile photos
- Auth-service `GetUserProfile` includes `profile_images` array but requires viewer context
- Current implementation: Profile photos retrieved from database for simplicity
- **Future improvement**: Consider adding `latest_profile_photo` field to auth-service `User` proto

## Benefits

1. **Service Isolation**: Training service no longer directly queries users table
2. **Consistency**: User data comes from single source of truth (auth-service)
3. **Scalability**: Auth service can cache user data independently
4. **Resilience**: Falls back to DB if auth-service unavailable

## TODO / Future Improvements

1. **Add GetUserByCode to auth-service**: Currently uses ListUsers search which is less efficient
2. **Add profile photo to User proto**: Avoid separate DB query for profile photos
3. **Caching**: Consider adding caching layer for frequently accessed users
4. **Metrics**: Add metrics for auth-service call latency and failures

## Testing

When testing:
1. Test with auth-service available - should use gRPC calls
2. Test with auth-service unavailable - should fall back to DB queries
3. Verify profile photos are still retrieved correctly
4. Verify both GetUserByID and GetUserByCode work correctly
