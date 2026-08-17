package handler_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/handler"
	"metarang/training-service/internal/models"
	"metarang/training-service/internal/repository"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func newReplyClient(t *testing.T, mc *testutil.MockCommentRepo, mu *testutil.MockUserRepo) trainingpb.ReplyServiceClient {
	t.Helper()
	if mc == nil {
		mc = &testutil.MockCommentRepo{}
	}
	if mu == nil {
		mu = &testutil.MockUserRepo{}
	}
	svc := service.NewReplyService(mc, mu)
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterReplyHandler(s, svc)
	})
	t.Cleanup(cleanup)
	return trainingpb.NewReplyServiceClient(conn)
}

func TestReplyHandler_GetReplies_DefaultsAndError(t *testing.T) {
	var gotPage, gotPer int32
	mc := &testutil.MockCommentRepo{
		GetRepliesFunc: func(ctx context.Context, commentID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			gotPage, gotPer = page, perPage
			return nil, 0, context.Canceled
		},
	}
	client := newReplyClient(t, mc, nil)
	_, err := client.GetReplies(context.Background(), &trainingpb.GetRepliesRequest{CommentId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
	if gotPage != 1 || gotPer != 10 {
		t.Fatalf("defaults page=%d per=%d", gotPage, gotPer)
	}
}

func TestReplyHandler_GetReplies_EnrichesUserStatsAndParent(t *testing.T) {
	pid := uint64(8)
	liked := false
	mc := &testutil.MockCommentRepo{
		GetRepliesFunc: func(ctx context.Context, commentID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			return []*models.Comment{{
				ID: 20, UserID: 3, ParentID: &pid, CommentableID: 1,
				Content: "r", CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}}, 1, nil
		},
		GetCommentStatsFunc: func(ctx context.Context, commentID uint64) (*models.CommentStats, error) {
			return &models.CommentStats{LikesCount: 1, RepliesCount: 0}, nil
		},
	}
	mu := &testutil.MockUserRepo{
		GetUserByIDFunc: func(ctx context.Context, userID uint64) (*repository.UserBasic, error) {
			return &repository.UserBasic{ID: userID, Name: "R", Code: "r1", ProfilePhoto: "u.png"}, nil
		},
	}
	client := newReplyClient(t, mc, mu)
	resp, err := client.GetReplies(context.Background(), &trainingpb.GetRepliesRequest{
		CommentId:  8,
		Pagination: &commonpb.PaginationRequest{Page: 1, PerPage: 10},
	})
	if err != nil || len(resp.Replies) != 1 {
		t.Fatal(err, resp)
	}
	r := resp.Replies[0]
	if r.ParentId != 8 || r.User == nil || r.User.ProfilePhoto == "" || r.Stats == nil {
		t.Fatalf("reply=%+v", r)
	}
	_ = liked
}

func TestReplyHandler_AddUpdateDeleteSuccess(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			if commentID == 1 {
				return &models.Comment{ID: 1, UserID: 1, CommentableID: 9}, nil
			}
			return &models.Comment{ID: commentID, UserID: 4, Content: "z"}, nil
		},
		AddReplyFunc: func(ctx context.Context, parentCommentID, userID uint64, content string) (*models.Comment, error) {
			return &models.Comment{ID: 20, UserID: userID, CommentableID: 9, Content: content, CreatedAt: time.Now()}, nil
		},
		UpdateReplyFunc: func(ctx context.Context, replyID, userID uint64, content string) error {
			return nil
		},
		DeleteReplyFunc: func(ctx context.Context, replyID, userID uint64) error {
			return nil
		},
	}
	client := newReplyClient(t, mc, nil)
	resp, err := client.AddReply(context.Background(), &trainingpb.AddReplyRequest{
		ParentCommentId: 1, UserId: 4, Content: "hi",
	})
	if err != nil || resp.Id != 20 {
		t.Fatalf("add err=%v resp=%+v", err, resp)
	}

	resp, err = client.UpdateReply(context.Background(), &trainingpb.UpdateReplyRequest{
		ReplyId: 8, UserId: 4, Content: "z",
	})
	if err != nil || resp.Content != "z" {
		t.Fatalf("update err=%v resp=%+v", err, resp)
	}

	if _, err := client.DeleteReply(context.Background(), &trainingpb.DeleteReplyRequest{ReplyId: 8, UserId: 4}); err != nil {
		t.Fatal(err)
	}
}

func TestReplyHandler_AddReply_PermissionDeniedAndValidation(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 6}, nil
		},
	}
	client := newReplyClient(t, mc, nil)
	_, err := client.AddReply(context.Background(), &trainingpb.AddReplyRequest{
		ParentCommentId: 1, UserId: 6, Content: "nope",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("own reply got %v", err)
	}

	_, err = client.AddReply(context.Background(), &trainingpb.AddReplyRequest{
		ParentCommentId: 1, UserId: 7, Content: strings.Repeat("x", 2001),
	})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("too long got %v", err)
	}
}

func TestReplyHandler_UpdateReply_PermissionDenied(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 1}, nil
		},
	}
	client := newReplyClient(t, mc, nil)
	_, err := client.UpdateReply(context.Background(), &trainingpb.UpdateReplyRequest{
		ReplyId: 8, UserId: 2, Content: "z",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}
}

func TestReplyHandler_DeleteReply_Error(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		DeleteReplyFunc: func(ctx context.Context, replyID, userID uint64) error {
			return context.Canceled
		},
	}
	client := newReplyClient(t, mc, nil)
	_, err := client.DeleteReply(context.Background(), &trainingpb.DeleteReplyRequest{ReplyId: 1, UserId: 2})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestReplyHandler_AddReplyInteraction_OwnReplyAndSuccess(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 3}, nil
		},
		AddReplyInteractionFunc: func(ctx context.Context, replyID, userID uint64, liked bool, ipAddress string) error {
			return nil
		},
	}
	client := newReplyClient(t, mc, nil)
	_, err := client.AddReplyInteraction(context.Background(), &trainingpb.AddReplyInteractionRequest{
		ReplyId: 9, UserId: 3, Liked: true,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("own interaction got %v", err)
	}

	_, err = client.AddReplyInteraction(context.Background(), &trainingpb.AddReplyInteractionRequest{
		ReplyId: 9, UserId: 4, Liked: false, IpAddress: "10.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
}
