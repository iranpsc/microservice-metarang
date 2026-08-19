package client_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "metarang/shared/pb/auth"
	"metarang/training-service/internal/client"
	"metarang/training-service/tests/internal/testutil"
)

type stubUserService struct {
	authpb.UnimplementedUserServiceServer
	getUserFunc        func(context.Context, *authpb.GetUserRequest) (*authpb.User, error)
	listUsersFunc      func(context.Context, *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error)
	getUserProfileFunc func(context.Context, *authpb.GetUserProfileRequest) (*authpb.GetUserProfileResponse, error)
}

func (s *stubUserService) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.User, error) {
	if s.getUserFunc != nil {
		return s.getUserFunc(ctx, req)
	}
	return &authpb.User{Id: req.UserId, Name: "Test", Code: "t1", Email: "t@example.com"}, nil
}

func (s *stubUserService) ListUsers(ctx context.Context, req *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
	if s.listUsersFunc != nil {
		return s.listUsersFunc(ctx, req)
	}
	return &authpb.ListUsersResponse{}, nil
}

func (s *stubUserService) GetUserProfile(ctx context.Context, req *authpb.GetUserProfileRequest) (*authpb.GetUserProfileResponse, error) {
	if s.getUserProfileFunc != nil {
		return s.getUserProfileFunc(ctx, req)
	}
	return &authpb.GetUserProfileResponse{}, nil
}

func newAuthClient(t *testing.T, stub *stubUserService) (*client.AuthClient, func()) {
	t.Helper()
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		authpb.RegisterUserServiceServer(s, stub)
	})
	return client.NewAuthClientFromConn(conn), cleanup
}

func TestAuthClient_CloseNilConn(t *testing.T) {
	c := &client.AuthClient{}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthClient_GetUser_SuccessAndError(t *testing.T) {
	stub := &stubUserService{}
	c, cleanup := newAuthClient(t, stub)
	defer cleanup()

	user, err := c.GetUser(context.Background(), 42)
	if err != nil || user.Id != 42 || user.Code != "t1" {
		t.Fatalf("user=%+v err=%v", user, err)
	}

	stub.getUserFunc = func(context.Context, *authpb.GetUserRequest) (*authpb.User, error) {
		return nil, status.Error(codes.Internal, "fail")
	}
	_, err = c.GetUser(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}

	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthClient_GetUserByCode_SuccessNotFoundAndListError(t *testing.T) {
	stub := &stubUserService{
		listUsersFunc: func(_ context.Context, req *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
			if req.Search != "abc" {
				t.Fatalf("search=%s", req.Search)
			}
			return &authpb.ListUsersResponse{
				Data: []*authpb.UserListItem{{Id: 7, Code: "abc"}},
			}, nil
		},
		getUserFunc: func(_ context.Context, req *authpb.GetUserRequest) (*authpb.User, error) {
			if req.UserId != 7 {
				t.Fatalf("user=%d", req.UserId)
			}
			return &authpb.User{Id: 7, Code: "abc", Name: "Alice"}, nil
		},
	}
	c, cleanup := newAuthClient(t, stub)
	defer cleanup()

	user, err := c.GetUserByCode(context.Background(), "abc")
	if err != nil || user.Id != 7 || user.Name != "Alice" {
		t.Fatalf("user=%+v err=%v", user, err)
	}

	stub.listUsersFunc = func(context.Context, *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
		return &authpb.ListUsersResponse{Data: []*authpb.UserListItem{{Id: 1, Code: "other"}}}, nil
	}
	_, err = c.GetUserByCode(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found")
	}

	stub.listUsersFunc = func(context.Context, *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
		return nil, errors.New("list failed")
	}
	_, err = c.GetUserByCode(context.Background(), "abc")
	if err == nil {
		t.Fatal("expected list error")
	}
}

func TestAuthClient_GetUserProfile_SuccessAndError(t *testing.T) {
	stub := &stubUserService{
		getUserProfileFunc: func(_ context.Context, req *authpb.GetUserProfileRequest) (*authpb.GetUserProfileResponse, error) {
			if req.UserId != 5 {
				t.Fatalf("user=%d", req.UserId)
			}
			return &authpb.GetUserProfileResponse{}, nil
		},
	}
	c, cleanup := newAuthClient(t, stub)
	defer cleanup()

	if _, err := c.GetUserProfile(context.Background(), 5); err != nil {
		t.Fatal(err)
	}

	stub.getUserProfileFunc = func(context.Context, *authpb.GetUserProfileRequest) (*authpb.GetUserProfileResponse, error) {
		return nil, errors.New("profile failed")
	}
	_, err := c.GetUserProfile(context.Background(), 5)
	if err == nil {
		t.Fatal("expected profile error")
	}
}
