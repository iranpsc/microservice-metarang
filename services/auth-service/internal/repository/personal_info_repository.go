package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
)

type PersonalInfoRepository interface {
	FindByUserID(ctx context.Context, userID uint64) (*models.PersonalInfo, error)
	Upsert(ctx context.Context, personalInfo *models.PersonalInfo) error
}

type personalInfoRepository struct {
	db *sql.DB
}

func NewPersonalInfoRepository(db *sql.DB) PersonalInfoRepository {
	return &personalInfoRepository{db: db}
}

func (r *personalInfoRepository) FindByUserID(ctx context.Context, userID uint64) (*models.PersonalInfo, error) {
	query := `
		SELECT id, user_id, occupation, education, memory, loved_city, loved_country,
			loved_language, problem_solving, prediction, about, passions
		FROM personal_infos
		WHERE user_id = ?
		LIMIT 1
	`

	personalInfo := &models.PersonalInfo{}
	var passionsJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&personalInfo.ID,
		&personalInfo.UserID,
		&personalInfo.Occupation,
		&personalInfo.Education,
		&personalInfo.Memory,
		&personalInfo.LovedCity,
		&personalInfo.LovedCountry,
		&personalInfo.LovedLanguage,
		&personalInfo.ProblemSolving,
		&personalInfo.Prediction,
		&personalInfo.About,
		&passionsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find personal info: %w", err)
	}

	// Parse passions JSON
	if passionsJSON.Valid {
		var passions map[string]bool
		if err := json.Unmarshal([]byte(passionsJSON.String), &passions); err == nil {
			personalInfo.Passions = passions
		} else {
			// If JSON is invalid, use defaults
			personalInfo.Passions = models.DefaultPassions()
		}
	} else {
		// If passions is null, use defaults
		personalInfo.Passions = models.DefaultPassions()
	}

	return personalInfo, nil
}

func (r *personalInfoRepository) Upsert(ctx context.Context, personalInfo *models.PersonalInfo) error {
	// Marshal passions to JSON
	passionsJSON, err := json.Marshal(personalInfo.Passions)
	if err != nil {
		return fmt.Errorf("failed to marshal passions: %w", err)
	}

	now := time.Now()

	// Check if record exists
	existing, err := r.FindByUserID(ctx, personalInfo.UserID)
	if err != nil {
		return fmt.Errorf("failed to check existing personal info: %w", err)
	}

	if existing != nil {
		// Update existing record
		query := `
			UPDATE personal_infos
			SET occupation = ?, education = ?, memory = ?, loved_city = ?,
				loved_country = ?, loved_language = ?, problem_solving = ?,
				prediction = ?, about = ?, passions = ?, updated_at = ?
			WHERE user_id = ?
		`
		_, err := r.db.ExecContext(ctx, query,
			personalInfo.Occupation,
			personalInfo.Education,
			personalInfo.Memory,
			personalInfo.LovedCity,
			personalInfo.LovedCountry,
			personalInfo.LovedLanguage,
			personalInfo.ProblemSolving,
			personalInfo.Prediction,
			personalInfo.About,
			string(passionsJSON),
			now,
			personalInfo.UserID,
		)
		if err != nil {
			return fmt.Errorf("failed to update personal info: %w", err)
		}
		personalInfo.ID = existing.ID
	} else {
		// Create new record
		query := `
			INSERT INTO personal_infos (user_id, occupation, education, memory, loved_city,
				loved_country, loved_language, problem_solving, prediction, about, passions,
				created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		result, err := r.db.ExecContext(ctx, query,
			personalInfo.UserID,
			personalInfo.Occupation,
			personalInfo.Education,
			personalInfo.Memory,
			personalInfo.LovedCity,
			personalInfo.LovedCountry,
			personalInfo.LovedLanguage,
			personalInfo.ProblemSolving,
			personalInfo.Prediction,
			personalInfo.About,
			string(passionsJSON),
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to create personal info: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert id: %w", err)
		}
		personalInfo.ID = uint64(id)
	}

	return nil
}
