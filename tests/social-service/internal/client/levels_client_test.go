package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "metarang/shared/pb/levels"
	"metarang/social-service/internal/client"
)

type stubActivityClient struct {
	pb.ActivityServiceClient
	recordFollowerFunc func(ctx context.Context, in *pb.RecordFollowerRequest, opts ...grpc.CallOption) (*pb.RecordFollowerResponse, error)
}

func (s *stubActivityClient) RecordFollower(
	ctx context.Context,
	in *pb.RecordFollowerRequest,
	opts ...grpc.CallOption,
) (*pb.RecordFollowerResponse, error) {
	if s.recordFollowerFunc != nil {
		return s.recordFollowerFunc(ctx, in, opts...)
	}
	return &pb.RecordFollowerResponse{Success: true}, nil
}

func TestLevelsClient_RecordFollower_Success(t *testing.T) {
	stub := &stubActivityClient{}
	var got uint64
	stub.recordFollowerFunc = func(_ context.Context, in *pb.RecordFollowerRequest, _ ...grpc.CallOption) (*pb.RecordFollowerResponse, error) {
		got = in.UserId
		return &pb.RecordFollowerResponse{Success: true}, nil
	}
	c := client.NewLevelsClientFromGRPC(stub)
	require.NoError(t, c.RecordFollower(context.Background(), 42))
	require.Equal(t, uint64(42), got)
	require.NoError(t, c.Close())
}

func TestLevelsClient_RecordFollower_SuccessFalse(t *testing.T) {
	stub := &stubActivityClient{
		recordFollowerFunc: func(context.Context, *pb.RecordFollowerRequest, ...grpc.CallOption) (*pb.RecordFollowerResponse, error) {
			return &pb.RecordFollowerResponse{Success: false}, nil
		},
	}
	c := client.NewLevelsClientFromGRPC(stub)
	err := c.RecordFollower(context.Background(), 7)
	require.Error(t, err)
	require.Contains(t, err.Error(), "record follower failed")
}

func TestLevelsClient_RecordFollower_GRPCError(t *testing.T) {
	stub := &stubActivityClient{
		recordFollowerFunc: func(context.Context, *pb.RecordFollowerRequest, ...grpc.CallOption) (*pb.RecordFollowerResponse, error) {
			return nil, errors.New("rpc down")
		},
	}
	c := client.NewLevelsClientFromGRPC(stub)
	err := c.RecordFollower(context.Background(), 7)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to record follower")
}

func TestNewLevelsClient_InvalidAddress(t *testing.T) {
	t.Setenv("GRPC_TLS_ENABLED", "true")
	t.Setenv("GRPC_TLS_CA_FILE", t.TempDir()+"/missing-ca.pem")
	_, err := client.NewLevelsClient("localhost:1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to connect to levels service")
}
