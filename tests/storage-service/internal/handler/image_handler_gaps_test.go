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
)

func TestImageHandler_GetImages_EmptyList(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}))

	resp, err := client.GetImages(ctx, &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   3,
	})
	if err != nil {
		t.Fatalf("GetImages: %v", err)
	}
	if resp == nil || len(resp.Images) != 0 {
		t.Fatalf("expected empty images, got %+v", resp)
	}
}

func TestImageHandler_GetImages_TypeFilter(t *testing.T) {
	ctx := context.Background()
	client, mock := newImageGRPCClient(t)
	now := time.Now()
	imageType := "avatar"

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? AND type = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(9), "avatar").
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(3), "App\\Models\\User", uint64(9), "http://example.com/a.jpg", &imageType, now, now))

	resp, err := client.GetImages(ctx, &storagepb.GetImagesRequest{
		ImageableType: "App\\Models\\User",
		ImageableId:   9,
		Type:          "avatar",
	})
	if err != nil {
		t.Fatalf("GetImages: %v", err)
	}
	if len(resp.Images) != 1 || resp.Images[0].Type != "avatar" {
		t.Fatalf("unexpected images: %+v", resp.Images)
	}
}

func TestImageHandler_DeleteImage_RepoError(t *testing.T) {
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
		WillReturnError(sql.ErrConnDone)

	_, err := client.DeleteImage(ctx, &storagepb.DeleteImageRequest{ImageId: 4})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}
