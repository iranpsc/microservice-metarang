package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/support-service/internal/models"
	"metarang/support-service/internal/repository"
)

func reportColumns() []string {
	return []string{"id", "subject", "title", "content", "url", "user_id", "status", "created_at", "updated_at"}
}

func imageColumns() []string {
	return []string{"id", "imageable_type", "imageable_id", "url", "created_at", "updated_at"}
}

func TestReportRepository_Create(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO reports").WithArgs(
		"displayError", "t", "c", "https://x", uint64(9), int32(0),
	).WillReturnResult(sqlmock.NewResult(21, 1))
	r := repository.NewReportRepository(db)
	got, err := r.Create(context.Background(), &models.Report{
		Subject: "displayError", Title: "t", Content: "c", URL: "https://x", UserID: 9, Status: 0,
	})
	if err != nil || got.ID != 21 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewReportRepository(db)
	_, err = r.Create(context.Background(), &models.Report{Subject: "s", Title: "t", UserID: 1})
	if err == nil {
		t.Fatal("expected create error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO reports").WillReturnResult(sqlmock.NewErrorResult(errors.New("no id")))
	r = repository.NewReportRepository(db)
	_, err = r.Create(context.Background(), &models.Report{Subject: "s", Title: "t", UserID: 1})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
}

func TestReportRepository_GetByID_WithImagesNoRowsAndErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("FROM reports").WithArgs(uint64(6)).
		WillReturnRows(sqlmock.NewRows(reportColumns()).AddRow(
			uint64(6), "displayError", "t", "c", "https://x", uint64(9), int32(0), now, now,
		))
	mock.ExpectQuery("FROM images").WithArgs(uint64(6)).
		WillReturnRows(sqlmock.NewRows(imageColumns()).AddRow(
			uint64(1), `App\Models\Report`, uint64(6), "pic.png", now, now,
		))
	r := repository.NewReportRepository(db)
	got, err := r.GetByID(context.Background(), 6)
	if err != nil || got == nil || got.ID != 6 || len(got.Images) != 1 || got.Images[0].URL != "pic.png" {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM reports").WillReturnError(sql.ErrNoRows)
	r = repository.NewReportRepository(db)
	got, err = r.GetByID(context.Background(), 99)
	if err != nil || got != nil {
		t.Fatalf("expected nil %+v err=%v", got, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewReportRepository(db)
	_, err = r.GetByID(context.Background(), 6)
	if err == nil {
		t.Fatal("expected report query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM reports").WithArgs(uint64(6)).
		WillReturnRows(sqlmock.NewRows(reportColumns()).AddRow(
			uint64(6), "displayError", "t", "c", "https://x", uint64(9), int32(0), now, now,
		))
	mock.ExpectQuery("FROM images").WillReturnError(sql.ErrConnDone)
	r = repository.NewReportRepository(db)
	_, err = r.GetByID(context.Background(), 6)
	if err == nil {
		t.Fatal("expected images error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM reports").WithArgs(uint64(6)).
		WillReturnRows(sqlmock.NewRows(reportColumns()).AddRow(
			uint64(6), "displayError", "t", "c", "https://x", uint64(9), int32(0), now, now,
		))
	mock.ExpectQuery("FROM images").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewReportRepository(db)
	_, err = r.GetByID(context.Background(), 6)
	if err == nil {
		t.Fatal("expected image scan error")
	}
}

func TestReportRepository_GetByUserID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reports").WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(3))
	mock.ExpectQuery("FROM reports").WithArgs(uint64(9), int32(10), int32(10)).
		WillReturnRows(sqlmock.NewRows(reportColumns()).AddRow(
			uint64(2), "FPSError", "t", "c", "https://x", uint64(9), int32(0), now, now,
		))
	r := repository.NewReportRepository(db)
	list, total, err := r.GetByUserID(context.Background(), 9, 2, 10)
	if err != nil || total != 3 || len(list) != 1 || list[0].Subject != "FPSError" {
		t.Fatalf("err=%v total=%d list=%+v", err, total, list)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewReportRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 9, 1, 10)
	if err == nil {
		t.Fatal("expected count error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reports").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("FROM reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewReportRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 9, 1, 10)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reports").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("FROM reports").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewReportRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 9, 1, 10)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestReportRepository_CreateImage(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO images").WithArgs(uint64(6), "pic.png").
		WillReturnResult(sqlmock.NewResult(1, 1))
	r := repository.NewReportRepository(db)
	if err := r.CreateImage(context.Background(), 6, "pic.png"); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO images").WillReturnError(sql.ErrConnDone)
	r = repository.NewReportRepository(db)
	if err := r.CreateImage(context.Background(), 6, "pic.png"); err == nil {
		t.Fatal("expected create image error")
	}
}
