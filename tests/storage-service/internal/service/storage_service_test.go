package service_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/repository"
	"metarang/storage-service/internal/service"
)

func TestStorageService_UploadFileAndGetFile(t *testing.T) {
	tempDir := t.TempDir()
	ftpDir := filepath.Join(tempDir, "ftp")
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatalf("NewChunkManager: %v", err)
	}
	mockFTP := ftp.NewMockFTPClient(ftpDir, "http://cdn.example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, filepath.Join(tempDir, "uploads"))

	data := []byte("hello storage")
	url, err := svc.UploadFile("photo.jpg", "image/jpeg", data, "avatars")
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	// Find uploaded remote path from mock storage.
	var remotePath string
	_ = filepath.Walk(ftpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() != "" {
			rel, _ := filepath.Rel(ftpDir, path)
			remotePath = filepath.ToSlash(rel)
		}
		return nil
	})
	if remotePath == "" {
		t.Fatal("uploaded file not found on mock FTP")
	}

	got, contentType, err := svc.GetFile(remotePath)
	if err != nil {
		t.Fatalf("GetFile from FTP: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch: %q", got)
	}
	if contentType != "image/jpeg" {
		t.Fatalf("contentType = %q, want image/jpeg", contentType)
	}
}

func TestStorageService_GetFile_LocalUpload(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	profileDir := filepath.Join(uploadBase, "profile")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("local-file")
	if err := os.WriteFile(filepath.Join(profileDir, "pic.png"), content, 0644); err != nil {
		t.Fatal(err)
	}

	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, uploadBase)

	got, contentType, err := svc.GetFile("profile/pic.png")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch")
	}
	if contentType != "image/png" {
		t.Fatalf("contentType = %q, want image/png", contentType)
	}
}

func TestStorageService_GetFile_ContentTypes(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	if err := os.MkdirAll(uploadBase, 0755); err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"doc.pdf":   "application/pdf",
		"clip.mp4":  "video/mp4",
		"clip.webm": "video/webm",
		"clip.mov":  "video/quicktime",
		"pic.gif":   "image/gif",
	}

	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, uploadBase)

	for name, wantMIME := range tests {
		if err := os.WriteFile(filepath.Join(uploadBase, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		_, got, err := svc.GetFile(name)
		if err != nil {
			t.Fatalf("GetFile(%q): %v", name, err)
		}
		if got != wantMIME {
			t.Fatalf("GetFile(%q): mime = %q, want %q", name, got, wantMIME)
		}
	}
}

func TestStorageService_GetFile_UploadPrefixPath(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	nested := filepath.Join(uploadBase, "uploads", "profile")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("legacy")
	if err := os.WriteFile(filepath.Join(nested, "legacy.png"), content, 0644); err != nil {
		t.Fatal(err)
	}

	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"), chunkManager, uploadBase)

	got, contentType, err := svc.GetFile("upload/profile/legacy.png")
	if err != nil || !bytes.Equal(got, content) || contentType != "image/png" {
		t.Fatalf("GetFile: err=%v got=%q type=%q", err, got, contentType)
	}
}

func TestStorageService_GetFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, filepath.Join(tempDir, "uploads"))
	if _, _, err := svc.GetFile("missing/file.bin"); err == nil {
		t.Fatal("expected error")
	}
}

func TestStorageService_DeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	ftpDir := filepath.Join(tempDir, "ftp")
	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(ftpDir, "http://example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, filepath.Join(tempDir, "uploads"))

	remotePath := "files/remove-me.txt"
	if err := mockFTP.UploadFile(remotePath, bytes.NewReader([]byte("bye"))); err != nil {
		t.Fatalf("setup upload: %v", err)
	}
	if err := svc.DeleteFile(remotePath); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ftpDir, remotePath)); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}

func TestStorageService_NewStorageService_EmptyBaseDir(t *testing.T) {
	tempDir := t.TempDir()
	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, "")
	if svc == nil {
		t.Fatal("expected service")
	}
}

func TestImageService_CreateGetDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	imageType := "profile"
	now := time.Now()
	repo := repository.NewImageRepository(db)
	svc := service.NewImageService(repo, nil)

	mock.ExpectExec("INSERT INTO images").
		WithArgs("App\\Models\\User", uint64(10), "http://example.com/p.jpg", &imageType).
		WillReturnResult(sqlmock.NewResult(1, 1))

	created, err := svc.CreateImage(context.Background(), "App\\Models\\User", 10, "http://example.com/p.jpg", &imageType)
	if err != nil || created.ID != 1 {
		t.Fatalf("CreateImage: err=%v created=%+v", err, created)
	}

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(1), "App\\Models\\User", uint64(10), "http://example.com/p.jpg", &imageType, now, now))

	images, err := svc.GetImages(context.Background(), "App\\Models\\User", 10, "")
	if err != nil || len(images) != 1 {
		t.Fatalf("GetImages: err=%v len=%d", err, len(images))
	}

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(1), "App\\Models\\User", uint64(10), "http://example.com/p.jpg", &imageType, now, now))
	mock.ExpectExec("DELETE FROM images WHERE id = \\?").
		WithArgs(uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteImage(context.Background(), 1); err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
}

func TestImageService_DeleteImage_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(404)).
		WillReturnError(sql.ErrNoRows)

	svc := service.NewImageService(repository.NewImageRepository(db), nil)
	if err := svc.DeleteImage(context.Background(), 404); err == nil {
		t.Fatal("expected error for missing image")
	}
}

func TestChunkManager_SaveDuplicateChunkAndCleanupMissing(t *testing.T) {
	tempDir := t.TempDir()
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}

	session, err := cm.GetOrCreateSession("dup-upload", "f.txt", "text/plain", 2, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.SaveChunk(session, 0, []byte("aa")); err != nil {
		t.Fatal(err)
	}
	if err := cm.SaveChunk(session, 0, []byte("aa")); err != nil {
		t.Fatal(err)
	}
	if cm.GetProgress(session) != 50.0 {
		t.Fatalf("progress = %f, want 50", cm.GetProgress(session))
	}

	if err := cm.CleanupSession("missing-session"); err != nil {
		t.Fatalf("CleanupSession missing: %v", err)
	}
}

func TestChunkManager_GetProgressZeroTotal(t *testing.T) {
	tempDir := t.TempDir()
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	session, err := cm.GetOrCreateSession("zero", "f.txt", "text/plain", 0, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if cm.GetProgress(session) != 0 {
		t.Fatalf("progress = %f, want 0", cm.GetProgress(session))
	}
}
