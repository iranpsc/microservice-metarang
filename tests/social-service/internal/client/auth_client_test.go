package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metarang/shared/pb/auth"
	"metarang/social-service/internal/client"
)

type stubUserServiceClient struct {
	pb.UserServiceClient
	getProfileLimitationsFunc func(ctx context.Context, in *pb.GetProfileLimitationsRequest, opts ...grpc.CallOption) (*pb.GetProfileLimitationsResponse, error)
	calls                     []*pb.GetProfileLimitationsRequest
}

func (s *stubUserServiceClient) GetProfileLimitations(
	ctx context.Context,
	in *pb.GetProfileLimitationsRequest,
	opts ...grpc.CallOption,
) (*pb.GetProfileLimitationsResponse, error) {
	s.calls = append(s.calls, in)
	if s.getProfileLimitationsFunc != nil {
		return s.getProfileLimitationsFunc(ctx, in, opts...)
	}
	return &pb.GetProfileLimitationsResponse{}, nil
}

func TestAuthClient_CanFollow_UsesCallerUserIDFromRequest(t *testing.T) {
	followTrue := true
	stub := &stubUserServiceClient{}
	stub.getProfileLimitationsFunc = func(
		ctx context.Context,
		in *pb.GetProfileLimitationsRequest,
		_ ...grpc.CallOption,
	) (*pb.GetProfileLimitationsResponse, error) {
		// Simulate auth-service after the service-token + CallerUserId fix:
		// no user context is required; CallerUserId in the request is authoritative.
		if in.CallerUserId == 0 {
			return nil, status.Error(codes.InvalidArgument, "caller_user_id is required")
		}
		if in.CallerUserId == 1 && in.TargetUserId == 2 {
			return &pb.GetProfileLimitationsResponse{
				Data: &pb.ProfileLimitation{
					LimiterUserId: 2,
					LimitedUserId: 1,
					Options:       &pb.ProfileLimitationOptions{Follow: &followTrue},
				},
			}, nil
		}
		if in.CallerUserId == 2 && in.TargetUserId == 2 {
			return &pb.GetProfileLimitationsResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "unexpected call: %+v", in)
	}

	authClient := client.NewAuthClientFromGRPC(stub, nil)
	allowed, err := authClient.CanFollow(context.Background(), 1, 2)
	require.NoError(t, err)
	require.True(t, allowed)
	require.Len(t, stub.calls, 2)
	require.Equal(t, uint64(1), stub.calls[0].CallerUserId)
	require.Equal(t, uint64(2), stub.calls[0].TargetUserId)
	require.Equal(t, uint64(2), stub.calls[1].CallerUserId)
	require.Equal(t, uint64(2), stub.calls[1].TargetUserId)
}

func TestAuthClient_CanFollow_PropagatesUnauthenticatedFromAuthService(t *testing.T) {
	// Regression: before auth-service accepted service-token + CallerUserId,
	// GetProfileLimitations returned Unauthenticated "user context not found"
	// and follow failed with "failed to check profile limitation".
	stub := &stubUserServiceClient{}
	stub.getProfileLimitationsFunc = func(
		context.Context,
		*pb.GetProfileLimitationsRequest,
		...grpc.CallOption,
	) (*pb.GetProfileLimitationsResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "user context not found")
	}

	authClient := client.NewAuthClientFromGRPC(stub, nil)
	allowed, err := authClient.CanFollow(context.Background(), 1, 2)
	require.False(t, allowed)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get profile limitations")
	require.Contains(t, err.Error(), "user context not found")
}

func TestAuthClient_CanFollow_BlockedByTargetLimitation(t *testing.T) {
	followFalse := false
	stub := &stubUserServiceClient{}
	stub.getProfileLimitationsFunc = func(
		_ context.Context,
		in *pb.GetProfileLimitationsRequest,
		_ ...grpc.CallOption,
	) (*pb.GetProfileLimitationsResponse, error) {
		if in.CallerUserId == 1 && in.TargetUserId == 2 {
			return &pb.GetProfileLimitationsResponse{
				Data: &pb.ProfileLimitation{
					LimiterUserId: 2,
					LimitedUserId: 1,
					Options:       &pb.ProfileLimitationOptions{Follow: &followFalse},
				},
			}, nil
		}
		return &pb.GetProfileLimitationsResponse{}, nil
	}

	authClient := client.NewAuthClientFromGRPC(stub, nil)
	allowed, err := authClient.CanFollow(context.Background(), 1, 2)
	require.NoError(t, err)
	require.False(t, allowed)
}
