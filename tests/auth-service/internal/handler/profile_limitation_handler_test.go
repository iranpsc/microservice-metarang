package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

type mockProfileLimitationService struct {
	createFunc          func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error)
	updateFunc          func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error)
	deleteFunc          func(ctx context.Context, limitationID, limiterUserID uint64) error
	getByIDFunc         func(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error)
	getBetweenUsersFunc func(ctx context.Context, callerUserID, targetUserID uint64) (*models.ProfileLimitation, error)
}

func (m *mockProfileLimitationService) Create(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, limiterUserID, limitedUserID, options, note)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProfileLimitationService) Update(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, limitationID, limiterUserID, options, note)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProfileLimitationService) Delete(ctx context.Context, limitationID, limiterUserID uint64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, limitationID, limiterUserID)
	}
	return errors.New("not implemented")
}

func (m *mockProfileLimitationService) GetByID(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, limitationID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProfileLimitationService) GetBetweenUsers(ctx context.Context, callerUserID, targetUserID uint64) (*models.ProfileLimitation, error) {
	if m.getBetweenUsersFunc != nil {
		return m.getBetweenUsersFunc(ctx, callerUserID, targetUserID)
	}
	return nil, errors.New("not implemented")
}

func boolPtr(v bool) *bool { return &v }

func plStringPtr(v string) *string { return &v }

func fullOptions(follow bool) *pb.ProfileLimitationOptions {
	f := follow
	t := true
	return &pb.ProfileLimitationOptions{
		Follow:                &f,
		SendMessage:           &t,
		Share:                 &t,
		SendTicket:            &t,
		ViewProfileImages:     &t,
		ViewFeaturesLocations: &t,
	}
}

func TestProfileLimitationHandler_CreateProfileLimitation(t *testing.T) {
	ctx := authenticatedContext(1)

	t.Run("successful creation", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.createFunc = func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
			n := sql.NullString{}
			if note.Present && note.Value != nil {
				n = sql.NullString{String: *note.Value, Valid: true}
			}
			return &models.ProfileLimitation{
				ID:            1,
				LimiterUserID: limiterUserID,
				LimitedUserID: limitedUserID,
				Options:       options,
				Note:          n,
			}, nil
		}

		h := handler.NewProfileLimitationHandler(mockService)
		note := "Test note"
		req := &pb.CreateProfileLimitationRequest{
			LimiterUserId: 1,
			LimitedUserId: 2,
			Options: &pb.ProfileLimitationOptions{
				Follow:                boolPtr(false),
				SendMessage:           boolPtr(false),
				Share:                 boolPtr(true),
				SendTicket:            boolPtr(true),
				ViewProfileImages:     boolPtr(false),
				ViewFeaturesLocations: boolPtr(true),
			},
			Note: &note,
		}

		resp, err := h.CreateProfileLimitation(ctx, req)
		if err != nil {
			t.Fatalf("CreateProfileLimitation failed: %v", err)
		}
		if resp.Data == nil || resp.Data.Id != 1 {
			t.Fatalf("unexpected response: %+v", resp.Data)
		}
		if resp.Data.Note == nil || *resp.Data.Note != "Test note" {
			t.Errorf("Expected note 'Test note', got %v", resp.Data.Note)
		}
	})

	t.Run("nil request", func(t *testing.T) {
		h := handler.NewProfileLimitationHandler(&mockProfileLimitationService{})
		_, err := h.CreateProfileLimitation(ctx, nil)
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("nil options", func(t *testing.T) {
		h := handler.NewProfileLimitationHandler(&mockProfileLimitationService{})
		_, err := h.CreateProfileLimitation(ctx, &pb.CreateProfileLimitationRequest{
			LimiterUserId: 1,
			LimitedUserId: 2,
		})
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("missing option key", func(t *testing.T) {
		h := handler.NewProfileLimitationHandler(&mockProfileLimitationService{})
		_, err := h.CreateProfileLimitation(ctx, &pb.CreateProfileLimitationRequest{
			LimiterUserId: 1,
			LimitedUserId: 2,
			Options: &pb.ProfileLimitationOptions{
				Follow: boolPtr(true),
			},
		})
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("already exists", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.createFunc = func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
			return nil, service.ErrProfileLimitationAlreadyExists
		}
		h := handler.NewProfileLimitationHandler(mockService)
		_, err := h.CreateProfileLimitation(ctx, &pb.CreateProfileLimitationRequest{
			LimiterUserId: 1,
			LimitedUserId: 2,
			Options:       fullOptions(true),
		})
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.PermissionDenied {
			t.Fatalf("expected PermissionDenied, got %v", err)
		}
	})
}

func TestProfileLimitationHandler_UpdateProfileLimitation(t *testing.T) {
	ctx := authenticatedContext(1)

	t.Run("successful update retains note when omitted", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.updateFunc = func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
			if note.Present {
				t.Fatal("expected note to be omitted")
			}
			return &models.ProfileLimitation{
				ID:            limitationID,
				LimiterUserID: limiterUserID,
				LimitedUserID: 2,
				Options:       options,
				Note:          sql.NullString{String: "kept", Valid: true},
			}, nil
		}

		h := handler.NewProfileLimitationHandler(mockService)
		resp, err := h.UpdateProfileLimitation(ctx, &pb.UpdateProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 1,
			Options:       fullOptions(false),
		})
		if err != nil {
			t.Fatalf("UpdateProfileLimitation failed: %v", err)
		}
		if resp.Data.Note == nil || *resp.Data.Note != "kept" {
			t.Fatalf("expected retained note, got %v", resp.Data.Note)
		}
	})

	t.Run("explicit clear note", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.updateFunc = func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
			if !note.Present || note.Value != nil {
				t.Fatalf("expected clear note update, got %+v", note)
			}
			return &models.ProfileLimitation{
				ID:            limitationID,
				LimiterUserID: limiterUserID,
				LimitedUserID: 2,
				Options:       options,
				Note:          sql.NullString{Valid: false},
			}, nil
		}

		h := handler.NewProfileLimitationHandler(mockService)
		empty := ""
		resp, err := h.UpdateProfileLimitation(ctx, &pb.UpdateProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 1,
			Options:       fullOptions(true),
			Note:          &empty,
		})
		if err != nil {
			t.Fatalf("UpdateProfileLimitation failed: %v", err)
		}
		if resp.Data.Note == nil || *resp.Data.Note != "" {
			t.Fatalf("expected empty note present for limiter, got %v", resp.Data.Note)
		}
	})

	t.Run("unauthorized update", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.updateFunc = func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
			return nil, service.ErrUnauthorized
		}
		h := handler.NewProfileLimitationHandler(mockService)
		_, err := h.UpdateProfileLimitation(ctx, &pb.UpdateProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 2,
			Options:       fullOptions(true),
		})
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.PermissionDenied {
			t.Fatalf("expected PermissionDenied, got %v", err)
		}
	})

	t.Run("nil options", func(t *testing.T) {
		h := handler.NewProfileLimitationHandler(&mockProfileLimitationService{})
		_, err := h.UpdateProfileLimitation(ctx, &pb.UpdateProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 1,
		})
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})
}

func TestProfileLimitationHandler_DeleteProfileLimitation(t *testing.T) {
	ctx := authenticatedContext(1)

	t.Run("successful delete", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.deleteFunc = func(ctx context.Context, limitationID, limiterUserID uint64) error {
			return nil
		}
		h := handler.NewProfileLimitationHandler(mockService)
		_, err := h.DeleteProfileLimitation(ctx, &pb.DeleteProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 1,
		})
		if err != nil {
			t.Fatalf("DeleteProfileLimitation failed: %v", err)
		}
	})

	t.Run("unauthorized delete", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.deleteFunc = func(ctx context.Context, limitationID, limiterUserID uint64) error {
			return service.ErrUnauthorized
		}
		h := handler.NewProfileLimitationHandler(mockService)
		_, err := h.DeleteProfileLimitation(ctx, &pb.DeleteProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 2,
		})
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.PermissionDenied {
			t.Fatalf("expected PermissionDenied, got %v", err)
		}
	})
}

func TestProfileLimitationHandler_NoteVisibility(t *testing.T) {
	ctx := authenticatedContext(10)
	mockService := &mockProfileLimitationService{}
	mockService.createFunc = func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note service.NoteUpdate) (*models.ProfileLimitation, error) {
		return &models.ProfileLimitation{
			ID:            1,
			LimiterUserID: limiterUserID,
			LimitedUserID: limitedUserID,
			Options:       options,
			Note:          sql.NullString{String: "secret", Valid: true},
		}, nil
	}
	h := handler.NewProfileLimitationHandler(mockService)

	// Create response always uses limiter as caller, so note is present.
	resp, err := h.CreateProfileLimitation(ctx, &pb.CreateProfileLimitationRequest{
		LimitedUserId: 20,
		Options:       fullOptions(true),
		Note:          plStringPtr("secret"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.Note == nil || *resp.Data.Note != "secret" {
		t.Fatalf("limiter should see note, got %v", resp.Data.Note)
	}
}

func TestMapProfileLimitationError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{"not found", service.ErrProfileLimitationNotFound, codes.NotFound},
		{"already exists", service.ErrProfileLimitationAlreadyExists, codes.PermissionDenied},
		{"invalid options", service.ErrInvalidOptions, codes.InvalidArgument},
		{"note too long", service.ErrNoteTooLong, codes.InvalidArgument},
		{"user not found", service.ErrUserNotFound, codes.NotFound},
		{"unauthorized", service.ErrUnauthorized, codes.PermissionDenied},
		{"internal error", errors.New("some internal error"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.MapProfileLimitationError(tt.err, "en")
			st, ok := status.FromError(err)
			if !ok {
				t.Fatal("Expected gRPC status error")
			}
			if st.Code() != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
			}
		})
	}
}
