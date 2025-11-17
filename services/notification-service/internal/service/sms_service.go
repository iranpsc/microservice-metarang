package service

import (
	"context"

	"metargb/notification-service/internal/models"
)

// SMSService exposes SMS-related operations to transport handlers.
type SMSService interface {
	SendSMS(ctx context.Context, payload models.SMSPayload) (string, error)
	SendOTP(ctx context.Context, payload models.OTPPayload) (string, error)
}

type smsService struct {
	channel SMSChannel
}

// NewSMSService creates a default SMS service backed by the provided channel.
func NewSMSService(channel SMSChannel) SMSService {
	if channel == nil {
		channel = NewSMSChannel()
	}
	return &smsService{
		channel: channel,
	}
}

func (s *smsService) SendSMS(ctx context.Context, payload models.SMSPayload) (string, error) {
	return s.channel.SendSMS(ctx, payload)
}

func (s *smsService) SendOTP(ctx context.Context, payload models.OTPPayload) (string, error) {
	return s.channel.SendOTP(ctx, payload)
}
