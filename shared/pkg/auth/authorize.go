package auth

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// HasValidServiceToken reports whether the incoming context carries a valid inter-service token.
func HasValidServiceToken(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	tokens := md.Get(ServiceTokenMetadataKey)
	if len(tokens) == 0 || tokens[0] == "" {
		return false
	}
	return ValidateServiceToken(tokens[0])
}

// RequireAuthenticatedUserID returns the authenticated user ID from context.
func RequireAuthenticatedUserID(ctx context.Context) (uint64, error) {
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if userCtx.UserID == 0 {
		return 0, status.Error(codes.Unauthenticated, "authentication required")
	}
	return userCtx.UserID, nil
}

// AuthorizeSelfOrService allows the call when a valid service token is present,
// or when the authenticated user matches userID.
func AuthorizeSelfOrService(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	if HasValidServiceToken(ctx) {
		return nil
	}
	authenticatedID, err := RequireAuthenticatedUserID(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	if authenticatedID != userID {
		return status.Error(codes.PermissionDenied, "access denied")
	}
	return nil
}
