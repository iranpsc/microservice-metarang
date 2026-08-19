package handler

import (
	"context"

	sharedauth "metarang/shared/pkg/auth"
)

// authenticatedUserID returns the authenticated caller’s user ID from gRPC context.
func authenticatedUserID(ctx context.Context) (uint64, error) {
	return sharedauth.RequireAuthenticatedUserID(ctx)
}

// authorizeSelfOrService allows the call for the matching authenticated user or a valid service token.
func authorizeSelfOrService(ctx context.Context, userID uint64) error {
	return sharedauth.AuthorizeSelfOrService(ctx, userID)
}
