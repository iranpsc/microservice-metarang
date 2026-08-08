package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"metarang/dynasty-service/internal/client"
	pb "metarang/shared/pb/commercial"
)

type stubWalletServiceClient struct {
	pb.WalletServiceClient
	addBalanceFunc func(context.Context, *pb.AddBalanceRequest, ...grpc.CallOption) (*pb.AddBalanceResponse, error)
	getWalletFunc  func(context.Context, *pb.GetWalletRequest, ...grpc.CallOption) (*pb.WalletResponse, error)
	lastAdd        *pb.AddBalanceRequest
}

func (s *stubWalletServiceClient) AddBalance(ctx context.Context, in *pb.AddBalanceRequest, opts ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
	s.lastAdd = in
	if s.addBalanceFunc != nil {
		return s.addBalanceFunc(ctx, in, opts...)
	}
	return &pb.AddBalanceResponse{Success: true}, nil
}

func (s *stubWalletServiceClient) GetWallet(ctx context.Context, in *pb.GetWalletRequest, opts ...grpc.CallOption) (*pb.WalletResponse, error) {
	if s.getWalletFunc != nil {
		return s.getWalletFunc(ctx, in, opts...)
	}
	return &pb.WalletResponse{}, nil
}

func (s *stubWalletServiceClient) CreateWallet(context.Context, *pb.CreateWalletRequest, ...grpc.CallOption) (*pb.WalletResponse, error) {
	return nil, nil
}
func (s *stubWalletServiceClient) DeductBalance(context.Context, *pb.DeductBalanceRequest, ...grpc.CallOption) (*pb.DeductBalanceResponse, error) {
	return nil, nil
}
func (s *stubWalletServiceClient) LockBalance(context.Context, *pb.LockBalanceRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}
func (s *stubWalletServiceClient) UnlockBalance(context.Context, *pb.UnlockBalanceRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func TestCommercialClient_IncrementWalletPSC_AndSatisfaction(t *testing.T) {
	stub := &stubWalletServiceClient{}
	c := client.NewCommercialClientFromGRPC(stub, nil)

	require.NoError(t, c.IncrementWalletPSC(context.Background(), 5, 12.5))
	require.NotNil(t, stub.lastAdd)
	assert.Equal(t, "psc", stub.lastAdd.Asset)
	assert.Equal(t, uint64(5), stub.lastAdd.UserId)
	assert.Equal(t, 12.5, stub.lastAdd.Amount)

	require.NoError(t, c.IncrementSatisfaction(context.Background(), 5, 3))
	assert.Equal(t, "satisfaction", stub.lastAdd.Asset)

	stub.addBalanceFunc = func(context.Context, *pb.AddBalanceRequest, ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
		return &pb.AddBalanceResponse{Success: false, Message: "nope"}, nil
	}
	err := c.IncrementWalletPSC(context.Background(), 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add balance failed")

	stub.addBalanceFunc = func(context.Context, *pb.AddBalanceRequest, ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
		return nil, errors.New("rpc fail")
	}
	err = c.IncrementSatisfaction(context.Background(), 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add satisfaction")
}

func TestCommercialClient_GetWallet_AndClose(t *testing.T) {
	stub := &stubWalletServiceClient{
		getWalletFunc: func(context.Context, *pb.GetWalletRequest, ...grpc.CallOption) (*pb.WalletResponse, error) {
			return &pb.WalletResponse{}, nil
		},
	}
	c := client.NewCommercialClientFromGRPC(stub, nil)
	resp, err := c.GetWallet(context.Background(), 9)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, c.Close())

	stub.getWalletFunc = func(context.Context, *pb.GetWalletRequest, ...grpc.CallOption) (*pb.WalletResponse, error) {
		return nil, errors.New("down")
	}
	_, err = c.GetWallet(context.Background(), 9)
	require.Error(t, err)
}
