package service

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"metargb/notifications-service/internal/models"
	"metargb/notifications-service/internal/repository"
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
	// #region agent log
	logEntry, _ := json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "A",
		"location": "notification_service.go:52", "message": "SendNotification service entry",
		"data": map[string]interface{}{"userID": input.UserID, "type": input.Type, "title": input.Title, "repoNil": s.repo == nil},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	notification := &models.Notification{
		UserID:    input.UserID,
		Type:      input.Type,
		Title:     input.Title,
		Message:   input.Message,
		Data:      input.Data,
		CreatedAt: time.Now(),
	}

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "A",
		"location": "notification_service.go:62", "message": "Before repo.CreateNotification call",
		"data": map[string]interface{}{"userID": notification.UserID, "type": notification.Type, "title": notification.Title},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	id, err := s.repo.CreateNotification(ctx, notification)

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "A",
		"location": "notification_service.go:70", "message": "After repo.CreateNotification call",
		"data": map[string]interface{}{"id": id, "error": err != nil, "errorMsg": func() string { if err != nil { return err.Error() } else { return "" } }()},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

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
	// #region agent log
	logEntry, _ := json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "B",
		"location": "notification_service.go:85", "message": "GetNotifications service entry",
		"data": map[string]interface{}{"userID": userID, "filter": filter, "repoNil": s.repo == nil},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	result, total, err := s.repo.ListNotifications(ctx, userID, filter)

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "B",
		"location": "notification_service.go:95", "message": "After repo.ListNotifications call",
		"data": map[string]interface{}{"count": len(result), "total": total, "error": err != nil, "errorMsg": func() string { if err != nil { return err.Error() } else { return "" } }()},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	return result, total, err
}

func (s *notificationService) MarkAsRead(ctx context.Context, notificationID string, userID uint64) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uint64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
