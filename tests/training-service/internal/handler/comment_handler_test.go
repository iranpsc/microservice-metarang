package handler_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/handler"
	"metarang/training-service/internal/models"
	"metarang/training-service/internal/repository"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func newCommentClient(t *testing.T, mc *testutil.MockCommentRepo, mu *testutil.MockUserRepo) trainingpb.CommentServiceClient {
	t.Helper()
	if mc == nil {
		mc = &testutil.MockCommentRepo{}
	}
	if mu == nil {
		mu = &testutil.MockUserRepo{}
	}
	svc := service.NewCommentService(mc, mu)
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterCommentHandler(s, svc)
	})
	t.Cleanup(cleanup)
	return trainingpb.NewCommentServiceClient(conn)
}

func TestCommentHandler_GetComments_DefaultsUserFromMetadataAndHeader(t *testing.T) {
	var gotPage, gotPer int32
	parent := uint64(0)
	_ = parent
	pid := uint64(99)
	mc := &testutil.MockCommentRepo{
		GetCommentsFunc: func(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			gotPage, gotPer = page, perPage
			return []*models.Comment{{
				ID: 1, UserID: 2, ParentID: &pid, CommentableID: videoID,
				Content: "hi", CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}}, 1, nil
		},
		GetUserInteractionsForCommentsFunc: func(ctx context.Context, commentIDs []uint64, userID uint64) (map[uint64]bool, error) {
			if userID != 44 {
				t.Fatalf("userID=%d", userID)
			}
			return map[uint64]bool{1: true}, nil
		},
		GetCommentStatsFunc: func(ctx context.Context, commentID uint64) (*models.CommentStats, error) {
			return &models.CommentStats{LikesCount: 2, DislikesCount: 1, RepliesCount: 3}, nil
		},
	}
	mu := &testutil.MockUserRepo{
		GetUserByIDFunc: func(ctx context.Context, userID uint64) (*repository.UserBasic, error) {
			return &repository.UserBasic{ID: userID, Name: "Ann", Code: "a1", ProfilePhoto: "p.jpg"}, nil
		},
	}
	client := newCommentClient(t, mc, mu)
	md := metadata.Pairs("x-user-id", "44")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	var header metadata.MD
	resp, err := client.GetComments(ctx, &trainingpb.GetCommentsRequest{VideoId: 10}, grpc.Header(&header))
	if err != nil {
		t.Fatal(err)
	}
	if gotPage != 1 || gotPer != 10 {
		t.Fatalf("defaults page=%d per=%d", gotPage, gotPer)
	}
	if len(resp.Comments) != 1 || resp.Comments[0].ParentId != 99 || resp.Comments[0].User.Name != "Ann" {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.Comments[0].UserInteraction == nil || !*resp.Comments[0].UserInteraction {
		t.Fatalf("interaction=%v", resp.Comments[0].UserInteraction)
	}
	if len(header.Get("x-comment-user-interactions")) != 1 {
		t.Fatalf("header=%v", header)
	}
}

func TestCommentHandler_GetComments_InternalError(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentsFunc: func(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			return nil, 0, context.Canceled
		},
	}
	client := newCommentClient(t, mc, nil)
	_, err := client.GetComments(context.Background(), &trainingpb.GetCommentsRequest{
		VideoId:    1,
		Pagination: &commonpb.PaginationRequest{Page: 2, PerPage: 5},
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestCommentHandler_AddComment_SuccessAndValidation(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		AddCommentFunc: func(ctx context.Context, videoID, userID uint64, content string) (*models.Comment, error) {
			return &models.Comment{ID: 5, UserID: userID, CommentableID: videoID, Content: content, CreatedAt: time.Now()}, nil
		},
	}
	client := newCommentClient(t, mc, nil)
	resp, err := client.AddComment(context.Background(), &trainingpb.AddCommentRequest{
		VideoId: 1, UserId: 2, Content: "hello world",
	})
	if err != nil || resp.Id != 5 || resp.Content != "hello world" {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}

	_, err = client.AddComment(context.Background(), &trainingpb.AddCommentRequest{
		VideoId: 1, UserId: 2, Content: "",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("empty content got %v", err)
	}
}

func TestCommentHandler_UpdateComment_PermissionDenied(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 1}, nil
		},
	}
	client := newCommentClient(t, mc, nil)
	_, err := client.UpdateComment(context.Background(), &trainingpb.UpdateCommentRequest{
		CommentId: 1, UserId: 99, Content: "nope",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}
}

func TestCommentHandler_DeleteComment_Error(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		DeleteCommentFunc: func(ctx context.Context, commentID, userID uint64) error {
			return context.Canceled
		},
	}
	client := newCommentClient(t, mc, nil)
	_, err := client.DeleteComment(context.Background(), &trainingpb.DeleteCommentRequest{CommentId: 1, UserId: 2})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestCommentHandler_AddCommentInteraction_OwnCommentAndEmptyIP(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 7}, nil
		},
	}
	client := newCommentClient(t, mc, nil)
	_, err := client.AddCommentInteraction(context.Background(), &trainingpb.AddCommentInteractionRequest{
		CommentId: 3, UserId: 7, Liked: true,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}
}

func TestCommentHandler_ReportComment_NotFoundNilAndTooLong(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			if commentID == 1 {
				return nil, context.Canceled
			}
			if commentID == 2 {
				return nil, nil
			}
			return &models.Comment{ID: commentID, UserID: 5, CommentableID: 10}, nil
		},
	}
	client := newCommentClient(t, mc, nil)

	_, err := client.ReportComment(context.Background(), &trainingpb.ReportCommentRequest{CommentId: 1, UserId: 9, Content: "x"})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("repo error got %v", err)
	}

	_, err = client.ReportComment(context.Background(), &trainingpb.ReportCommentRequest{CommentId: 2, UserId: 9, Content: "x"})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("nil comment got %v", err)
	}

	_, err = client.ReportComment(context.Background(), &trainingpb.ReportCommentRequest{
		CommentId: 3, UserId: 9, Content: strings.Repeat("a", 2001),
	})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("too long got %v", err)
	}
}

func TestCommentHandler_ReportComment_OwnComment(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 9, CommentableID: 10}, nil
		},
	}
	client := newCommentClient(t, mc, nil)
	_, err := client.ReportComment(context.Background(), &trainingpb.ReportCommentRequest{
		CommentId: 3, UserId: 9, Content: "spam",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}
}
