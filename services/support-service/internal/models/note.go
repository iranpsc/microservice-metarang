package models

import (
	"time"
)

// Note represents a personal note
type Note struct {
	ID         uint64    `db:"id"`
	Title      string    `db:"title"`
	Content    string    `db:"content"`
	Attachment string    `db:"attachment"`
	UserID     uint64    `db:"user_id"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
