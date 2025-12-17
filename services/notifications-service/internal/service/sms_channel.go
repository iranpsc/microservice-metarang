package service

import (
	"context"
	"log"
	"os"

	"metargb/notifications-service/internal/errs"
	"metargb/notifications-service/internal/models"
)

type noopSMSChannel struct{}

// NewSMSChannel returns an SMS channel implementation based on the SMS_PROVIDER environment variable.
// Supported providers: "kavenegar" (defaults to noop if not configured or provider not supported).
func NewSMSChannel() SMSChannel {
	provider := os.Getenv("SMS_PROVIDER")
	apiKey := os.Getenv("SMS_API_KEY")
	sender := os.Getenv("SMS_SENDER")

	log.Printf("SMS Channel initialization: provider=%s, apiKey set=%v, sender=%s", provider, apiKey != "", sender)

	switch provider {
	case "kavenegar":
		if apiKey == "" {
			// Return noop if API key is not configured
			log.Println("Warning: SMS_PROVIDER is 'kavenegar' but SMS_API_KEY is not set, using noop channel")
			return &noopSMSChannel{}
		}
		// Default sender if not provided (from Laravel config)
		if sender == "" {
			sender = "10008663"
		}
		log.Printf("Initializing Kavenegar SMS channel with sender: %s", sender)
		return NewKavenegarSMSChannel(apiKey, sender)
	default:
		// Return noop for unknown providers or when provider is not set
		if provider == "" {
			log.Println("Warning: SMS_PROVIDER is not set, using noop channel")
		} else {
			log.Printf("Warning: Unknown SMS_PROVIDER '%s', using noop channel", provider)
		}
		return &noopSMSChannel{}
	}
}

func (c *noopSMSChannel) SendSMS(ctx context.Context, payload models.SMSPayload) (string, error) {
	return "", errs.ErrNotImplemented
}

func (c *noopSMSChannel) SendOTP(ctx context.Context, payload models.OTPPayload) (string, error) {
	return "", errs.ErrNotImplemented
}
