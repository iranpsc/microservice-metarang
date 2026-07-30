package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/handler"
	"metarang/storage-service/internal/service"
)

func setupHTTPHandler(t *testing.T) (*handler.HTTPHandler, string) {
	t.Helper()
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	storageBase := filepath.Join(tempDir, "storage", "app")
	ftpClient := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	storageService := service.NewStorageService(ftpClient, chunkManager, storageBase)
	return handler.NewHTTPHandler(storageService, storageBase), storageBase
}

func TestHTTPHandler_HandleChunkUpload(t *testing.T) {
	t.Cleanup(func() { _ = os.RemoveAll("uploads") })
	h, _ := setupHTTPHandler(t)

	t.Run("missing file field returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
		w := httptest.NewRecorder()
		h.HandleChunkUpload(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("single file upload", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		_, _ = part.Write([]byte("test file content"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		h.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response["path"] == nil || response["name"] == nil || response["mime_type"] == nil {
			t.Fatalf("unexpected response: %v", response)
		}
	})

	t.Run("in progress chunk", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		_, _ = part.Write([]byte("chunk 1"))
		_ = writer.WriteField("upload_id", "test-upload-123")
		_ = writer.WriteField("chunk_index", "0")
		_ = writer.WriteField("total_chunks", "3")
		_ = writer.WriteField("total_size", "45")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		h.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		if response["done"] == nil {
			t.Fatalf("expected done field: %v", response)
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/upload", nil)
		w := httptest.NewRecorder()
		h.HandleChunkUpload(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/upload", nil)
		w := httptest.NewRecorder()
		h.HandleChunkUpload(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", w.Code)
		}
	})
}

func TestHTTPHandler_ServeUploads(t *testing.T) {
	tempDir := t.TempDir()
	uploadRoot := filepath.Join(tempDir, "uploads")
	profileDir := filepath.Join(uploadRoot, "profile")
	_ = os.MkdirAll(profileDir, 0755)
	_ = os.WriteFile(filepath.Join(profileDir, "photo.jpg"), []byte("jpeg-bytes"), 0644)

	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	storageService := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"), chunkManager, uploadRoot)
	h := handler.NewHTTPHandler(storageService, uploadRoot)

	req := httptest.NewRequest(http.MethodGet, "/uploads/profile/photo.jpg", nil)
	w := httptest.NewRecorder()
	h.ServeUploads(w, req)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), []byte("jpeg-bytes")) {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.Bytes())
	}

	req = httptest.NewRequest(http.MethodGet, "/uploads/", nil)
	w = httptest.NewRecorder()
	h.ServeUploads(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for empty path, got %d", w.Code)
	}
}

func TestHTTPHandler_HandleHealthCheck(t *testing.T) {
	h, _ := setupHTTPHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealthCheck(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	if response["status"] != "healthy" {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestHTTPHandler_CompletedUploadPathFormat(t *testing.T) {
	t.Cleanup(func() { _ = os.RemoveAll("uploads") })
	h, _ := setupHTTPHandler(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "document.pdf")
	_, _ = part.Write([]byte("pdf"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.HandleChunkUpload(w, req)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	path, _ := response["path"].(string)
	if !strings.HasPrefix(path, "uploads/") || !strings.HasSuffix(path, "/") {
		t.Fatalf("unexpected path: %q", path)
	}
}

func TestHTTPHandler_StartHTTPServer(t *testing.T) {
	h, _ := setupHTTPHandler(t)
	go func() { _ = handler.StartHTTPServer(h, "18059") }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:18059/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("HTTP server did not become ready")
}
