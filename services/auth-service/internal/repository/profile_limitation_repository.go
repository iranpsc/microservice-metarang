package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
)

type ProfileLimitationRepository interface {
	Create(ctx context.Context, limitation *models.ProfileLimitation) error
	FindByID(ctx context.Context, id uint64) (*models.ProfileLimitation, error)
	FindByLimiterAndLimited(ctx context.Context, limiterUserID, limitedUserID uint64) (*models.ProfileLimitation, error)
	FindBetweenUsers(ctx context.Context, userID1, userID2 uint64) (*models.ProfileLimitation, error)
	Update(ctx context.Context, limitation *models.ProfileLimitation) error
	Delete(ctx context.Context, id uint64) error
	ExistsForLimiterAndLimited(ctx context.Context, limiterUserID, limitedUserID uint64) (bool, error)
}

type profileLimitationRepository struct {
	db *sql.DB
}

func NewProfileLimitationRepository(db *sql.DB) ProfileLimitationRepository {
	return &profileLimitationRepository{db: db}
}

func (r *profileLimitationRepository) Create(ctx context.Context, limitation *models.ProfileLimitation) error {
	optionsJSON, err := json.Marshal(limitation.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	query := `
		INSERT INTO profile_limitations (limiter_user_id, limited_user_id, options, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	var note sql.NullString
	if limitation.Note.Valid {
		note = limitation.Note
	}

	result, err := r.db.ExecContext(ctx, query,
		limitation.LimiterUserID,
		limitation.LimitedUserID,
		string(optionsJSON),
		note,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create profile limitation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	limitation.ID = uint64(id)
	limitation.CreatedAt = now
	limitation.UpdatedAt = now

	return nil
}

func (r *profileLimitationRepository) FindByID(ctx context.Context, id uint64) (*models.ProfileLimitation, error) {
	query := `
		SELECT id, limiter_user_id, limited_user_id, options, note, created_at, updated_at
		FROM profile_limitations
		WHERE id = ?
	`

	limitation := &models.ProfileLimitation{}
	var optionsJSON string
	var note sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&limitation.ID,
		&limitation.LimiterUserID,
		&limitation.LimitedUserID,
		&optionsJSON,
		&note,
		&limitation.CreatedAt,
		&limitation.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find profile limitation by id: %w", err)
	}

	if err := json.Unmarshal([]byte(optionsJSON), &limitation.Options); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	limitation.Note = note
	return limitation, nil
}

func (r *profileLimitationRepository) FindByLimiterAndLimited(ctx context.Context, limiterUserID, limitedUserID uint64) (*models.ProfileLimitation, error) {
	query := `
		SELECT id, limiter_user_id, limited_user_id, options, note, created_at, updated_at
		FROM profile_limitations
		WHERE limiter_user_id = ? AND limited_user_id = ?
		LIMIT 1
	`

	limitation := &models.ProfileLimitation{}
	var optionsJSON string
	var note sql.NullString

	err := r.db.QueryRowContext(ctx, query, limiterUserID, limitedUserID).Scan(
		&limitation.ID,
		&limitation.LimiterUserID,
		&limitation.LimitedUserID,
		&optionsJSON,
		&note,
		&limitation.CreatedAt,
		&limitation.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find profile limitation: %w", err)
	}

	if err := json.Unmarshal([]byte(optionsJSON), &limitation.Options); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	limitation.Note = note
	return limitation, nil
}

func (r *profileLimitationRepository) FindBetweenUsers(ctx context.Context, userID1, userID2 uint64) (*models.ProfileLimitation, error) {
	// Check both directions: user1 limiting user2, or user2 limiting user1
	query := `
		SELECT id, limiter_user_id, limited_user_id, options, note, created_at, updated_at
		FROM profile_limitations
		WHERE (limiter_user_id = ? AND limited_user_id = ?)
		   OR (limiter_user_id = ? AND limited_user_id = ?)
		LIMIT 1
	`

	limitation := &models.ProfileLimitation{}
	var optionsJSON string
	var note sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID1, userID2, userID2, userID1).Scan(
		&limitation.ID,
		&limitation.LimiterUserID,
		&limitation.LimitedUserID,
		&optionsJSON,
		&note,
		&limitation.CreatedAt,
		&limitation.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find profile limitation between users: %w", err)
	}

	if err := json.Unmarshal([]byte(optionsJSON), &limitation.Options); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	limitation.Note = note
	return limitation, nil
}

func (r *profileLimitationRepository) Update(ctx context.Context, limitation *models.ProfileLimitation) error {
	optionsJSON, err := json.Marshal(limitation.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	query := `
		UPDATE profile_limitations
		SET options = ?, note = ?, updated_at = ?
		WHERE id = ?
	`

	var note sql.NullString
	if limitation.Note.Valid {
		note = limitation.Note
	}

	_, err = r.db.ExecContext(ctx, query,
		string(optionsJSON),
		note,
		time.Now(),
		limitation.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile limitation: %w", err)
	}

	limitation.UpdatedAt = time.Now()
	return nil
}

func (r *profileLimitationRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM profile_limitations WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile limitation: %w", err)
	}
	return nil
}

func (r *profileLimitationRepository) ExistsForLimiterAndLimited(ctx context.Context, limiterUserID, limitedUserID uint64) (bool, error) {
	query := `
		SELECT COUNT(*) FROM profile_limitations
		WHERE limiter_user_id = ? AND limited_user_id = ?
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, limiterUserID, limitedUserID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}
