package handler_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metarang/shared/pb/support"

	"metarang/support-service/internal/handler"
	"metarang/support-service/internal/models"
	"metarang/support-service/internal/service"
	"metarang/support-service/tests/internal/testutil"
)

func noteClient(t *testing.T, repo *testutil.MockNoteRepo) (pb.NoteServiceClient, func()) {
	t.Helper()
	svc := service.NewNoteService(repo)
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterNoteHandler(s, svc)
	})
	return pb.NewNoteServiceClient(conn), cleanup
}

func TestNoteHandler_CreateNote_TooManyAttachmentsAndInternal(t *testing.T) {
	client, cleanup := noteClient(t, &testutil.MockNoteRepo{})
	defer cleanup()
	atts := make([]string, 6)
	for i := range atts {
		atts[i] = "u"
	}
	_, err := client.CreateNote(context.Background(), &pb.CreateNoteRequest{
		UserId: 1, Title: "t", Content: "c", Attachments: atts,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}

	repo := &testutil.MockNoteRepo{
		CreateFunc: func(ctx context.Context, note *models.Note) (*models.Note, error) {
			return nil, errString("db")
		},
	}
	client, cleanup = noteClient(t, repo)
	defer cleanup()
	_, err = client.CreateNote(context.Background(), &pb.CreateNoteRequest{UserId: 1, Title: "t", Content: "c"})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestNoteHandler_GetNotes_Internal(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64) ([]*models.Note, error) {
			return nil, errString("list")
		},
	}
	client, cleanup := noteClient(t, repo)
	defer cleanup()
	_, err := client.GetNotes(context.Background(), &pb.GetNotesRequest{UserId: 3})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestNoteHandler_GetNote_SuccessAndNotFound(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return &models.Note{
				ID: noteID, UserID: 2, Title: "x", Content: "y",
				UpdatedAt: time.Date(2024, 3, 15, 10, 11, 12, 0, time.UTC),
			}, nil
		},
	}
	client, cleanup := noteClient(t, repo)
	defer cleanup()
	resp, err := client.GetNote(context.Background(), &pb.GetNoteRequest{NoteId: 4, UserId: 2})
	if err != nil || resp.Title != "x" || resp.Date == "" || resp.Time != "10:11:12" {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}

	repo = &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return nil, nil
		},
	}
	client, cleanup = noteClient(t, repo)
	defer cleanup()
	_, err = client.GetNote(context.Background(), &pb.GetNoteRequest{NoteId: 4, UserId: 2})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("got %v", err)
	}
}

func TestNoteHandler_DeleteNote_Unauthorized(t *testing.T) {
	repo := &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return false, nil
		},
	}
	client, cleanup := noteClient(t, repo)
	defer cleanup()
	_, err := client.DeleteNote(context.Background(), &pb.DeleteNoteRequest{NoteId: 3, UserId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}
}

func TestNoteHandler_UpdateNote_ValidationAndNotFound(t *testing.T) {
	client, cleanup := noteClient(t, &testutil.MockNoteRepo{})
	defer cleanup()
	_, err := client.UpdateNote(context.Background(), &pb.UpdateNoteRequest{
		NoteId: 1, UserId: 1, Title: strings.Repeat("t", 131), Content: "c",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}

	repo := &testutil.MockNoteRepo{
		CheckUserOwnershipFunc: func(ctx context.Context, noteID, userID uint64) (bool, error) {
			return true, nil
		},
		GetByIDFunc: func(ctx context.Context, noteID uint64) (*models.Note, error) {
			return nil, nil
		},
	}
	client, cleanup = noteClient(t, repo)
	defer cleanup()
	_, err = client.UpdateNote(context.Background(), &pb.UpdateNoteRequest{
		NoteId: 2, UserId: 1, Title: "new", Content: "nc",
	})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("got %v", err)
	}
}
