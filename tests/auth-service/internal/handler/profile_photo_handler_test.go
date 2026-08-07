package handler_test

import (
	"context"
	"errors"
	"metarang/auth-service/internal/handler"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

type mockProfilePhotoService struct {
	gatewayURL      string
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

func (m *mockProfilePhotoService) ResolvePhotoURL(url string) string {
	return service.ResolvePublicURL(m.gatewayURL, url)
}

func newProfilePhotoHandler(mock *mockProfilePhotoService) *handler.ProfilePhotoHandler {
	return handler.NewProfilePhotoHandler(mock)
}

func TestProfilePhotoHandler_ListProfilePhotos(t *testing.T) {
	ctx := authenticatedContext(1)

	t.Run("successful list with full URLs - database records from auth-service, files from storage-service", func(t *testing.T) {
		mockService := &mockProfilePhotoService{gatewayURL: "https://api.example.com"}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return []*models.Image{
				{ID: 1, URL: "/uploads/profile/photo1.jpg"},
				{ID: 2, URL: "/uploads/profile/photo2.jpg"},
				{ID: 3, URL: "https://storage.example.com/photo3.jpg"},
			}, nil
		}

		h := newProfilePhotoHandler(mockService)
		req := &pb.ListProfilePhotosRequest{UserId: 1}
		resp, err := h.ListProfilePhotos(ctx, req)
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		if len(resp.Data) != 3 {
			t.Errorf("Expected 3 photos, got %d", len(resp.Data))
		}

		expectedURL1 := "https://api.example.com/uploads/profile/photo1.jpg"
		if resp.Data[0].Url != expectedURL1 {
			t.Errorf("Expected first photo URL %s, got %s", expectedURL1, resp.Data[0].Url)
		}

		expectedURL2 := "https://api.example.com/uploads/profile/photo2.jpg"
		if resp.Data[1].Url != expectedURL2 {
			t.Errorf("Expected second photo URL %s, got %s", expectedURL2, resp.Data[1].Url)
		}

		expectedURL3 := "https://storage.example.com/photo3.jpg"
		if resp.Data[2].Url != expectedURL3 {
			t.Errorf("Expected third photo URL %s, got %s", expectedURL3, resp.Data[2].Url)
		}

		for i, photo := range resp.Data {
			if photo.Id == 0 {
				t.Errorf("Photo at index %d: id is required and must not be zero", i)
			}
			if photo.Url == "" {
				t.Errorf("Photo at index %d: url is required and must not be empty", i)
			}
			if !(strings.HasPrefix(photo.Url, "http://") || strings.HasPrefix(photo.Url, "https://")) {
				t.Errorf("Photo at index %d: url must be a full URL, got %s", i, photo.Url)
			}
		}
	})

	t.Run("successful list with gateway URL having trailing slash", func(t *testing.T) {
		mockService := &mockProfilePhotoService{gatewayURL: "https://api.example.com/"}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return []*models.Image{{ID: 1, URL: "/uploads/profile/photo1.jpg"}}, nil
		}

		h := newProfilePhotoHandler(mockService)
		resp, err := h.ListProfilePhotos(ctx, &pb.ListProfilePhotosRequest{UserId: 1})
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		expectedURL := "https://api.example.com/uploads/profile/photo1.jpg"
		if resp.Data[0].Url != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, resp.Data[0].Url)
		}
	})

	t.Run("successful list with empty gateway URL", func(t *testing.T) {
		mockService := &mockProfilePhotoService{gatewayURL: ""}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return []*models.Image{{ID: 1, URL: "/uploads/profile/photo1.jpg"}}, nil
		}

		h := newProfilePhotoHandler(mockService)
		resp, err := h.ListProfilePhotos(ctx, &pb.ListProfilePhotosRequest{UserId: 1})
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		expectedURL := "/uploads/profile/photo1.jpg"
		if resp.Data[0].Url != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, resp.Data[0].Url)
		}
	})

	t.Run("successful list with empty result", func(t *testing.T) {
		mockService := &mockProfilePhotoService{gatewayURL: "https://api.example.com"}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return []*models.Image{}, nil
		}

		h := newProfilePhotoHandler(mockService)
		resp, err := h.ListProfilePhotos(ctx, &pb.ListProfilePhotosRequest{UserId: 1})
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		if resp.Data == nil {
			t.Error("Expected data array to be initialized, got nil")
		}
		if len(resp.Data) != 0 {
			t.Errorf("Expected 0 photos, got %d", len(resp.Data))
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newProfilePhotoHandler(&mockProfilePhotoService{})
		_, err := h.ListProfilePhotos(context.Background(), &pb.ListProfilePhotosRequest{})
		if err == nil {
			t.Fatal("Expected error for unauthenticated request")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Unauthenticated {
			t.Errorf("Expected Unauthenticated, got %v", err)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.listPhotosFunc = func(ctx context.Context, userID uint64) ([]*models.Image, error) {
			return nil, errors.New("database error")
		}

		h := newProfilePhotoHandler(mockService)
		_, err := h.ListProfilePhotos(ctx, &pb.ListProfilePhotosRequest{UserId: 1})
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
	ctx := authenticatedContext(1)

	t.Run("successful upload", func(t *testing.T) {
		mockService := &mockProfilePhotoService{gatewayURL: "https://api.example.com"}
		mockService.uploadPhotoFunc = func(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
			return &models.Image{ID: 1, URL: "https://api.example.com/uploads/profile/test.jpg"}, nil
		}

		h := newProfilePhotoHandler(mockService)
		resp, err := h.UploadProfilePhoto(ctx, &pb.UploadProfilePhotoRequest{
			UserId:      1,
			ImageData:   []byte{1, 2, 3},
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		})
		if err != nil {
			t.Fatalf("UploadProfilePhoto failed: %v", err)
		}

		if resp.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Id)
		}
		expectedURL := "https://api.example.com/uploads/profile/test.jpg"
		if resp.Url != expectedURL {
			t.Errorf("Expected full URL %s, got %s", expectedURL, resp.Url)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newProfilePhotoHandler(&mockProfilePhotoService{})
		_, err := h.UploadProfilePhoto(context.Background(), &pb.UploadProfilePhotoRequest{
			ImageData:   []byte{1, 2, 3},
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		})
		if err == nil {
			t.Fatal("Expected error for unauthenticated request")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Unauthenticated {
			t.Errorf("Expected Unauthenticated, got %v", err)
		}
	})

	t.Run("missing image_data", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.uploadPhotoFunc = func(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
			return nil, service.ErrImageRequired
		}

		h := newProfilePhotoHandler(mockService)
		_, err := h.UploadProfilePhoto(ctx, &pb.UploadProfilePhotoRequest{
			UserId:      1,
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		})
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

		h := newProfilePhotoHandler(mockService)
		_, err := h.UploadProfilePhoto(ctx, &pb.UploadProfilePhotoRequest{
			UserId:      1,
			ImageData:   []byte{1, 2, 3},
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
		})
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
	ctx := authenticatedContext(1)

	t.Run("successful get with full URL", func(t *testing.T) {
		mockService := &mockProfilePhotoService{gatewayURL: "https://api.example.com"}
		mockService.getPhotoFunc = func(ctx context.Context, id uint64) (*models.Image, error) {
			return &models.Image{ID: id, URL: "/uploads/profile/photo.jpg"}, nil
		}

		h := newProfilePhotoHandler(mockService)
		resp, err := h.GetProfilePhoto(ctx, &pb.GetProfilePhotoRequest{ProfilePhotoId: 1})
		if err != nil {
			t.Fatalf("GetProfilePhoto failed: %v", err)
		}

		expectedURL := "https://api.example.com/uploads/profile/photo.jpg"
		if resp.Url != expectedURL {
			t.Errorf("Expected full URL %s, got %s", expectedURL, resp.Url)
		}
	})

	t.Run("missing profile_photo_id", func(t *testing.T) {
		h := newProfilePhotoHandler(&mockProfilePhotoService{})
		_, err := h.GetProfilePhoto(ctx, &pb.GetProfilePhotoRequest{ProfilePhotoId: 0})
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

		h := newProfilePhotoHandler(mockService)
		_, err := h.GetProfilePhoto(ctx, &pb.GetProfilePhotoRequest{ProfilePhotoId: 999})
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
	ctx := authenticatedContext(1)

	t.Run("successful delete", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.deletePhotoFunc = func(ctx context.Context, userID uint64, id uint64) error {
			return nil
		}

		h := newProfilePhotoHandler(mockService)
		_, err := h.DeleteProfilePhoto(ctx, &pb.DeleteProfilePhotoRequest{UserId: 1, ProfilePhotoId: 1})
		if err != nil {
			t.Fatalf("DeleteProfilePhoto failed: %v", err)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newProfilePhotoHandler(&mockProfilePhotoService{})
		_, err := h.DeleteProfilePhoto(context.Background(), &pb.DeleteProfilePhotoRequest{ProfilePhotoId: 1})
		if err == nil {
			t.Fatal("Expected error for unauthenticated request")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Unauthenticated {
			t.Errorf("Expected Unauthenticated, got %v", err)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := &mockProfilePhotoService{}
		mockService.deletePhotoFunc = func(ctx context.Context, userID uint64, id uint64) error {
			return service.ErrPhotoUnauthorized
		}

		h := newProfilePhotoHandler(mockService)
		_, err := h.DeleteProfilePhoto(ctx, &pb.DeleteProfilePhotoRequest{UserId: 1, ProfilePhotoId: 999})
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

		h := newProfilePhotoHandler(mockService)
		_, err := h.DeleteProfilePhoto(ctx, &pb.DeleteProfilePhotoRequest{UserId: 1, ProfilePhotoId: 999})
		if err == nil {
			t.Fatal("Expected error")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound, got %v", err)
		}
	})
}

func TestPrependGatewayURL(t *testing.T) {
	tests := []struct {
		name        string
		gatewayURL  string
		inputURL    string
		expectedURL string
	}{
		{"relative URL with gateway URL", "https://api.example.com", "/uploads/profile/photo.jpg", "https://api.example.com/uploads/profile/photo.jpg"},
		{"relative URL without leading slash", "https://api.example.com", "uploads/profile/photo.jpg", "https://api.example.com/uploads/profile/photo.jpg"},
		{"already full HTTP URL", "https://api.example.com", "http://storage.example.com/photo.jpg", "http://storage.example.com/photo.jpg"},
		{"already full HTTPS URL", "https://api.example.com", "https://storage.example.com/photo.jpg", "https://storage.example.com/photo.jpg"},
		{"empty gateway URL", "", "/uploads/profile/photo.jpg", "/uploads/profile/photo.jpg"},
		{"gateway URL with trailing slash", "https://api.example.com/", "/uploads/profile/photo.jpg", "https://api.example.com/uploads/profile/photo.jpg"},
		{"empty input URL", "https://api.example.com", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.PrependGatewayURL(tt.gatewayURL, tt.inputURL)
			if result != tt.expectedURL {
				t.Errorf("expected %s, got %s", tt.expectedURL, result)
			}
		})
	}
}
