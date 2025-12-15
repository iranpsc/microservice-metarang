package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// UserRepositoryInterface defines the interface for user repository operations
type UserRepositoryInterface interface {
	GetUserBasicByCode(ctx context.Context, code string) (*UserBasic, error)
	GetUserByID(ctx context.Context, userID uint64) (*UserBasic, error)
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUserBasicByCode retrieves basic user information by code
func (r *UserRepository) GetUserBasicByCode(ctx context.Context, code string) (*UserBasic, error) {
	query := `
		SELECT id, name, code, email
		FROM users
		WHERE code = ?
		LIMIT 1
	`

	var user UserBasic
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&user.ID,
		&user.Name,
		&user.Code,
		&user.Email,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by code: %w", err)
	}

	// Get latest profile photo
	photoQuery := `
		SELECT url
		FROM images
		WHERE imageable_type = 'App\\Models\\User' AND imageable_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var photoURL sql.NullString
	r.db.QueryRowContext(ctx, photoQuery, user.ID).Scan(&photoURL)
	if photoURL.Valid {
		user.ProfilePhoto = photoURL.String
	}

	return &user, nil
}

// GetUserByID retrieves basic user information by ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID uint64) (*UserBasic, error) {
	query := `
		SELECT id, name, code, email
		FROM users
		WHERE id = ?
		LIMIT 1
	`

	var user UserBasic
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Code,
		&user.Email,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Get latest profile photo
	photoQuery := `
		SELECT url
		FROM images
		WHERE imageable_type = 'App\\Models\\User' AND imageable_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var photoURL sql.NullString
	r.db.QueryRowContext(ctx, photoQuery, user.ID).Scan(&photoURL)
	if photoURL.Valid {
		user.ProfilePhoto = photoURL.String
	}

	return &user, nil
}

// UserBasic represents basic user information
type UserBasic struct {
	ID           uint64
	Name         string
	Code         string
	Email        string
	ProfilePhoto string
}
