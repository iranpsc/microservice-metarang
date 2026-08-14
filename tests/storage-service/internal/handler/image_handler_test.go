package handler_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	storagepb "metarang/shared/pb/storage"
	"metarang/storage-service/internal/ftp"
	"metarang/storage-service/internal/handler"
	"metarang/storage-service/internal/repository"
	"metarang/storage-service/internal/service"
	"metarang/storage-service/tests/internal/testutil"
)

func newImageGRPCClient(t *testing.T) (storagepb.ImageServiceClient, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	imageRepo := repository.NewImageRepository(db)
	imageService := service.NewImageService(imageRepo, ftp.NewFTPClient("localhost", "21", "", "", "http://example.com"))
	s := testutil.NewBufGRPCTestServer()
	handler.RegisterImageHandler(s.Server, imageService)
	s.Start(t)
	return storagepb.NewImageServiceClient(s.GRPCClientConn(t)), mock
}

func TestImageHandler_CreateImage(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)

	imageType := "profile"
	mock.ExpectExec("INSERT INTO images").
		WithArgs("App\\Models\\User", uint64(5), "http://example.com/a.jpg", &imageType).
		WillReturnResult(sqlmock.NewResult(11, 1))

	resp, err := client.CreateImage(ctx, &storagepb.CreateImageRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   5,
		Url:           "http://example.com/a.jpg",
		Type:          "profile",
	})
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	if resp.Id != 11 || resp.Type != "profile" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestImageHandler_CreateImage_NoType(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)

	mock.ExpectExec("INSERT INTO images").
		WithArgs("App\\Models\\User", uint64(1), "http://example.com/b.jpg", (*string)(nil)).
		WillReturnResult(sqlmock.NewResult(2, 1))

	resp, err := client.CreateImage(ctx, &storagepb.CreateImageRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   1,
		Url:           "http://example.com/b.jpg",
	})
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	if resp.Type != "" {
		t.Fatalf("expected empty type, got %q", resp.Type)
	}
}

func TestImageHandler_CreateImage_ServiceError(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)

	mock.ExpectExec("INSERT INTO images").WillReturnError(sql.ErrConnDone)

	_, err := client.CreateImage(ctx, &storagepb.CreateImageRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   1,
		Url:           "http://example.com/x.jpg",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestImageHandler_GetImages(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)
	now := time.Now()
	imageType := "profile"

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(3), "App\\Models\\User", uint64(9), "http://example.com/p.jpg", &imageType, now, now))

	resp, err := client.GetImages(ctx, &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   9,
	})
	if err != nil {
		t.Fatalf("GetImages: %v", err)
	}
	if len(resp.Images) != 1 || resp.Images[0].Url != "http://example.com/p.jpg" {
		t.Fatalf("unexpected images: %+v", resp.Images)
	}
}

func TestImageHandler_DeleteImage(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)
	now := time.Now()
	imageType := "profile"

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(4), "App\\Models\\User", uint64(1), "http://example.com/z.jpg", &imageType, now, now))
	mock.ExpectExec("DELETE FROM images WHERE id = \\?").
		WithArgs(uint64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err := client.DeleteImage(ctx, &storagepb.DeleteImageRequest{ImageId: 4})
	if err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
}

func TestImageHandler_GetImages_Error(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images").
		WillReturnError(sql.ErrConnDone)

	_, err := client.GetImages(ctx, &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   1,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestImageHandler_GetImages_NilType(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)
	now := time.Now()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(8), "App\\Models\\User", uint64(2), "http://img", nil, now, now))

	resp, err := client.GetImages(ctx, &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   2,
	})
	if err != nil || len(resp.Images) != 1 || resp.Images[0].Type != "" {
		t.Fatalf("GetImages: err=%v resp=%+v", err, resp)
	}
}

func TestImageHandler_DeleteImage_NotFound(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(99)).
		WillReturnError(sql.ErrNoRows)

	_, err := client.DeleteImage(ctx, &storagepb.DeleteImageRequest{ImageId: 99})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}
