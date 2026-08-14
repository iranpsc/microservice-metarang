package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"metarang/training-service/internal/models"
	"metarang/training-service/internal/repository"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func TestCommentService_GetComments_WithUserInteractions(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentsFunc: func(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			return []*models.Comment{
				{ID: 1, UserID: 2, CommentableID: videoID, Content: "a", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: 2, UserID: 3, CommentableID: videoID, Content: "b", CreatedAt: time.Now()},
			}, 2, nil
		},
		GetUserInteractionsForCommentsFunc: func(ctx context.Context, commentIDs []uint64, userID uint64) (map[uint64]bool, error) {
			if userID != 9 || len(commentIDs) != 2 {
				t.Fatalf("ids=%v user=%d", commentIDs, userID)
			}
			return map[uint64]bool{1: true}, nil
		},
		GetCommentStatsFunc: func(ctx context.Context, commentID uint64) (*models.CommentStats, error) {
			return &models.CommentStats{LikesCount: 1}, nil
		},
	}
	mu := &testutil.MockUserRepo{
		GetUserByIDFunc: func(ctx context.Context, userID uint64) (*repository.UserBasic, error) {
			return &repository.UserBasic{ID: userID, Name: "U"}, nil
		},
	}
	svc := service.NewCommentService(mc, mu)
	list, total, err := svc.GetComments(context.Background(), 10, 1, 10, 9)
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("list=%d total=%d err=%v", len(list), total, err)
	}
	if list[0].UserInteraction == nil || !*list[0].UserInteraction {
		t.Fatalf("expected liked on first comment")
	}
	if list[1].UserInteraction != nil {
		t.Fatalf("second should have no interaction")
	}
}

func TestCommentService_GetComments_RepoErrorAndInteractionErrorIgnored(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentsFunc: func(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			return nil, 0, errors.New("db")
		},
	}
	svc := service.NewCommentService(mc, &testutil.MockUserRepo{})
	_, _, err := svc.GetComments(context.Background(), 1, 1, 10, 0)
	if err == nil {
		t.Fatal("expected error")
	}

	mc.GetCommentsFunc = func(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
		return []*models.Comment{{ID: 1, UserID: 2, CreatedAt: time.Now()}}, 1, nil
	}
	mc.GetUserInteractionsForCommentsFunc = func(ctx context.Context, commentIDs []uint64, userID uint64) (map[uint64]bool, error) {
		return nil, errors.New("batch")
	}
	list, total, err := svc.GetComments(context.Background(), 1, 1, 10, 5)
	if err != nil || total != 1 || len(list) != 1 || list[0].UserInteraction != nil {
		t.Fatalf("interaction errors should be ignored: %+v err=%v", list, err)
	}
}

func TestCommentService_GetCommentByID_NotFoundAndRepoError(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc := service.NewCommentService(mc, &testutil.MockUserRepo{})
	_, err := svc.GetCommentByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected not found")
	}
	mc.GetCommentByIDFunc = func(ctx context.Context, commentID uint64) (*models.Comment, error) {
		return nil, context.Canceled
	}
	_, err = svc.GetCommentByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestCommentService_UpdateComment_ValidationNotFoundAndUpdateError(t *testing.T) {
	svc := service.NewCommentService(&testutil.MockCommentRepo{}, &testutil.MockUserRepo{})
	_, err := svc.UpdateComment(context.Background(), 1, 1, "")
	if err == nil {
		t.Fatal("empty content")
	}
	_, err = svc.UpdateComment(context.Background(), 1, 1, strings.Repeat("a", 2001))
	if err == nil {
		t.Fatal("too long")
	}

	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc = service.NewCommentService(mc, &testutil.MockUserRepo{})
	_, err = svc.UpdateComment(context.Background(), 1, 1, "ok")
	if err == nil {
		t.Fatal("not found")
	}

	mc.GetCommentByIDFunc = func(ctx context.Context, commentID uint64) (*models.Comment, error) {
		return &models.Comment{ID: 1, UserID: 1}, nil
	}
	mc.UpdateCommentFunc = func(ctx context.Context, commentID, userID uint64, content string) error {
		return errors.New("update failed")
	}
	_, err = svc.UpdateComment(context.Background(), 1, 1, "ok")
	if err == nil {
		t.Fatal("expected update error")
	}
}

func TestCommentService_ReportComment_ValidationAndNotFound(t *testing.T) {
	svc := service.NewCommentService(&testutil.MockCommentRepo{}, &testutil.MockUserRepo{})
	if err := svc.ReportComment(context.Background(), 1, 1, 2, ""); err == nil {
		t.Fatal("empty")
	}
	if err := svc.ReportComment(context.Background(), 1, 1, 2, strings.Repeat("a", 2001)); err == nil {
		t.Fatal("too long")
	}

	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc = service.NewCommentService(mc, &testutil.MockUserRepo{})
	if err := svc.ReportComment(context.Background(), 1, 1, 2, "spam"); err == nil {
		t.Fatal("not found")
	}
}

func TestCommentService_AddCommentInteraction_NotFound(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc := service.NewCommentService(mc, &testutil.MockUserRepo{})
	if err := svc.AddCommentInteraction(context.Background(), 1, 2, true, "ip"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestCommentService_AddComment_RepoError(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		AddCommentFunc: func(ctx context.Context, videoID, userID uint64, content string) (*models.Comment, error) {
			return nil, errors.New("db")
		},
	}
	svc := service.NewCommentService(mc, &testutil.MockUserRepo{})
	_, err := svc.AddComment(context.Background(), 1, 2, "hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommentService_IsNotAuthorized(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: 1, UserID: 5}, nil
		},
	}
	svc := service.NewCommentService(mc, &testutil.MockUserRepo{})
	err := svc.AddCommentInteraction(context.Background(), 1, 5, true, "ip")
	if !errors.Is(err, service.ErrNotAuthorized) {
		t.Fatalf("got %v", err)
	}
}
