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

func noteColumns() []string {
	return []string{"id", "title", "content", "attachments", "user_id", "created_at", "updated_at"}
}

func TestNoteRepository_Create_WithAndWithoutAttachments(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO notes").WithArgs("t", "c", `["a.png","b.pdf"]`, uint64(4)).
		WillReturnResult(sqlmock.NewResult(11, 1))
	r := repository.NewNoteRepository(db)
	got, err := r.Create(context.Background(), &models.Note{
		Title: "t", Content: "c", Attachments: []string{"a.png", "b.pdf"}, UserID: 4,
	})
	if err != nil || got.ID != 11 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO notes").WithArgs("t", "c", nil, uint64(4)).
		WillReturnResult(sqlmock.NewResult(12, 1))
	r = repository.NewNoteRepository(db)
	got, err = r.Create(context.Background(), &models.Note{Title: "t", Content: "c", UserID: 4})
	if err != nil || got.ID != 12 {
		t.Fatalf("err=%v got=%+v", err, got)
	}
}

func TestNoteRepository_Create_ExecAndLastInsertIDErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO notes").WillReturnError(sql.ErrConnDone)
	r := repository.NewNoteRepository(db)
	_, err := r.Create(context.Background(), &models.Note{Title: "t", Content: "c", UserID: 1})
	if err == nil {
		t.Fatal("expected create error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO notes").WillReturnResult(sqlmock.NewErrorResult(errors.New("no id")))
	r = repository.NewNoteRepository(db)
	_, err = r.Create(context.Background(), &models.Note{Title: "t", Content: "c", UserID: 1})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
}

func TestNoteRepository_GetByID_FoundEmptyInvalidJSONAndErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("FROM notes").WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows(noteColumns()).AddRow(
			uint64(2), "t", "c", `["u1"]`, uint64(4), now, now,
		))
	r := repository.NewNoteRepository(db)
	n, err := r.GetByID(context.Background(), 2)
	if err != nil || n == nil || n.Title != "t" || len(n.Attachments) != 1 {
		t.Fatalf("n=%+v err=%v", n, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows(noteColumns()).AddRow(
			uint64(3), "t", "c", nil, uint64(4), now, now,
		))
	r = repository.NewNoteRepository(db)
	n, err = r.GetByID(context.Background(), 3)
	if err != nil || n == nil || n.Attachments != nil {
		t.Fatalf("expected empty attachments %+v err=%v", n, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").WillReturnError(sql.ErrNoRows)
	r = repository.NewNoteRepository(db)
	n, err = r.GetByID(context.Background(), 9)
	if err != nil || n != nil {
		t.Fatalf("expected nil %+v err=%v", n, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").WillReturnError(sql.ErrConnDone)
	r = repository.NewNoteRepository(db)
	_, err = r.GetByID(context.Background(), 9)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows(noteColumns()).AddRow(
			uint64(4), "t", "c", "{not-json", uint64(4), now, now,
		))
	r = repository.NewNoteRepository(db)
	_, err = r.GetByID(context.Background(), 4)
	if err == nil {
		t.Fatal("expected json decode error")
	}
}

func TestNoteRepository_GetByUserID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("FROM notes").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows(noteColumns()).AddRow(
			uint64(1), "t", "c", `["a"]`, uint64(4), now, now,
		).AddRow(
			uint64(2), "t2", "c2", "", uint64(4), now, now,
		))
	r := repository.NewNoteRepository(db)
	list, err := r.GetByUserID(context.Background(), 4)
	if err != nil || len(list) != 2 || list[0].Title != "t" {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").WillReturnError(sql.ErrConnDone)
	r = repository.NewNoteRepository(db)
	_, err = r.GetByUserID(context.Background(), 4)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewNoteRepository(db)
	_, err = r.GetByUserID(context.Background(), 4)
	if err == nil {
		t.Fatal("expected scan error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM notes").
		WillReturnRows(sqlmock.NewRows(noteColumns()).AddRow(
			uint64(1), "t", "c", "{bad", uint64(4), now, now,
		))
	r = repository.NewNoteRepository(db)
	_, err = r.GetByUserID(context.Background(), 4)
	if err == nil {
		t.Fatal("expected attachments json error")
	}
}

func TestNoteRepository_UpdateDeleteAndOwnership(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("UPDATE notes").WithArgs("n", "nc", `["z"]`, uint64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r := repository.NewNoteRepository(db)
	if err := r.Update(context.Background(), &models.Note{
		ID: 2, Title: "n", Content: "nc", Attachments: []string{"z"},
	}); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE notes").WithArgs("n", "nc", nil, uint64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r = repository.NewNoteRepository(db)
	if err := r.Update(context.Background(), &models.Note{ID: 2, Title: "n", Content: "nc"}); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE notes").WillReturnError(sql.ErrConnDone)
	r = repository.NewNoteRepository(db)
	if err := r.Update(context.Background(), &models.Note{ID: 2, Title: "n", Content: "nc"}); err == nil {
		t.Fatal("expected update error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("DELETE FROM notes").WithArgs(uint64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r = repository.NewNoteRepository(db)
	if err := r.Delete(context.Background(), 2); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("DELETE FROM notes").WillReturnError(sql.ErrConnDone)
	r = repository.NewNoteRepository(db)
	if err := r.Delete(context.Background(), 2); err == nil {
		t.Fatal("expected delete error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM notes").WithArgs(uint64(2), uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	r = repository.NewNoteRepository(db)
	ok, err := r.CheckUserOwnership(context.Background(), 2, 4)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM notes").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	r = repository.NewNoteRepository(db)
	ok, err = r.CheckUserOwnership(context.Background(), 2, 9)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM notes").WillReturnError(sql.ErrConnDone)
	r = repository.NewNoteRepository(db)
	_, err = r.CheckUserOwnership(context.Background(), 2, 4)
	if err == nil {
		t.Fatal("expected ownership error")
	}
}
