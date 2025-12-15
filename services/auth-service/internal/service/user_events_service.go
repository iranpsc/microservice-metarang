package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

var (
	ErrUserEventNotFound       = errors.New("user event not found")
	ErrUserEventReportExists   = errors.New("user event report already exists")
	ErrInvalidCitizenCode      = errors.New("invalid citizen code")
	ErrEventDescriptionTooLong = errors.New("event description exceeds maximum length of 500 characters")
	ErrResponseTooLong         = errors.New("response exceeds maximum length of 300 characters")
	ErrUserEventReportNotFound = errors.New("user event report not found")
)

type UserEventsService interface {
	ListUserEvents(ctx context.Context, userID uint64, page int32) ([]*models.UserEvent, string, string, error)
	GetUserEvent(ctx context.Context, userID, eventID uint64) (*models.UserEvent, *models.UserEventReport, []*models.UserEventReportResponse, error)
	ReportUserEvent(ctx context.Context, userID, eventID uint64, suspeciousCitizen *string, eventDescription string) (*models.UserEventReport, error)
	SendReportResponse(ctx context.Context, userID, eventID uint64, responserName, response string) (*models.UserEventReportResponse, error)
	CloseEventReport(ctx context.Context, userID, eventID uint64) error
}

type userEventsService struct {
	activityRepo repository.ActivityRepository
	userRepo     repository.UserRepository
}

func NewUserEventsService(
	activityRepo repository.ActivityRepository,
	userRepo repository.UserRepository,
) UserEventsService {
	return &userEventsService{
		activityRepo: activityRepo,
		userRepo:     userRepo,
	}
}

// ListUserEvents lists user events with simple pagination (10 per page)
func (s *userEventsService) ListUserEvents(ctx context.Context, userID uint64, page int32) ([]*models.UserEvent, string, string, error) {
	if page < 1 {
		page = 1
	}

	events, err := s.activityRepo.GetUserEventsByUserID(ctx, userID, page)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to list user events: %w", err)
	}

	// Simple pagination: if we got 11 items, there's a next page
	var nextPageURL, prevPageURL string
	if len(events) > 10 {
		// Remove the extra item used for pagination check
		events = events[:10]
		nextPageURL = fmt.Sprintf("/api/events?page=%d", page+1)
	}
	if page > 1 {
		prevPageURL = fmt.Sprintf("/api/events?page=%d", page-1)
	}

	return events, nextPageURL, prevPageURL, nil
}

// GetUserEvent gets a specific user event with its report and responses
func (s *userEventsService) GetUserEvent(ctx context.Context, userID, eventID uint64) (*models.UserEvent, *models.UserEventReport, []*models.UserEventReportResponse, error) {
	// Get the event, ensuring it belongs to the user
	event, err := s.activityRepo.GetUserEventByID(ctx, userID, eventID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get user event: %w", err)
	}
	if event == nil {
		return nil, nil, nil, ErrUserEventNotFound
	}

	// Get the report if it exists
	report, err := s.activityRepo.GetUserEventReportByEventID(ctx, eventID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get user event report: %w", err)
	}

	var responses []*models.UserEventReportResponse
	if report != nil {
		// Get responses for the report
		responses, err = s.activityRepo.GetUserEventReportResponses(ctx, report.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get user event report responses: %w", err)
		}
	}

	return event, report, responses, nil
}

// ReportUserEvent creates a report for a user event
func (s *userEventsService) ReportUserEvent(ctx context.Context, userID, eventID uint64, suspeciousCitizen *string, eventDescription string) (*models.UserEventReport, error) {
	// Validate event description
	eventDescription = strings.TrimSpace(eventDescription)
	if eventDescription == "" {
		return nil, errors.New("event_description is required")
	}
	if len(eventDescription) > 500 {
		return nil, ErrEventDescriptionTooLong
	}

	// Verify the event exists and belongs to the user
	event, err := s.activityRepo.GetUserEventByID(ctx, userID, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify user event: %w", err)
	}
	if event == nil {
		return nil, ErrUserEventNotFound
	}

	// Check if report already exists
	existingReport, err := s.activityRepo.GetUserEventReportByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing report: %w", err)
	}
	if existingReport != nil {
		return nil, ErrUserEventReportExists
	}

	// Validate citizen code if provided
	var suspeciousCitizenNull sql.NullString
	if suspeciousCitizen != nil && *suspeciousCitizen != "" {
		// Verify the citizen code exists
		citizen, err := s.userRepo.FindByCode(ctx, *suspeciousCitizen)
		if err != nil {
			return nil, fmt.Errorf("failed to validate citizen code: %w", err)
		}
		if citizen == nil {
			return nil, ErrInvalidCitizenCode
		}
		suspeciousCitizenNull = sql.NullString{String: *suspeciousCitizen, Valid: true}
	}

	// Create the report
	report := &models.UserEventReport{
		UserEventID:       eventID,
		SuspeciousCitizen: suspeciousCitizenNull,
		EventDescription:  eventDescription,
		Status:            0, // Initial status
		Closed:            false,
	}

	if err := s.activityRepo.CreateUserEventReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create user event report: %w", err)
	}

	return report, nil
}

// SendReportResponse sends a response to a user event report
func (s *userEventsService) SendReportResponse(ctx context.Context, userID, eventID uint64, responserName, response string) (*models.UserEventReportResponse, error) {
	// Validate response
	response = strings.TrimSpace(response)
	if response == "" {
		return nil, errors.New("response is required")
	}
	if len(response) > 300 {
		return nil, ErrResponseTooLong
	}

	// Verify the event exists and belongs to the user
	event, err := s.activityRepo.GetUserEventByID(ctx, userID, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify user event: %w", err)
	}
	if event == nil {
		return nil, ErrUserEventNotFound
	}

	// Get the report
	report, err := s.activityRepo.GetUserEventReportByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user event report: %w", err)
	}
	if report == nil {
		return nil, ErrUserEventReportNotFound
	}

	// Create the response
	reportResponse := &models.UserEventReportResponse{
		UserEventReportID: report.ID,
		Response:          response,
		ResponserName:     responserName,
	}

	if err := s.activityRepo.CreateUserEventReportResponse(ctx, reportResponse); err != nil {
		return nil, fmt.Errorf("failed to create user event report response: %w", err)
	}

	// Update report status to 1 (handled)
	if err := s.activityRepo.UpdateUserEventReportStatus(ctx, report.ID, 1); err != nil {
		return nil, fmt.Errorf("failed to update report status: %w", err)
	}

	return reportResponse, nil
}

// CloseEventReport closes a user event report
func (s *userEventsService) CloseEventReport(ctx context.Context, userID, eventID uint64) error {
	// Verify the event exists and belongs to the user
	event, err := s.activityRepo.GetUserEventByID(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("failed to verify user event: %w", err)
	}
	if event == nil {
		return ErrUserEventNotFound
	}

	// Get the report
	report, err := s.activityRepo.GetUserEventReportByEventID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get user event report: %w", err)
	}
	if report == nil {
		return ErrUserEventReportNotFound
	}

	// Close the report
	if err := s.activityRepo.CloseUserEventReport(ctx, report.ID); err != nil {
		return fmt.Errorf("failed to close user event report: %w", err)
	}

	return nil
}
