package service

import (
	"context"

	"metargb/notification-service/internal/models"
)

// EmailService exposes email-related operations to transport handlers.
type EmailService interface {
	SendEmail(ctx context.Context, payload models.EmailPayload) (string, error)
}

type emailService struct {
	channel EmailChannel
}

// NewEmailService creates a default email service backed by the provided channel.
func NewEmailService(channel EmailChannel) EmailService {
	if channel == nil {
		channel = NewEmailChannel()
	}
	return &emailService{
		channel: channel,
	}
}

func (s *emailService) SendEmail(ctx context.Context, payload models.EmailPayload) (string, error) {
	return s.channel.SendEmail(ctx, payload)
}
