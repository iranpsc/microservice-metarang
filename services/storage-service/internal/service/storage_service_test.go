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

func newTestStorageService(t *testing.T) (*service.StorageService, string) {
	t.Helper()
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatalf("NewChunkManager: %v", err)
	}
	uploadBase := filepath.Join(tempDir, "uploads")
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com/uploads")
	return service.NewStorageService(mockFTP, chunkManager, uploadBase), uploadBase
}

func TestStorageService_UploadFileAndGetFile(t *testing.T) {
	tempDir := t.TempDir()
	ftpDir := filepath.Join(tempDir, "ftp")
	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(ftpDir, "http://cdn.example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, filepath.Join(tempDir, "uploads"))

	data := []byte("payload")
	if _, err := svc.UploadFile("doc.pdf", "application/pdf", data, "docs"); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	var remotePath string
	_ = filepath.Walk(ftpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(ftpDir, path)
			remotePath = filepath.ToSlash(rel)
		}
		return nil
	})

	got, contentType, err := svc.GetFile(remotePath)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if !bytes.Equal(got, data) || contentType != "application/pdf" {
		t.Fatalf("GetFile: got %q type %q", got, contentType)
	}
}

func TestStorageService_GetFile_Local(t *testing.T) {
	svc, uploadBase := newTestStorageService(t)
	profileDir := filepath.Join(uploadBase, "profile")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("local")
	if err := os.WriteFile(filepath.Join(profileDir, "pic.png"), content, 0644); err != nil {
		t.Fatal(err)
	}

	got, contentType, err := svc.GetFile("profile/pic.png")
	if err != nil || !bytes.Equal(got, content) || contentType != "image/png" {
		t.Fatalf("GetFile: err=%v got=%q type=%q", err, got, contentType)
	}

	for name, want := range map[string]string{
		"photo.jpg":  "image/jpeg",
		"photo.jpeg": "image/jpeg",
	} {
		if err := os.WriteFile(filepath.Join(uploadBase, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		_, ct, err := svc.GetFile(name)
		if err != nil || ct != want {
			t.Fatalf("GetFile(%q): ct=%q err=%v", name, ct, err)
		}
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

	cm, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	svc := service.NewStorageService(mockFTP, cm, uploadBase)

	for name, wantMIME := range tests {
		if err := os.WriteFile(filepath.Join(uploadBase, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		_, got, err := svc.GetFile(name)
		if err != nil || got != wantMIME {
			t.Fatalf("GetFile(%q): mime=%q err=%v", name, got, err)
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

	cm, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"), cm, uploadBase)

	got, contentType, err := svc.GetFile("upload/profile/legacy.png")
	if err != nil || !bytes.Equal(got, content) || contentType != "image/png" {
		t.Fatalf("GetFile: err=%v got=%q type=%q", err, got, contentType)
	}
}

func TestStorageService_GetFile_NotFound(t *testing.T) {
	svc, _ := newTestStorageService(t)
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

	if err := mockFTP.UploadFile("drop.txt", bytes.NewReader([]byte("x"))); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteFile("drop.txt"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
}

func TestImageService_CRUD(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	imageType := "profile"
	now := time.Now()
	repo := repository.NewImageRepository(db)
	svc := service.NewImageService(repo, ftp.NewFTPClient("localhost", "21", "", "", "http://example.com"))

	mock.ExpectExec("INSERT INTO images").
		WithArgs("App\\Models\\User", uint64(10), "http://example.com/p.jpg", &imageType).
		WillReturnResult(sqlmock.NewResult(1, 1))

	created, err := svc.CreateImage(context.Background(), "App\\Models\\User", 10, "http://example.com/p.jpg", &imageType)
	if err != nil || created.ID != 1 {
		t.Fatalf("CreateImage: err=%v id=%d", err, created.ID)
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
		t.Fatal("expected error")
	}
}

func TestChunkManager_DuplicateChunk(t *testing.T) {
	cm, err := service.NewChunkManager(filepath.Join(t.TempDir(), "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	session, err := cm.GetOrCreateSession("dup", "f.txt", "text/plain", 2, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	_ = cm.SaveChunk(session, 0, []byte("aa"))
	_ = cm.SaveChunk(session, 0, []byte("aa"))
	if cm.GetProgress(session) != 50.0 {
		t.Fatalf("progress = %f", cm.GetProgress(session))
	}
	_ = cm.CleanupSession("missing")
}

func TestNewStorageService_EmptyBaseDir(t *testing.T) {
	tempDir := t.TempDir()
	cm, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	if svc := service.NewStorageService(mockFTP, cm, ""); svc == nil {
		t.Fatal("expected service")
	}
}
