package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/service"
)

func TestHandleChunkUploadProfilePath(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	chunkTemp := filepath.Join(tempDir, "chunks")

	chunkManager, err := service.NewChunkManager(chunkTemp)
	if err != nil {
		t.Fatalf("NewChunkManager: %v", err)
	}

	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://localhost/uploads")
	svc := service.NewStorageService(mockFTP, chunkManager, uploadBase)

	finished, progress, publicDir, filename, mimeType, err := svc.HandleChunkUpload(
		"test-upload-1",
		"photo.jpg",
		"image/jpeg",
		[]byte("fake-jpeg-data"),
		0,
		1,
		int64(len("fake-jpeg-data")),
		"/uploads/profile",
	)
	if err != nil {
		t.Fatalf("HandleChunkUpload: %v", err)
	}
	if !finished || progress != 100.0 {
		t.Fatalf("unexpected progress: finished=%v progress=%v", finished, progress)
	}
	if publicDir != "/uploads/profile/" {
		t.Fatalf("publicDir = %q, want /uploads/profile/", publicDir)
	}
	if filename == "" {
		t.Fatal("expected generated filename")
	}
	if mimeType != "image/jpeg" {
		t.Fatalf("mimeType = %q, want image/jpeg", mimeType)
	}

	localFile := filepath.Join(uploadBase, "profile", filename)
	if _, err := os.Stat(localFile); err != nil {
		t.Fatalf("expected file at %s: %v", localFile, err)
	}
}

func TestHandleChunkUpload_MultiChunk(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

	finished, progress, _, _, _, err := svc.HandleChunkUpload("mc", "f.txt", "text/plain", []byte("part1"), 0, 2, 10, "docs")
	if err != nil || finished || progress != 50.0 {
		t.Fatalf("chunk 0: err=%v finished=%v progress=%v", err, finished, progress)
	}

	finished, progress, dir, name, _, err := svc.HandleChunkUpload("mc", "f.txt", "text/plain", []byte("part2"), 1, 2, 10, "docs")
	if err != nil || !finished || progress != 100.0 {
		t.Fatalf("chunk 1: err=%v finished=%v progress=%v", err, finished, progress)
	}
	if _, err := os.Stat(filepath.Join(uploadBase, "docs", name)); err != nil {
		t.Fatalf("missing assembled file under docs/: dir=%q name=%q err=%v", dir, name, err)
	}
}

func TestChunkManager_NewChunkManager_InvalidBaseDir(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.NewChunkManager(filePath); err == nil {
		t.Fatal("expected error when base path is a file")
	}
}

func TestHandleChunkUpload_StripsContentTypeCharset(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), cm, uploadBase)

	_, _, _, _, mimeType, err := svc.HandleChunkUpload(
		"charset-upload", "pic.jpg", "image/jpeg; charset=utf-8", []byte("data"), 0, 1, 4, "profile",
	)
	if err != nil || mimeType != "image/jpeg" {
		t.Fatalf("HandleChunkUpload: err=%v mime=%q", err, mimeType)
	}
}

func TestHandleChunkUpload_FileWithoutExtension(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

	finished, _, _, filename, _, err := svc.HandleChunkUpload(
		"no-ext", "README", "text/plain", []byte("content"), 0, 1, 7, "docs",
	)
	if err != nil || !finished || filename == "" || filepath.Ext(filename) != "" {
		t.Fatalf("HandleChunkUpload: err=%v finished=%v filename=%q", err, finished, filename)
	}
}
