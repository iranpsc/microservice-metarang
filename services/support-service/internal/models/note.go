// Package models defines domain types for the support service.
package models

import (
	"time"
)

type Note struct {
	ID          uint64    `db:"id"`
	Title       string    `db:"title"`
	Content     string    `db:"content"`
	Attachments []string  `db:"attachments"`
	UserID      uint64    `db:"user_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
