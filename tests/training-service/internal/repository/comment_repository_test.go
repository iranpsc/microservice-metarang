package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/training-service/internal/repository"
)

func commentColumns() []string {
	return []string{"id", "user_id", "parent_id", "commentable_type", "commentable_id", "content", "created_at", "updated_at"}
}

func addCommentRow(rows *sqlmock.Rows, id, userID, videoID uint64, parent interface{}, content string) *sqlmock.Rows {
	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	return rows.AddRow(id, userID, parent, `App\Models\Video`, videoID, content, now, now)
}

func TestCommentRepository_GetComments_CountAndQueryErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnError(sql.ErrConnDone)
	r := repository.NewCommentRepository(db)
	_, _, err := r.GetComments(context.Background(), 10, 1, 10)
	if err == nil {
		t.Fatal("count error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT").WithArgs(uint64(10)).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(int32(1)))
	mock.ExpectQuery("SELECT id, user_id, parent_id").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	_, _, err = r.GetComments(context.Background(), 10, 1, 10)
	if err == nil {
		t.Fatal("query error")
	}
}

func TestCommentRepository_GetCommentByID(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(5)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 5, 2, 10, nil, "hi"))
	r := repository.NewCommentRepository(db)
	c, err := r.GetCommentByID(context.Background(), 5)
	if err != nil || c == nil || c.Content != "hi" {
		t.Fatalf("c=%+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WillReturnError(sql.ErrNoRows)
	r = repository.NewCommentRepository(db)
	c, err = r.GetCommentByID(context.Background(), 9)
	if err != nil || c != nil {
		t.Fatalf("expected nil %+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	_, err = r.GetCommentByID(context.Background(), 9)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommentRepository_AddComment(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO comments").WithArgs(uint64(2), uint64(10), "hello").
		WillReturnResult(sqlmock.NewResult(15, 1))
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(15)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 15, 2, 10, nil, "hello"))
	r := repository.NewCommentRepository(db)
	c, err := r.AddComment(context.Background(), 10, 2, "hello")
	if err != nil || c == nil || c.ID != 15 {
		t.Fatalf("c=%+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO comments").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	_, err = r.AddComment(context.Background(), 10, 2, "hello")
	if err == nil {
		t.Fatal("expected insert error")
	}
}

func TestCommentRepository_UpdateComment(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("UPDATE comments").WithArgs("new", uint64(5), uint64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r := repository.NewCommentRepository(db)
	if err := r.UpdateComment(context.Background(), 5, 2, "new"); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE comments").WillReturnResult(sqlmock.NewResult(0, 0))
	r = repository.NewCommentRepository(db)
	if err := r.UpdateComment(context.Background(), 5, 2, "new"); err == nil {
		t.Fatal("expected not found/unauthorized")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE comments").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	if err := r.UpdateComment(context.Background(), 5, 2, "new"); err == nil {
		t.Fatal("expected exec error")
	}
}

func TestCommentRepository_DeleteComment(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(5)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 5, 2, 10, nil, "hi"))
	mock.ExpectExec("DELETE FROM interactions").WithArgs(uint64(5)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM comments").WithArgs(uint64(5)).WillReturnResult(sqlmock.NewResult(0, 1))
	r := repository.NewCommentRepository(db)
	if err := r.DeleteComment(context.Background(), 5, 2); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WillReturnError(sql.ErrNoRows)
	r = repository.NewCommentRepository(db)
	if err := r.DeleteComment(context.Background(), 5, 2); err == nil {
		t.Fatal("expected not found")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(5)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 5, 9, 10, nil, "hi"))
	r = repository.NewCommentRepository(db)
	if err := r.DeleteComment(context.Background(), 5, 2); err == nil {
		t.Fatal("expected unauthorized")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(5)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 5, 2, 10, nil, "hi"))
	mock.ExpectExec("DELETE FROM interactions").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	if err := r.DeleteComment(context.Background(), 5, 2); err == nil {
		t.Fatal("expected interaction delete error")
	}
}

func TestCommentRepository_GetReplies(t *testing.T) {
	db, mock := newSQLMock(t)
	pid := uint64(5)
	mock.ExpectQuery("SELECT COUNT").WithArgs(uint64(5)).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(int32(1)))
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(5), int32(10), int32(0)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 8, 3, 10, pid, "r"))
	r := repository.NewCommentRepository(db)
	list, total, err := r.GetReplies(context.Background(), 5, 1, 10)
	if err != nil || total != 1 || len(list) != 1 || list[0].ParentID == nil {
		t.Fatalf("err=%v total=%d n=%d", err, total, len(list))
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	_, _, err = r.GetReplies(context.Background(), 5, 1, 10)
	if err == nil {
		t.Fatal("count error")
	}
}

func TestCommentRepository_AddReply_TopLevelAndNested(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(5)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 5, 2, 10, nil, "p"))
	mock.ExpectExec("INSERT INTO comments").WithArgs(uint64(7), uint64(5), uint64(10), "reply").
		WillReturnResult(sqlmock.NewResult(20, 1))
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(20)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 20, 7, 10, uint64(5), "reply"))
	r := repository.NewCommentRepository(db)
	c, err := r.AddReply(context.Background(), 5, 7, "reply")
	if err != nil || c == nil || c.ID != 20 {
		t.Fatalf("c=%+v err=%v", c, err)
	}

	top := uint64(5)
	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(8)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 8, 2, 10, top, "nested"))
	mock.ExpectExec("INSERT INTO comments").WithArgs(uint64(7), uint64(5), uint64(10), "r2").
		WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(21)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 21, 7, 10, top, "r2"))
	r = repository.NewCommentRepository(db)
	c, err = r.AddReply(context.Background(), 8, 7, "r2")
	if err != nil || c.ID != 21 {
		t.Fatalf("nested c=%+v err=%v", c, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WillReturnError(sql.ErrNoRows)
	r = repository.NewCommentRepository(db)
	_, err = r.AddReply(context.Background(), 1, 2, "x")
	if err == nil {
		t.Fatal("expected parent not found")
	}
}

func TestCommentRepository_UpdateReplyAndDeleteReplyDelegate(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("UPDATE comments").WithArgs("z", uint64(8), uint64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r := repository.NewCommentRepository(db)
	if err := r.UpdateReply(context.Background(), 8, 3, "z"); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT id, user_id, parent_id").WithArgs(uint64(8)).
		WillReturnRows(addCommentRow(sqlmock.NewRows(commentColumns()), 8, 3, 10, uint64(5), "r"))
	mock.ExpectExec("DELETE FROM interactions").WithArgs(uint64(8)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM comments").WithArgs(uint64(8)).WillReturnResult(sqlmock.NewResult(0, 1))
	r = repository.NewCommentRepository(db)
	if err := r.DeleteReply(context.Background(), 8, 3); err != nil {
		t.Fatal(err)
	}
}

func TestCommentRepository_GetCommentStats(t *testing.T) {
	db, mock := newSQLMock(t)
	for _, n := range []int32{4, 1, 2} {
		mock.ExpectQuery("SELECT COUNT").WithArgs(uint64(5)).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(n))
	}
	r := repository.NewCommentRepository(db)
	st, err := r.GetCommentStats(context.Background(), 5)
	if err != nil || st.LikesCount != 4 || st.DislikesCount != 1 || st.RepliesCount != 2 {
		t.Fatalf("st=%+v err=%v", st, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	_, err = r.GetCommentStats(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommentRepository_GetUserInteractionsForComments(t *testing.T) {
	r := repository.NewCommentRepository(nil)
	m, err := r.GetUserInteractionsForComments(context.Background(), nil, 1)
	if err != nil || len(m) != 0 {
		t.Fatalf("empty ids m=%v err=%v", m, err)
	}
	m, err = r.GetUserInteractionsForComments(context.Background(), []uint64{1}, 0)
	if err != nil || len(m) != 0 {
		t.Fatalf("zero user m=%v err=%v", m, err)
	}

	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT likeable_id, liked").WithArgs(uint64(9), uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "liked"}).AddRow(uint64(1), true).AddRow(uint64(2), false))
	r = repository.NewCommentRepository(db)
	m, err = r.GetUserInteractionsForComments(context.Background(), []uint64{1, 2}, 9)
	if err != nil || m[1] != true || m[2] != false {
		t.Fatalf("m=%v err=%v", m, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT likeable_id, liked").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	_, err = r.GetUserInteractionsForComments(context.Background(), []uint64{1}, 9)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommentRepository_ReportCommentAndReplyInteraction(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO comment_reports").WithArgs(uint64(3), uint64(10), uint64(5), "spam").
		WillReturnResult(sqlmock.NewResult(1, 1))
	r := repository.NewCommentRepository(db)
	if err := r.ReportComment(context.Background(), 10, 5, 3, "spam"); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO comment_reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	if err := r.ReportComment(context.Background(), 10, 5, 3, "spam"); err == nil {
		t.Fatal("expected report error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO interactions").WillReturnResult(sqlmock.NewResult(1, 1))
	r = repository.NewCommentRepository(db)
	if err := r.AddReplyInteraction(context.Background(), 8, 3, false, "ip"); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO interactions").WillReturnError(sql.ErrConnDone)
	r = repository.NewCommentRepository(db)
	if err := r.AddCommentInteraction(context.Background(), 8, 3, true, "ip"); err == nil {
		t.Fatal("expected interaction error")
	}
}
