package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/notifications-service/internal/errs"
	"metarang/notifications-service/internal/models"
	"metarang/notifications-service/internal/service"
)

func TestNewKavenegarSMSChannel_EmptyKeyUsesNoop(t *testing.T) {
	ch := service.NewKavenegarSMSChannel("", "10008663")
	_, err := ch.SendSMS(context.Background(), models.SMSPayload{Phone: "09120000000", Message: "hi"})
	assert.ErrorIs(t, err, errs.ErrNotImplemented)
}

func TestNewSMSChannel_KavenegarConfigured(t *testing.T) {
	ch := service.NewSMSChannel(service.SMSChannelConfig{
		Provider: "kavenegar",
		APIKey:   "693835337A3377547771646A327733396D6D79393539744E6A5372487644456F3448434C773974337234733D",
		Sender:   "",
	})
	_, err := ch.SendSMS(context.Background(), models.SMSPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone number is required")
}

func TestNewSMSChannel_KavenegarWithSender(t *testing.T) {
	ch := service.NewSMSChannel(service.SMSChannelConfig{
		Provider: "kavenegar",
		APIKey:   "valid-api-key",
		Sender:   "10001",
	})
	_, err := ch.SendOTP(context.Background(), models.OTPPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone number is required")
}

func TestKavenegarSendSMSValidation(t *testing.T) {
	ch := service.NewKavenegarSMSChannel("valid-api-key", "10008663")

	_, err := ch.SendSMS(context.Background(), models.SMSPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone number is required")

	_, err = ch.SendSMS(context.Background(), models.SMSPayload{Phone: "09120000000"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "message is required"))
}

func TestKavenegarSendOTPValidation(t *testing.T) {
	ch := service.NewKavenegarSMSChannel("valid-api-key", "10008663")

	_, err := ch.SendOTP(context.Background(), models.OTPPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone number is required")

	_, err = ch.SendOTP(context.Background(), models.OTPPayload{Phone: "09120000000"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OTP code is required")
}
