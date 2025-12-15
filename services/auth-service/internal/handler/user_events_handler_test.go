package handler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

// Mock implementations

type mockUserEventsService struct {
	listUserEventsFunc     func(ctx context.Context, userID uint64, page int32) ([]*models.UserEvent, string, string, error)
	getUserEventFunc       func(ctx context.Context, userID, eventID uint64) (*models.UserEvent, *models.UserEventReport, []*models.UserEventReportResponse, error)
	reportUserEventFunc    func(ctx context.Context, userID, eventID uint64, suspeciousCitizen *string, eventDescription string) (*models.UserEventReport, error)
	sendReportResponseFunc func(ctx context.Context, userID, eventID uint64, responserName, response string) (*models.UserEventReportResponse, error)
	closeEventReportFunc   func(ctx context.Context, userID, eventID uint64) error
}

type mockUserRepo struct {
	findByIDFunc func(ctx context.Context, id uint64) (*models.User, error)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint64) (*models.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

// Test ListUserEvents

func TestUserEventsHandler_ListUserEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.listUserEventsFunc = func(ctx context.Context, userID uint64, page int32) ([]*models.UserEvent, string, string, error) {
			events := []*models.UserEvent{
				{
					ID:        1,
					UserID:    userID,
					Event:     "ورود به حساب کاربری",
					IP:        "192.168.1.1",
					Device:    "Chrome",
					Status:    1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}
			return events, "/api/events?page=2", "", nil
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.ListUserEventsRequest{
			UserId: 1,
			Page:   1,
		}

		resp, err := handler.ListUserEvents(ctx, req)
		if err != nil {
			t.Fatalf("ListUserEvents failed: %v", err)
		}

		if len(resp.Data) != 1 {
			t.Errorf("Expected 1 event, got %d", len(resp.Data))
		}
		if resp.Pagination.NextPageUrl != "/api/events?page=2" {
			t.Errorf("Expected next page URL, got %s", resp.Pagination.NextPageUrl)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.listUserEventsFunc = func(ctx context.Context, userID uint64, page int32) ([]*models.UserEvent, string, string, error) {
			return nil, "", "", errors.New("database error")
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.ListUserEventsRequest{
			UserId: 1,
			Page:   1,
		}

		_, err := handler.ListUserEvents(ctx, req)
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

// Test GetUserEvent

func TestUserEventsHandler_GetUserEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get with report", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.getUserEventFunc = func(ctx context.Context, userID, eventID uint64) (*models.UserEvent, *models.UserEventReport, []*models.UserEventReportResponse, error) {
			event := &models.UserEvent{
				ID:        eventID,
				UserID:    userID,
				Event:     "ورود به حساب کاربری",
				IP:        "192.168.1.1",
				Device:    "Chrome",
				Status:    1,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			report := &models.UserEventReport{
				ID:               1,
				UserEventID:      eventID,
				EventDescription: "Suspicious activity detected",
				Status:           0,
				Closed:           false,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			return event, report, nil, nil
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.GetUserEventRequest{
			UserId:  1,
			EventId: 1,
		}

		resp, err := handler.GetUserEvent(ctx, req)
		if err != nil {
			t.Fatalf("GetUserEvent failed: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Expected event data")
		}
		if resp.Data.Report == nil {
			t.Fatal("Expected report data")
		}
		if resp.Data.Report.EventDescription != "Suspicious activity detected" {
			t.Errorf("Expected event description, got %s", resp.Data.Report.EventDescription)
		}
	})

	t.Run("event not found", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.getUserEventFunc = func(ctx context.Context, userID, eventID uint64) (*models.UserEvent, *models.UserEventReport, []*models.UserEventReportResponse, error) {
			return nil, nil, nil, service.ErrUserEventNotFound
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.GetUserEventRequest{
			UserId:  1,
			EventId: 999,
		}

		_, err := handler.GetUserEvent(ctx, req)
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

// Test ReportUserEvent

func TestUserEventsHandler_ReportUserEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("successful report", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.reportUserEventFunc = func(ctx context.Context, userID, eventID uint64, suspeciousCitizen *string, eventDescription string) (*models.UserEventReport, error) {
			report := &models.UserEventReport{
				ID:               1,
				UserEventID:      eventID,
				EventDescription: eventDescription,
				Status:           0,
				Closed:           false,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			if suspeciousCitizen != nil {
				report.SuspeciousCitizen = sql.NullString{String: *suspeciousCitizen, Valid: true}
			}
			return report, nil
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.ReportUserEventRequest{
			UserId:           1,
			EventId:          1,
			EventDescription: "Suspicious activity detected",
		}

		resp, err := handler.ReportUserEvent(ctx, req)
		if err != nil {
			t.Fatalf("ReportUserEvent failed: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Expected report data")
		}
		if resp.Data.EventDescription != "Suspicious activity detected" {
			t.Errorf("Expected event description, got %s", resp.Data.EventDescription)
		}
	})

	t.Run("missing event_description", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.ReportUserEventRequest{
			UserId:  1,
			EventId: 1,
			// EventDescription is empty
		}

		_, err := handler.ReportUserEvent(ctx, req)
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

	t.Run("event_description too long", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		longDescription := make([]byte, 501)
		for i := range longDescription {
			longDescription[i] = 'a'
		}

		req := &pb.ReportUserEventRequest{
			UserId:           1,
			EventId:          1,
			EventDescription: string(longDescription),
		}

		_, err := handler.ReportUserEvent(ctx, req)
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

	t.Run("event not found", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.reportUserEventFunc = func(ctx context.Context, userID, eventID uint64, suspeciousCitizen *string, eventDescription string) (*models.UserEventReport, error) {
			return nil, service.ErrUserEventNotFound
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.ReportUserEventRequest{
			UserId:           1,
			EventId:          999,
			EventDescription: "Suspicious activity",
		}

		_, err := handler.ReportUserEvent(ctx, req)
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

	t.Run("invalid citizen code", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.reportUserEventFunc = func(ctx context.Context, userID, eventID uint64, suspeciousCitizen *string, eventDescription string) (*models.UserEventReport, error) {
			return nil, service.ErrInvalidCitizenCode
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		citizenCode := "INVALID"
		req := &pb.ReportUserEventRequest{
			UserId:            1,
			EventId:           1,
			SuspeciousCitizen: citizenCode,
			EventDescription:  "Suspicious activity",
		}

		_, err := handler.ReportUserEvent(ctx, req)
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
}

// Test SendReportResponse

func TestUserEventsHandler_SendReportResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("successful response", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.sendReportResponseFunc = func(ctx context.Context, userID, eventID uint64, responserName, response string) (*models.UserEventReportResponse, error) {
			return &models.UserEventReportResponse{
				ID:                1,
				UserEventReportID: 1,
				Response:          response,
				ResponserName:     responserName,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}, nil
		}

		mockUserRepo := &mockUserRepo{}
		mockUserRepo.findByIDFunc = func(ctx context.Context, id uint64) (*models.User, error) {
			return &models.User{
				ID:   id,
				Name: "Test User",
			}, nil
		}

		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.SendReportResponseRequest{
			UserId:   1,
			EventId:  1,
			Response: "We have reviewed your report and taken action.",
		}

		resp, err := handler.SendReportResponse(ctx, req)
		if err != nil {
			t.Fatalf("SendReportResponse failed: %v", err)
		}

		if resp.Data == nil {
			t.Fatal("Expected response data")
		}
		if resp.Data.Response != "We have reviewed your report and taken action." {
			t.Errorf("Expected response text, got %s", resp.Data.Response)
		}
		if resp.Data.ResponserName != "Test User" {
			t.Errorf("Expected responser name, got %s", resp.Data.ResponserName)
		}
	})

	t.Run("missing response", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.SendReportResponseRequest{
			UserId:  1,
			EventId: 1,
			// Response is empty
		}

		_, err := handler.SendReportResponse(ctx, req)
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

	t.Run("response too long", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		longResponse := make([]byte, 301)
		for i := range longResponse {
			longResponse[i] = 'a'
		}

		req := &pb.SendReportResponseRequest{
			UserId:   1,
			EventId:  1,
			Response: string(longResponse),
		}

		_, err := handler.SendReportResponse(ctx, req)
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

	t.Run("event not found", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.sendReportResponseFunc = func(ctx context.Context, userID, eventID uint64, responserName, response string) (*models.UserEventReportResponse, error) {
			return nil, service.ErrUserEventNotFound
		}

		mockUserRepo := &mockUserRepo{}
		mockUserRepo.findByIDFunc = func(ctx context.Context, id uint64) (*models.User, error) {
			return &models.User{ID: id, Name: "Test User"}, nil
		}

		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.SendReportResponseRequest{
			UserId:   1,
			EventId:  999,
			Response: "Response text",
		}

		_, err := handler.SendReportResponse(ctx, req)
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

// Test CloseEventReport

func TestUserEventsHandler_CloseEventReport(t *testing.T) {
	ctx := context.Background()

	t.Run("successful close", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.closeEventReportFunc = func(ctx context.Context, userID, eventID uint64) error {
			return nil
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.CloseEventReportRequest{
			UserId:  1,
			EventId: 1,
		}

		_, err := handler.CloseEventReport(ctx, req)
		if err != nil {
			t.Fatalf("CloseEventReport failed: %v", err)
		}
	})

	t.Run("event not found", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.closeEventReportFunc = func(ctx context.Context, userID, eventID uint64) error {
			return service.ErrUserEventNotFound
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.CloseEventReportRequest{
			UserId:  1,
			EventId: 999,
		}

		_, err := handler.CloseEventReport(ctx, req)
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

	t.Run("report not found", func(t *testing.T) {
		mockService := &mockUserEventsService{}
		mockService.closeEventReportFunc = func(ctx context.Context, userID, eventID uint64) error {
			return service.ErrUserEventReportNotFound
		}

		mockUserRepo := &mockUserRepo{}
		handler := &userEventsHandler{
			service:  mockService,
			userRepo: mockUserRepo,
		}

		req := &pb.CloseEventReportRequest{
			UserId:  1,
			EventId: 1,
		}

		_, err := handler.CloseEventReport(ctx, req)
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
