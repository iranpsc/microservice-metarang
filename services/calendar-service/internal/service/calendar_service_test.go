package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/calendar-service/internal/models"
)

// Mock CalendarRepository
type mockCalendarRepository struct {
	getEventsFunc             func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error)
	getEventByIDFunc          func(ctx context.Context, id uint64) (*models.Calendar, error)
	filterByDateRangeFunc     func(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error)
	getLatestVersionTitleFunc func(ctx context.Context) (string, error)
	getEventStatsFunc         func(ctx context.Context, eventID uint64) (*models.CalendarStats, error)
	getUserInteractionFunc    func(ctx context.Context, eventID, userID uint64) (*models.Interaction, error)
	addInteractionFunc        func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error
	incrementViewFunc         func(ctx context.Context, eventID uint64, ipAddress string) error
}

func (m *mockCalendarRepository) GetEvents(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
	if m.getEventsFunc != nil {
		return m.getEventsFunc(ctx, eventType, search, date, userID, page, perPage)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockCalendarRepository) GetEventByID(ctx context.Context, id uint64) (*models.Calendar, error) {
	if m.getEventByIDFunc != nil {
		return m.getEventByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarRepository) FilterByDateRange(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
	if m.filterByDateRangeFunc != nil {
		return m.filterByDateRangeFunc(ctx, startDate, endDate)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarRepository) GetLatestVersionTitle(ctx context.Context) (string, error) {
	if m.getLatestVersionTitleFunc != nil {
		return m.getLatestVersionTitleFunc(ctx)
	}
	return "", errors.New("not implemented")
}

func (m *mockCalendarRepository) GetEventStats(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
	if m.getEventStatsFunc != nil {
		return m.getEventStatsFunc(ctx, eventID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarRepository) GetUserInteraction(ctx context.Context, eventID, userID uint64) (*models.Interaction, error) {
	if m.getUserInteractionFunc != nil {
		return m.getUserInteractionFunc(ctx, eventID, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarRepository) AddInteraction(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
	if m.addInteractionFunc != nil {
		return m.addInteractionFunc(ctx, eventID, userID, liked, ipAddress)
	}
	return errors.New("not implemented")
}

func (m *mockCalendarRepository) IncrementView(ctx context.Context, eventID uint64, ipAddress string) error {
	if m.incrementViewFunc != nil {
		return m.incrementViewFunc(ctx, eventID, ipAddress)
	}
	return errors.New("not implemented")
}

func TestCalendarService_GetEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get events", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.getEventsFunc = func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
			return []*models.Calendar{
				{
					ID:        1,
					Title:     "Test Event",
					IsVersion: false,
					StartsAt:  time.Now(),
				},
			}, 1, nil
		}

		service := NewCalendarService(mockRepo)
		events, total, err := service.GetEvents(ctx, "event", "", "", 0, 1, 10)

		if err != nil {
			t.Fatalf("GetEvents failed: %v", err)
		}

		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}

		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.getEventsFunc = func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
			return nil, 0, errors.New("database error")
		}

		service := NewCalendarService(mockRepo)
		_, _, err := service.GetEvents(ctx, "event", "", "", 0, 1, 10)

		if err == nil {
			t.Fatal("Expected error")
		}
	})
}

func TestCalendarService_GetEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get event", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.getEventByIDFunc = func(ctx context.Context, id uint64) (*models.Calendar, error) {
			return &models.Calendar{
				ID:        1,
				Title:     "Test Event",
				IsVersion: false,
				StartsAt:  time.Now(),
			}, nil
		}

		service := NewCalendarService(mockRepo)
		event, err := service.GetEvent(ctx, 1, 0)

		if err != nil {
			t.Fatalf("GetEvent failed: %v", err)
		}

		if event.ID != 1 {
			t.Errorf("Expected event ID 1, got %d", event.ID)
		}
	})

	t.Run("event not found", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.getEventByIDFunc = func(ctx context.Context, id uint64) (*models.Calendar, error) {
			return nil, nil
		}

		service := NewCalendarService(mockRepo)
		_, err := service.GetEvent(ctx, 999, 0)

		if err == nil {
			t.Fatal("Expected error")
		}

		if err.Error() != "event not found" {
			t.Errorf("Expected 'event not found' error, got %v", err)
		}
	})
}

func TestCalendarService_AddInteraction(t *testing.T) {
	ctx := context.Background()

	t.Run("successful like", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			if liked != 1 {
				t.Errorf("Expected liked 1, got %d", liked)
			}
			return nil
		}

		service := NewCalendarService(mockRepo)
		err := service.AddInteraction(ctx, 1, 123, 1, "192.168.1.1")

		if err != nil {
			t.Fatalf("AddInteraction failed: %v", err)
		}
	})

	t.Run("successful dislike", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			if liked != 0 {
				t.Errorf("Expected liked 0, got %d", liked)
			}
			return nil
		}

		service := NewCalendarService(mockRepo)
		err := service.AddInteraction(ctx, 1, 123, 0, "192.168.1.1")

		if err != nil {
			t.Fatalf("AddInteraction failed: %v", err)
		}
	})

	t.Run("successful remove interaction", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			if liked != -1 {
				t.Errorf("Expected liked -1, got %d", liked)
			}
			return nil
		}

		service := NewCalendarService(mockRepo)
		err := service.AddInteraction(ctx, 1, 123, -1, "192.168.1.1")

		if err != nil {
			t.Fatalf("AddInteraction failed: %v", err)
		}
	})

	t.Run("invalid liked value", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		service := NewCalendarService(mockRepo)

		err := service.AddInteraction(ctx, 1, 123, 2, "192.168.1.1")

		if err == nil {
			t.Fatal("Expected error")
		}

		if err.Error() != "invalid liked value: must be -1, 0, or 1" {
			t.Errorf("Expected invalid liked value error, got %v", err)
		}
	})
}

func TestCalendarService_GetLatestVersionTitle(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get latest version", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.getLatestVersionTitleFunc = func(ctx context.Context) (string, error) {
			return "v1.2.3", nil
		}

		service := NewCalendarService(mockRepo)
		version, err := service.GetLatestVersionTitle(ctx)

		if err != nil {
			t.Fatalf("GetLatestVersionTitle failed: %v", err)
		}

		if version != "v1.2.3" {
			t.Errorf("Expected version v1.2.3, got %s", version)
		}
	})
}

func TestCalendarService_FilterByDateRange(t *testing.T) {
	ctx := context.Background()

	t.Run("successful filter by date range", func(t *testing.T) {
		mockRepo := &mockCalendarRepository{}
		mockRepo.filterByDateRangeFunc = func(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
			if startDate != "1403/01/01" || endDate != "1403/01/05" {
				t.Errorf("Expected dates 1403/01/01-1403/01/05, got %s-%s", startDate, endDate)
			}
			return []*models.Calendar{
				{
					ID:        1,
					Title:     "Test Event",
					IsVersion: false,
					StartsAt:  time.Now(),
				},
			}, nil
		}

		service := NewCalendarService(mockRepo)
		events, err := service.FilterByDateRange(ctx, "1403/01/01", "1403/01/05")

		if err != nil {
			t.Fatalf("FilterByDateRange failed: %v", err)
		}

		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})
}
