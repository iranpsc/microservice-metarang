package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// UserRepository provides access to user data for follow resources
type UserRepository interface {
	GetUserBasicInfo(ctx context.Context, userID uint64) (*UserBasicInfo, error)
	GetUserLevel(ctx context.Context, userID uint64) (string, error)
	IsUserOnline(ctx context.Context, userID uint64) (bool, error)
}

type UserBasicInfo struct {
	ID           uint64
	Name         string
	Code         string
	ProfilePhoto string // Latest profile photo URL; empty when none
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserBasicInfo(ctx context.Context, userID uint64) (*UserBasicInfo, error) {
	// Profile photo lives on the shared images table (same source as auth-service
	// user lists). Do not call auth-service ListProfilePhotos here: that RPC
	// authenticates as the caller and ignores the listed user's id.
	query := `
		SELECT id, name, code,
			(SELECT url FROM images
			 WHERE imageable_type = 'App\\Models\\User' AND imageable_id = users.id
			 ORDER BY created_at DESC
			 LIMIT 1) AS profile_photo
		FROM users
		WHERE id = ?
	`
	info := &UserBasicInfo{}
	var profilePhoto sql.NullString
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&info.ID, &info.Name, &info.Code, &profilePhoto)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user basic info: %w", err)
	}
	if profilePhoto.Valid {
		info.ProfilePhoto = profilePhoto.String
	}
	return info, nil
}

func (r *userRepository) GetUserLevel(ctx context.Context, userID uint64) (string, error) {
	// Get user score first
	var score int32
	query := `SELECT score FROM users WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&score)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user score: %w", err)
	}

	// Get the latest level for this score
	query = `
		SELECT slug FROM levels
		WHERE score <= ?
		ORDER BY score DESC
		LIMIT 1
	`
	var levelSlug string
	err = r.db.QueryRowContext(ctx, query, score).Scan(&levelSlug)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user level: %w", err)
	}
	return levelSlug, nil
}

func (r *userRepository) IsUserOnline(ctx context.Context, userID uint64) (bool, error) {
	// User is online if last_seen is within the last 2 minutes
	query := `
		SELECT last_seen FROM users WHERE id = ?
	`
	var lastSeen sql.NullTime
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&lastSeen)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get last_seen: %w", err)
	}

	if !lastSeen.Valid {
		return false, nil
	}

	// Check if last_seen is within the last 2 minutes
	twoMinutesAgo := time.Now().Add(-2 * time.Minute)
	return lastSeen.Time.After(twoMinutesAgo), nil
}
