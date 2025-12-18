package handler

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

// mockProfilePhotoService is a mock implementation for testing
type mockProfilePhotoService struct {
	listPhotosFunc  func(ctx context.Context, userID uint64) ([]*models.Image, error)
	uploadPhotoFunc func(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error)
	getPhotoFunc    func(ctx context.Context, id uint64) (*models.Image, error)
	deletePhotoFunc func(ctx context.Context, userID uint64, id uint64) error
}

func (m *mockProfilePhotoService) ListProfilePhotos(ctx context.Context, userID uint64) ([]*models.Image, error) {
	if m.listPhotosFunc != nil {
		return m.listPhotosFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockProfilePhotoService) UploadProfilePhoto(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
	if m.uploadPhotoFunc != nil {
		return m.uploadPhotoFunc(ctx, userID, imageData, filename, contentType)
	}
	return nil, nil
}

func (m *mockProfilePhotoService) GetProfilePhoto(ctx context.Context, id uint64) (*models.Image, error) {
	if m.getPhotoFunc != nil {
		return m.getPhotoFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockProfilePhotoService) DeleteProfilePhoto(ctx context.Context, userID uint64, id uint64) error {
	if m.deletePhotoFunc != nil {
		return m.deletePhotoFunc(ctx, userID, id)
	}
	return nil
}

func TestProfilePhotoHandler_ListProfilePhotos(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return []*models.Image{
				{ID: 1, URL: "https://example.com/photo1.jpg"},
				{ID: 2, URL: "https://example.com/photo2.jpg"},
			}, nil
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.ListProfilePhotosRequest{UserId: 1}
		resp, err := handler.ListProfilePhotos(ctx, req)
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		if len(resp.Data) != 2 {
			t.Errorf("Expected 2 photos, got %d", len(resp.Data))
		}
		if resp.Data[0].Id != 1 {
			t.Errorf("Expected first photo ID 1, got %d", resp.Data[0].Id)
		}
	})

	t.Run("missing user_id", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.ListProfilePhotosRequest{UserId: 0}
		_, err := handler.ListProfilePhotos(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing user_id")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", err)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return nil, errors.New("database error")
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.ListProfilePhotosRequest{UserId: 1}
		_, err := handler.ListProfilePhotos(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Internal {
			t.Errorf("Expected Internal error, got %v", err)
		}
	})
}

func TestProfilePhotoHandler_UploadProfilePhoto(t *testing.T) {
	ctx := context.Background()

	t.Run("successful upload", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.uploadPhotoFunc = func(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
			return &models.Image{
				ID:  1,
				URL: "https://example.com/photo.jpg",
			}, nil
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.UploadProfilePhotoRequest{
			UserId:      1,
			ImageData:   []byte{1, 2, 3},
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		}

		resp, err := handler.UploadProfilePhoto(ctx, req)
		if err != nil {
			t.Fatalf("UploadProfilePhoto failed: %v", err)
		}

		if resp.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Id)
		}
		if resp.Url == "" {
			t.Error("Expected URL to be set")
		}
	})

	t.Run("missing user_id", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.UploadProfilePhotoRequest{
			ImageData:   []byte{1, 2, 3},
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		}

		_, err := handler.UploadProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing user_id")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", err)
		}
	})

	t.Run("missing image_data", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.UploadProfilePhotoRequest{
			UserId:      1,
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		}

		_, err := handler.UploadProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing image_data")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", err)
		}
	})

	t.Run("invalid image error", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.uploadPhotoFunc = func(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
			return nil, service.ErrInvalidImage
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.UploadProfilePhotoRequest{
			UserId:      1,
			ImageData:   []byte{1, 2, 3},
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		}

		_, err := handler.UploadProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", err)
		}
	})
}

func TestProfilePhotoHandler_GetProfilePhoto(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.getPhotoFunc = func(ctx context.Context, id uint64) (*models.Image, error) {
			return &models.Image{
				ID:  id,
				URL: "https://example.com/photo.jpg",
			}, nil
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.GetProfilePhotoRequest{ProfilePhotoId: 1}
		resp, err := handler.GetProfilePhoto(ctx, req)
		if err != nil {
			t.Fatalf("GetProfilePhoto failed: %v", err)
		}

		if resp.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Id)
		}
	})

	t.Run("missing profile_photo_id", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.GetProfilePhotoRequest{ProfilePhotoId: 0}
		_, err := handler.GetProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing profile_photo_id")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", err)
		}
	})

	t.Run("photo not found", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.getPhotoFunc = func(ctx context.Context, id uint64) (*models.Image, error) {
			return nil, service.ErrProfilePhotoNotFound
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.GetProfilePhotoRequest{ProfilePhotoId: 999}
		_, err := handler.GetProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound, got %v", err)
		}
	})
}

func TestProfilePhotoHandler_DeleteProfilePhoto(t *testing.T) {
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.deletePhotoFunc = func(ctx context.Context, userID uint64, id uint64) error {
			return nil
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.DeleteProfilePhotoRequest{
			UserId:         1,
			ProfilePhotoId: 1,
		}

		_, err := handler.DeleteProfilePhoto(ctx, req)
		if err != nil {
			t.Fatalf("DeleteProfilePhoto failed: %v", err)
		}
	})

	t.Run("missing user_id", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.DeleteProfilePhotoRequest{
			ProfilePhotoId: 1,
		}

		_, err := handler.DeleteProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error for missing user_id")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", err)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.deletePhotoFunc = func(ctx context.Context, userID uint64, id uint64) error {
			return service.ErrUnauthorized
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.DeleteProfilePhotoRequest{
			UserId:         1,
			ProfilePhotoId: 999,
		}

		_, err := handler.DeleteProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.PermissionDenied {
			t.Errorf("Expected PermissionDenied, got %v", err)
		}
	})

	t.Run("photo not found", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.deletePhotoFunc = func(ctx context.Context, userID uint64, id uint64) error {
			return service.ErrProfilePhotoNotFound
		}

		handler := &profilePhotoHandler{profilePhotoService: mockService}

		req := &pb.DeleteProfilePhotoRequest{
			UserId:         1,
			ProfilePhotoId: 999,
		}

		_, err := handler.DeleteProfilePhoto(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound, got %v", err)
		}
	})
}
