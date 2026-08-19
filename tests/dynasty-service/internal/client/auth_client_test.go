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
	pb "metarang/shared/pb/auth"
)

type stubUserServiceClient struct {
	pb.UserServiceClient
	getUserFunc        func(context.Context, *pb.GetUserRequest, ...grpc.CallOption) (*pb.User, error)
	getUserProfileFunc func(context.Context, *pb.GetUserProfileRequest, ...grpc.CallOption) (*pb.GetUserProfileResponse, error)
	listUsersFunc      func(context.Context, *pb.ListUsersRequest, ...grpc.CallOption) (*pb.ListUsersResponse, error)
}

func (s *stubUserServiceClient) GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.User, error) {
	if s.getUserFunc != nil {
		return s.getUserFunc(ctx, in, opts...)
	}
	return &pb.User{Id: in.UserId}, nil
}
func (s *stubUserServiceClient) GetUserProfile(ctx context.Context, in *pb.GetUserProfileRequest, opts ...grpc.CallOption) (*pb.GetUserProfileResponse, error) {
	if s.getUserProfileFunc != nil {
		return s.getUserProfileFunc(ctx, in, opts...)
	}
	return &pb.GetUserProfileResponse{}, nil
}
func (s *stubUserServiceClient) ListUsers(ctx context.Context, in *pb.ListUsersRequest, opts ...grpc.CallOption) (*pb.ListUsersResponse, error) {
	if s.listUsersFunc != nil {
		return s.listUsersFunc(ctx, in, opts...)
	}
	return &pb.ListUsersResponse{}, nil
}
func (s *stubUserServiceClient) UpdateProfile(context.Context, *pb.UpdateProfileRequest, ...grpc.CallOption) (*pb.User, error) {
	return nil, nil
}
func (s *stubUserServiceClient) GetUserLevels(context.Context, *pb.GetUserLevelsRequest, ...grpc.CallOption) (*pb.GetUserLevelsResponse, error) {
	return nil, nil
}
func (s *stubUserServiceClient) GetUserWallet(context.Context, *pb.GetUserWalletRequest, ...grpc.CallOption) (*pb.UserWalletResponse, error) {
	return nil, nil
}
func (s *stubUserServiceClient) GetUserLevel(context.Context, *pb.GetUserLevelRequest, ...grpc.CallOption) (*pb.UserLevelResponse, error) {
	return nil, nil
}
func (s *stubUserServiceClient) GetProfileLimitations(context.Context, *pb.GetProfileLimitationsRequest, ...grpc.CallOption) (*pb.GetProfileLimitationsResponse, error) {
	return nil, nil
}
func (s *stubUserServiceClient) GetUserFeaturesCount(context.Context, *pb.GetUserFeaturesCountRequest, ...grpc.CallOption) (*pb.GetUserFeaturesCountResponse, error) {
	return nil, nil
}

type stubKYCServiceClient struct {
	pb.KYCServiceClient
	getKYCFunc func(context.Context, *pb.GetKYCRequest, ...grpc.CallOption) (*pb.KYCResponse, error)
}

func (s *stubKYCServiceClient) GetKYC(ctx context.Context, in *pb.GetKYCRequest, opts ...grpc.CallOption) (*pb.KYCResponse, error) {
	if s.getKYCFunc != nil {
		return s.getKYCFunc(ctx, in, opts...)
	}
	return &pb.KYCResponse{}, nil
}
func (s *stubKYCServiceClient) UpdateKYC(context.Context, *pb.UpdateKYCRequest, ...grpc.CallOption) (*pb.KYCResponse, error) {
	return nil, nil
}
func (s *stubKYCServiceClient) ListBankAccounts(context.Context, *pb.ListBankAccountsRequest, ...grpc.CallOption) (*pb.ListBankAccountsResponse, error) {
	return nil, nil
}
func (s *stubKYCServiceClient) CreateBankAccount(context.Context, *pb.CreateBankAccountRequest, ...grpc.CallOption) (*pb.BankAccountResponse, error) {
	return nil, nil
}
func (s *stubKYCServiceClient) GetBankAccount(context.Context, *pb.GetBankAccountRequest, ...grpc.CallOption) (*pb.BankAccountResponse, error) {
	return nil, nil
}
func (s *stubKYCServiceClient) UpdateBankAccount(context.Context, *pb.UpdateBankAccountRequest, ...grpc.CallOption) (*pb.BankAccountResponse, error) {
	return nil, nil
}
func (s *stubKYCServiceClient) DeleteBankAccount(context.Context, *pb.DeleteBankAccountRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func TestAuthClient_Methods(t *testing.T) {
	user := &stubUserServiceClient{}
	kyc := &stubKYCServiceClient{}
	c := client.NewAuthClientFromGRPC(user, kyc, nil)

	u, err := c.GetUser(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), u.Id)

	_, err = c.GetUserProfile(context.Background(), 3)
	require.NoError(t, err)

	_, err = c.ListUsers(context.Background(), "ali", 1)
	require.NoError(t, err)

	_, err = c.GetKYC(context.Background(), 3)
	require.NoError(t, err)
	require.NoError(t, c.Close())

	user.getUserFunc = func(context.Context, *pb.GetUserRequest, ...grpc.CallOption) (*pb.User, error) {
		return nil, errors.New("fail")
	}
	_, err = c.GetUser(context.Background(), 1)
	require.Error(t, err)

	user.getUserProfileFunc = func(context.Context, *pb.GetUserProfileRequest, ...grpc.CallOption) (*pb.GetUserProfileResponse, error) {
		return nil, errors.New("fail")
	}
	_, err = c.GetUserProfile(context.Background(), 1)
	require.Error(t, err)

	user.listUsersFunc = func(context.Context, *pb.ListUsersRequest, ...grpc.CallOption) (*pb.ListUsersResponse, error) {
		return nil, errors.New("fail")
	}
	_, err = c.ListUsers(context.Background(), "x", 1)
	require.Error(t, err)

	kyc.getKYCFunc = func(context.Context, *pb.GetKYCRequest, ...grpc.CallOption) (*pb.KYCResponse, error) {
		return nil, errors.New("fail")
	}
	_, err = c.GetKYC(context.Background(), 1)
	require.Error(t, err)
}
