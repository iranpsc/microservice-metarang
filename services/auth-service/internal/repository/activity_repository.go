package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
)

type ActivityRepository interface {
	// User Events
	CreateUserEvent(ctx context.Context, event *models.UserEvent) error
	
	// User Activities
	CreateActivity(ctx context.Context, activity *models.UserActivity) error
	GetLatestActivity(ctx context.Context, userID uint64) (*models.UserActivity, error)
	UpdateActivity(ctx context.Context, activity *models.UserActivity) error
	GetTotalActivityMinutes(ctx context.Context, userID uint64) (int32, error)
	
	// User Logs
	GetUserLog(ctx context.Context, userID uint64) (*models.UserLog, error)
	CreateUserLog(ctx context.Context, log *models.UserLog) error
	UpdateUserLog(ctx context.Context, log *models.UserLog) error
	IncrementLogField(ctx context.Context, userID uint64, field string, amount float64) error
}

type activityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) ActivityRepository {
	return &activityRepository{db: db}
}

// CreateUserEvent creates a new user event record
func (r *activityRepository) CreateUserEvent(ctx context.Context, event *models.UserEvent) error {
	query := `
		INSERT INTO user_events (user_id, event, ip, device, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		event.UserID, event.Event, event.IP, event.Device, event.Status,
		time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create user event: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	event.ID = uint64(id)
	
	return nil
}

// CreateActivity creates a new activity record
func (r *activityRepository) CreateActivity(ctx context.Context, activity *models.UserActivity) error {
	query := `
		INSERT INTO user_activities (user_id, start, ip, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		activity.UserID, activity.Start, activity.IP, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create activity: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	activity.ID = uint64(id)
	
	return nil
}

// GetLatestActivity gets the most recent activity for a user
func (r *activityRepository) GetLatestActivity(ctx context.Context, userID uint64) (*models.UserActivity, error) {
	query := `
		SELECT id, user_id, start, end, total, ip, created_at, updated_at
		FROM user_activities
		WHERE user_id = ?
		ORDER BY id DESC
		LIMIT 1
	`
	activity := &models.UserActivity{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&activity.ID, &activity.UserID, &activity.Start, &activity.End,
		&activity.Total, &activity.IP, &activity.CreatedAt, &activity.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest activity: %w", err)
	}
	return activity, nil
}

// UpdateActivity updates an activity record
func (r *activityRepository) UpdateActivity(ctx context.Context, activity *models.UserActivity) error {
	query := `
		UPDATE user_activities
		SET end = ?, total = ?, ip = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		activity.End, activity.Total, activity.IP, time.Now(), activity.ID)
	if err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}
	return nil
}

// GetTotalActivityMinutes calculates total minutes from all activities
func (r *activityRepository) GetTotalActivityMinutes(ctx context.Context, userID uint64) (int32, error) {
	query := `
		SELECT COALESCE(SUM(total), 0) as total_minutes
		FROM user_activities
		WHERE user_id = ?
	`
	var total int32
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total activity minutes: %w", err)
	}
	return total, nil
}

// GetUserLog retrieves the user log record
func (r *activityRepository) GetUserLog(ctx context.Context, userID uint64) (*models.UserLog, error) {
	query := `
		SELECT id, user_id, transactions_count, followers_count, deposit_amount, 
		       activity_hours, score, created_at, updated_at
		FROM user_logs
		WHERE user_id = ?
	`
	log := &models.UserLog{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&log.ID, &log.UserID, &log.TransactionsCount, &log.FollowersCount,
		&log.DepositAmount, &log.ActivityHours, &log.Score,
		&log.CreatedAt, &log.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user log: %w", err)
	}
	return log, nil
}

// CreateUserLog creates a new user log record
func (r *activityRepository) CreateUserLog(ctx context.Context, log *models.UserLog) error {
	query := `
		INSERT INTO user_logs (user_id, transactions_count, followers_count, 
		                       deposit_amount, activity_hours, score, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		log.UserID, log.TransactionsCount, log.FollowersCount,
		log.DepositAmount, log.ActivityHours, log.Score,
		time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create user log: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	log.ID = uint64(id)
	
	return nil
}

// UpdateUserLog updates a user log record
func (r *activityRepository) UpdateUserLog(ctx context.Context, log *models.UserLog) error {
	query := `
		UPDATE user_logs
		SET transactions_count = ?, followers_count = ?, deposit_amount = ?,
		    activity_hours = ?, score = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		log.TransactionsCount, log.FollowersCount, log.DepositAmount,
		log.ActivityHours, log.Score, time.Now(), log.ID)
	if err != nil {
		return fmt.Errorf("failed to update user log: %w", err)
	}
	return nil
}

// IncrementLogField increments a specific field in user log
func (r *activityRepository) IncrementLogField(ctx context.Context, userID uint64, field string, amount float64) error {
	// Validate field name to prevent SQL injection
	validFields := map[string]bool{
		"transactions_count": true,
		"followers_count":    true,
		"deposit_amount":     true,
		"activity_hours":     true,
		"score":              true,
	}
	
	if !validFields[field] {
		return fmt.Errorf("invalid field name: %s", field)
	}
	
	query := fmt.Sprintf(`
		UPDATE user_logs
		SET %s = %s + ?, updated_at = ?
		WHERE user_id = ?
	`, field, field)
	
	_, err := r.db.ExecContext(ctx, query, amount, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to increment log field %s: %w", field, err)
	}
	return nil
}

