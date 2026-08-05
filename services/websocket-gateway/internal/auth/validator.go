// Package auth provides token validation for WebSocket connections.
package auth

import (
	"context"
	"fmt"

	pb "metarang/shared/pb/auth"
	grpcutil "metarang/shared/pkg/grpc"
)

// Validator validates Sanctum tokens via auth-service.
type Validator struct {
	client pb.AuthServiceClient
}

// NewValidator dials auth-service and returns a token validator.
func NewValidator(ctx context.Context, address string) (*Validator, error) {
	conn, err := grpcutil.DialContext(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("dial auth service: %w", err)
	}

	return &Validator{client: pb.NewAuthServiceClient(conn)}, nil
}

// ValidateToken checks whether the token is valid and returns the user ID.
func (v *Validator) ValidateToken(ctx context.Context, token string) (uint64, error) {
	resp, err := v.client.ValidateToken(ctx, &pb.ValidateTokenRequest{Token: token})
	if err != nil {
		return 0, err
	}
	if resp == nil || !resp.Valid {
		return 0, fmt.Errorf("invalid token")
	}
	return resp.UserId, nil
}
