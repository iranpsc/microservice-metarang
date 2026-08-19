package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"metarang/training-service/internal/models"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func TestReplyService_GetReplies_Success(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetRepliesFunc: func(ctx context.Context, commentID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			return []*models.Comment{{ID: 2, UserID: 3, Content: "r"}}, 1, nil
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	list, total, err := svc.GetReplies(context.Background(), 1, 1, 10)
	if err != nil || total != 1 || len(list) != 1 || list[0].Comment.ID != 2 {
		t.Fatalf("list=%v total=%d err=%v", list, total, err)
	}
}

func TestReplyService_GetReplies_RepoError(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetRepliesFunc: func(ctx context.Context, commentID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
			return nil, 0, errors.New("db")
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	_, _, err := svc.GetReplies(context.Background(), 1, 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReplyService_AddReply_ValidationAndParentMissing(t *testing.T) {
	svc := service.NewReplyService(&testutil.MockCommentRepo{}, &testutil.MockUserRepo{})
	_, err := svc.AddReply(context.Background(), 1, 2, "")
	if err == nil {
		t.Fatal("empty")
	}
	_, err = svc.AddReply(context.Background(), 1, 2, strings.Repeat("a", 2001))
	if err == nil {
		t.Fatal("too long")
	}

	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc = service.NewReplyService(mc, &testutil.MockUserRepo{})
	_, err = svc.AddReply(context.Background(), 1, 2, "hi")
	if err == nil {
		t.Fatal("parent not found")
	}

	mc.GetCommentByIDFunc = func(ctx context.Context, commentID uint64) (*models.Comment, error) {
		return nil, context.Canceled
	}
	_, err = svc.AddReply(context.Background(), 1, 2, "hi")
	if err == nil {
		t.Fatal("parent repo error")
	}
}

func TestReplyService_AddReply_InsertError(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: 1, UserID: 4, CommentableID: 9}, nil
		},
		AddReplyFunc: func(ctx context.Context, parentCommentID, userID uint64, content string) (*models.Comment, error) {
			return nil, errors.New("insert")
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	_, err := svc.AddReply(context.Background(), 1, 9, "reply")
	if err == nil {
		t.Fatal("expected insert error")
	}
}

func TestReplyService_UpdateReply_ValidationNotFoundAndUpdateError(t *testing.T) {
	svc := service.NewReplyService(&testutil.MockCommentRepo{}, &testutil.MockUserRepo{})
	_, err := svc.UpdateReply(context.Background(), 1, 1, "")
	if err == nil {
		t.Fatal("empty")
	}
	_, err = svc.UpdateReply(context.Background(), 1, 1, strings.Repeat("a", 2001))
	if err == nil {
		t.Fatal("too long")
	}

	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc = service.NewReplyService(mc, &testutil.MockUserRepo{})
	_, err = svc.UpdateReply(context.Background(), 1, 1, "ok")
	if err == nil {
		t.Fatal("not found")
	}

	mc.GetCommentByIDFunc = func(ctx context.Context, commentID uint64) (*models.Comment, error) {
		return &models.Comment{ID: 1, UserID: 1}, nil
	}
	mc.UpdateReplyFunc = func(ctx context.Context, replyID, userID uint64, content string) error {
		return errors.New("upd")
	}
	_, err = svc.UpdateReply(context.Background(), 1, 1, "ok")
	if err == nil {
		t.Fatal("update error")
	}
}

func TestReplyService_AddReplyInteraction_NotFound(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return nil, nil
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	if err := svc.AddReplyInteraction(context.Background(), 3, 2, true, "ip"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestReplyService_AddReply_Success(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 1, CommentableID: 9}, nil
		},
		AddReplyFunc: func(ctx context.Context, parentCommentID, userID uint64, content string) (*models.Comment, error) {
			return &models.Comment{ID: 20, UserID: userID, CommentableID: 9, Content: content}, nil
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	d, err := svc.AddReply(context.Background(), 1, 4, "reply")
	if err != nil || d.Comment.ID != 20 {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestReplyService_UpdateReply_SuccessReloadAndUnauthorized(t *testing.T) {
	calls := 0
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			calls++
			if calls == 1 {
				return &models.Comment{ID: commentID, UserID: 2}, nil
			}
			return &models.Comment{ID: commentID, UserID: 2, Content: "upd"}, nil
		},
		UpdateReplyFunc: func(ctx context.Context, replyID, userID uint64, content string) error {
			return nil
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	d, err := svc.UpdateReply(context.Background(), 8, 2, "upd")
	if err != nil || d.Comment.Content != "upd" {
		t.Fatalf("d=%+v err=%v", d, err)
	}

	mc.GetCommentByIDFunc = func(ctx context.Context, commentID uint64) (*models.Comment, error) {
		return &models.Comment{ID: commentID, UserID: 9}, nil
	}
	_, err = svc.UpdateReply(context.Background(), 8, 2, "upd")
	if !errors.Is(err, service.ErrNotAuthorized) {
		t.Fatalf("got %v", err)
	}

	calls = 0
	mc.GetCommentByIDFunc = func(ctx context.Context, commentID uint64) (*models.Comment, error) {
		calls++
		if calls == 1 {
			return &models.Comment{ID: commentID, UserID: 2}, nil
		}
		return nil, context.Canceled
	}
	if _, err := svc.UpdateReply(context.Background(), 8, 2, "upd"); err == nil {
		t.Fatal("expected reload error")
	}
}

func TestReplyService_DeleteAndInteractSuccess(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: commentID, UserID: 1}, nil
		},
		DeleteReplyFunc: func(ctx context.Context, replyID, userID uint64) error {
			return nil
		},
		AddReplyInteractionFunc: func(ctx context.Context, replyID, userID uint64, liked bool, ipAddress string) error {
			return nil
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	if err := svc.DeleteReply(context.Background(), 8, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddReplyInteraction(context.Background(), 8, 2, false, "ip"); err != nil {
		t.Fatal(err)
	}
}

func TestReplyService_AddReply_NotAuthorizedWrap(t *testing.T) {
	mc := &testutil.MockCommentRepo{
		GetCommentByIDFunc: func(ctx context.Context, commentID uint64) (*models.Comment, error) {
			return &models.Comment{ID: 1, UserID: 5}, nil
		},
	}
	svc := service.NewReplyService(mc, &testutil.MockUserRepo{})
	_, err := svc.AddReply(context.Background(), 1, 5, "hi")
	if !errors.Is(err, service.ErrNotAuthorized) {
		t.Fatalf("got %v", err)
	}
}
