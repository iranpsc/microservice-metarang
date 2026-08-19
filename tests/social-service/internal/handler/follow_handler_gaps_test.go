package handler_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metarang/shared/pb/social"
	"metarang/social-service/internal/handler"
	"metarang/social-service/internal/models"
	"metarang/social-service/internal/testutil"
)

func TestFollowHandler_GetFollowing_Error(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterFollowHandler(gs, &stubFollowSvc{
			getFollowing: func(context.Context, uint64) ([]*models.FollowResource, error) {
				return nil, errors.New("db")
			},
		})
	})
	defer cleanup()
	cli := pb.NewFollowServiceClient(conn)
	_, err := cli.GetFollowing(context.Background(), &pb.GetFollowingRequest{UserId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestFollowHandler_Unfollow_ServiceError(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterFollowHandler(gs, &stubFollowSvc{
			unfollow: func(context.Context, uint64, uint64) error {
				return errors.New("delete failed")
			},
		})
	})
	defer cleanup()
	cli := pb.NewFollowServiceClient(conn)
	_, err := cli.Unfollow(context.Background(), &pb.UnfollowRequest{UserId: 1, TargetUserId: 2})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestFollowHandler_Remove_ServiceError(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterFollowHandler(gs, &stubFollowSvc{
			remove: func(context.Context, uint64, uint64) error {
				return errors.New("remove failed")
			},
		})
	})
	defer cleanup()
	cli := pb.NewFollowServiceClient(conn)
	_, err := cli.Remove(context.Background(), &pb.RemoveRequest{UserId: 1, TargetUserId: 2})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestFollowHandler_Follow_OK(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterFollowHandler(gs, &stubFollowSvc{})
	})
	defer cleanup()
	cli := pb.NewFollowServiceClient(conn)
	_, err := cli.Follow(context.Background(), &pb.FollowRequest{UserId: 1, TargetUserId: 2})
	if err != nil {
		t.Fatal(err)
	}
}
