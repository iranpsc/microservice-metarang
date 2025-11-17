package service

import (
	"context"

	"metargb/notification-service/internal/errs"
	"metargb/notification-service/internal/models"
)

type noopSMSChannel struct{}

// NewSMSChannel returns a placeholder SMS channel implementation.
func NewSMSChannel() SMSChannel {
	return &noopSMSChannel{}
}

func (c *noopSMSChannel) SendSMS(ctx context.Context, payload models.SMSPayload) (string, error) {
	return "", errs.ErrNotImplemented
}

func (c *noopSMSChannel) SendOTP(ctx context.Context, payload models.OTPPayload) (string, error) {
	return "", errs.ErrNotImplemented
}
