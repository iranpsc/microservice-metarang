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

func TestTicketRepository_Create_WithAndWithoutReceiver(t *testing.T) {
	db, mock := newSQLMock(t)
	rid := uint64(20)
	dept := models.DeptTechnicalSupport
	mock.ExpectExec("INSERT INTO tickets").WithArgs(
		"hello", "body", "a.png", int32(models.TicketStatusNew), &dept, int32(0), int32(123456), uint64(10), &rid,
	).WillReturnResult(sqlmock.NewResult(42, 1))

	r := repository.NewTicketRepository(db)
	got, err := r.Create(context.Background(), &models.Ticket{
		Title: "hello", Content: "body", Attachment: "a.png", Status: models.TicketStatusNew,
		Department: &dept, Importance: 0, Code: 123456, UserID: 10, ReceiverID: &rid,
	})
	if err != nil || got.ID != 42 {
		t.Fatalf("err=%v id=%d", err, got.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO tickets").WithArgs(
		"hello", "body", "", int32(0), nil, int32(0), int32(100000), uint64(3), nil,
	).WillReturnResult(sqlmock.NewResult(7, 1))
	r = repository.NewTicketRepository(db)
	got, err = r.Create(context.Background(), &models.Ticket{
		Title: "hello", Content: "body", UserID: 3, Code: 100000,
	})
	if err != nil || got.ID != 7 {
		t.Fatalf("err=%v id=%d", err, got.ID)
	}
}

func TestTicketRepository_Create_ExecAndLastInsertIDErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO tickets").WillReturnError(sql.ErrConnDone)
	r := repository.NewTicketRepository(db)
	_, err := r.Create(context.Background(), &models.Ticket{Title: "t", Content: "c", UserID: 1})
	if err == nil {
		t.Fatal("expected create error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO tickets").WillReturnResult(sqlmock.NewErrorResult(errors.New("no insert id")))
	r = repository.NewTicketRepository(db)
	_, err = r.Create(context.Background(), &models.Ticket{Title: "t", Content: "c", UserID: 1})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
}

func TestTicketRepository_GetByID_FoundWithResponsesAndReceiver(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	rid := uint64(8)
	mock.ExpectQuery("FROM tickets t").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows(ticketJoinColumns()).AddRow(
			uint64(3), "t", "c", "att.png", int32(0), "technical_support", int32(0), int32(111111),
			uint64(5), rid, now, now,
			"Alice", "A1", "Bob", "B1",
			"sp.jpg", "rp.jpg",
		))
	mock.ExpectQuery("FROM ticket_responses").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows(responseColumns()).AddRow(
			uint64(1), uint64(3), "hi", "", "Alice", uint64(5), now, now,
		))

	r := repository.NewTicketRepository(db)
	got, err := r.GetByID(context.Background(), 3)
	if err != nil || got == nil || got.ID != 3 || got.ReceiverID == nil || *got.ReceiverID != 8 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if got.ReceiverName == nil || *got.ReceiverName != "Bob" || got.ReceiverCode == nil || *got.ReceiverCode != "B1" {
		t.Fatalf("receiver names %+v", got)
	}
	if len(got.Responses) != 1 || got.Responses[0].Response != "hi" {
		t.Fatalf("responses=%+v", got.Responses)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTicketRepository_GetByID_NoRowsQueryErrorAndResponsesError(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("FROM tickets t").WillReturnError(sql.ErrNoRows)
	r := repository.NewTicketRepository(db)
	got, err := r.GetByID(context.Background(), 9)
	if err != nil || got != nil {
		t.Fatalf("expected nil ticket, got %+v err=%v", got, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM tickets t").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	_, err = r.GetByID(context.Background(), 9)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM tickets t").WithArgs(uint64(3)).
		WillReturnRows(addTicketJoinRow(sqlmock.NewRows(ticketJoinColumns()), 3, 5, nil, nil))
	mock.ExpectQuery("FROM ticket_responses").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	_, err = r.GetByID(context.Background(), 3)
	if err == nil {
		t.Fatal("expected responses error")
	}
}

func TestTicketRepository_GetByUserID_SentAndReceivedPagination(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE user_id").WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(12))
	mock.ExpectQuery("WHERE t.user_id").WithArgs(uint64(10), int32(10), int32(10)).
		WillReturnRows(addTicketJoinRow(sqlmock.NewRows(ticketJoinColumns()), 2, 10, nil, "technical_support"))

	r := repository.NewTicketRepository(db)
	list, total, err := r.GetByUserID(context.Background(), 10, 2, 10, false)
	if err != nil || total != 12 || len(list) != 1 || list[0].ID != 2 {
		t.Fatalf("err=%v total=%d n=%d", err, total, len(list))
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE reciever_id").WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("WHERE t.reciever_id").WithArgs(uint64(10), int32(5), int32(0)).
		WillReturnRows(sqlmock.NewRows(ticketJoinColumns()).AddRow(
			uint64(4), "t", "c", "", int32(0), nil, int32(0), int32(1),
			uint64(1), uint64(10), fixedTime(), fixedTime(),
			"Alice", "A1", "Bob", "B1",
			nil, nil,
		))
	r = repository.NewTicketRepository(db)
	list, total, err = r.GetByUserID(context.Background(), 10, 1, 5, true)
	if err != nil || total != 1 || len(list) != 1 || list[0].ReceiverID == nil || *list[0].ReceiverID != 10 {
		t.Fatalf("err=%v total=%d list=%+v", err, total, list)
	}
}

func TestTicketRepository_GetByUserID_CountQueryAndScanErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE user_id").WillReturnError(sql.ErrConnDone)
	r := repository.NewTicketRepository(db)
	_, _, err := r.GetByUserID(context.Background(), 1, 1, 10, false)
	if err == nil {
		t.Fatal("expected count error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE reciever_id").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("WHERE t.reciever_id").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 1, 1, 10, true)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE user_id").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("WHERE t.user_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewTicketRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 1, 1, 10, false)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestTicketRepository_UpdateAndUpdateStatus(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("UPDATE tickets").WithArgs("nt", "nc", "x.png", int32(models.TicketStatusNew), uint64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r := repository.NewTicketRepository(db)
	if err := r.Update(context.Background(), &models.Ticket{
		ID: 9, Title: "nt", Content: "nc", Attachment: "x.png", Status: models.TicketStatusNew,
	}); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE tickets").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	if err := r.Update(context.Background(), &models.Ticket{ID: 1, Title: "t", Content: "c"}); err == nil {
		t.Fatal("expected update error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE tickets SET status").WithArgs(int32(models.TicketStatusClosed), uint64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r = repository.NewTicketRepository(db)
	if err := r.UpdateStatus(context.Background(), 4, models.TicketStatusClosed); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE tickets SET status").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	if err := r.UpdateStatus(context.Background(), 4, models.TicketStatusClosed); err == nil {
		t.Fatal("expected status error")
	}
}

func TestTicketRepository_GetResponsesByTicketID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("FROM ticket_responses").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows(responseColumns()).AddRow(
			uint64(1), uint64(3), "hi", "a.png", "Bob", uint64(2), now, now,
		))
	r := repository.NewTicketRepository(db)
	list, err := r.GetResponsesByTicketID(context.Background(), 3)
	if err != nil || len(list) != 1 || list[0].ResponserName != "Bob" {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM ticket_responses").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	_, err = r.GetResponsesByTicketID(context.Background(), 3)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM ticket_responses").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewTicketRepository(db)
	_, err = r.GetResponsesByTicketID(context.Background(), 3)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestTicketRepository_CreateResponse(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO ticket_responses").WithArgs(
		uint64(3), "ok", "a.png", "Alice", uint64(5),
	).WillReturnResult(sqlmock.NewResult(15, 1))
	r := repository.NewTicketRepository(db)
	got, err := r.CreateResponse(context.Background(), &models.TicketResponse{
		TicketID: 3, Response: "ok", Attachment: "a.png", ResponserName: "Alice", ResponserID: 5,
	})
	if err != nil || got.ID != 15 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO ticket_responses").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	_, err = r.CreateResponse(context.Background(), &models.TicketResponse{TicketID: 1, Response: "x"})
	if err == nil {
		t.Fatal("expected create error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO ticket_responses").WillReturnResult(sqlmock.NewErrorResult(errors.New("no id")))
	r = repository.NewTicketRepository(db)
	_, err = r.CreateResponse(context.Background(), &models.TicketResponse{TicketID: 1, Response: "x"})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
}

func TestTicketRepository_CheckUserOwnership(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE id").WithArgs(uint64(3), uint64(5), uint64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	r := repository.NewTicketRepository(db)
	ok, err := r.CheckUserOwnership(context.Background(), 3, 5)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE id").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	r = repository.NewTicketRepository(db)
	ok, err = r.CheckUserOwnership(context.Background(), 3, 9)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tickets WHERE id").WillReturnError(sql.ErrConnDone)
	r = repository.NewTicketRepository(db)
	_, err = r.CheckUserOwnership(context.Background(), 3, 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTicketRepository_GetTicketSenderReceiver_NilReceiverAndError(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("SELECT user_id, reciever_id FROM tickets").WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "reciever_id"}).AddRow(uint64(1), nil))
	r := repository.NewTicketRepository(db)
	sender, receiver, err := r.GetTicketSenderReceiver(context.Background(), 4)
	if err != nil || sender != 1 || receiver != 0 {
		t.Fatalf("sender=%d receiver=%d err=%v", sender, receiver, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT user_id, reciever_id FROM tickets").WithArgs(uint64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "reciever_id"}).AddRow(uint64(2), uint64(9)))
	r = repository.NewTicketRepository(db)
	sender, receiver, err = r.GetTicketSenderReceiver(context.Background(), 5)
	if err != nil || sender != 2 || receiver != 9 {
		t.Fatalf("sender=%d receiver=%d err=%v", sender, receiver, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT user_id, reciever_id FROM tickets").WillReturnError(sql.ErrNoRows)
	r = repository.NewTicketRepository(db)
	_, _, err = r.GetTicketSenderReceiver(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error")
	}
}
