package repository_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func fixedTime() time.Time {
	return time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
}

func ticketJoinColumns() []string {
	return []string{
		"id", "title", "content", "attachment", "status", "department", "importance", "code",
		"user_id", "reciever_id", "created_at", "updated_at",
		"sender_name", "sender_code", "receiver_name", "receiver_code",
		"sender_photo_url", "receiver_photo_url",
	}
}

func addTicketJoinRow(rows *sqlmock.Rows, id, userID uint64, receiverID interface{}, dept interface{}) *sqlmock.Rows {
	now := fixedTime()
	return rows.AddRow(
		id, "t", "c", "att.png", int32(0), dept, int32(0), int32(111111),
		userID, receiverID, now, now,
		"Alice", "A1", nil, nil,
		nil, nil,
	)
}

func responseColumns() []string {
	return []string{"id", "ticket_id", "response", "attachment", "responser_name", "responser_id", "created_at", "updated_at"}
}
