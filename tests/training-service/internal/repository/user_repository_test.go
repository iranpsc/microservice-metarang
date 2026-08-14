package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/training-service/internal/repository"
)

func TestUserRepository_GetUserByID_DBFallback(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "email"}).
			AddRow(uint64(4), "Ada", "a1", "ada@example.com"))
	mock.ExpectQuery("SELECT url").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("photo.jpg"))

	r := repository.NewUserRepository(db, nil)
	u, err := r.GetUserByID(context.Background(), 4)
	if err != nil || u == nil || u.Name != "Ada" || u.ProfilePhoto != "photo.jpg" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepository_GetUserByID_NotFoundAndError(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WillReturnError(sql.ErrNoRows)
	r := repository.NewUserRepository(db, nil)
	u, err := r.GetUserByID(context.Background(), 9)
	if err != nil || u != nil {
		t.Fatalf("expected nil %+v err=%v", u, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserRepository(db, nil)
	_, err = r.GetUserByID(context.Background(), 9)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserRepository_GetUserByID_PhotoMissing(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "email"}).
			AddRow(uint64(4), "Ada", "a1", "ada@example.com"))
	mock.ExpectQuery("SELECT url").WithArgs(uint64(4)).WillReturnError(sql.ErrNoRows)

	r := repository.NewUserRepository(db, nil)
	u, err := r.GetUserByID(context.Background(), 4)
	if err != nil || u.ProfilePhoto != "" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
}

func TestUserRepository_GetUserBasicByCode_DBFallback(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WithArgs("c1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "email"}).
			AddRow(uint64(8), "Bob", "c1", "b@example.com"))
	mock.ExpectQuery("SELECT url").WithArgs(uint64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("b.png"))

	r := repository.NewUserRepository(db, nil)
	u, err := r.GetUserBasicByCode(context.Background(), "c1")
	if err != nil || u == nil || u.Code != "c1" || u.ProfilePhoto != "b.png" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
}

func TestUserRepository_GetUserBasicByCode_NotFoundAndError(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WillReturnError(sql.ErrNoRows)
	r := repository.NewUserRepository(db, nil)
	u, err := r.GetUserBasicByCode(context.Background(), "missing")
	if err != nil || u != nil {
		t.Fatalf("expected nil %+v err=%v", u, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, code, email").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserRepository(db, nil)
	_, err = r.GetUserBasicByCode(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}
