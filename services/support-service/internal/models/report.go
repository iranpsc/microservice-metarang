package models

import (
	"time"
)

// Report represents a general user report
type Report struct {
	ID        uint64    `db:"id"`
	Subject   string    `db:"subject"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	URL       string    `db:"url"`
	UserID    uint64    `db:"user_id"`
	Status    int32     `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// ReportWithImages includes report with its associated images
type ReportWithImages struct {
	Report
	Images []Image
}

// Image represents an image attached to a report
type Image struct {
	ID            uint64    `db:"id"`
	ImageableType string    `db:"imageable_type"`
	ImageableID   uint64    `db:"imageable_id"`
	URL           string    `db:"url"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// CommentReport represents a report for a comment
type CommentReport struct {
	ID              uint64    `db:"id"`
	UserID          uint64    `db:"user_id"`
	CommentableType string    `db:"commentable_type"`
	CommentableID   uint64    `db:"commentable_id"`
	CommentID       uint64    `db:"comment_id"`
	Content         string    `db:"content"`
	Status          int32     `db:"status"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}
