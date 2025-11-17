package repository

import (
	"context"
	"database/sql"

	"metargb/notification-service/internal/errs"
	"metargb/notification-service/internal/models"
)

// NotificationRepository handles database interactions for notifications.
type NotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new repository instance.
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{
		db: db,
	}
}

// CreateNotification persists a notification record.
func (r *NotificationRepository) CreateNotification(ctx context.Context, notification *models.Notification) (string, error) {
	return "", errs.ErrNotImplemented
}

// ListNotifications retrieves notifications for a user along with the total count.
func (r *NotificationRepository) ListNotifications(ctx context.Context, userID uint64, filter models.NotificationFilter) ([]models.Notification, int64, error) {
	return nil, 0, errs.ErrNotImplemented
}

// MarkAsRead marks a single notification as read.
func (r *NotificationRepository) MarkAsRead(ctx context.Context, notificationID string, userID uint64) error {
	return errs.ErrNotImplemented
}

// MarkAllAsRead marks all notifications as read for a user.
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uint64) error {
	return errs.ErrNotImplemented
}
