package client

import (
	"context"
	"fmt"
	"time"

	pb "metargb/shared/pb/notifications"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NotificationClient wraps gRPC client for Notification Service
type NotificationClient struct {
	client pb.NotificationServiceClient
	conn   *grpc.ClientConn
}

// NewNotificationClient creates a new Notification Service client
func NewNotificationClient(address string) (*NotificationClient, error) {
	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service at %s: %w", address, err)
	}

	return &NotificationClient{
		client: pb.NewNotificationServiceClient(conn),
		conn:   conn,
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
func (c *NotificationClient) SendNotification(ctx context.Context, userID uint64, notificationType, title, message string, data map[string]string) error {
	req := &pb.SendNotificationRequest{
		UserId:    userID,
		Type:      notificationType,
		Title:     title,
		Message:   message,
		Data:      data,
		SendSms:   false,
		SendEmail: false,
	}

	_, err := c.client.SendNotification(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
