package handler

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

// mockProfileLimitationService is a mock implementation for testing
type mockProfileLimitationService struct {
	createFunc          func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error)
	updateFunc          func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error)
	deleteFunc          func(ctx context.Context, limitationID, limiterUserID uint64) error
	getByIDFunc         func(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error)
	getBetweenUsersFunc func(ctx context.Context, callerUserID, targetUserID uint64) (*models.ProfileLimitation, error)
	validateOptionsFunc func(options models.ProfileLimitationOptions) error
}

func (m *mockProfileLimitationService) Create(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, limiterUserID, limitedUserID, options, note)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProfileLimitationService) Update(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
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

func (m *mockProfileLimitationService) ValidateOptions(options models.ProfileLimitationOptions) error {
	if m.validateOptionsFunc != nil {
		return m.validateOptionsFunc(options)
	}
	return nil
}

func TestProfileLimitationHandler_CreateProfileLimitation(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.createFunc = func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
			return &models.ProfileLimitation{
				ID:            1,
				LimiterUserID: limiterUserID,
				LimitedUserID: limitedUserID,
				Options:       options,
				Note:          sql.NullString{String: note, Valid: note != ""},
			}, nil
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.CreateProfileLimitationRequest{
			LimiterUserId: 1,
			LimitedUserId: 2,
			Options: &pb.ProfileLimitationOptions{
				Follow:                false,
				SendMessage:           false,
				Share:                 true,
				SendTicket:            true,
				ViewProfileImages:     false,
				ViewFeaturesLocations: true,
			},
			Note: "Test note",
		}

		resp, err := handler.CreateProfileLimitation(ctx, req)
		if err != nil {
			t.Fatalf("CreateProfileLimitation failed: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Expected data to be returned")
		}
		if resp.Data.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Data.Id)
		}
		if resp.Data.Note != "Test note" {
			t.Errorf("Expected note 'Test note', got '%s'", resp.Data.Note)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.createFunc = func(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
			return nil, service.ErrProfileLimitationAlreadyExists
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.CreateProfileLimitationRequest{
			LimiterUserId: 1,
			LimitedUserId: 2,
			Options: &pb.ProfileLimitationOptions{
				Follow:                true,
				SendMessage:           true,
				Share:                 true,
				SendTicket:            true,
				ViewProfileImages:     true,
				ViewFeaturesLocations: true,
			},
		}

		_, err := handler.CreateProfileLimitation(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.FailedPrecondition {
			t.Errorf("Expected FailedPrecondition, got %v", st.Code())
		}
	})
}

func TestProfileLimitationHandler_UpdateProfileLimitation(t *testing.T) {
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.updateFunc = func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
			return &models.ProfileLimitation{
				ID:            limitationID,
				LimiterUserID: limiterUserID,
				LimitedUserID: 2,
				Options:       options,
				Note:          sql.NullString{String: note, Valid: note != ""},
			}, nil
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.UpdateProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 1,
			Options: &pb.ProfileLimitationOptions{
				Follow:                false,
				SendMessage:           false,
				Share:                 true,
				SendTicket:            true,
				ViewProfileImages:     false,
				ViewFeaturesLocations: true,
			},
			Note: "Updated note",
		}

		resp, err := handler.UpdateProfileLimitation(ctx, req)
		if err != nil {
			t.Fatalf("UpdateProfileLimitation failed: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Expected data to be returned")
		}
		if resp.Data.Note != "Updated note" {
			t.Errorf("Expected note 'Updated note', got '%s'", resp.Data.Note)
		}
	})

	t.Run("unauthorized update", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.updateFunc = func(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
			return nil, service.ErrUnauthorized
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.UpdateProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 2, // Not the owner
			Options: &pb.ProfileLimitationOptions{
				Follow:                true,
				SendMessage:           true,
				Share:                 true,
				SendTicket:            true,
				ViewProfileImages:     true,
				ViewFeaturesLocations: true,
			},
		}

		_, err := handler.UpdateProfileLimitation(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.PermissionDenied {
			t.Errorf("Expected PermissionDenied, got %v", st.Code())
		}
	})
}

func TestProfileLimitationHandler_DeleteProfileLimitation(t *testing.T) {
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.deleteFunc = func(ctx context.Context, limitationID, limiterUserID uint64) error {
			return nil
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.DeleteProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 1,
		}

		_, err := handler.DeleteProfileLimitation(ctx, req)
		if err != nil {
			t.Fatalf("DeleteProfileLimitation failed: %v", err)
		}
	})

	t.Run("unauthorized delete", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.deleteFunc = func(ctx context.Context, limitationID, limiterUserID uint64) error {
			return service.ErrUnauthorized
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.DeleteProfileLimitationRequest{
			LimitationId:  1,
			LimiterUserId: 2, // Not the owner
		}

		_, err := handler.DeleteProfileLimitation(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.PermissionDenied {
			t.Errorf("Expected PermissionDenied, got %v", st.Code())
		}
	})
}

func TestProfileLimitationHandler_GetProfileLimitation(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.getByIDFunc = func(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error) {
			return &models.ProfileLimitation{
				ID:            limitationID,
				LimiterUserID: 1,
				LimitedUserID: 2,
				Options:       models.DefaultOptions(),
				Note:          sql.NullString{String: "Test note", Valid: true},
			}, nil
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.GetProfileLimitationRequest{
			LimitationId: 1,
		}

		resp, err := handler.GetProfileLimitation(ctx, req)
		if err != nil {
			t.Fatalf("GetProfileLimitation failed: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Expected data to be returned")
		}
		if resp.Data.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Data.Id)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockService := &mockProfileLimitationService{}
		mockService.getByIDFunc = func(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error) {
			return nil, service.ErrProfileLimitationNotFound
		}

		handler := &profileLimitationHandler{
			limitationService: mockService,
		}

		req := &pb.GetProfileLimitationRequest{
			LimitationId: 99999,
		}

		_, err := handler.GetProfileLimitation(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound, got %v", st.Code())
		}
	})
}

func TestMapProfileLimitationError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "not found",
			err:          service.ErrProfileLimitationNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "already exists",
			err:          service.ErrProfileLimitationAlreadyExists,
			expectedCode: codes.FailedPrecondition,
		},
		{
			name:         "invalid options",
			err:          service.ErrInvalidOptions,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "note too long",
			err:          service.ErrNoteTooLong,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "user not found",
			err:          service.ErrUserNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "unauthorized",
			err:          service.ErrUnauthorized,
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "internal error",
			err:          errors.New("some internal error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapProfileLimitationError(tt.err)
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
