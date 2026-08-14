package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/storage-service/internal/repository"
)

func TestImageRepository_GetImageByID_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE id = \\?").
		WithArgs(uint64(12)).
		WillReturnError(sql.ErrConnDone)

	repo := repository.NewImageRepository(db)
	image, err := repo.GetImageByID(context.Background(), 12)
	if err == nil || image != nil {
		t.Fatalf("GetImageByID: err=%v image=%+v", err, image)
	}
}

func TestImageRepository_DeleteImage_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM images WHERE id = \\?").
		WithArgs(uint64(3)).
		WillReturnError(sql.ErrConnDone)

	repo := repository.NewImageRepository(db)
	if err := repo.DeleteImage(context.Background(), 3); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestImageRepository_GetImages_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT id, imageable_type, imageable_id, url, type, created_at, updated_at FROM images WHERE imageable_type = \\? AND imageable_id = \\? ORDER BY created_at DESC").
		WithArgs("App\\Models\\User", uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_type", "imageable_id", "url", "type", "created_at", "updated_at"}).
			AddRow("not-a-uint64", "App\\Models\\User", uint64(1), "http://x", nil, now, now))

	repo := repository.NewImageRepository(db)
	if _, err := repo.GetImages(context.Background(), "App\\Models\\User", 1, ""); err == nil {
		t.Fatal("expected scan error")
	}
}
