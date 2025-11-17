package models

import (
	"time"
)

// Ticket status constants matching Laravel
const (
	TicketStatusNew        = 0
	TicketStatusAnswered   = 1
	TicketStatusResolved   = 2
	TicketStatusUnresolved = 3
	TicketStatusTracking   = 4
	TicketStatusClosed     = 5
)

// Department constants matching Laravel Departments enum
const (
	DeptTechnicalSupport = "technical_support"
	DeptCitizensSafety   = "citizens_safety"
	DeptInvestment       = "investment"
	DeptInspection       = "inspection"
	DeptProtection       = "protection"
	DeptZTB              = "ztb"
)

// Ticket represents a support ticket
type Ticket struct {
	ID         uint64    `db:"id"`
	Title      string    `db:"title"`
	Content    string    `db:"content"`
	Attachment string    `db:"attachment"`
	Status     int32     `db:"status"`
	Department *string   `db:"department"`
	Importance int32     `db:"importance"`
	Code       int32     `db:"code"`
	UserID     uint64    `db:"user_id"`
	ReceiverID *uint64   `db:"reciever_id"` // Note: Laravel uses 'reciever' (typo)
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// TicketResponse represents a response to a ticket
type TicketResponse struct {
	ID            uint64    `db:"id"`
	TicketID      uint64    `db:"ticket_id"`
	Response      string    `db:"response"`
	Attachment    string    `db:"attachment"`
	ResponserName string    `db:"responser_name"`
	ResponserID   uint64    `db:"responser_id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// TicketWithRelations includes ticket with sender, receiver, and responses
type TicketWithRelations struct {
	Ticket
	SenderName           string
	SenderCode           string
	SenderProfilePhoto   *string
	ReceiverName         *string
	ReceiverCode         *string
	ReceiverProfilePhoto *string
	Responses            []TicketResponse
}

// IsClosed checks if the ticket is closed
func (t *Ticket) IsClosed() bool {
	return t.Status == TicketStatusClosed
}

// IsOpen checks if the ticket is open
func (t *Ticket) IsOpen() bool {
	return t.Status != TicketStatusClosed
}

// GetDepartmentTitle returns Persian title for department (matching Laravel)
func GetDepartmentTitle(dept string) string {
	switch dept {
	case DeptTechnicalSupport:
		return "پشتیبانی فنی"
	case DeptCitizensSafety:
		return "امنیت شهروندان"
	case DeptInvestment:
		return "سرمایه گذاری"
	case DeptInspection:
		return "بازرسی"
	case DeptProtection:
		return "حراست"
	case DeptZTB:
		return "مدیریت کل ز ت ب"
	default:
		return ""
	}
}
