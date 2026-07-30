package handler_test

import (
	"context"
	"errors"
	"testing"

	"metarang/commercial-service/internal/handler"
	pb "metarang/shared/pb/commercial"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubUserVariableService struct {
	err error
}

func (s *stubUserVariableService) CreateUserVariables(ctx context.Context, userID uint64) error {
	return s.err
}

func TestUserVariableHandler_CreateUserVariables(t *testing.T) {
	h := handler.NewUserVariableHandler(&stubUserVariableService{})
	_, err := h.CreateUserVariables(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = h.CreateUserVariables(context.Background(), &pb.CreateUserVariablesRequest{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = h.CreateUserVariables(context.Background(), &pb.CreateUserVariablesRequest{UserId: 1})
	require.NoError(t, err)

	h = handler.NewUserVariableHandler(&stubUserVariableService{err: errors.New("fail")})
	_, err = h.CreateUserVariables(context.Background(), &pb.CreateUserVariablesRequest{UserId: 1})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}
