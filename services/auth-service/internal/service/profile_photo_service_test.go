package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/auth-service/internal/models"
)

// fakeProfilePhotoRepository is a mock implementation for testing
type fakeProfilePhotoRepository struct {
	photos       map[uint64]*models.Image
	userPhotos   map[uint64][]*models.Image
	createCount  int
	deleteCount  int
	findByIDFunc func(ctx context.Context, id uint64) (*models.Image, error)
	checkOwnFunc func(ctx context.Context, id uint64, userID uint64) (bool, error)
}

func newFakeProfilePhotoRepository() *fakeProfilePhotoRepository {
	return &fakeProfilePhotoRepository{
		photos:     make(map[uint64]*models.Image),
		userPhotos: make(map[uint64][]*models.Image),
	}
}

func (r *fakeProfilePhotoRepository) Create(ctx context.Context, userID uint64, url string) (*models.Image, error) {
	r.createCount++
	id := uint64(len(r.photos) + 1)
	image := &models.Image{
		ID:            id,
		ImageableType: "App\\Models\\User",
		ImageableID:   userID,
		URL:           url,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	r.photos[id] = image
	r.userPhotos[userID] = append(r.userPhotos[userID], image)
	return image, nil
}

func (r *fakeProfilePhotoRepository) FindByUserID(ctx context.Context, userID uint64) ([]*models.Image, error) {
	return r.userPhotos[userID], nil
}

func (r *fakeProfilePhotoRepository) FindByID(ctx context.Context, id uint64) (*models.Image, error) {
	if r.findByIDFunc != nil {
		return r.findByIDFunc(ctx, id)
	}
	return r.photos[id], nil
}

func (r *fakeProfilePhotoRepository) Delete(ctx context.Context, id uint64) error {
	r.deleteCount++
	if _, exists := r.photos[id]; !exists {
		return errors.New("profile photo not found")
	}
	delete(r.photos, id)
	// Remove from userPhotos
	for userID, photos := range r.userPhotos {
		for i, photo := range photos {
			if photo.ID == id {
				r.userPhotos[userID] = append(photos[:i], photos[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (r *fakeProfilePhotoRepository) CheckOwnership(ctx context.Context, id uint64, userID uint64) (bool, error) {
	if r.checkOwnFunc != nil {
		return r.checkOwnFunc(ctx, id, userID)
	}
	photo, exists := r.photos[id]
	if !exists {
		return false, nil
	}
	return photo.ImageableID == userID, nil
}

func TestProfilePhotoService_ListProfilePhotos(t *testing.T) {
	ctx := context.Background()
	repo := newFakeProfilePhotoRepository()
	service, _ := NewProfilePhotoService(repo, "")

	t.Run("successful list", func(t *testing.T) {
		userID := uint64(1)
		// Create some photos
		_, _ = repo.Create(ctx, userID, "https://example.com/photo1.jpg")
		_, _ = repo.Create(ctx, userID, "https://example.com/photo2.jpg")

		photos, err := service.ListProfilePhotos(ctx, userID)
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		if len(photos) != 2 {
			t.Errorf("Expected 2 photos, got %d", len(photos))
		}
	})

	t.Run("empty list for user with no photos", func(t *testing.T) {
		userID := uint64(999)
		photos, err := service.ListProfilePhotos(ctx, userID)
		if err != nil {
			t.Fatalf("ListProfilePhotos failed: %v", err)
		}

		if len(photos) != 0 {
			t.Errorf("Expected 0 photos, got %d", len(photos))
		}
	})
}

func TestProfilePhotoService_UploadProfilePhoto(t *testing.T) {
	ctx := context.Background()
	repo := newFakeProfilePhotoRepository()
	service, _ := NewProfilePhotoService(repo, "")

	t.Run("successful upload", func(t *testing.T) {
		userID := uint64(1)
		imageData := make([]byte, 100) // Small test image
		filename := "test.jpg"
		contentType := "image/jpeg"

		photo, err := service.UploadProfilePhoto(ctx, userID, imageData, filename, contentType)
		if err != nil {
			t.Fatalf("UploadProfilePhoto failed: %v", err)
		}

		if photo.ID == 0 {
			t.Error("Expected photo ID to be set")
		}
		if photo.URL == "" {
			t.Error("Expected photo URL to be set")
		}
		if photo.ImageableID != userID {
			t.Errorf("Expected ImageableID %d, got %d", userID, photo.ImageableID)
		}
	})

	t.Run("empty image data", func(t *testing.T) {
		userID := uint64(1)
		imageData := []byte{}
		filename := "test.jpg"
		contentType := "image/jpeg"

		_, err := service.UploadProfilePhoto(ctx, userID, imageData, filename, contentType)
		if err != ErrImageRequired {
			t.Errorf("Expected ErrImageRequired, got %v", err)
		}
	})

	t.Run("file too large", func(t *testing.T) {
		userID := uint64(1)
		imageData := make([]byte, 2*1024*1024) // 2 MB, exceeds limit
		filename := "test.jpg"
		contentType := "image/jpeg"

		_, err := service.UploadProfilePhoto(ctx, userID, imageData, filename, contentType)
		if err != ErrInvalidImage {
			t.Errorf("Expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("invalid content type", func(t *testing.T) {
		userID := uint64(1)
		imageData := make([]byte, 100)
		filename := "test.gif"
		contentType := "image/gif"

		_, err := service.UploadProfilePhoto(ctx, userID, imageData, filename, contentType)
		if err != ErrInvalidImage {
			t.Errorf("Expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("invalid file extension", func(t *testing.T) {
		userID := uint64(1)
		imageData := make([]byte, 100)
		filename := "test.gif"
		contentType := "image/jpeg"

		_, err := service.UploadProfilePhoto(ctx, userID, imageData, filename, contentType)
		if err != ErrInvalidImage {
			t.Errorf("Expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("PNG file upload", func(t *testing.T) {
		userID := uint64(1)
		imageData := make([]byte, 100)
		filename := "test.png"
		contentType := "image/png"

		photo, err := service.UploadProfilePhoto(ctx, userID, imageData, filename, contentType)
		if err != nil {
			t.Fatalf("UploadProfilePhoto failed: %v", err)
		}

		if photo.ID == 0 {
			t.Error("Expected photo ID to be set")
		}
	})
}

func TestProfilePhotoService_GetProfilePhoto(t *testing.T) {
	ctx := context.Background()
	repo := newFakeProfilePhotoRepository()
	service, _ := NewProfilePhotoService(repo, "")

	t.Run("successful get", func(t *testing.T) {
		userID := uint64(1)
		photo, _ := repo.Create(ctx, userID, "https://example.com/photo.jpg")

		result, err := service.GetProfilePhoto(ctx, photo.ID)
		if err != nil {
			t.Fatalf("GetProfilePhoto failed: %v", err)
		}

		if result.ID != photo.ID {
			t.Errorf("Expected ID %d, got %d", photo.ID, result.ID)
		}
		if result.URL != photo.URL {
			t.Errorf("Expected URL %s, got %s", photo.URL, result.URL)
		}
	})

	t.Run("photo not found", func(t *testing.T) {
		_, err := service.GetProfilePhoto(ctx, 999)
		if err != ErrProfilePhotoNotFound {
			t.Errorf("Expected ErrProfilePhotoNotFound, got %v", err)
		}
	})

	t.Run("repository returns nil", func(t *testing.T) {
		repo.findByIDFunc = func(ctx context.Context, id uint64) (*models.Image, error) {
			return nil, nil
		}

		_, err := service.GetProfilePhoto(ctx, 1)
		if err != ErrProfilePhotoNotFound {
			t.Errorf("Expected ErrProfilePhotoNotFound, got %v", err)
		}
	})
}

func TestProfilePhotoService_DeleteProfilePhoto(t *testing.T) {
	ctx := context.Background()
	repo := newFakeProfilePhotoRepository()
	service, _ := NewProfilePhotoService(repo, "")

	t.Run("successful delete", func(t *testing.T) {
		userID := uint64(1)
		photo, _ := repo.Create(ctx, userID, "https://example.com/photo.jpg")

		err := service.DeleteProfilePhoto(ctx, userID, photo.ID)
		if err != nil {
			t.Fatalf("DeleteProfilePhoto failed: %v", err)
		}

		// Verify photo is deleted
		result, _ := repo.FindByID(ctx, photo.ID)
		if result != nil {
			t.Error("Expected photo to be deleted")
		}
	})

	t.Run("unauthorized - photo belongs to different user", func(t *testing.T) {
		userID1 := uint64(1)
		userID2 := uint64(2)
		photo, _ := repo.Create(ctx, userID1, "https://example.com/photo.jpg")

		err := service.DeleteProfilePhoto(ctx, userID2, photo.ID)
		if err != ErrUnauthorized {
			t.Errorf("Expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("photo not found", func(t *testing.T) {
		userID := uint64(1)
		repo.checkOwnFunc = func(ctx context.Context, id uint64, userID uint64) (bool, error) {
			return false, nil
		}

		err := service.DeleteProfilePhoto(ctx, userID, 999)
		if err != ErrUnauthorized {
			t.Errorf("Expected ErrUnauthorized, got %v", err)
		}
	})
}
