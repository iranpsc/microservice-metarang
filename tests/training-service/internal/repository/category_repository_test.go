package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/training-service/internal/repository"
)

func categoryColumns() []string {
	return []string{"id", "name", "slug", "description", "image", "icon", "created_at", "updated_at"}
}

func subCategoryColumns() []string {
	return []string{"id", "video_category_id", "name", "slug", "description", "image", "icon", "created_at", "updated_at"}
}

func TestCategoryRepository_GetCategories(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM video_categories").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(int32(1)))
	mock.ExpectQuery("SELECT id, name, slug").WithArgs(int32(30), int32(0)).
		WillReturnRows(sqlmock.NewRows(categoryColumns()).
			AddRow(uint64(1), "Basics", "basics", "d", "img.png", nil, now, now))

	r := repository.NewCategoryRepository(db)
	list, total, err := r.GetCategories(context.Background(), 1, 30)
	if err != nil || total != 1 || len(list) != 1 || list[0].Slug != "basics" {
		t.Fatalf("err=%v total=%d n=%d", err, total, len(list))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCategoryRepository_GetCategories_Errors(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM video_categories").WillReturnError(sql.ErrConnDone)
	r := repository.NewCategoryRepository(db)
	_, _, err := r.GetCategories(context.Background(), 1, 30)
	if err == nil {
		t.Fatal("count error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM video_categories").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(int32(1)))
	mock.ExpectQuery("SELECT id, name, slug").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, _, err = r.GetCategories(context.Background(), 1, 30)
	if err == nil {
		t.Fatal("query error")
	}
}

func TestCategoryRepository_GetCategoryByID_FoundNotFoundError(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	icon := "i.svg"
	mock.ExpectQuery("SELECT id, name, slug").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows(categoryColumns()).
			AddRow(uint64(3), "N", "n", "d", "img", icon, now, now))
	r := repository.NewCategoryRepository(db)
	c, err := r.GetCategoryByID(context.Background(), 3)
	if err != nil || c == nil || c.Name != "N" || c.Icon == nil {
		t.Fatalf("c=%+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, slug").WillReturnError(sql.ErrNoRows)
	r = repository.NewCategoryRepository(db)
	c, err = r.GetCategoryByID(context.Background(), 9)
	if err != nil || c != nil {
		t.Fatalf("expected nil, got %+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, slug").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetCategoryByID(context.Background(), 9)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCategoryRepository_GetCategoryBySlug(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	mock.ExpectQuery("SELECT id, name, slug").WithArgs("basics").
		WillReturnRows(sqlmock.NewRows(categoryColumns()).
			AddRow(uint64(1), "Basics", "basics", "d", "img", nil, now, now))
	r := repository.NewCategoryRepository(db)
	c, err := r.GetCategoryBySlug(context.Background(), "basics")
	if err != nil || c == nil || c.Slug != "basics" {
		t.Fatalf("c=%+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, slug").WillReturnError(sql.ErrNoRows)
	r = repository.NewCategoryRepository(db)
	c, err = r.GetCategoryBySlug(context.Background(), "missing")
	if err != nil || c != nil {
		t.Fatalf("expected nil %+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, name, slug").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetCategoryBySlug(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCategoryRepository_GetSubCategoriesByCategoryID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	mock.ExpectQuery("SELECT id, video_category_id").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows(subCategoryColumns()).
			AddRow(uint64(2), uint64(4), "Sub", "sub", "d", "img", nil, now, now))
	r := repository.NewCategoryRepository(db)
	list, err := r.GetSubCategoriesByCategoryID(context.Background(), 4)
	if err != nil || len(list) != 1 || list[0].Slug != "sub" {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, video_category_id").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetSubCategoriesByCategoryID(context.Background(), 4)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCategoryRepository_GetSubCategoryByID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	mock.ExpectQuery("SELECT id, video_category_id").WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows(subCategoryColumns()).
			AddRow(uint64(2), uint64(4), "Sub", "sub", "d", "img", "i.svg", now, now))
	r := repository.NewCategoryRepository(db)
	s, err := r.GetSubCategoryByID(context.Background(), 2)
	if err != nil || s == nil || s.Name != "Sub" {
		t.Fatalf("s=%+v err=%v", s, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, video_category_id").WillReturnError(sql.ErrNoRows)
	r = repository.NewCategoryRepository(db)
	s, err = r.GetSubCategoryByID(context.Background(), 9)
	if err != nil || s != nil {
		t.Fatalf("expected nil %+v err=%v", s, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, video_category_id").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetSubCategoryByID(context.Background(), 9)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCategoryRepository_GetSubCategoryBySlugs(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	mock.ExpectQuery("FROM video_sub_categories vsc").WithArgs("cat", "sub").
		WillReturnRows(sqlmock.NewRows(subCategoryColumns()).
			AddRow(uint64(2), uint64(4), "Sub", "sub", "d", "img", nil, now, now))
	r := repository.NewCategoryRepository(db)
	s, err := r.GetSubCategoryBySlugs(context.Background(), "cat", "sub")
	if err != nil || s == nil || s.Slug != "sub" {
		t.Fatalf("s=%+v err=%v", s, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM video_sub_categories vsc").WillReturnError(sql.ErrNoRows)
	r = repository.NewCategoryRepository(db)
	s, err = r.GetSubCategoryBySlugs(context.Background(), "a", "b")
	if err != nil || s != nil {
		t.Fatalf("expected nil %+v err=%v", s, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM video_sub_categories vsc").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetSubCategoryBySlugs(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error")
	}
}

func expectFourCounts(mock sqlmock.Sqlmock, id uint64, vals ...int32) {
	for _, v := range vals {
		mock.ExpectQuery("SELECT COUNT").WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(v))
	}
}

func TestCategoryRepository_GetCategoryStats(t *testing.T) {
	db, mock := newSQLMock(t)
	expectFourCounts(mock, 5, 4, 10, 2, 1)
	r := repository.NewCategoryRepository(db)
	st, err := r.GetCategoryStats(context.Background(), 5)
	if err != nil || st.VideosCount != 4 || st.ViewsCount != 10 || st.LikesCount != 2 || st.DislikesCount != 1 {
		t.Fatalf("st=%+v err=%v", st, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetCategoryStats(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCategoryRepository_GetSubCategoryStats(t *testing.T) {
	db, mock := newSQLMock(t)
	expectFourCounts(mock, 8, 3, 6, 1, 0)
	r := repository.NewCategoryRepository(db)
	st, err := r.GetSubCategoryStats(context.Background(), 8)
	if err != nil || st.VideosCount != 3 || st.ViewsCount != 6 {
		t.Fatalf("st=%+v err=%v", st, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetSubCategoryStats(context.Background(), 8)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCategoryRepository_GetSubCategoryStatsByCategoryID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := time.Now()
	mock.ExpectQuery("SELECT id, video_category_id").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows(subCategoryColumns()).
			AddRow(uint64(2), uint64(4), "Sub", "sub", "d", "img", nil, now, now))
	expectFourCounts(mock, 2, 1, 2, 3, 4)

	r := repository.NewCategoryRepository(db)
	m, err := r.GetSubCategoryStatsByCategoryID(context.Background(), 4)
	if err != nil || m[2] == nil || m[2].VideosCount != 1 || m[2].DislikesCount != 4 {
		t.Fatalf("m=%+v err=%v", m, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, video_category_id").WillReturnError(sql.ErrConnDone)
	r = repository.NewCategoryRepository(db)
	_, err = r.GetSubCategoryStatsByCategoryID(context.Background(), 4)
	if err == nil {
		t.Fatal("expected subcategories error")
	}
}
