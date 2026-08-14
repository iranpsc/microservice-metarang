package ftp_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"metarang/storage-service/internal/ftp"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestMockFTPClient_UploadFile_WriteError(t *testing.T) {
	client := ftp.NewMockFTPClient(t.TempDir(), "http://example.com")
	if err := client.UploadFile("fail.txt", errReader{}); err == nil {
		t.Fatal("expected upload error")
	}
}

func TestMockFTPClient_UploadFile_InvalidDir(t *testing.T) {
	client := ftp.NewMockFTPClient(t.TempDir(), "http://example.com")
	if err := client.UploadFile(string(filepath.Separator), bytes.NewReader([]byte("x"))); err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestMockFTPClient_DeleteMissingFile(t *testing.T) {
	client := ftp.NewMockFTPClient(t.TempDir(), "http://example.com")
	if err := client.DeleteFile("missing.txt"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestMockFTPClient_UploadCreatesNestedDirs(t *testing.T) {
	base := t.TempDir()
	client := ftp.NewMockFTPClient(base, "http://example.com")
	data := []byte("nested")
	if err := client.UploadFile(filepath.Join("a", "b", "c.txt"), bytes.NewReader(data)); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	rc, err := client.DownloadFile(filepath.Join("a", "b", "c.txt"))
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch")
	}
	if _, err := os.Stat(filepath.Join(base, "a", "b", "c.txt")); err != nil {
		t.Fatal(err)
	}
}
