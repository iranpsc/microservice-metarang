package service

import (
	"context"
	"fmt"

	"metargb/calendar-service/internal/models"
	"metargb/calendar-service/internal/repository"
)

type CalendarService struct {
	repo *repository.CalendarRepository
}

func NewCalendarService(repo *repository.CalendarRepository) *CalendarService {
	return &CalendarService{repo: repo}
}

// GetEvents retrieves events with filtering
func (s *CalendarService) GetEvents(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
	return s.repo.GetEvents(ctx, eventType, search, date, userID, page, perPage)
}

// GetEvent retrieves a single event by ID
func (s *CalendarService) GetEvent(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("event not found")
	}

	return event, nil
}

// FilterByDateRange retrieves events within a date range
func (s *CalendarService) FilterByDateRange(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
	return s.repo.FilterByDateRange(ctx, startDate, endDate)
}

// GetLatestVersionTitle retrieves the latest version title
func (s *CalendarService) GetLatestVersionTitle(ctx context.Context) (string, error) {
	return s.repo.GetLatestVersionTitle(ctx)
}

// GetEventStats retrieves statistics for an event
func (s *CalendarService) GetEventStats(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
	return s.repo.GetEventStats(ctx, eventID)
}

// GetUserInteraction retrieves user's interaction with an event
func (s *CalendarService) GetUserInteraction(ctx context.Context, eventID, userID uint64) (*models.Interaction, error) {
	return s.repo.GetUserInteraction(ctx, eventID, userID)
}

// AddInteraction adds or updates a user's interaction
func (s *CalendarService) AddInteraction(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
	// Validate liked value (-1, 0, or 1)
	if liked < -1 || liked > 1 {
		return fmt.Errorf("invalid liked value: must be -1, 0, or 1")
	}

	return s.repo.AddInteraction(ctx, eventID, userID, liked, ipAddress)
}

// IncrementView increments view count for an event
func (s *CalendarService) IncrementView(ctx context.Context, eventID uint64, ipAddress string) error {
	return s.repo.IncrementView(ctx, eventID, ipAddress)
}

