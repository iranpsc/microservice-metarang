package service

import (
	"context"
	"fmt"
	"metargb/support-service/internal/models"
	"metargb/support-service/internal/repository"
)

type UserEventService interface {
	CreateUserEvent(ctx context.Context, userID uint64, title, description, eventDate string) (*models.UserEvent, error)
	GetUserEvents(ctx context.Context, userID uint64, page, perPage int32) ([]*models.UserEvent, int, error)
	GetUserEvent(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error)
	ReportUserEvent(ctx context.Context, eventID uint64, suspiciousCitizen, eventDescription string) (*models.UserEventReport, error)
	SendEventReportResponse(ctx context.Context, reportID uint64, responderName, response string) error
}

type userEventService struct {
	userEventRepo repository.UserEventRepository
}

func NewUserEventService(userEventRepo repository.UserEventRepository) UserEventService {
	return &userEventService{
		userEventRepo: userEventRepo,
	}
}

func (s *userEventService) CreateUserEvent(ctx context.Context, userID uint64, title, description, eventDate string) (*models.UserEvent, error) {
	// Note: The proto expects title, description, event_date but Laravel UserEvent has event, ip, device, status
	// We'll store the title as event field
	event := &models.UserEvent{
		UserID: userID,
		Event:  title,
		IP:     "0.0.0.0", // Default IP - should be provided by gateway
		Device: "unknown", // Default device - should be provided by gateway
		Status: true,
	}

	return s.userEventRepo.Create(ctx, event)
}

func (s *userEventService) GetUserEvents(ctx context.Context, userID uint64, page, perPage int32) ([]*models.UserEvent, int, error) {
	if perPage <= 0 {
		perPage = 10
	}
	if page <= 0 {
		page = 1
	}

	return s.userEventRepo.GetByUserID(ctx, userID, page, perPage)
}

func (s *userEventService) GetUserEvent(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
	return s.userEventRepo.GetByID(ctx, eventID)
}

func (s *userEventService) ReportUserEvent(ctx context.Context, eventID uint64, suspiciousCitizen, eventDescription string) (*models.UserEventReport, error) {
	var suspiciousCitizenPtr *string
	if suspiciousCitizen != "" {
		suspiciousCitizenPtr = &suspiciousCitizen
	}

	report := &models.UserEventReport{
		UserEventID:       eventID,
		SuspeciousCitizen: suspiciousCitizenPtr,
		EventDescription:  eventDescription,
		Status:            0, // Default status
		Closed:            false,
	}

	return s.userEventRepo.CreateReport(ctx, report)
}

func (s *userEventService) SendEventReportResponse(ctx context.Context, reportID uint64, responderName, response string) error {
	// Create response
	reportResponse := &models.UserEventReportResponse{
		UserEventReportID: reportID,
		Response:          response,
		ResponserName:     responderName,
	}

	_, err := s.userEventRepo.CreateReportResponse(ctx, reportResponse)
	if err != nil {
		return fmt.Errorf("failed to create response: %w", err)
	}

	// Update report status to 1 (matching Laravel)
	err = s.userEventRepo.UpdateReportStatus(ctx, reportID, 1)
	if err != nil {
		return fmt.Errorf("failed to update report status: %w", err)
	}

	return nil
}
