package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	storagepb "metarang/shared/pb/storage"
	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/handler"
	"metarang/storage-service/internal/repository"
	"metarang/storage-service/internal/service"
)

const bufSize = 1024 * 1024

func newBufServer(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	register(s)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(func() { s.Stop() })

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func newStorageService(t *testing.T) *service.StorageService {
	t.Helper()
	tempDir := t.TempDir()
	cm, err := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	if err != nil {
		t.Fatal(err)
	}
	return service.NewStorageService(
		ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com"),
		cm,
		filepath.Join(tempDir, "uploads"),
	)
}

func TestStorageHandler_UploadFile(t *testing.T) {
	conn := newBufServer(t, func(s *grpc.Server) {
		handler.RegisterStorageHandler(s, newStorageService(t))
	})
	client := storagepb.NewFileStorageServiceClient(conn)

	stream, err := client.UploadFile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = stream.Send(&storagepb.UploadFileRequest{Data: &storagepb.UploadFileRequest_Metadata{Metadata: &storagepb.FileMetadata{
		Filename: "doc.txt", ContentType: "text/plain", UploadPath: "docs",
	}}})
	_ = stream.Send(&storagepb.UploadFileRequest{Data: &storagepb.UploadFileRequest_ChunkData{ChunkData: []byte("hello")}})
	resp, err := stream.CloseAndRecv()
	if err != nil || !resp.Success || resp.FileSize != 5 {
		t.Fatalf("UploadFile: err=%v resp=%+v", err, resp)
	}
}

func TestStorageHandler_UploadFile_NoMetadata(t *testing.T) {
	conn := newBufServer(t, func(s *grpc.Server) {
		handler.RegisterStorageHandler(s, newStorageService(t))
	})
	client := storagepb.NewFileStorageServiceClient(conn)

	stream, _ := client.UploadFile(context.Background())
	_ = stream.Send(&storagepb.UploadFileRequest{Data: &storagepb.UploadFileRequest_ChunkData{ChunkData: []byte("x")}})
	_, err := stream.CloseAndRecv()
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestStorageHandler_GetFilesByEntity(t *testing.T) {
	conn := newBufServer(t, func(s *grpc.Server) {
		handler.RegisterStorageHandler(s, newStorageService(t))
	})
	client := storagepb.NewFileStorageServiceClient(conn)

	resp, err := client.GetFilesByEntity(context.Background(), &storagepb.GetFilesByEntityRequest{
		ImageableType: "App\\Models\\User", ImageableId: 1,
	})
	if err != nil || len(resp.Files) != 0 {
		t.Fatalf("GetFilesByEntity: err=%v resp=%+v", err, resp)
	}
}

func TestStorageHandler_ChunkUploadValidation(t *testing.T) {
	conn := newBufServer(t, func(s *grpc.Server) {
		handler.RegisterStorageHandler(s, newStorageService(t))
	})
	client := storagepb.NewFileStorageServiceClient(conn)
	ctx := context.Background()

	_, err := client.ChunkUpload(ctx, &storagepb.ChunkUploadRequest{Filename: "a.txt", ChunkIndex: 0, TotalChunks: 1, ChunkData: []byte("x")})
	if st, _ := status.FromError(err); st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}

	inProgress, err := client.ChunkUpload(ctx, &storagepb.ChunkUploadRequest{
		UploadId: "p1", Filename: "big.bin", ContentType: "application/octet-stream",
		ChunkData: []byte("a"), ChunkIndex: 0, TotalChunks: 2, TotalSize: 2,
	})
	if err != nil || inProgress.IsFinished {
		t.Fatalf("expected in-progress chunk: err=%v resp=%+v", err, inProgress)
	}

	finished, err := client.ChunkUpload(ctx, &storagepb.ChunkUploadRequest{
		UploadId: "u1", Filename: "n.txt", ContentType: "text/plain",
		ChunkData: []byte("data"), ChunkIndex: 0, TotalChunks: 1, TotalSize: 4,
	})
	if err != nil || !finished.IsFinished {
		t.Fatalf("ChunkUpload finished: err=%v resp=%+v", err, finished)
	}
}

func TestStorageHandler_DeleteFile_Error(t *testing.T) {
	conn := newBufServer(t, func(s *grpc.Server) {
		handler.RegisterStorageHandler(s, newStorageService(t))
	})
	client := storagepb.NewFileStorageServiceClient(conn)

	_, err := client.DeleteFile(context.Background(), &storagepb.DeleteFileRequest{FilePath: "missing.txt"})
	if st, _ := status.FromError(err); st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestStorageHandler_GetFileAndDelete(t *testing.T) {
	tempDir := t.TempDir()
	uploadBase := filepath.Join(tempDir, "uploads")
	_ = os.MkdirAll(uploadBase, 0755)
	_ = os.WriteFile(filepath.Join(uploadBase, "f.txt"), []byte("data"), 0644)
	cm, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	mockFTP := ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://example.com")
	svc := service.NewStorageService(mockFTP, cm, uploadBase)
	_ = mockFTP.UploadFile("remote.txt", bytes.NewReader([]byte("remote")))

	conn := newBufServer(t, func(s *grpc.Server) { handler.RegisterStorageHandler(s, svc) })
	client := storagepb.NewFileStorageServiceClient(conn)

	stream, _ := client.GetFile(context.Background(), &storagepb.GetFileRequest{FilePath: "f.txt"})
	resp, err := stream.Recv()
	if err != nil || !bytes.Equal(resp.Data, []byte("data")) {
		t.Fatalf("GetFile local: err=%v data=%q", err, resp.GetData())
	}

	_, err = client.DeleteFile(context.Background(), &storagepb.DeleteFileRequest{FilePath: "remote.txt"})
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
}

func TestImageHandler_CreateGetDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	imageType := "profile"
	now := time.Now()
	imageService := service.NewImageService(repository.NewImageRepository(db), ftp.NewFTPClient("h", "21", "", "", "http://x"))

	conn := newBufServer(t, func(s *grpc.Server) { handler.RegisterImageHandler(s, imageService) })
	client := storagepb.NewImageServiceClient(conn)

	mock.ExpectExec("INSERT INTO images").WithArgs("App\\Models\\User", uint64(1), "http://img", &imageType).
		WillReturnResult(sqlmock.NewResult(5, 1))
	created, err := client.CreateImage(context.Background(), &storagepb.CreateImageRequest{
		ImageableType: "App\\Models\\User", ImageableId: 1, Url: "http://img", Type: "profile",
	})
	if err != nil || created.Id != 5 {
		t.Fatalf("CreateImage: err=%v resp=%+v", err, created)
	}

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(5), "App\\Models\\User", uint64(1), "http://img", &imageType, now, now))
	images, err := client.GetImages(context.Background(), &storagepb.GetImagesRequest{ImageableType: "App\\Models\\User", ImageableId: 1})
	if err != nil || len(images.Images) != 1 {
		t.Fatalf("GetImages: err=%v resp=%+v", err, images)
	}

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(5)).WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
		AddRow(uint64(5), "App\\Models\\User", uint64(1), "http://img", &imageType, now, now))
	mock.ExpectExec("DELETE FROM images WHERE id = \\?").WithArgs(uint64(5)).WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := client.DeleteImage(context.Background(), &storagepb.DeleteImageRequest{ImageId: 5}); err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
}

func TestImageHandler_GetImages_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	imageService := service.NewImageService(repository.NewImageRepository(db), nil)
	conn := newBufServer(t, func(s *grpc.Server) { handler.RegisterImageHandler(s, imageService) })
	client := storagepb.NewImageServiceClient(conn)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images").
		WillReturnError(errors.New("db down"))

	_, err = client.GetImages(context.Background(), &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User", ImageableId: 1,
	})
	if st, _ := status.FromError(err); st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestHTTPHandler_RegisterRoutesAndServeUploads(t *testing.T) {
	tempDir := t.TempDir()
	uploadRoot := filepath.Join(tempDir, "uploads")
	_ = os.MkdirAll(filepath.Join(uploadRoot, "profile"), 0755)
	_ = os.WriteFile(filepath.Join(uploadRoot, "profile", "a.jpg"), []byte("jpg"), 0644)
	cm, _ := service.NewChunkManager(filepath.Join(tempDir, "chunks"))
	svc := service.NewStorageService(ftp.NewMockFTPClient(filepath.Join(tempDir, "ftp"), "http://x"), cm, uploadRoot)
	h := handler.NewHTTPHandler(svc, uploadRoot)

	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}

	req := httptest.NewRequest(http.MethodGet, "/uploads/../bad", nil)
	w := httptest.NewRecorder()
	h.ServeUploads(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for traversal, got %d", w.Code)
	}
}

func TestImageHandler_GetImages_NilType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	imageService := service.NewImageService(repository.NewImageRepository(db), nil)
	conn := newBufServer(t, func(s *grpc.Server) { handler.RegisterImageHandler(s, imageService) })
	client := storagepb.NewImageServiceClient(conn)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(8), "App\\Models\\User", uint64(2), "http://img", nil, now, now))

	resp, err := client.GetImages(context.Background(), &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User", ImageableId: 2,
	})
	if err != nil || len(resp.Images) != 1 || resp.Images[0].Type != "" {
		t.Fatalf("GetImages: err=%v resp=%+v", err, resp)
	}
}
