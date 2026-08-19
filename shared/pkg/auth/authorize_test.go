package auth

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthorizeSelfOrService_Self(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserContextKey{}, &UserContext{UserID: 7})
	if err := AuthorizeSelfOrService(ctx, 7); err != nil {
		t.Fatalf("expected self access allowed: %v", err)
	}
}

func TestAuthorizeSelfOrService_OtherUserDenied(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserContextKey{}, &UserContext{UserID: 7})
	err := AuthorizeSelfOrService(ctx, 99)
	if err == nil {
		t.Fatal("expected permission denied")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", status.Code(err))
	}
}

func TestAuthorizeSelfOrService_ServiceToken(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_SECRET", "test-secret")
	md := metadata.Pairs(ServiceTokenMetadataKey, "test-secret")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	if err := AuthorizeSelfOrService(ctx, 99); err != nil {
		t.Fatalf("expected service token to allow access: %v", err)
	}
}

func TestRequireAuthenticatedUserID(t *testing.T) {
	_, err := RequireAuthenticatedUserID(context.Background())
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}

	ctx := context.WithValue(context.Background(), UserContextKey{}, &UserContext{UserID: 42})
	id, err := RequireAuthenticatedUserID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected 42, got %d", id)
	}
}
