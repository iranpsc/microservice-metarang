package service_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/storage-service/internal/repository"
	"metarang/storage-service/internal/service"
)

func TestImageService_CreateImage_RepoError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO images").WillReturnError(sql.ErrConnDone)

	svc := service.NewImageService(repository.NewImageRepository(db), nil)
	imageType := "profile"
	_, err = svc.CreateImage(context.Background(), "App\\Models\\User", 1, "http://example.com/a.jpg", &imageType)
	if err == nil || !strings.Contains(err.Error(), "failed to create image") {
		t.Fatalf("expected wrapped create error, got %v", err)
	}
}

func TestImageService_GetImages_TypeFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	imageType := "kyc"
	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? AND type = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(8), "kyc").
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(2), "App\\Models\\User", uint64(8), "http://example.com/k.jpg", &imageType, now, now))

	svc := service.NewImageService(repository.NewImageRepository(db), nil)
	images, err := svc.GetImages(context.Background(), "App\\Models\\User", 8, "kyc")
	if err != nil || len(images) != 1 || images[0].URL != "http://example.com/k.jpg" {
		t.Fatalf("GetImages: err=%v images=%+v", err, images)
	}
}

func TestImageService_DeleteImage_GetDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(7)).
		WillReturnError(sql.ErrConnDone)

	svc := service.NewImageService(repository.NewImageRepository(db), nil)
	err = svc.DeleteImage(context.Background(), 7)
	if err == nil || !strings.Contains(err.Error(), "failed to get image") {
		t.Fatalf("expected get image error, got %v", err)
	}
}

func TestImageService_DeleteImage_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	imageType := "profile"
	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow(uint64(5), "App\\Models\\User", uint64(1), "http://example.com/z.jpg", &imageType, now, now))
	mock.ExpectExec("DELETE FROM images WHERE id = \\?").
		WithArgs(uint64(5)).
		WillReturnError(sql.ErrConnDone)

	svc := service.NewImageService(repository.NewImageRepository(db), nil)
	err = svc.DeleteImage(context.Background(), 5)
	if err == nil || !strings.Contains(err.Error(), "failed to delete image record") {
		t.Fatalf("expected delete record error, got %v", err)
	}
}
