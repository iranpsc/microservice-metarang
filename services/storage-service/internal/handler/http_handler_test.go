package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metargb/storage-service/internal/ftp"
	"metargb/storage-service/internal/service"
)

// mockStorageService is a mock storage service for testing
type mockStorageService struct {
	handleChunkUploadFunc func(uploadID, filename, contentType string, chunkData []byte, chunkIndex, totalChunks int32, totalSize int64, uploadPath string) (bool, float64, string, string, string, error)
}

func (m *mockStorageService) HandleChunkUpload(uploadID, filename, contentType string, chunkData []byte, chunkIndex, totalChunks int32, totalSize int64, uploadPath string) (bool, float64, string, string, string, error) {
	if m.handleChunkUploadFunc != nil {
		return m.handleChunkUploadFunc(uploadID, filename, contentType, chunkData, chunkIndex, totalChunks, totalSize, uploadPath)
	}
	return false, 0, "", "", "", fmt.Errorf("not implemented")
}

func TestHTTPHandler_HandleChunkUpload(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatalf("Failed to create chunk manager: %v", err)
	}

	storageBase := filepath.Join(tempDir, "storage", "app")
	ftpClient := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	storageService := service.NewStorageService(ftpClient, chunkManager, storageBase)
	handler := NewHTTPHandler(storageService)

	t.Run("missing file field returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
		req.Header.Set("Content-Type", "multipart/form-data")
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["success"] != false {
			t.Error("Expected success to be false")
		}
	})

	t.Run("single file upload returns completed response", func(t *testing.T) {
		// Create a test file
		fileContent := []byte("test file content")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check for completed upload response format: { "path": "...", "name": "...", "mime_type": "..." }
		if response["path"] == nil {
			t.Error("Expected 'path' field in response")
		}
		if response["name"] == nil {
			t.Error("Expected 'name' field in response")
		}
		if response["mime_type"] == nil {
			t.Error("Expected 'mime_type' field in response")
		}

		// Verify path format: upload/{mime}/{date}/
		path, ok := response["path"].(string)
		if !ok {
			t.Fatal("Path should be a string")
		}
		if !strings.HasPrefix(path, "upload/") {
			t.Errorf("Path should start with 'upload/', got: %s", path)
		}
		if !strings.HasSuffix(path, "/") {
			t.Errorf("Path should end with '/', got: %s", path)
		}
	})

	t.Run("chunk upload in progress returns done percentage", func(t *testing.T) {
		// Create a test file
		fileContent := []byte("chunk 1 content")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}

		// Set chunk metadata
		writer.WriteField("upload_id", "test-upload-123")
		writer.WriteField("chunk_index", "0")
		writer.WriteField("total_chunks", "3")
		writer.WriteField("total_size", "45")
		writer.WriteField("filename", "test.txt")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check for in-progress response format: { "done": <float> }
		if response["done"] == nil {
			t.Error("Expected 'done' field in response for in-progress upload")
		}

		done, ok := response["done"].(float64)
		if !ok {
			t.Fatal("Done should be a float64")
		}
		if done < 0 || done > 100 {
			t.Errorf("Done percentage should be between 0 and 100, got: %f", done)
		}

		// Should not have path, name, or mime_type for in-progress
		if response["path"] != nil {
			t.Error("Should not have 'path' field for in-progress upload")
		}
		if response["name"] != nil {
			t.Error("Should not have 'name' field for in-progress upload")
		}
		if response["mime_type"] != nil {
			t.Error("Should not have 'mime_type' field for in-progress upload")
		}
	})

	t.Run("OPTIONS request returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/upload", nil)
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("non-POST method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/upload", nil)
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("file saved to correct location", func(t *testing.T) {
		// Create a test file with specific content type
		fileContent := []byte("test image content")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "image.jpg")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify file was saved
		path, ok := response["path"].(string)
		if !ok {
			t.Fatal("Path should be a string")
		}
		name, ok := response["name"].(string)
		if !ok {
			t.Fatal("Name should be a string")
		}

		// Construct expected file path
		// Path is like "upload/image-jpeg/2024-01-15/"
		// Name is like "image_abc123.jpg"
		// Full path should be: storage/app/upload/image-jpeg/2024-01-15/image_abc123.jpg
		expectedPath := filepath.Join(storageBase, path, name)
		expectedPath = filepath.Clean(expectedPath)

		// Check if file exists
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("File was not saved to expected location: %s", expectedPath)
		}

		// Verify file content
		savedContent, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}
		if !bytes.Equal(savedContent, fileContent) {
			t.Error("Saved file content does not match original")
		}
	})

	t.Run("response format matches Laravel API exactly", func(t *testing.T) {
		// Test completed upload response
		fileContent := []byte("test content")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "document.pdf")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify exact response format: { "path": "...", "name": "...", "mime_type": "..." }
		// Should have exactly 3 fields
		if len(response) != 3 {
			t.Errorf("Expected exactly 3 fields in response, got %d: %v", len(response), response)
		}

		// Verify field names match exactly
		requiredFields := []string{"path", "name", "mime_type"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}

		// Verify path format
		path := response["path"].(string)
		if !strings.HasPrefix(path, "upload/") {
			t.Errorf("Path should start with 'upload/', got: %s", path)
		}
		if !strings.HasSuffix(path, "/") {
			t.Errorf("Path should end with '/', got: %s", path)
		}

		// Verify mime_type format
		mimeType := response["mime_type"].(string)
		if mimeType == "" {
			t.Error("mime_type should not be empty")
		}
	})
}

func TestHTTPHandler_HandleHealthCheck(t *testing.T) {
	tempDir := t.TempDir()
	chunkManager, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	ftpClient := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	storageService := service.NewStorageService(ftpClient, chunkManager, filepath.Join(tempDir, "storage", "app"))
	handler := NewHTTPHandler(storageService)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Error("Expected status to be 'healthy'")
	}
}

// Helper function to create multipart form data
func createMultipartFormData(filename string, content []byte, fields map[string]string) (string, *bytes.Buffer, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", nil, err
	}
	if _, err := part.Write(content); err != nil {
		return "", nil, err
	}

	// Add fields
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return "", nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return "", nil, err
	}

	return writer.FormDataContentType(), body, nil
}

func TestChunkUpload_CompleteFlow(t *testing.T) {
	// Test complete chunk upload flow with multiple chunks
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatalf("Failed to create chunk manager: %v", err)
	}

	storageBase := filepath.Join(tempDir, "storage", "app")
	ftpClient := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	storageService := service.NewStorageService(ftpClient, chunkManager, storageBase)
	handler := NewHTTPHandler(storageService)

	uploadID := "test-upload-complete"
	totalChunks := int32(3)
	chunkSize := 10

	// Upload chunks 0, 1, 2
	for i := int32(0); i < totalChunks; i++ {
		chunkContent := []byte(fmt.Sprintf("chunk %d content", i))
		contentType, body, err := createMultipartFormData("test.txt", chunkContent, map[string]string{
			"upload_id":    uploadID,
			"chunk_index":  fmt.Sprintf("%d", i),
			"total_chunks": fmt.Sprintf("%d", totalChunks),
			"total_size":   fmt.Sprintf("%d", int(totalChunks)*chunkSize),
			"filename":     "test.txt",
		})
		if err != nil {
			t.Fatalf("Failed to create multipart form: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()

		handler.HandleChunkUpload(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Chunk %d upload failed with status %d: %s", i, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if i < totalChunks-1 {
			// In-progress: should have "done" field
			if response["done"] == nil {
				t.Errorf("Chunk %d: Expected 'done' field for in-progress upload", i)
			}
			done := response["done"].(float64)
			expectedDone := float64(i+1) / float64(totalChunks) * 100.0
			if done < expectedDone-1 || done > expectedDone+1 {
				t.Errorf("Chunk %d: Expected done ~%.1f, got %.1f", i, expectedDone, done)
			}
		} else {
			// Last chunk: should have completed response
			if response["path"] == nil {
				t.Error("Last chunk: Expected 'path' field in completed response")
			}
			if response["name"] == nil {
				t.Error("Last chunk: Expected 'name' field in completed response")
			}
			if response["mime_type"] == nil {
				t.Error("Last chunk: Expected 'mime_type' field in completed response")
			}
		}
	}

	// Verify final file was assembled correctly
	// Check if any file was saved in the upload directory
	files, err := filepath.Glob(filepath.Join(storageBase, "upload", "*", "*", "*"))
	if err == nil && len(files) > 0 {
		// File was saved, verify content
		savedContent, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}

		// Expected content: "chunk 0 contentchunk 1 contentchunk 2 content"
		expectedContent := []byte("chunk 0 contentchunk 1 contentchunk 2 content")
		if !bytes.Equal(savedContent, expectedContent) {
			t.Errorf("File content mismatch. Expected %d bytes, got %d bytes", len(expectedContent), len(savedContent))
		}
	} else {
		t.Error("No file was saved after complete chunk upload")
	}
}
