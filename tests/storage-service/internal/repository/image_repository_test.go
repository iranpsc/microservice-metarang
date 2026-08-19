package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/storage-service/internal/models"
	"metarang/storage-service/internal/repository"
)

func TestImageRepository_CreateImage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	imageType := "profile"
	image := &models.Image{
		ImageableType: "App\\Models\\User",
		ImageableID:   42,
		URL:           "http://example.com/photo.jpg",
		Type:          &imageType,
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO images (imageable_type, imageable_id, url, type, created_at, updated_at)`)).
		WithArgs(image.ImageableType, image.ImageableID, image.URL, image.Type).
		WillReturnResult(sqlmock.NewResult(7, 1))

	repo := repository.NewImageRepository(db)
	if err := repo.CreateImage(context.Background(), image); err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	if image.ID != 7 {
		t.Fatalf("ID = %d, want 7", image.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestImageRepository_CreateImage_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	image := &models.Image{ImageableType: "App\\Models\\User", ImageableID: 1, URL: "http://x/y.jpg"}
	mock.ExpectExec("INSERT INTO images").WillReturnError(sql.ErrConnDone)

	repo := repository.NewImageRepository(db)
	if err := repo.CreateImage(context.Background(), image); err == nil {
		t.Fatal("expected error")
	}
}

func TestImageRepository_GetImages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	imageType := "profile"
	rows := sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
		AddRow(uint64(1), "App\\Models\\User", uint64(42), "http://example.com/a.jpg", &imageType, now, now)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? AND type = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(42), "profile").
		WillReturnRows(rows)

	repo := repository.NewImageRepository(db)
	images, err := repo.GetImages(context.Background(), "App\\Models\\User", 42, "profile")
	if err != nil {
		t.Fatalf("GetImages: %v", err)
	}
	if len(images) != 1 || images[0].URL != "http://example.com/a.jpg" {
		t.Fatalf("unexpected images: %+v", images)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestImageRepository_GetImages_NoTypeFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\Feature", uint64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}))

	repo := repository.NewImageRepository(db)
	images, err := repo.GetImages(context.Background(), "App\\Models\\Feature", 99, "")
	if err != nil {
		t.Fatalf("GetImages: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("expected empty slice, got %d", len(images))
	}
}

func TestImageRepository_GetImageByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
		AddRow(uint64(5), "App\\Models\\User", uint64(1), "http://example.com/x.jpg", nil, now, now)

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(5)).
		WillReturnRows(rows)

	repo := repository.NewImageRepository(db)
	image, err := repo.GetImageByID(context.Background(), 5)
	if err != nil || image == nil || image.ID != 5 {
		t.Fatalf("GetImageByID: err=%v image=%+v", err, image)
	}
}

func TestImageRepository_GetImageByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(999)).
		WillReturnError(sql.ErrNoRows)

	repo := repository.NewImageRepository(db)
	image, err := repo.GetImageByID(context.Background(), 999)
	if err != nil || image != nil {
		t.Fatalf("GetImageByID: err=%v image=%+v", err, image)
	}
}

func TestImageRepository_CreateImage_LastInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	image := &models.Image{ImageableType: "App\\Models\\User", ImageableID: 1, URL: "http://x/y.jpg"}
	mock.ExpectExec("INSERT INTO images").
		WillReturnResult(sqlmock.NewErrorResult(errors.New("no last insert id")))

	repo := repository.NewImageRepository(db)
	if err := repo.CreateImage(context.Background(), image); err == nil {
		t.Fatal("expected error")
	}
}

func TestImageRepository_GetImages_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WillReturnError(errors.New("query failed"))

	repo := repository.NewImageRepository(db)
	if _, err := repo.GetImages(context.Background(), "App\\Models\\User", 1, ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestImageRepository_DeleteImage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM images WHERE id = \\?").
		WithArgs(uint64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := repository.NewImageRepository(db)
	if err := repo.DeleteImage(context.Background(), 3); err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
