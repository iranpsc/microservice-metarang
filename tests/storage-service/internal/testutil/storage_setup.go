package testutil

import (
	"path/filepath"
	"testing"

	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/service"
)

func NewTestStorageService(t *testing.T) (*service.StorageService, string) {
	t.Helper()
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatalf("NewChunkManager: %v", err)
	}
	uploadBase := filepath.Join(tempDir, "uploads")
	ftpClient := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com/uploads")
	return service.NewStorageService(ftpClient, chunkManager, uploadBase), uploadBase
}
