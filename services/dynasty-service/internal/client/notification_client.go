package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "metargb/shared/pb/notifications"
)

// NotificationClient wraps gRPC client for Notifications Service
type NotificationClient struct {
	notificationClient pb.NotificationServiceClient
	conn               *grpc.ClientConn
}

// NewNotificationClient creates a new Notifications Service client
func NewNotificationClient(address string) (*NotificationClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notifications service at %s: %w", address, err)
	}

	return &NotificationClient{
		notificationClient: pb.NewNotificationServiceClient(conn),
		conn:               conn,
	}, nil
}

// Close closes the gRPC connection
func (c *NotificationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendNotification sends a notification to a user
func (c *NotificationClient) SendNotification(ctx context.Context, userID uint64, notificationType, title, message string, data map[string]string, sendSMS, sendEmail bool) error {
	req := &pb.SendNotificationRequest{
		UserId:     userID,
		Type:       notificationType,
		Title:      title,
		Message:    message,
		Data:       data,
		SendSms:    sendSMS,
		SendEmail:  sendEmail,
	}

	resp, err := c.notificationClient.SendNotification(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	if !resp.Sent {
		return fmt.Errorf("notification was not sent")
	}

	return nil
}

