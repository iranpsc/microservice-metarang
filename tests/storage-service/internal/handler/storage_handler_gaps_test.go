package handler_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	storagepb "metarang/shared/pb/storage"
	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/handler"
	"metarang/storage-service/internal/service"
	"metarang/storage-service/tests/internal/testutil"
)

func TestStorageHandler_UploadFile_FTPError(t *testing.T) {
	tempDir := t.TempDir()
	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(
		&testutil.FailingFTPClient{UploadErr: errors.New("ftp stor failed")},
		chunkManager,
		filepath.Join(tempDir, "uploads"),
	)

	s := testutil.NewBufGRPCTestServer()
	handler.RegisterStorageHandler(s.Server, svc)
	s.Start(t)
	client := storagepb.NewFileStorageServiceClient(s.GRPCClientConn(t))

	stream, err := client.UploadFile(context.Background())
	if err != nil {
		t.Fatalf("UploadFile stream: %v", err)
	}
	if err := stream.Send(&storagepb.UploadFileRequest{
		Data: &storagepb.UploadFileRequest_Metadata{
			Metadata: &storagepb.FileMetadata{
				Filename:    "doc.txt",
				ContentType: "text/plain",
				UploadPath:  "docs",
			},
		},
	}); err != nil {
		t.Fatalf("Send metadata: %v", err)
	}
	if err := stream.Send(&storagepb.UploadFileRequest{
		Data: &storagepb.UploadFileRequest_ChunkData{ChunkData: []byte("hello")},
	}); err != nil {
		t.Fatalf("Send chunk: %v", err)
	}
	_, err = stream.CloseAndRecv()
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestStorageHandler_ChunkUpload_HandleFailure(t *testing.T) {
	tempDir := t.TempDir()
	chunkDir := filepath.Join(tempDir, "chunks")
	chunkManager, err := service.NewChunkManager(chunkDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chunkDir, "blocked-chunk"), []byte("not-a-dir"), 0644); err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(
		ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"),
		chunkManager,
		filepath.Join(tempDir, "uploads"),
	)

	s := testutil.NewBufGRPCTestServer()
	handler.RegisterStorageHandler(s.Server, svc)
	s.Start(t)
	client := storagepb.NewFileStorageServiceClient(s.GRPCClientConn(t))

	_, err = client.ChunkUpload(context.Background(), &storagepb.ChunkUploadRequest{
		UploadId:    "blocked-chunk",
		Filename:    "note.txt",
		ContentType: "text/plain",
		ChunkData:   []byte("chunk-bytes"),
		ChunkIndex:  0,
		TotalChunks: 1,
		TotalSize:   11,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestStorageHandler_GetFile_ContentType(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	if err := os.MkdirAll(uploadBase, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploadBase, "photo.png"), []byte("png-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	chunkManager, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewStorageService(
		ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"),
		chunkManager,
		uploadBase,
	)

	s := testutil.NewBufGRPCTestServer()
	handler.RegisterStorageHandler(s.Server, svc)
	s.Start(t)
	client := storagepb.NewFileStorageServiceClient(s.GRPCClientConn(t))

	stream, err := client.GetFile(context.Background(), &storagepb.GetFileRequest{FilePath: "photo.png"})
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if !bytes.Equal(resp.Data, []byte("png-bytes")) {
		t.Fatalf("data = %q", resp.Data)
	}
	if resp.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", resp.ContentType)
	}
	if resp.FileSize != int64(len("png-bytes")) {
		t.Fatalf("FileSize = %d", resp.FileSize)
	}
}
