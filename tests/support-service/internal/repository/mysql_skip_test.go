package repository_test

import (
	"context"
	"testing"

	"metarang/support-service/internal/repository"
	"metarang/support-service/tests/internal/testutil"
)

func TestTicketRepository_GetByID_Integration(t *testing.T) {
	db := testutil.OpenMySQLOrSkip(t)
	defer db.Close()

	r := repository.NewTicketRepository(db)
	_, err := r.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("integration GetByID: %v", err)
	}
}

func TestNoteRepository_GetByUserID_Integration(t *testing.T) {
	db := testutil.OpenMySQLOrSkip(t)
	defer db.Close()

	r := repository.NewNoteRepository(db)
	_, err := r.GetByUserID(context.Background(), 1)
	if err != nil {
		t.Fatalf("integration GetByUserID: %v", err)
	}
}

func TestReportRepository_GetByUserID_Integration(t *testing.T) {
	db := testutil.OpenMySQLOrSkip(t)
	defer db.Close()

	r := repository.NewReportRepository(db)
	_, _, err := r.GetByUserID(context.Background(), 1, 1, 5)
	if err != nil {
		t.Fatalf("integration GetByUserID: %v", err)
	}
}

func TestUserEventRepository_GetByUserID_Integration(t *testing.T) {
	db := testutil.OpenMySQLOrSkip(t)
	defer db.Close()

	r := repository.NewUserEventRepository(db)
	_, _, err := r.GetByUserID(context.Background(), 1, 1, 5)
	if err != nil {
		t.Fatalf("integration GetByUserID: %v", err)
	}
}
