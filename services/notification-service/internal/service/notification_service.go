package service

import (
	"context"
	"time"

	"metargb/notification-service/internal/models"
	"metargb/notification-service/internal/repository"
)

// SendNotificationInput represents the information required to dispatch a notification.
type SendNotificationInput struct {
	UserID    uint64
	Type      string
	Title     string
	Message   string
	Data      map[string]string
	SendSMS   bool
	SendEmail bool

	SMSPayload   *models.SMSPayload
	EmailPayload *models.EmailPayload
}

// NotificationService encapsulates business logic for notifications.
type NotificationService interface {
	SendNotification(ctx context.Context, input SendNotificationInput) (*models.NotificationResult, error)
	GetNotifications(ctx context.Context, userID uint64, filter models.NotificationFilter) ([]models.Notification, int64, error)
	MarkAsRead(ctx context.Context, notificationID string, userID uint64) error
	MarkAllAsRead(ctx context.Context, userID uint64) error
}

type notificationService struct {
	repo         *repository.NotificationRepository
	smsChannel   SMSChannel
	emailChannel EmailChannel
}

// NewNotificationService creates a notification service implementation.
func NewNotificationService(
	repo *repository.NotificationRepository,
	smsChannel SMSChannel,
	emailChannel EmailChannel,
) NotificationService {
	return &notificationService{
		repo:         repo,
		smsChannel:   smsChannel,
		emailChannel: emailChannel,
	}
}

func (s *notificationService) SendNotification(ctx context.Context, input SendNotificationInput) (*models.NotificationResult, error) {
	notification := &models.Notification{
		UserID:    input.UserID,
		Type:      input.Type,
		Title:     input.Title,
		Message:   input.Message,
		Data:      input.Data,
		CreatedAt: time.Now(),
	}

	id, err := s.repo.CreateNotification(ctx, notification)
	if err != nil {
		return nil, err
	}

	if input.SendSMS && s.smsChannel != nil && input.SMSPayload != nil {
		if _, err := s.smsChannel.SendSMS(ctx, *input.SMSPayload); err != nil {
			return &models.NotificationResult{ID: id, Sent: false}, err
		}
	}

	if input.SendEmail && s.emailChannel != nil && input.EmailPayload != nil {
		if _, err := s.emailChannel.SendEmail(ctx, *input.EmailPayload); err != nil {
			return &models.NotificationResult{ID: id, Sent: false}, err
		}
	}

	return &models.NotificationResult{
		ID:   id,
		Sent: true,
	}, nil
}

func (s *notificationService) GetNotifications(ctx context.Context, userID uint64, filter models.NotificationFilter) ([]models.Notification, int64, error) {
	return s.repo.ListNotifications(ctx, userID, filter)
}

func (s *notificationService) MarkAsRead(ctx context.Context, notificationID string, userID uint64) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uint64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
