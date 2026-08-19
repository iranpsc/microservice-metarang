package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"metarang/support-service/internal/models"
	"metarang/support-service/internal/service"
	"metarang/support-service/tests/internal/testutil"
)

func TestNoteService_CreateNote_CreateAndReloadErrors(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		CreateFunc: func(ctx context.Context, note *models.Note) (*models.Note, error) {
			return nil, errors.New("insert failed")
		},
	}
	svc := service.NewNoteService(repo)
	_, err := svc.CreateNote(context.Background(), 1, "t", "c", nil)
	if err == nil || err.Error() != "insert failed" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockNoteRepo{
		CreateFunc: func(ctx context.Context, note *models.Note) (*models.Note, error) {
			note.ID = 4
			return note, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return nil, errors.New("reload")
		},
	}
	svc = service.NewNoteService(repo)
	_, err = svc.CreateNote(context.Background(), 1, "t", "c", nil)
	if err == nil || err.Error() != "reload" {
		t.Fatalf("err=%v", err)
	}
}

func TestNoteService_GetNotes_RepoError(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64) ([]*models.Note, error) {
			return nil, errors.New("list failed")
		},
	}
	svc := service.NewNoteService(repo)
	_, err := svc.GetNotes(context.Background(), 1)
	if err == nil || err.Error() != "list failed" {
		t.Fatalf("err=%v", err)
	}
}

func TestNoteService_GetNote_OwnershipErrorAndGetByID(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return false, errors.New("own check")
		},
	}
	svc := service.NewNoteService(repo)
	_, err := svc.GetNote(context.Background(), 1, 2)
	if err == nil || !strings.Contains(err.Error(), "failed to check ownership") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return nil, errors.New("missing")
		},
	}
	svc = service.NewNoteService(repo)
	_, err = svc.GetNote(context.Background(), 1, 2)
	if err == nil || err.Error() != "missing" {
		t.Fatalf("err=%v", err)
	}
}

func TestNoteService_UpdateNote_ErrorsNilNoteAndReplaceAttachmentsFalse(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return false, errors.New("own")
		},
	}
	svc := service.NewNoteService(repo)
	_, err := svc.UpdateNote(context.Background(), 1, 2, "a", "b", nil, true)
	if err == nil || !strings.Contains(err.Error(), "failed to check ownership") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return nil, errors.New("db")
		},
	}
	svc = service.NewNoteService(repo)
	_, err = svc.UpdateNote(context.Background(), 1, 2, "a", "b", nil, true)
	if err == nil || !strings.Contains(err.Error(), "failed to get note") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return nil, nil
		},
	}
	svc = service.NewNoteService(repo)
	_, err = svc.UpdateNote(context.Background(), 1, 2, "a", "b", nil, true)
	if err == nil || err.Error() != "note not found" {
		t.Fatalf("err=%v", err)
	}

	note := &models.Note{
		ID: 1, UserID: 2, Title: "old", Content: "oldc",
		Attachments: []string{"keep.png"}, UpdatedAt: time.Now(),
	}
	var saved *models.Note
	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			cp := *note
			return &cp, nil
		},
		UpdateFunc: func(ctx context.Context, n *models.Note) error {
			cp := *n
			saved = &cp
			*note = *n
			return nil
		},
	}
	svc = service.NewNoteService(repo)
	_, err = svc.UpdateNote(context.Background(), 1, 2, "n", "nc", []string{"new.png"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if saved == nil || len(saved.Attachments) != 1 || saved.Attachments[0] != "keep.png" {
		t.Fatalf("replaceAttachments=false must keep existing files: %+v", saved)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return &models.Note{ID: 1, UserID: 2, Title: "old", Content: "oldc"}, nil
		},
		UpdateFunc: func(ctx context.Context, n *models.Note) error {
			return errors.New("write")
		},
	}
	svc = service.NewNoteService(repo)
	_, err = svc.UpdateNote(context.Background(), 1, 2, "n", "nc", []string{"x"}, true)
	if err == nil || !strings.Contains(err.Error(), "failed to update note") {
		t.Fatalf("err=%v", err)
	}
}

func TestNoteService_DeleteNote_OwnershipAndDeleteErrors(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return false, errors.New("own")
		},
	}
	svc := service.NewNoteService(repo)
	err := svc.DeleteNote(context.Background(), 1, 2)
	if err == nil || !strings.Contains(err.Error(), "failed to check ownership") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return false, nil
		},
	}
	svc = service.NewNoteService(repo)
	err = svc.DeleteNote(context.Background(), 1, 2)
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		DeleteFunc: func(ctx context.Context, noteID uint64) error {
			return errors.New("delete failed")
		},
	}
	svc = service.NewNoteService(repo)
	err = svc.DeleteNote(context.Background(), 1, 2)
	if err == nil || err.Error() != "delete failed" {
		t.Fatalf("err=%v", err)
	}
}
