package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"metargb/calendar-service/internal/models"
	calendarpb "metargb/shared/pb/calendar"
	commonpb "metargb/shared/pb/common"
)

// Mock CalendarService
type mockCalendarService struct {
	getEventsFunc             func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error)
	getEventFunc              func(ctx context.Context, eventID, userID uint64) (*models.Calendar, error)
	filterByDateRangeFunc     func(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error)
	getLatestVersionTitleFunc func(ctx context.Context) (string, error)
	getEventStatsFunc         func(ctx context.Context, eventID uint64) (*models.CalendarStats, error)
	getUserInteractionFunc    func(ctx context.Context, eventID, userID uint64) (*models.Interaction, error)
	addInteractionFunc        func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error
	incrementViewFunc         func(ctx context.Context, eventID uint64, ipAddress string) error
}

func (m *mockCalendarService) GetEvents(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
	if m.getEventsFunc != nil {
		return m.getEventsFunc(ctx, eventType, search, date, userID, page, perPage)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockCalendarService) GetEvent(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
	if m.getEventFunc != nil {
		return m.getEventFunc(ctx, eventID, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarService) FilterByDateRange(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
	if m.filterByDateRangeFunc != nil {
		return m.filterByDateRangeFunc(ctx, startDate, endDate)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarService) GetLatestVersionTitle(ctx context.Context) (string, error) {
	if m.getLatestVersionTitleFunc != nil {
		return m.getLatestVersionTitleFunc(ctx)
	}
	return "", errors.New("not implemented")
}

func (m *mockCalendarService) GetEventStats(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
	if m.getEventStatsFunc != nil {
		return m.getEventStatsFunc(ctx, eventID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarService) GetUserInteraction(ctx context.Context, eventID, userID uint64) (*models.Interaction, error) {
	if m.getUserInteractionFunc != nil {
		return m.getUserInteractionFunc(ctx, eventID, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCalendarService) AddInteraction(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
	if m.addInteractionFunc != nil {
		return m.addInteractionFunc(ctx, eventID, userID, liked, ipAddress)
	}
	return errors.New("not implemented")
}

func (m *mockCalendarService) IncrementView(ctx context.Context, eventID uint64, ipAddress string) error {
	if m.incrementViewFunc != nil {
		return m.incrementViewFunc(ctx, eventID, ipAddress)
	}
	return errors.New("not implemented")
}

func TestCalendarHandler_GetEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get events with pagination", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getEventsFunc = func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
			return []*models.Calendar{
				{
					ID:        1,
					Title:     "Test Event",
					Content:   "Test Content",
					IsVersion: false,
					StartsAt:  time.Now(),
				},
			}, 1, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 5, DislikesCount: 2}, nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetEventsRequest{
			Type: "event",
			Pagination: &commonpb.PaginationRequest{
				Page:    1,
				PerPage: 10,
			},
		}

		resp, err := handler.GetEvents(ctx, req)
		if err != nil {
			t.Fatalf("GetEvents failed: %v", err)
		}

		if len(resp.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(resp.Events))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination metadata")
		}

		if resp.Pagination.Total != 1 {
			t.Errorf("Expected total 1, got %d", resp.Pagination.Total)
		}
	})

	t.Run("successful get events with date filter (no pagination)", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getEventsFunc = func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
			if date != "1403/01/01" {
				t.Errorf("Expected date 1403/01/01, got %s", date)
			}
			return []*models.Calendar{
				{
					ID:        1,
					Title:     "Test Event",
					Content:   "Test Content",
					IsVersion: false,
					StartsAt:  time.Now(),
				},
			}, 1, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 5, DislikesCount: 2}, nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetEventsRequest{
			Type: "event",
			Date: "1403/01/01",
		}

		resp, err := handler.GetEvents(ctx, req)
		if err != nil {
			t.Fatalf("GetEvents failed: %v", err)
		}

		if len(resp.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(resp.Events))
		}

		if resp.Pagination != nil {
			t.Error("Expected no pagination metadata when date filter is provided")
		}
	})

	t.Run("get events with user interaction", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getEventsFunc = func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
			return []*models.Calendar{
				{
					ID:        1,
					Title:     "Test Event",
					Content:   "Test Content",
					IsVersion: false,
					StartsAt:  time.Now(),
				},
			}, 1, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 5, DislikesCount: 2}, nil
		}
		mockService.getUserInteractionFunc = func(ctx context.Context, eventID, userID uint64) (*models.Interaction, error) {
			return &models.Interaction{
				ID:    1,
				Liked: true,
			}, nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetEventsRequest{
			Type:   "event",
			UserId: 123,
			Pagination: &commonpb.PaginationRequest{
				Page:    1,
				PerPage: 10,
			},
		}

		resp, err := handler.GetEvents(ctx, req)
		if err != nil {
			t.Fatalf("GetEvents failed: %v", err)
		}

		if len(resp.Events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(resp.Events))
		}

		if resp.Events[0].UserInteraction == nil {
			t.Error("Expected user interaction to be set")
		}

		if !resp.Events[0].UserInteraction.HasLiked {
			t.Error("Expected HasLiked to be true")
		}

		if resp.Events[0].UserInteraction.HasDisliked {
			t.Error("Expected HasDisliked to be false")
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getEventsFunc = func(ctx context.Context, eventType, search, date string, userID uint64, page, perPage int32) ([]*models.Calendar, int32, error) {
			return nil, 0, errors.New("database error")
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetEventsRequest{
			Type: "event",
		}

		_, err := handler.GetEvents(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("Expected Internal error code, got %v", st.Code())
		}
	})
}

func TestCalendarHandler_GetEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get event", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getEventFunc = func(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
			if eventID != 1 {
				t.Errorf("Expected eventID 1, got %d", eventID)
			}
			return &models.Calendar{
				ID:        1,
				Title:     "Test Event",
				Content:   "Test Content",
				IsVersion: false,
				StartsAt:  time.Now(),
			}, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 5, DislikesCount: 2}, nil
		}
		mockService.incrementViewFunc = func(ctx context.Context, eventID uint64, ipAddress string) error {
			return nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetEventRequest{
			EventId: 1,
		}

		resp, err := handler.GetEvent(ctx, req)
		if err != nil {
			t.Fatalf("GetEvent failed: %v", err)
		}

		if resp.Id != 1 {
			t.Errorf("Expected event ID 1, got %d", resp.Id)
		}

		if resp.Views != 10 {
			t.Errorf("Expected views 10, got %d", resp.Views)
		}
	})

	t.Run("event not found", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getEventFunc = func(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
			return nil, errors.New("event not found")
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetEventRequest{
			EventId: 999,
		}

		_, err := handler.GetEvent(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound error code, got %v", st.Code())
		}
	})
}

func TestCalendarHandler_FilterByDateRange(t *testing.T) {
	ctx := context.Background()

	t.Run("successful filter by date range", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.filterByDateRangeFunc = func(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
			if startDate != "1403/01/01" || endDate != "1403/01/05" {
				t.Errorf("Expected dates 1403/01/01-1403/01/05, got %s-%s", startDate, endDate)
			}
			return []*models.Calendar{
				{
					ID:        1,
					Title:     "Test Event",
					Color:     "#FFAA00",
					IsVersion: false,
					StartsAt:  time.Now(),
					EndsAt:    func() *time.Time { t := time.Now().Add(24 * time.Hour); return &t }(),
				},
			}, nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.FilterByDateRangeRequest{
			StartDate: "1403/01/01",
			EndDate:   "1403/01/05",
		}

		resp, err := handler.FilterByDateRange(ctx, req)
		if err != nil {
			t.Fatalf("FilterByDateRange failed: %v", err)
		}

		if len(resp.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(resp.Events))
		}

		if resp.Events[0].Id != 1 {
			t.Errorf("Expected event ID 1, got %d", resp.Events[0].Id)
		}

		if resp.Events[0].Color != "#FFAA00" {
			t.Errorf("Expected color #FFAA00, got %s", resp.Events[0].Color)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.filterByDateRangeFunc = func(ctx context.Context, startDate, endDate string) ([]*models.Calendar, error) {
			return nil, errors.New("database error")
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.FilterByDateRangeRequest{
			StartDate: "1403/01/01",
			EndDate:   "1403/01/05",
		}

		_, err := handler.FilterByDateRange(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("Expected Internal error code, got %v", st.Code())
		}
	})
}

func TestCalendarHandler_GetLatestVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get latest version", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getLatestVersionTitleFunc = func(ctx context.Context) (string, error) {
			return "v1.2.3", nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetLatestVersionRequest{}

		resp, err := handler.GetLatestVersion(ctx, req)
		if err != nil {
			t.Fatalf("GetLatestVersion failed: %v", err)
		}

		if resp.VersionTitle != "v1.2.3" {
			t.Errorf("Expected version v1.2.3, got %s", resp.VersionTitle)
		}
	})

	t.Run("no version found", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.getLatestVersionTitleFunc = func(ctx context.Context) (string, error) {
			return "", nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.GetLatestVersionRequest{}

		resp, err := handler.GetLatestVersion(ctx, req)
		if err != nil {
			t.Fatalf("GetLatestVersion failed: %v", err)
		}

		if resp.VersionTitle != "" {
			t.Errorf("Expected empty version, got %s", resp.VersionTitle)
		}
	})
}

func TestCalendarHandler_AddInteraction(t *testing.T) {
	ctx := context.Background()

	t.Run("successful like interaction", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			if eventID != 1 {
				t.Errorf("Expected eventID 1, got %d", eventID)
			}
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
			if liked != 1 {
				t.Errorf("Expected liked 1, got %d", liked)
			}
			return nil
		}
		mockService.getEventFunc = func(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
			return &models.Calendar{
				ID:        1,
				Title:     "Test Event",
				Content:   "Test Content",
				IsVersion: false,
				StartsAt:  time.Now(),
			}, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 6, DislikesCount: 2}, nil
		}
		mockService.incrementViewFunc = func(ctx context.Context, eventID uint64, ipAddress string) error {
			return nil
		}

		handler := &CalendarHandler{service: mockService}

		// Create context with metadata for IP address
		md := metadata.New(map[string]string{
			"x-forwarded-for": "192.168.1.1",
		})
		ctxWithMD := metadata.NewIncomingContext(ctx, md)

		req := &calendarpb.AddInteractionRequest{
			EventId: 1,
			UserId:  123,
			Liked:   1,
		}

		resp, err := handler.AddInteraction(ctxWithMD, req)
		if err != nil {
			t.Fatalf("AddInteraction failed: %v", err)
		}

		if resp.Id != 1 {
			t.Errorf("Expected event ID 1, got %d", resp.Id)
		}
	})

	t.Run("successful dislike interaction", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			if liked != 0 {
				t.Errorf("Expected liked 0, got %d", liked)
			}
			return nil
		}
		mockService.getEventFunc = func(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
			return &models.Calendar{
				ID:        1,
				Title:     "Test Event",
				Content:   "Test Content",
				IsVersion: false,
				StartsAt:  time.Now(),
			}, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 5, DislikesCount: 3}, nil
		}
		mockService.incrementViewFunc = func(ctx context.Context, eventID uint64, ipAddress string) error {
			return nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.AddInteractionRequest{
			EventId: 1,
			UserId:  123,
			Liked:   0,
		}

		_, err := handler.AddInteraction(ctx, req)
		if err != nil {
			t.Fatalf("AddInteraction failed: %v", err)
		}
	})

	t.Run("successful remove interaction", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			if liked != -1 {
				t.Errorf("Expected liked -1, got %d", liked)
			}
			return nil
		}
		mockService.getEventFunc = func(ctx context.Context, eventID, userID uint64) (*models.Calendar, error) {
			return &models.Calendar{
				ID:        1,
				Title:     "Test Event",
				Content:   "Test Content",
				IsVersion: false,
				StartsAt:  time.Now(),
			}, nil
		}
		mockService.getEventStatsFunc = func(ctx context.Context, eventID uint64) (*models.CalendarStats, error) {
			return &models.CalendarStats{ViewsCount: 10, LikesCount: 4, DislikesCount: 2}, nil
		}
		mockService.incrementViewFunc = func(ctx context.Context, eventID uint64, ipAddress string) error {
			return nil
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.AddInteractionRequest{
			EventId: 1,
			UserId:  123,
			Liked:   -1,
		}

		_, err := handler.AddInteraction(ctx, req)
		if err != nil {
			t.Fatalf("AddInteraction failed: %v", err)
		}
	})

	t.Run("invalid liked value", func(t *testing.T) {
		mockService := &mockCalendarService{}
		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.AddInteractionRequest{
			EventId: 1,
			UserId:  123,
			Liked:   2, // Invalid value
		}

		_, err := handler.AddInteraction(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument error code, got %v", st.Code())
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockCalendarService{}
		mockService.addInteractionFunc = func(ctx context.Context, eventID, userID uint64, liked int32, ipAddress string) error {
			return errors.New("database error")
		}

		handler := &CalendarHandler{service: mockService}

		req := &calendarpb.AddInteractionRequest{
			EventId: 1,
			UserId:  123,
			Liked:   1,
		}

		_, err := handler.AddInteraction(ctx, req)
		if err == nil {
			t.Fatal("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("Expected Internal error code, got %v", st.Code())
		}
	})
}

func TestBuildEventResponse(t *testing.T) {
	t.Run("event response (not version)", func(t *testing.T) {
		event := &models.Calendar{
			ID:        1,
			Title:     "Test Event",
			Content:   "Test Content",
			Color:     "#FFAA00",
			IsVersion: false,
			StartsAt:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		}
		endsAt := time.Date(2024, 1, 5, 18, 0, 0, 0, time.UTC)
		event.EndsAt = &endsAt
		btnName := "Click Here"
		event.BtnName = &btnName
		btnLink := "https://example.com"
		event.BtnLink = &btnLink
		image := "https://example.com/image.jpg"
		event.Image = &image

		stats := &models.CalendarStats{
			ViewsCount:    10,
			LikesCount:    5,
			DislikesCount: 2,
		}

		userInteraction := &calendarpb.UserInteraction{
			HasLiked:    true,
			HasDisliked: false,
		}

		resp := buildEventResponse(event, stats, userInteraction)

		if resp.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Id)
		}

		if resp.Title != "Test Event" {
			t.Errorf("Expected title 'Test Event', got %s", resp.Title)
		}

		if resp.Views != 10 {
			t.Errorf("Expected views 10, got %d", resp.Views)
		}

		if resp.Likes != 5 {
			t.Errorf("Expected likes 5, got %d", resp.Likes)
		}

		if resp.Color != "#FFAA00" {
			t.Errorf("Expected color #FFAA00, got %s", resp.Color)
		}

		if resp.UserInteraction == nil {
			t.Error("Expected user interaction to be set")
		}
	})

	t.Run("version response", func(t *testing.T) {
		versionTitle := "v1.2.3"
		event := &models.Calendar{
			ID:           2,
			Title:        "Version Update",
			Content:      "Version content",
			IsVersion:    true,
			VersionTitle: &versionTitle,
			StartsAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		}

		resp := buildEventResponse(event, nil, nil)

		if resp.Id != 2 {
			t.Errorf("Expected ID 2, got %d", resp.Id)
		}

		if resp.VersionTitle != "v1.2.3" {
			t.Errorf("Expected version title v1.2.3, got %s", resp.VersionTitle)
		}

		if resp.Views != 0 {
			t.Errorf("Expected views 0 for version, got %d", resp.Views)
		}

		if resp.UserInteraction != nil {
			t.Error("Expected no user interaction for version")
		}
	})
}
