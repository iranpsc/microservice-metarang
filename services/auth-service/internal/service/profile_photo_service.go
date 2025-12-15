package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

// StorageClient interface for uploading files (allows mocking in tests)
type StorageClient interface {
	UploadFile(ctx context.Context, imageData []byte, filename, contentType string, userID uint64) (string, error)
}

var (
	ErrProfilePhotoNotFound = errors.New("profile photo not found")
	ErrPhotoUnauthorized    = errors.New("unauthorized: profile photo does not belong to user")
	ErrInvalidImage         = errors.New("invalid image: must be PNG or JPEG, ≤1 MB")
	ErrImageRequired        = errors.New("image is required")
)

type ProfilePhotoService interface {
	// ListProfilePhotos returns all profile photos for a user, ordered by creation time
	ListProfilePhotos(ctx context.Context, userID uint64) ([]*models.Image, error)
	// UploadProfilePhoto uploads a new profile photo for a user
	UploadProfilePhoto(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error)
	// GetProfilePhoto retrieves a profile photo by ID
	GetProfilePhoto(ctx context.Context, id uint64) (*models.Image, error)
	// DeleteProfilePhoto deletes a profile photo (with ownership check)
	DeleteProfilePhoto(ctx context.Context, userID uint64, id uint64) error
}

type profilePhotoService struct {
	repo          repository.ProfilePhotoRepository
	storageClient StorageClient
}

func NewProfilePhotoService(repo repository.ProfilePhotoRepository, storageClient StorageClient) ProfilePhotoService {
	return &profilePhotoService{
		repo:          repo,
		storageClient: storageClient,
	}
}

// ListProfilePhotos returns all profile photos for a user
func (s *profilePhotoService) ListProfilePhotos(ctx context.Context, userID uint64) ([]*models.Image, error) {
	photos, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list profile photos: %w", err)
	}
	return photos, nil
}

// UploadProfilePhoto uploads a new profile photo
func (s *profilePhotoService) UploadProfilePhoto(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
	// Validate image
	if len(imageData) == 0 {
		return nil, ErrImageRequired
	}

	// Validate file size (≤1 MB = 1024 * 1024 bytes)
	const maxSize = 1024 * 1024
	if len(imageData) > maxSize {
		return nil, ErrInvalidImage
	}

	// Validate content type
	contentType = strings.ToLower(contentType)
	if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/jpg" {
		return nil, ErrInvalidImage
	}

	// Validate filename extension
	filenameLower := strings.ToLower(filename)
	if !strings.HasSuffix(filenameLower, ".png") && !strings.HasSuffix(filenameLower, ".jpg") && !strings.HasSuffix(filenameLower, ".jpeg") {
		return nil, ErrInvalidImage
	}

	// Upload to storage service if available
	var url string
	if s.storageClient != nil {
		uploadURL, err := s.storageClient.UploadFile(ctx, imageData, filename, contentType, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to upload to storage service: %w", err)
		}
		url = uploadURL
	} else {
		// Fallback: construct URL assuming file was uploaded via HTTP endpoint
		// In production, this should always use the storage service
		url = fmt.Sprintf("/uploads/profile/%s", filename)
	}

	// Create image record in database
	image, err := s.repo.Create(ctx, userID, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile photo record: %w", err)
	}

	return image, nil
}

// GetProfilePhoto retrieves a profile photo by ID
func (s *profilePhotoService) GetProfilePhoto(ctx context.Context, id uint64) (*models.Image, error) {
	photo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile photo: %w", err)
	}
	if photo == nil {
		return nil, ErrProfilePhotoNotFound
	}
	return photo, nil
}

// DeleteProfilePhoto deletes a profile photo with ownership check
func (s *profilePhotoService) DeleteProfilePhoto(ctx context.Context, userID uint64, id uint64) error {
	// Check ownership
	owns, err := s.repo.CheckOwnership(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}
	if !owns {
		return ErrPhotoUnauthorized
	}

	// Delete the record
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete profile photo: %w", err)
	}

	return nil
}
