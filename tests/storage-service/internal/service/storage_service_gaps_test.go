package service_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/service"
	"metarang/storage-service/tests/internal/testutil"
)

func TestStorageService_UploadFile_FTPError(t *testing.T) {
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(
		&testutil.FailingFTPClient{UploadErr: errors.New("stor failed")},
		chunkManager,
		filepath.Join(tempDir, "uploads"),
	)

	_, err = svc.UploadFile("photo.jpg", "image/jpeg", []byte("data"), "avatars")
	if err == nil {
		t.Fatal("expected upload error")
	}
	if !strings.Contains(err.Error(), "failed to upload file") {
		t.Fatalf("error = %v", err)
	}
}

func TestStorageService_GetFile_UnknownExtension(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	if err := os.MkdirAll(uploadBase, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploadBase, "blob.xyz"), []byte("bin"), 0644); err != nil {
		t.Fatal(err)
	}

	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"), chunkManager, uploadBase)

	got, contentType, err := svc.GetFile("blob.xyz")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if !bytes.Equal(got, []byte("bin")) {
		t.Fatalf("content = %q", got)
	}
	if contentType != "application/octet-stream" {
		t.Fatalf("contentType = %q, want application/octet-stream", contentType)
	}
}

func TestStorageService_GetFile_FTPFallback(t *testing.T) {
	tempDir := t.TempDir()
	ftpDir := filepath.Join(tempDir, "ftp")
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	mockFTP := ftp.NewMockFTPClient(ftpDir, "http://cdn.example.com")
	svc := service.NewStorageService(mockFTP, chunkManager, filepath.Join(tempDir, "uploads"))

	data := []byte("only-on-ftp")
	if err := mockFTP.UploadFile("remote/only.bin", bytes.NewReader(data)); err != nil {
		t.Fatalf("setup upload: %v", err)
	}

	got, contentType, err := svc.GetFile("remote/only.bin")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("content = %q", got)
	}
	if contentType != "application/octet-stream" {
		t.Fatalf("contentType = %q, want application/octet-stream", contentType)
	}
}

func TestStorageService_GetFile_DownloadReadError(t *testing.T) {
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(
		&testutil.FailingFTPClient{DownloadBody: testutil.ErrReadCloser{Err: errors.New("broken stream")}},
		chunkManager,
		filepath.Join(tempDir, "uploads"),
	)

	if _, _, err := svc.GetFile("missing/remote.bin"); err == nil {
		t.Fatal("expected read error")
	}
}

func TestStorageService_GetFile_LeadingSlash(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	profileDir := filepath.Join(uploadBase, "profile")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("slash-path")
	if err := os.WriteFile(filepath.Join(profileDir, "pic.jpeg"), content, 0644); err != nil {
		t.Fatal(err)
	}

	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"), chunkManager, uploadBase)

	got, contentType, err := svc.GetFile("/profile/pic.jpeg")
	if err != nil || !bytes.Equal(got, content) || contentType != "image/jpeg" {
		t.Fatalf("GetFile: err=%v got=%q type=%q", err, got, contentType)
	}
}

func TestHandleChunkUpload_SessionReuse(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

	finished, progress, _, _, _, err := svc.HandleChunkUpload("reuse-id", "a.txt", "text/plain", []byte("aaa"), 0, 3, 9, "docs")
	if err != nil || finished || progress < 33 || progress > 34 {
		t.Fatalf("chunk 0: err=%v finished=%v progress=%v", err, finished, progress)
	}

	finished, progress, _, _, _, err = svc.HandleChunkUpload("reuse-id", "changed.txt", "text/plain", []byte("bbb"), 1, 9, 99, "other")
	if err != nil || finished {
		t.Fatalf("chunk 1 reuse: err=%v finished=%v", err, finished)
	}
	if progress < 60 || progress > 70 {
		t.Fatalf("progress = %v, want ~66 for original 3-chunk session", progress)
	}
}

func TestHandleChunkUpload_AssembleMissingChunk(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	chunkTemp := filepath.Join(tempDir, "chunks")
	chunkManager, err := service.NewChunkManager(chunkTemp)
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

	if _, _, _, _, _, err := svc.HandleChunkUpload("asm", "f.txt", "text/plain", []byte("a"), 0, 2, 2, "docs"); err != nil {
		t.Fatalf("chunk 0: %v", err)
	}
	if err := os.Remove(filepath.Join(chunkTemp, "asm", "chunk_0")); err != nil {
		t.Fatalf("remove chunk: %v", err)
	}

	_, _, _, _, _, err = svc.HandleChunkUpload("asm", "f.txt", "text/plain", []byte("b"), 1, 2, 2, "docs")
	if err == nil {
		t.Fatal("expected assemble error for missing chunk")
	}
	if !strings.Contains(err.Error(), "failed to assemble file") {
		t.Fatalf("error = %v", err)
	}
}

func TestHandleChunkUpload_CleanupSessionAfterAssemble(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	chunkTemp := filepath.Join(tempDir, "chunks")
	chunkManager, err := service.NewChunkManager(chunkTemp)
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

	finished, _, _, _, _, err := svc.HandleChunkUpload("clean-me", "f.txt", "text/plain", []byte("done"), 0, 1, 4, "docs")
	if err != nil || !finished {
		t.Fatalf("HandleChunkUpload: err=%v finished=%v", err, finished)
	}
	if _, err := os.Stat(filepath.Join(chunkTemp, "clean-me")); !os.IsNotExist(err) {
		t.Fatalf("session temp dir should be removed, err=%v", err)
	}

	finished, progress, _, _, _, err := svc.HandleChunkUpload("clean-me", "f.txt", "text/plain", []byte("again"), 0, 2, 10, "docs")
	if err != nil || finished || progress != 50.0 {
		t.Fatalf("reused id should start a new session: err=%v finished=%v progress=%v", err, finished, progress)
	}
}

func TestHandleChunkUpload_NormalizeUploadSubdir(t *testing.T) {
	tests := []struct {
		name          string
		uploadPath    string
		wantPublicDir string
		defaultLayout bool
	}{
		{name: "uploads only", uploadPath: "uploads", defaultLayout: true},
		{name: "uploads with slashes", uploadPath: "uploads/", defaultLayout: true},
		{name: "whitespace and trailing slashes", uploadPath: "  /uploads/profile/  ", wantPublicDir: "/uploads/profile/"},
		{name: "whitespace around uploads", uploadPath: "  uploads  ", defaultLayout: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			uploadBase := filepath.Join(tempDir, "uploads")
			chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
			if err != nil {
				t.Fatal(err)
			}
			svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

			if tt.defaultLayout {
				t.Cleanup(func() { _ = os.RemoveAll("uploads") })
			}

			finished, _, publicDir, filename, _, err := svc.HandleChunkUpload(
				"norm-"+tt.name, "photo.jpg", "image/jpeg", []byte("jpeg"), 0, 1, 4, tt.uploadPath,
			)
			if err != nil || !finished || filename == "" {
				t.Fatalf("HandleChunkUpload: err=%v finished=%v filename=%q", err, finished, filename)
			}
			if tt.defaultLayout {
				if !strings.HasPrefix(publicDir, "uploads/image-jpeg/") || !strings.HasSuffix(publicDir, "/") {
					t.Fatalf("publicDir = %q, want default mime/date layout", publicDir)
				}
				return
			}
			if publicDir != tt.wantPublicDir {
				t.Fatalf("publicDir = %q, want %q", publicDir, tt.wantPublicDir)
			}
		})
	}
}

func TestHandleChunkUpload_SessionCreateError(t *testing.T) {
	tempDir := t.TempDir()
	chunkDir := filepath.Join(tempDir, "chunks")
	chunkManager, err := service.NewChunkManager(chunkDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chunkDir, "blocked"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, filepath.Join(tempDir, "uploads"))

	_, _, _, _, _, err = svc.HandleChunkUpload("blocked", "f.txt", "text/plain", []byte("x"), 0, 1, 1, "docs")
	if err == nil || !strings.Contains(err.Error(), "failed to create session") {
		t.Fatalf("expected session error, got %v", err)
	}
}

func TestHandleChunkUpload_SaveChunkError(t *testing.T) {
	tempDir := t.TempDir()
	chunkTemp := filepath.Join(tempDir, "chunks")
	chunkManager, err := service.NewChunkManager(chunkTemp)
	if err != nil {
		t.Fatal(err)
	}
	session, err := chunkManager.GetOrCreateSession("bad-save", "f.txt", "text/plain", 1, 4, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(session.TempDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session.TempDir, []byte("not-a-dir"), 0644); err != nil {
		t.Fatal(err)
	}

	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, filepath.Join(tempDir, "uploads"))
	_, _, _, _, _, err = svc.HandleChunkUpload("bad-save", "f.txt", "text/plain", []byte("data"), 0, 1, 4, "docs")
	if err == nil || !strings.Contains(err.Error(), "failed to save chunk") {
		t.Fatalf("expected save chunk error, got %v", err)
	}
}

func TestHandleChunkUpload_MkdirAllError(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	if err := os.WriteFile(uploadBase, []byte("not-a-dir"), 0644); err != nil {
		t.Fatal(err)
	}
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), chunkManager, uploadBase)

	_, _, _, _, _, err = svc.HandleChunkUpload("mkdir-fail", "f.txt", "text/plain", []byte("data"), 0, 1, 4, "docs")
	if err == nil || !strings.Contains(err.Error(), "failed to create storage directory") {
		t.Fatalf("expected mkdir error, got %v", err)
	}
}

func TestChunkManager_AssembleFile_MissingChunk(t *testing.T) {
	tempDir := t.TempDir()
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	session, err := cm.GetOrCreateSession("missing-chunk", "f.txt", "text/plain", 2, 4, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.SaveChunk(session, 0, []byte("aa")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(session.TempDir, "chunk_0")); err != nil {
		t.Fatal(err)
	}
	if err := cm.SaveChunk(session, 1, []byte("bb")); err != nil {
		t.Fatal(err)
	}

	_, _, _, err = cm.AssembleFile(session)
	if err == nil {
		t.Fatal("expected missing chunk error")
	}
}

func TestChunkManager_SaveChunk_ConcurrentDistinctIndexes(t *testing.T) {
	tempDir := t.TempDir()
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	const n int32 = 8
	session, err := cm.GetOrCreateSession("concurrent", "f.bin", "application/octet-stream", n, int64(n), "")
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, n)
	var wg sync.WaitGroup
	for i := int32(0); i < n; i++ {
		wg.Add(1)
		go func(idx int32) {
			defer wg.Done()
			errCh <- cm.SaveChunk(session, idx, []byte{byte(idx)})
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("SaveChunk: %v", err)
		}
	}
	if !cm.IsComplete(session) {
		t.Fatalf("expected complete session, progress=%v", cm.GetProgress(session))
	}
}

func TestChunkManager_GetOrCreateSession_ReusesPointer(t *testing.T) {
	tempDir := t.TempDir()
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := cm.GetOrCreateSession("same", "a.txt", "text/plain", 3, 30, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := cm.GetOrCreateSession("same", "other.txt", "text/plain", 9, 99, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("expected the same session instance")
	}
	if second.Filename != "a.txt" || second.TotalChunks != 3 {
		t.Fatalf("session should keep original metadata: %+v", second)
	}
}
