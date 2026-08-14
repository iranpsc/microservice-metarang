package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "metarang/shared/pb/commercial"
	"metarang/social-service/internal/client"
)

type stubWalletClient struct {
	pb.WalletServiceClient
	addBalanceFunc func(ctx context.Context, in *pb.AddBalanceRequest, opts ...grpc.CallOption) (*pb.AddBalanceResponse, error)
}

func (s *stubWalletClient) AddBalance(
	ctx context.Context,
	in *pb.AddBalanceRequest,
	opts ...grpc.CallOption,
) (*pb.AddBalanceResponse, error) {
	if s.addBalanceFunc != nil {
		return s.addBalanceFunc(ctx, in, opts...)
	}
	return &pb.AddBalanceResponse{Success: true}, nil
}

func TestCommercialClient_AddBalance_Success(t *testing.T) {
	stub := &stubWalletClient{}
	var got *pb.AddBalanceRequest
	stub.addBalanceFunc = func(_ context.Context, in *pb.AddBalanceRequest, _ ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
		got = in
		return &pb.AddBalanceResponse{Success: true}, nil
	}
	c := client.NewCommercialClientFromGRPC(stub)
	require.NoError(t, c.AddBalance(context.Background(), 5, "psc", 12.5))
	require.NotNil(t, got)
	require.Equal(t, uint64(5), got.UserId)
	require.Equal(t, "psc", got.Asset)
	require.Equal(t, 12.5, got.Amount)
	require.NoError(t, c.Close())
}

func TestCommercialClient_AddBalance_SuccessFalse(t *testing.T) {
	stub := &stubWalletClient{
		addBalanceFunc: func(context.Context, *pb.AddBalanceRequest, ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
			return &pb.AddBalanceResponse{Success: false, Message: "insufficient"}, nil
		},
	}
	c := client.NewCommercialClientFromGRPC(stub)
	err := c.AddBalance(context.Background(), 1, "psc", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "add balance failed")
	require.Contains(t, err.Error(), "insufficient")
}

func TestCommercialClient_AddBalance_GRPCError(t *testing.T) {
	stub := &stubWalletClient{
		addBalanceFunc: func(context.Context, *pb.AddBalanceRequest, ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
			return nil, errors.New("rpc down")
		},
	}
	c := client.NewCommercialClientFromGRPC(stub)
	err := c.AddBalance(context.Background(), 1, "psc", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to add balance")
}

func TestNewCommercialClient_InvalidAddress(t *testing.T) {
	t.Setenv("GRPC_TLS_ENABLED", "true")
	t.Setenv("GRPC_TLS_CA_FILE", t.TempDir()+"/missing-ca.pem")
	_, err := client.NewCommercialClient("localhost:1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to connect to commercial service")
}
