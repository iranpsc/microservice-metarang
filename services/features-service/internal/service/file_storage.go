package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	storagepb "metarang/shared/pb/storage"
)

var (
	ErrStorageUnavailable   = errors.New("storage service not available")
	ErrStorageUploadFailed  = errors.New("storage service upload failed")
	ErrInvalidFeatureImage  = errors.New("invalid image: must be PNG, JPG, or BMP, \u22641024 KB")
	ErrFeatureImageRequired = errors.New("image data is required")
)

const featureImageMaxSize = 1024 * 1024

// FileStorage uploads files via storage-service gRPC.
type FileStorage interface {
	UploadChunk(ctx context.Context, uploadID, uploadPath, filename, contentType string, data []byte) (relativePath string, err error)
}

type grpcFileStorage struct {
	client storagepb.FileStorageServiceClient
}

func NewGRPCFileStorage(client storagepb.FileStorageServiceClient) FileStorage {
	if client == nil {
		return nil
	}
	return &grpcFileStorage{client: client}
}

func (s *grpcFileStorage) UploadChunk(ctx context.Context, uploadID, uploadPath, filename, contentType string, data []byte) (string, error) {
	if s == nil || s.client == nil {
		return "", ErrStorageUnavailable
	}

	chunkResp, err := s.client.ChunkUpload(ctx, &storagepb.ChunkUploadRequest{
		UploadId:    uploadID,
		ChunkData:   data,
		ChunkIndex:  0,
		TotalChunks: 1,
		Filename:    filename,
		ContentType: contentType,
		TotalSize:   int64(len(data)),
		UploadPath:  uploadPath,
	})
	if err != nil {
		return "", fmt.Errorf("storage upload: %w", err)
	}
	if !chunkResp.Success {
		return "", fmt.Errorf("%w: %s", ErrStorageUploadFailed, chunkResp.Message)
	}
	if !chunkResp.IsFinished {
		return "", fmt.Errorf("%w: upload did not complete", ErrStorageUploadFailed)
	}

	dirPath := chunkResp.FileUrl
	name := chunkResp.FilePath
	if name == "" {
		name = chunkResp.FinalFilename
	}
	if dirPath == "" || name == "" {
		return "", fmt.Errorf("%w: incomplete file path returned", ErrStorageUploadFailed)
	}

	return strings.TrimSuffix(dirPath, "/") + "/" + name, nil
}

func newFeatureImageUploadID(featureID uint64, index int) string {
	return fmt.Sprintf("feature_image_%d_%d_%d", featureID, index, time.Now().UnixNano())
}

func featureImageUploadPath(featureID uint64) string {
	return fmt.Sprintf("uploads/features/%d", featureID)
}

func prependPublicURL(base, path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if base == "" {
		return path
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(path, "/")
}

func validateFeatureImage(imageData []byte, filename, contentType string) error {
	if len(imageData) == 0 {
		return ErrFeatureImageRequired
	}
	if len(imageData) > featureImageMaxSize {
		return ErrInvalidFeatureImage
	}

	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if !isAllowedFeatureImageContentType(contentType) {
		return ErrInvalidFeatureImage
	}

	if filename != "" && !isAllowedFeatureImageFilename(filename) {
		return ErrInvalidFeatureImage
	}
	return nil
}

func isAllowedFeatureImageContentType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/jpg", "image/bmp":
		return true
	default:
		return false
	}
}

func isAllowedFeatureImageFilename(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png", ".jpg", ".jpeg", ".bmp":
		return true
	default:
		return false
	}
}

func defaultFeatureImageFilename(contentType string, index int) string {
	ext := ".jpg"
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/png":
		ext = ".png"
	case "image/bmp":
		ext = ".bmp"
	}
	return fmt.Sprintf("image_%d%s", index+1, ext)
}
