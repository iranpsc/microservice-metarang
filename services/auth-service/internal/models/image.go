package models

import (
	"time"
)

// Image represents a polymorphic image record in the images table
type Image struct {
	ID            uint64    `db:"id"`
	ImageableType string    `db:"imageable_type"` // e.g., "App\\Models\\User"
	ImageableID   uint64    `db:"imageable_id"`
	URL           string    `db:"url"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
