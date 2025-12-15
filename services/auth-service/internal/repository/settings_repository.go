package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
)

type SettingsRepository interface {
	FindByUserID(ctx context.Context, userID uint64) (*models.Settings, error)
	FindByID(ctx context.Context, id uint64) (*models.Settings, error)
	Update(ctx context.Context, settings *models.Settings) error
	Create(ctx context.Context, settings *models.Settings) error
}

type settingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) FindByUserID(ctx context.Context, userID uint64) (*models.Settings, error) {
	query := `
		SELECT id, user_id, status, level, details, checkout_days_count, automatic_logout,
			privacy, notifications, created_at, updated_at
		FROM settings
		WHERE user_id = ?
		LIMIT 1
	`

	settings := &models.Settings{}
	var privacyJSON, notificationsJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.Status,
		&settings.Level,
		&settings.Details,
		&settings.CheckoutDaysCount,
		&settings.AutomaticLogout,
		&privacyJSON,
		&notificationsJSON,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default settings
		return &models.Settings{
			UserID:            userID,
			Status:            true,
			Level:             true,
			Details:           true,
			CheckoutDaysCount: 3,
			AutomaticLogout:   55,
			Privacy:           models.DefaultPrivacySettings(),
			Notifications:     models.DefaultNotificationSettings(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find settings: %w", err)
	}

	// Parse privacy JSON
	if privacyJSON.Valid && privacyJSON.String != "" {
		var privacy map[string]int
		if err := json.Unmarshal([]byte(privacyJSON.String), &privacy); err == nil {
			settings.Privacy = privacy
		} else {
			settings.Privacy = models.DefaultPrivacySettings()
		}
	} else {
		settings.Privacy = models.DefaultPrivacySettings()
	}

	// Parse notifications JSON
	if notificationsJSON.Valid && notificationsJSON.String != "" {
		var notifications map[string]bool
		if err := json.Unmarshal([]byte(notificationsJSON.String), &notifications); err == nil {
			settings.Notifications = notifications
		} else {
			settings.Notifications = models.DefaultNotificationSettings()
		}
	} else {
		settings.Notifications = models.DefaultNotificationSettings()
	}

	return settings, nil
}

func (r *settingsRepository) FindByID(ctx context.Context, id uint64) (*models.Settings, error) {
	query := `
		SELECT id, user_id, status, level, details, checkout_days_count, automatic_logout,
			privacy, notifications, created_at, updated_at
		FROM settings
		WHERE id = ?
		LIMIT 1
	`

	settings := &models.Settings{}
	var privacyJSON, notificationsJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.Status,
		&settings.Level,
		&settings.Details,
		&settings.CheckoutDaysCount,
		&settings.AutomaticLogout,
		&privacyJSON,
		&notificationsJSON,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find settings: %w", err)
	}

	// Parse privacy JSON
	if privacyJSON.Valid && privacyJSON.String != "" {
		var privacy map[string]int
		if err := json.Unmarshal([]byte(privacyJSON.String), &privacy); err == nil {
			settings.Privacy = privacy
		} else {
			settings.Privacy = models.DefaultPrivacySettings()
		}
	} else {
		settings.Privacy = models.DefaultPrivacySettings()
	}

	// Parse notifications JSON
	if notificationsJSON.Valid && notificationsJSON.String != "" {
		var notifications map[string]bool
		if err := json.Unmarshal([]byte(notificationsJSON.String), &notifications); err == nil {
			settings.Notifications = notifications
		} else {
			settings.Notifications = models.DefaultNotificationSettings()
		}
	} else {
		settings.Notifications = models.DefaultNotificationSettings()
	}

	return settings, nil
}

func (r *settingsRepository) Update(ctx context.Context, settings *models.Settings) error {
	// Marshal privacy to JSON
	privacyJSON, err := json.Marshal(settings.Privacy)
	if err != nil {
		return fmt.Errorf("failed to marshal privacy: %w", err)
	}

	// Marshal notifications to JSON
	notificationsJSON, err := json.Marshal(settings.Notifications)
	if err != nil {
		return fmt.Errorf("failed to marshal notifications: %w", err)
	}

	now := time.Now()
	query := `
		UPDATE settings
		SET status = ?, level = ?, details = ?, checkout_days_count = ?,
			automatic_logout = ?, privacy = ?, notifications = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, query,
		settings.Status,
		settings.Level,
		settings.Details,
		settings.CheckoutDaysCount,
		settings.AutomaticLogout,
		string(privacyJSON),
		string(notificationsJSON),
		now,
		settings.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	settings.UpdatedAt = now
	return nil
}

func (r *settingsRepository) Create(ctx context.Context, settings *models.Settings) error {
	// Marshal privacy to JSON
	privacyJSON, err := json.Marshal(settings.Privacy)
	if err != nil {
		return fmt.Errorf("failed to marshal privacy: %w", err)
	}

	// Marshal notifications to JSON
	notificationsJSON, err := json.Marshal(settings.Notifications)
	if err != nil {
		return fmt.Errorf("failed to marshal notifications: %w", err)
	}

	now := time.Now()
	query := `
		INSERT INTO settings (user_id, status, level, details, checkout_days_count,
			automatic_logout, privacy, notifications, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		settings.UserID,
		settings.Status,
		settings.Level,
		settings.Details,
		settings.CheckoutDaysCount,
		settings.AutomaticLogout,
		string(privacyJSON),
		string(notificationsJSON),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create settings: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	settings.ID = uint64(id)
	settings.CreatedAt = now
	settings.UpdatedAt = now
	return nil
}
