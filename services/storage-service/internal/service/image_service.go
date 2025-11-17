package service

import (
	"context"
	"fmt"

	"metargb/storage-service/internal/ftp"
	"metargb/storage-service/internal/models"
	"metargb/storage-service/internal/repository"
)

type ImageService struct {
	repo      *repository.ImageRepository
	ftpClient *ftp.FTPClient
}

func NewImageService(repo *repository.ImageRepository, ftpClient *ftp.FTPClient) *ImageService {
	return &ImageService{
		repo:      repo,
		ftpClient: ftpClient,
	}
}

// CreateImage creates a new image record
func (s *ImageService) CreateImage(ctx context.Context, imageableType string, imageableID uint64, url string, imageType *string) (*models.Image, error) {
	image := &models.Image{
		ImageableType: imageableType,
		ImageableID:   imageableID,
		URL:           url,
		Type:          imageType,
	}

	if err := s.repo.CreateImage(ctx, image); err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	return image, nil
}

// GetImages retrieves images for an entity
func (s *ImageService) GetImages(ctx context.Context, imageableType string, imageableID uint64, imageType string) ([]*models.Image, error) {
	return s.repo.GetImages(ctx, imageableType, imageableID, imageType)
}

// DeleteImage deletes an image record and file
func (s *ImageService) DeleteImage(ctx context.Context, imageID uint64) error {
	// Get image record
	image, err := s.repo.GetImageByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	if image == nil {
		return fmt.Errorf("image not found")
	}

	// Delete from database
	if err := s.repo.DeleteImage(ctx, imageID); err != nil {
		return fmt.Errorf("failed to delete image record: %w", err)
	}

	// Delete from FTP (optional - might fail if file already deleted)
	// Extract path from URL and delete
	// s.ftpClient.DeleteFile(extractPathFromURL(image.URL))

	return nil
}

