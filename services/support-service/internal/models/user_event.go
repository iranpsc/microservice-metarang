package models

import (
	"time"
)

// UserEvent represents a user event
type UserEvent struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	Event     string    `db:"event"`
	IP        string    `db:"ip"`
	Device    string    `db:"device"`
	Status    bool      `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// UserEventReport represents a report for a user event
type UserEventReport struct {
	ID                uint64    `db:"id"`
	UserEventID       uint64    `db:"user_event_id"`
	SuspeciousCitizen *string   `db:"suspecious_citizen"` // Note: Laravel uses 'suspecious' (typo)
	EventDescription  string    `db:"event_description"`
	Status            int32     `db:"status"`
	Closed            bool      `db:"closed"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// UserEventReportResponse represents a response to a user event report
type UserEventReportResponse struct {
	ID                uint64    `db:"id"`
	UserEventReportID uint64    `db:"user_event_report_id"`
	Response          string    `db:"response"`
	ResponserName     string    `db:"responser_name"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// UserEventWithReport includes event with its report
type UserEventWithReport struct {
	UserEvent
	Report    *UserEventReport
	Responses []UserEventReportResponse
}
