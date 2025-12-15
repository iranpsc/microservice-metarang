package repository

import (
	"context"
	"database/sql"
	"fmt"

	"metargb/calendar-service/internal/models"
	"metargb/shared/pkg/jalali"
)

// CalendarRepositoryInterface defines the interface for calendar repository operations
type CalendarRepositoryInterface interface {
	GetEvents(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error)
	GetEventByID(ctx context.Context, id uint64) (*models.Calendar, error)
	FilterByDateRange(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error)
	GetLatestVersionTitle(ctx context.Context) (string, error)
	GetEventStats(ctx context.Context, eventID uint64) (*models.CalendarStats, error)
	GetUserInteraction(ctx context.Context, eventID, userID uint64) (*models.Interaction, error)
	AddInteraction(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error
	IncrementView(ctx context.Context, eventID uint64, ipAddress string) error
}

type CalendarRepository struct {
	db *sql.DB
}

func NewCalendarRepository(db *sql.DB) *CalendarRepository {
	return &CalendarRepository{db: db}
}

// GetEvents retrieves events with optional filtering
// NOTE: When date is provided, returns all entries (no pagination) in descending order
// When date is not provided, uses pagination
func (r *CalendarRepository) GetEvents(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
	// Build query
	query := "SELECT id, slug, title, content, color, writer, is_version, version_title, btn_name, btn_link, image, starts_at, ends_at, created_at, updated_at FROM calendars WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM calendars WHERE 1=1"
	args := []interface{}{}

	// Filter by type
	if eventType == "version" {
		query += " AND is_version = 1"
		countQuery += " AND is_version = 1"
	} else {
		query += " AND is_version = 0"
		countQuery += " AND is_version = 0"
	}

	// Search by title
	if search != "" {
		query += " AND title LIKE ?"
		countQuery += " AND title LIKE ?"
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm)
	}

	// Filter by date (events active on that date)
	// When date is provided, no pagination - return all entries
	hasDateFilter := date != ""
	if hasDateFilter {
		carbonDate, err := jalali.JalaliToCarbon(date)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid jalali date: %w", err)
		}
		query += " AND DATE(starts_at) <= ? AND (ends_at IS NULL OR DATE(ends_at) >= ?)"
		countQuery += " AND DATE(starts_at) <= ? AND (ends_at IS NULL OR DATE(ends_at) >= ?)"
		args = append(args, carbonDate.Format("2006-01-02"), carbonDate.Format("2006-01-02"))
	}

	// Get total count (only needed for pagination)
	var total int32
	if !hasDateFilter {
		countArgs := make([]interface{}, len(args))
		copy(countArgs, args)
		err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count events: %w", err)
		}
	}

	// Add ordering
	query += " ORDER BY starts_at DESC"

	// Add pagination only if date filter is not provided
	if !hasDateFilter {
		offset := (page - 1) * perPage
		query += " LIMIT ? OFFSET ?"
		args = append(args, perPage, offset)
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []*models.Calendar
	for rows.Next() {
		var event models.Calendar
		if err := rows.Scan(
			&event.ID,
			&event.Slug,
			&event.Title,
			&event.Content,
			&event.Color,
			&event.Writer,
			&event.IsVersion,
			&event.VersionTitle,
			&event.BtnName,
			&event.BtnLink,
			&event.Image,
			&event.StartsAt,
			&event.EndsAt,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	// If date filter is provided, total is the count of returned events
	if hasDateFilter {
		total = int32(len(events))
	}

	return events, total, nil
}

// GetEventByID retrieves a single event by ID
func (r *CalendarRepository) GetEventByID(ctx context.Context, id uint64) (*models.Calendar, error) {
	query := "SELECT id, slug, title, content, color, writer, is_version, version_title, btn_name, btn_link, image, starts_at, ends_at, created_at, updated_at FROM calendars WHERE id = ?"

	var event models.Calendar
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.Slug,
		&event.Title,
		&event.Content,
		&event.Color,
		&event.Writer,
		&event.IsVersion,
		&event.VersionTitle,
		&event.BtnName,
		&event.BtnLink,
		&event.Image,
		&event.StartsAt,
		&event.EndsAt,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

// FilterByDateRange retrieves events within a date range
// NOTE: Returns only non-version events that overlap the requested range
// Overlap logic: events that start, end, or span entirely within the provided range
// Ordered by created_at DESC (latest first) per API documentation
func (r *CalendarRepository) FilterByDateRange(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
	start, err := jalali.JalaliToCarbon(startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}

	end, err := jalali.JalaliToCarbon(endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	// Overlap logic: event overlaps if:
	// 1. Event starts within range: starts_at BETWEEN start AND end
	// 2. Event ends within range: ends_at BETWEEN start AND end
	// 3. Event spans entire range: starts_at <= start AND (ends_at IS NULL OR ends_at >= end)
	query := `
		SELECT id, slug, title, content, color, writer, is_version, version_title, btn_name, btn_link, image, starts_at, ends_at, created_at, updated_at 
		FROM calendars 
		WHERE is_version = 0 
		AND (
			(DATE(starts_at) BETWEEN ? AND ?) OR
			(DATE(ends_at) BETWEEN ? AND ?) OR
			(DATE(starts_at) <= ? AND (ends_at IS NULL OR DATE(ends_at) >= ?))
		)
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, startStr, endStr, startStr, endStr, startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("failed to filter events: %w", err)
	}
	defer rows.Close()

	var events []*models.Calendar
	for rows.Next() {
		var event models.Calendar
		if err := rows.Scan(
			&event.ID,
			&event.Slug,
			&event.Title,
			&event.Content,
			&event.Color,
			&event.Writer,
			&event.IsVersion,
			&event.VersionTitle,
			&event.BtnName,
			&event.BtnLink,
			&event.Image,
			&event.StartsAt,
			&event.EndsAt,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}

// GetLatestVersionTitle retrieves the title of the latest version event
// NOTE: Laravel orders by starts_at DESC (line 126 in CalendarController)
func (r *CalendarRepository) GetLatestVersionTitle(ctx context.Context) (string, error) {
	query := "SELECT version_title FROM calendars WHERE is_version = 1 ORDER BY starts_at DESC LIMIT 1"

	var versionTitle sql.NullString
	err := r.db.QueryRowContext(ctx, query).Scan(&versionTitle)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get latest version: %w", err)
	}

	if versionTitle.Valid {
		return versionTitle.String, nil
	}
	return "", nil
}

// GetEventStats retrieves statistics for an event
func (r *CalendarRepository) GetEventStats(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
	stats := &models.CalendarStats{}

	// Get views count
	viewQuery := "SELECT COUNT(*) FROM views WHERE viewable_type = 'App\\\\Models\\\\Calendar' AND viewable_id = ?"
	r.db.QueryRowContext(ctx, viewQuery, eventID).Scan(&stats.ViewsCount)

	// Get likes count
	likeQuery := "SELECT COUNT(*) FROM interactions WHERE likeable_type = 'App\\\\Models\\\\Calendar' AND likeable_id = ? AND liked = 1"
	r.db.QueryRowContext(ctx, likeQuery, eventID).Scan(&stats.LikesCount)

	// Get dislikes count
	dislikeQuery := "SELECT COUNT(*) FROM interactions WHERE likeable_type = 'App\\\\Models\\\\Calendar' AND likeable_id = ? AND liked = 0"
	r.db.QueryRowContext(ctx, dislikeQuery, eventID).Scan(&stats.DislikesCount)

	return stats, nil
}

// GetUserInteraction retrieves user's interaction with an event
func (r *CalendarRepository) GetUserInteraction(ctx context.Context, eventID, userID uint64) (*models.Interaction, error) {
	query := "SELECT id, likeable_type, likeable_id, user_id, liked, ip_address, created_at, updated_at FROM interactions WHERE likeable_type = 'App\\\\Models\\\\Calendar' AND likeable_id = ? AND user_id = ?"

	var interaction models.Interaction
	err := r.db.QueryRowContext(ctx, query, eventID, userID).Scan(
		&interaction.ID,
		&interaction.LikeableType,
		&interaction.LikeableID,
		&interaction.UserID,
		&interaction.Liked,
		&interaction.IPAddress,
		&interaction.CreatedAt,
		&interaction.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user interaction: %w", err)
	}

	return &interaction, nil
}

// AddInteraction adds or updates a user's interaction
func (r *CalendarRepository) AddInteraction(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
	if liked == -1 {
		// Remove interaction
		query := "DELETE FROM interactions WHERE likeable_type = 'App\\\\Models\\\\Calendar' AND likeable_id = ? AND user_id = ?"
		_, err := r.db.ExecContext(ctx, query, eventID, userID)
		return err
	}

	// Upsert interaction
	query := `
		INSERT INTO interactions (likeable_type, likeable_id, user_id, liked, ip_address, created_at, updated_at) 
		VALUES ('App\\Models\\Calendar', ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE liked = ?, ip_address = ?, updated_at = NOW()
	`

	likedBool := liked == 1
	_, err := r.db.ExecContext(ctx, query, eventID, userID, likedBool, ipAddress, likedBool, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to add interaction: %w", err)
	}

	return nil
}

// IncrementView adds a view for an event
func (r *CalendarRepository) IncrementView(ctx context.Context, eventID uint64, ipAddress string) error {
	query := "INSERT INTO views (viewable_type, viewable_id, ip_address, created_at, updated_at) VALUES ('App\\\\Models\\\\Calendar', ?, ?, NOW(), NOW())"
	_, err := r.db.ExecContext(ctx, query, eventID, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to increment view: %w", err)
	}
	return nil
}
