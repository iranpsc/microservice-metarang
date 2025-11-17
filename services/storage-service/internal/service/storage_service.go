package service

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"metargb/storage-service/internal/ftp"
)

type StorageService struct {
	ftpClient    ftp.FTPClientInterface
	chunkManager *ChunkManager
}

func NewStorageService(ftpClient ftp.FTPClientInterface, chunkManager *ChunkManager) *StorageService {
	return &StorageService{
		ftpClient:    ftpClient,
		chunkManager: chunkManager,
	}
}

// UploadFile uploads a file to FTP server
func (s *StorageService) UploadFile(filename, contentType string, data []byte, uploadPath string) (string, error) {
	// Generate unique filename
	timestamp := time.Now().Unix()
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%d_%s%s", timestamp, filename[:len(filename)-len(ext)], ext)

	// Build remote path
	remotePath := filepath.Join(uploadPath, uniqueFilename)

	// Upload to FTP
	reader := bytes.NewReader(data)
	if err := s.ftpClient.UploadFile(remotePath, reader); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate URL
	url := s.ftpClient.GenerateURL(remotePath)

	return url, nil
}

// GetFile retrieves a file from FTP server
func (s *StorageService) GetFile(filePath string) ([]byte, string, error) {
	// Download from FTP
	reader, err := s.ftpClient.DownloadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	// Read file content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Determine content type from extension
	contentType := "application/octet-stream"
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".pdf":
		contentType = "application/pdf"
	case ".mp4":
		contentType = "video/mp4"
	}

	return data, contentType, nil
}

// DeleteFile deletes a file from FTP server
func (s *StorageService) DeleteFile(filePath string) error {
	return s.ftpClient.DeleteFile(filePath)
}

// HandleChunkUpload processes a chunk upload
func (s *StorageService) HandleChunkUpload(uploadID, filename, contentType string, chunkData []byte, chunkIndex, totalChunks int32, totalSize int64, uploadPath string) (bool, float64, string, string, string, error) {
	// Get or create session
	session, err := s.chunkManager.GetOrCreateSession(uploadID, filename, contentType, totalChunks, totalSize, uploadPath)
	if err != nil {
		return false, 0, "", "", "", fmt.Errorf("failed to create session: %w", err)
	}

	// Save the chunk
	if err := s.chunkManager.SaveChunk(session, chunkIndex, chunkData); err != nil {
		return false, 0, "", "", "", fmt.Errorf("failed to save chunk: %w", err)
	}

	// Get progress
	progress := s.chunkManager.GetProgress(session)

	// Check if upload is complete
	if !s.chunkManager.IsComplete(session) {
		return false, progress, "", "", "", nil
	}

	// Assemble file
	assembledData, finalPath, err := s.chunkManager.AssembleFile(session)
	if err != nil {
		s.chunkManager.CleanupSession(uploadID)
		return false, 0, "", "", "", fmt.Errorf("failed to assemble file: %w", err)
	}

	// Generate unique filename from the final path
	finalFilename := filepath.Base(finalPath)

	// Upload to FTP
	reader := bytes.NewReader(assembledData)
	if err := s.ftpClient.UploadFile(finalPath, reader); err != nil {
		s.chunkManager.CleanupSession(uploadID)
		return false, 0, "", "", "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate URL
	url := s.ftpClient.GenerateURL(finalPath)

	// Cleanup session
	s.chunkManager.CleanupSession(uploadID)

	return true, 100.0, url, finalPath, finalFilename, nil
}

