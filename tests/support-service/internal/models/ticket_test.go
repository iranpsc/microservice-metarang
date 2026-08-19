package models_test

import (
	"testing"

	"metarang/support-service/internal/models"
)

func TestTicket_IsClosedAndIsOpen(t *testing.T) {
	open := &models.Ticket{Status: models.TicketStatusNew}
	if open.IsClosed() || !open.IsOpen() {
		t.Fatal("new ticket should be open")
	}
	answered := &models.Ticket{Status: models.TicketStatusAnswered}
	if answered.IsClosed() || !answered.IsOpen() {
		t.Fatal("answered ticket should be open")
	}
	closed := &models.Ticket{Status: models.TicketStatusClosed}
	if !closed.IsClosed() || closed.IsOpen() {
		t.Fatal("closed ticket should be closed")
	}
}

func TestGetDepartmentTitle(t *testing.T) {
	cases := map[string]string{
		models.DeptTechnicalSupport: "پشتیبانی فنی",
		models.DeptCitizensSafety:   "امنیت شهروندان",
		models.DeptInvestment:       "سرمایه گذاری",
		models.DeptInspection:       "بازرسی",
		models.DeptProtection:       "حراست",
		models.DeptZTB:              "مدیریت کل ز ت ب",
		"unknown":                   "",
		"":                          "",
	}
	for in, want := range cases {
		if got := models.GetDepartmentTitle(in); got != want {
			t.Fatalf("dept=%q got=%q want=%q", in, got, want)
		}
	}
}

func TestTicketStatusConstants(t *testing.T) {
	if models.TicketStatusNew != 0 || models.TicketStatusAnswered != 1 || models.TicketStatusClosed != 5 {
		t.Fatalf("unexpected status constants")
	}
}
