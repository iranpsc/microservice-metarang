package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/repository"
)

var (
	ErrProfilePhotoNotFound = errors.New("profile photo not found")
	ErrPhotoUnauthorized    = errors.New("unauthorized: profile photo does not belong to user")
	ErrInvalidImage         = errors.New("invalid image: must be PNG or JPEG, ≤1 MB")
	ErrImageRequired        = errors.New("image is required")
)

const (
	profilePhotoMaxSize    = 1024 * 1024
	profilePhotoUploadPath = "/uploads/profile"
)

type ProfilePhotoService interface {
	ListProfilePhotos(ctx context.Context, userID uint64) ([]*models.Image, error)
	UploadProfilePhoto(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error)
	GetProfilePhoto(ctx context.Context, id uint64) (*models.Image, error)
	DeleteProfilePhoto(ctx context.Context, userID uint64, id uint64) error
	ResolvePhotoURL(url string) string
}

type profilePhotoService struct {
	repo          repository.ProfilePhotoRepository
	fileStorage   FileStorage
	apiGatewayURL string
}

func NewProfilePhotoService(repo repository.ProfilePhotoRepository, fileStorage FileStorage, apiGatewayURL string) ProfilePhotoService {
	return &profilePhotoService{
		repo:          repo,
		fileStorage:   fileStorage,
		apiGatewayURL: apiGatewayURL,
	}
}

func (s *profilePhotoService) ResolvePhotoURL(url string) string {
	return ResolvePublicURL(s.apiGatewayURL, url)
}

func (s *profilePhotoService) ListProfilePhotos(ctx context.Context, userID uint64) ([]*models.Image, error) {
	photos, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list profile photos: %w", err)
	}
	return photos, nil
}

func (s *profilePhotoService) UploadProfilePhoto(ctx context.Context, userID uint64, imageData []byte, filename, contentType string) (*models.Image, error) {
	if err := validateProfilePhotoFile(imageData, filename, contentType); err != nil {
		return nil, err
	}
	if s.fileStorage == nil {
		return nil, ErrStorageUnavailable
	}

	relativePath, err := s.fileStorage.UploadChunk(
		ctx,
		NewUploadID("profile_photo", userID),
		profilePhotoUploadPath,
		filename,
		contentType,
		imageData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload profile photo: %w", err)
	}

	fullURL := PrependGatewayURL(s.apiGatewayURL, relativePath)
	image, err := s.repo.Create(ctx, userID, fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile photo record: %w", err)
	}
	return image, nil
}

func validateProfilePhotoFile(imageData []byte, filename, contentType string) error {
	if len(imageData) == 0 {
		return ErrImageRequired
	}
	if len(imageData) > profilePhotoMaxSize {
		return ErrInvalidImage
	}

	contentType = strings.ToLower(contentType)
	if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/jpg" {
		return ErrInvalidImage
	}

	filenameLower := strings.ToLower(filename)
	if !strings.HasSuffix(filenameLower, ".png") && !strings.HasSuffix(filenameLower, ".jpg") && !strings.HasSuffix(filenameLower, ".jpeg") {
		return ErrInvalidImage
	}
	if filename == "" || contentType == "" {
		return ErrInvalidImage
	}
	return nil
}

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

func (s *profilePhotoService) DeleteProfilePhoto(ctx context.Context, userID uint64, id uint64) error {
	owns, err := s.repo.CheckOwnership(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}
	if !owns {
		return ErrPhotoUnauthorized
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete profile photo: %w", err)
	}
	return nil
}
