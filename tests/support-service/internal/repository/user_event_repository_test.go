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

func userEventColumns() []string {
	return []string{"id", "user_id", "event", "ip", "device", "status", "created_at", "updated_at"}
}

func userEventReportColumns() []string {
	return []string{"id", "user_event_id", "suspecious_citizen", "event_description", "status", "closed", "created_at", "updated_at"}
}

func userEventResponseColumns() []string {
	return []string{"id", "user_event_report_id", "response", "responser_name", "created_at", "updated_at"}
}

func TestUserEventRepository_Create(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_events").WithArgs(
		uint64(3), "login", "0.0.0.0", "unknown", true,
	).WillReturnResult(sqlmock.NewResult(50, 1))
	r := repository.NewUserEventRepository(db)
	got, err := r.Create(context.Background(), &models.UserEvent{
		UserID: 3, Event: "login", IP: "0.0.0.0", Device: "unknown", Status: true,
	})
	if err != nil || got.ID != 50 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_events").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.Create(context.Background(), &models.UserEvent{UserID: 1, Event: "e"})
	if err == nil {
		t.Fatal("expected create error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_events").WillReturnResult(sqlmock.NewErrorResult(errors.New("no id")))
	r = repository.NewUserEventRepository(db)
	_, err = r.Create(context.Background(), &models.UserEvent{UserID: 1, Event: "e"})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
}

func TestUserEventRepository_GetByID_WithReportResponsesAndNilReport(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("FROM user_events").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			uint64(1), uint64(3), "login", "1.1.1.1", "ios", true, now, now,
		))
	mock.ExpectQuery("FROM user_event_reports").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userEventReportColumns()).AddRow(
			uint64(7), uint64(1), "citizen", "desc", int32(0), false, now, now,
		))
	mock.ExpectQuery("FROM user_event_report_responses").WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows(userEventResponseColumns()).AddRow(
			uint64(9), uint64(7), "ok", "Admin", now, now,
		))
	r := repository.NewUserEventRepository(db)
	got, err := r.GetByID(context.Background(), 1)
	if err != nil || got == nil || got.Report == nil || got.Report.SuspeciousCitizen == nil {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if *got.Report.SuspeciousCitizen != "citizen" || len(got.Responses) != 1 {
		t.Fatalf("report=%+v responses=%+v", got.Report, got.Responses)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_events").WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			uint64(2), uint64(3), "login", "0.0.0.0", "unknown", true, now, now,
		))
	mock.ExpectQuery("FROM user_event_reports").WillReturnError(sql.ErrNoRows)
	r = repository.NewUserEventRepository(db)
	got, err = r.GetByID(context.Background(), 2)
	if err != nil || got == nil || got.Report != nil {
		t.Fatalf("expected nil report %+v err=%v", got, err)
	}
}

func TestUserEventRepository_GetByID_NotFoundAndNestedErrors(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectQuery("FROM user_events").WillReturnError(sql.ErrNoRows)
	r := repository.NewUserEventRepository(db)
	got, err := r.GetByID(context.Background(), 99)
	if err != nil || got != nil {
		t.Fatalf("expected nil %+v err=%v", got, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_events").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected query error")
	}

	now := fixedTime()
	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_events").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			uint64(1), uint64(3), "login", "0.0.0.0", "unknown", true, now, now,
		))
	mock.ExpectQuery("FROM user_event_reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected report error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_events").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			uint64(1), uint64(3), "login", "0.0.0.0", "unknown", true, now, now,
		))
	mock.ExpectQuery("FROM user_event_reports").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userEventReportColumns()).AddRow(
			uint64(7), uint64(1), nil, "desc", int32(0), false, now, now,
		))
	mock.ExpectQuery("FROM user_event_report_responses").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected responses error")
	}
}

func TestUserEventRepository_GetByUserID(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_events").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(4))
	mock.ExpectQuery("FROM user_events").WithArgs(uint64(3), int32(10), int32(10)).
		WillReturnRows(sqlmock.NewRows(userEventColumns()).AddRow(
			uint64(2), uint64(3), "login", "0.0.0.0", "unknown", true, now, now,
		))
	r := repository.NewUserEventRepository(db)
	list, total, err := r.GetByUserID(context.Background(), 3, 2, 10)
	if err != nil || total != 4 || len(list) != 1 || list[0].Event != "login" {
		t.Fatalf("err=%v total=%d list=%+v", err, total, list)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_events").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 3, 1, 10)
	if err == nil {
		t.Fatal("expected count error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_events").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("FROM user_events").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 3, 1, 10)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_events").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("FROM user_events").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewUserEventRepository(db)
	_, _, err = r.GetByUserID(context.Background(), 3, 1, 10)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestUserEventRepository_CreateReportAndStatusAndClose(t *testing.T) {
	cit := "citizen"
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_reports").WithArgs(
		uint64(1), &cit, "desc", int32(0), false,
	).WillReturnResult(sqlmock.NewResult(88, 1))
	r := repository.NewUserEventRepository(db)
	got, err := r.CreateReport(context.Background(), &models.UserEventReport{
		UserEventID: 1, SuspeciousCitizen: &cit, EventDescription: "desc", Status: 0, Closed: false,
	})
	if err != nil || got.ID != 88 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_reports").WithArgs(
		uint64(1), nil, "desc", int32(0), false,
	).WillReturnResult(sqlmock.NewResult(89, 1))
	r = repository.NewUserEventRepository(db)
	got, err = r.CreateReport(context.Background(), &models.UserEventReport{
		UserEventID: 1, EventDescription: "desc",
	})
	if err != nil || got.ID != 89 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.CreateReport(context.Background(), &models.UserEventReport{UserEventID: 1})
	if err == nil {
		t.Fatal("expected create report error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_reports").WillReturnResult(sqlmock.NewErrorResult(errors.New("no id")))
	r = repository.NewUserEventRepository(db)
	_, err = r.CreateReport(context.Background(), &models.UserEventReport{UserEventID: 1})
	if err == nil {
		t.Fatal("expected last insert id error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE user_event_reports SET status").WithArgs(int32(1), uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r = repository.NewUserEventRepository(db)
	if err := r.UpdateReportStatus(context.Background(), 7, 1); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE user_event_reports SET status").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	if err := r.UpdateReportStatus(context.Background(), 7, 1); err == nil {
		t.Fatal("expected status error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE user_event_reports SET closed").WithArgs(uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	r = repository.NewUserEventRepository(db)
	if err := r.CloseReport(context.Background(), 7); err != nil {
		t.Fatal(err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("UPDATE user_event_reports SET closed").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	if err := r.CloseReport(context.Background(), 7); err == nil {
		t.Fatal("expected close error")
	}
}

func TestUserEventRepository_CreateReportResponse(t *testing.T) {
	db, mock := newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_report_responses").WithArgs(
		uint64(7), "ok", "Admin",
	).WillReturnResult(sqlmock.NewResult(300, 1))
	r := repository.NewUserEventRepository(db)
	got, err := r.CreateReportResponse(context.Background(), &models.UserEventReportResponse{
		UserEventReportID: 7, Response: "ok", ResponserName: "Admin",
	})
	if err != nil || got.ID != 300 {
		t.Fatalf("err=%v got=%+v", err, got)
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_report_responses").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.CreateReportResponse(context.Background(), &models.UserEventReportResponse{UserEventReportID: 7})
	if err == nil {
		t.Fatal("expected create response error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectExec("INSERT INTO user_event_report_responses").WillReturnResult(sqlmock.NewErrorResult(errors.New("no id")))
	r = repository.NewUserEventRepository(db)
	_, err = r.CreateReportResponse(context.Background(), &models.UserEventReportResponse{UserEventReportID: 7})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
}

func TestUserEventRepository_GetReportByEventID_AndResponses(t *testing.T) {
	db, mock := newSQLMock(t)
	now := fixedTime()
	mock.ExpectQuery("FROM user_event_reports").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userEventReportColumns()).AddRow(
			uint64(7), uint64(1), nil, "desc", int32(0), true, now, now,
		))
	r := repository.NewUserEventRepository(db)
	rep, err := r.GetReportByEventID(context.Background(), 1)
	if err != nil || rep == nil || rep.SuspeciousCitizen != nil || !rep.Closed {
		t.Fatalf("rep=%+v err=%v", rep, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_event_reports").WillReturnError(sql.ErrNoRows)
	r = repository.NewUserEventRepository(db)
	rep, err = r.GetReportByEventID(context.Background(), 99)
	if err != nil || rep != nil {
		t.Fatalf("expected nil %+v err=%v", rep, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_event_reports").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.GetReportByEventID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_event_report_responses").WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows(userEventResponseColumns()).AddRow(
			uint64(9), uint64(7), "ok", "Admin", now, now,
		))
	r = repository.NewUserEventRepository(db)
	list, err := r.GetReportResponses(context.Background(), 7)
	if err != nil || len(list) != 1 || list[0].ResponserName != "Admin" {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_event_report_responses").WillReturnError(sql.ErrConnDone)
	r = repository.NewUserEventRepository(db)
	_, err = r.GetReportResponses(context.Background(), 7)
	if err == nil {
		t.Fatal("expected responses query error")
	}

	db, mock = newSQLMock(t)
	mock.ExpectQuery("FROM user_event_report_responses").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	r = repository.NewUserEventRepository(db)
	_, err = r.GetReportResponses(context.Background(), 7)
	if err == nil {
		t.Fatal("expected scan error")
	}
}
