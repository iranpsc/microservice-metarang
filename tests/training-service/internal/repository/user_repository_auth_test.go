package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "metarang/shared/pb/auth"
	"metarang/training-service/internal/client"
	"metarang/training-service/internal/repository"
	"metarang/training-service/tests/internal/testutil"
)

type stubUserService struct {
	authpb.UnimplementedUserServiceServer
	getUserFunc   func(context.Context, *authpb.GetUserRequest) (*authpb.User, error)
	listUsersFunc func(context.Context, *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error)
}

func (s *stubUserService) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.User, error) {
	if s.getUserFunc != nil {
		return s.getUserFunc(ctx, req)
	}
	return nil, status.Error(codes.NotFound, "missing")
}

func (s *stubUserService) ListUsers(ctx context.Context, req *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
	if s.listUsersFunc != nil {
		return s.listUsersFunc(ctx, req)
	}
	return &authpb.ListUsersResponse{}, nil
}

func newUserRepoWithAuth(t *testing.T, db *sql.DB, stub *stubUserService) *repository.UserRepository {
	t.Helper()
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		authpb.RegisterUserServiceServer(s, stub)
	})
	t.Cleanup(cleanup)
	return repository.NewUserRepository(db, client.NewAuthClientFromConn(conn))
}

func TestUserRepository_GetUserByID_AuthClientSuccess(t *testing.T) {
	db, mock := newSQLMock(t)
	stub := &stubUserService{
		getUserFunc: func(_ context.Context, req *authpb.GetUserRequest) (*authpb.User, error) {
			if req.UserId != 4 {
				t.Fatalf("user=%d", req.UserId)
			}
			return &authpb.User{Id: 4, Name: "Ada", Code: "a1", Email: "ada@example.com"}, nil
		},
	}
	mock.ExpectQuery("SELECT url").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("photo.jpg"))

	r := newUserRepoWithAuth(t, db, stub)
	u, err := r.GetUserByID(context.Background(), 4)
	if err != nil || u == nil || u.Name != "Ada" || u.ProfilePhoto != "photo.jpg" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
}

func TestUserRepository_GetUserByID_AuthClientFallbackToDB(t *testing.T) {
	db, mock := newSQLMock(t)
	stub := &stubUserService{
		getUserFunc: func(context.Context, *authpb.GetUserRequest) (*authpb.User, error) {
			return nil, status.Error(codes.Internal, "auth down")
		},
	}
	mock.ExpectQuery("SELECT id, name, code, email").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "email"}).
			AddRow(uint64(4), "Ada", "a1", "ada@example.com"))
	mock.ExpectQuery("SELECT url").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("photo.jpg"))

	r := newUserRepoWithAuth(t, db, stub)
	u, err := r.GetUserByID(context.Background(), 4)
	if err != nil || u == nil || u.Name != "Ada" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
}

func TestUserRepository_GetUserBasicByCode_AuthClientSuccess(t *testing.T) {
	db, mock := newSQLMock(t)
	stub := &stubUserService{
		listUsersFunc: func(_ context.Context, req *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
			return &authpb.ListUsersResponse{
				Data: []*authpb.UserListItem{{Id: 8, Code: "c1"}},
			}, nil
		},
		getUserFunc: func(_ context.Context, req *authpb.GetUserRequest) (*authpb.User, error) {
			return &authpb.User{Id: 8, Name: "Bob", Code: "c1", Email: "b@example.com"}, nil
		},
	}
	mock.ExpectQuery("SELECT url").WithArgs(uint64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("b.png"))

	r := newUserRepoWithAuth(t, db, stub)
	u, err := r.GetUserBasicByCode(context.Background(), "c1")
	if err != nil || u == nil || u.Code != "c1" || u.ProfilePhoto != "b.png" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
}

func TestUserRepository_GetUserBasicByCode_AuthClientFallbackToDB(t *testing.T) {
	db, mock := newSQLMock(t)
	stub := &stubUserService{
		listUsersFunc: func(context.Context, *authpb.ListUsersRequest) (*authpb.ListUsersResponse, error) {
			return nil, status.Error(codes.Internal, "auth down")
		},
	}
	mock.ExpectQuery("SELECT id, name, code, email").WithArgs("c1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "email"}).
			AddRow(uint64(8), "Bob", "c1", "b@example.com"))
	mock.ExpectQuery("SELECT url").WithArgs(uint64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("b.png"))

	r := newUserRepoWithAuth(t, db, stub)
	u, err := r.GetUserBasicByCode(context.Background(), "c1")
	if err != nil || u == nil || u.Name != "Bob" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
}
