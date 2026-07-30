package ftp_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"metarang/storage-service/internal/ftp"
)

func TestMockFTPClient_UploadDownloadDeleteGenerateURL(t *testing.T) {
	tempDir := t.TempDir()
	client := ftp.NewMockFTPClient(tempDir, "http://cdn.test/uploads")

	data := []byte("mock ftp payload")
	remotePath := "nested/dir/file.txt"
	if err := client.UploadFile(remotePath, bytes.NewReader(data)); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	url := client.GenerateURL(remotePath)
	wantURL := "http://cdn.test/uploads/" + remotePath
	if url != wantURL {
		t.Fatalf("GenerateURL = %q, want %q", url, wantURL)
	}

	rc, err := client.DownloadFile(remotePath)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch")
	}

	if err := client.DeleteFile(remotePath); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, remotePath)); !os.IsNotExist(err) {
		t.Fatal("file should be removed")
	}
}

func TestMockFTPClient_DownloadMissingFile(t *testing.T) {
	client := ftp.NewMockFTPClient(t.TempDir(), "http://example.com")
	if _, err := client.DownloadFile("missing.txt"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMockFTPClient_Close(t *testing.T) {
	client := ftp.NewMockFTPClient(t.TempDir(), "http://example.com")
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewFTPClient(t *testing.T) {
	client := ftp.NewFTPClient("host", "21", "user", "pass", "http://base")
	if client == nil {
		t.Fatal("expected client")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close on unconnected client: %v", err)
	}
}
