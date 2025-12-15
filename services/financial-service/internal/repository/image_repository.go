package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ImageRepository interface {
	FindImageURLByImageable(ctx context.Context, imageableType string, imageableID uint64) (string, error)
}

type imageRepository struct {
	db *sql.DB
}

func NewImageRepository(db *sql.DB) ImageRepository {
	return &imageRepository{db: db}
}

func (r *imageRepository) FindImageURLByImageable(ctx context.Context, imageableType string, imageableID uint64) (string, error) {
	query := `
		SELECT url
		FROM images
		WHERE imageable_type = ? AND imageable_id = ?
		LIMIT 1
	`

	var url string
	err := r.db.QueryRowContext(ctx, query, imageableType, imageableID).Scan(&url)
	if err == sql.ErrNoRows {
		return "", nil // No image found, return empty string
	}
	if err != nil {
		return "", fmt.Errorf("failed to find image: %w", err)
	}

	return url, nil
}
