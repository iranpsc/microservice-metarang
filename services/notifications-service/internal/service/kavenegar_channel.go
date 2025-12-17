package service

import (
	"context"
	"fmt"
	"log"

	"metargb/notifications-service/internal/models"

	"github.com/kavenegar/kavenegar-go"
)

type kavenegarSMSChannel struct {
	api    *kavenegar.Kavenegar
	sender string
}

// NewKavenegarSMSChannel creates a new Kavenegar SMS channel implementation.
func NewKavenegarSMSChannel(apiKey, sender string) SMSChannel {
	if apiKey == "" {
		log.Println("Warning: Kavenegar API key is empty, SMS sending will fail")
		return &noopSMSChannel{}
	}

	api := kavenegar.New(apiKey)
	return &kavenegarSMSChannel{
		api:    api,
		sender: sender,
	}
}

func (c *kavenegarSMSChannel) SendSMS(ctx context.Context, payload models.SMSPayload) (string, error) {
	if payload.Phone == "" {
		return "", fmt.Errorf("phone number is required")
	}

	// If template is provided, use Kavenegar Verify.Lookup for template-based sending
	if payload.Template != "" {
		// Extract token from the map
		var token string
		if val, ok := payload.Tokens["token"]; ok {
			token = val
		}
		// If no token in map, try "code" as fallback
		if token == "" {
			if val, ok := payload.Tokens["code"]; ok {
				token = val
			}
		}

		// Verify.Lookup API signature: (receptor, template, token, params)
		params := &kavenegar.VerifyLookupParam{}
		res, err := c.api.Verify.Lookup(payload.Phone, payload.Template, token, params)
		if err != nil {
			// Handle Kavenegar-specific errors
			switch err := err.(type) {
			case *kavenegar.APIError:
				return "", fmt.Errorf("kavenegar API error: %w", err)
			case *kavenegar.HTTPError:
				return "", fmt.Errorf("kavenegar HTTP error: %w", err)
			default:
				return "", fmt.Errorf("failed to send SMS via template: %w", err)
			}
		}
		return fmt.Sprintf("%d", res.MessageID), nil
	}

	// Regular SMS sending using Message.Send
	if payload.Message == "" {
		return "", fmt.Errorf("message is required when template is not provided")
	}

	res, err := c.api.Message.Send(c.sender, []string{payload.Phone}, payload.Message, nil)
	if err != nil {
		// Handle Kavenegar-specific errors
		switch err := err.(type) {
		case *kavenegar.APIError:
			return "", fmt.Errorf("kavenegar API error: %w", err)
		case *kavenegar.HTTPError:
			return "", fmt.Errorf("kavenegar HTTP error: %w", err)
		default:
			return "", fmt.Errorf("failed to send SMS: %w", err)
		}
	}

	// Message.Send returns a slice of MessageSendResult
	if len(res) == 0 {
		return "", fmt.Errorf("no response entries from Kavenegar")
	}

	return fmt.Sprintf("%d", res[0].MessageID), nil
}

func (c *kavenegarSMSChannel) SendOTP(ctx context.Context, payload models.OTPPayload) (string, error) {
	if payload.Phone == "" {
		return "", fmt.Errorf("phone number is required")
	}
	if payload.Code == "" {
		return "", fmt.Errorf("OTP code is required")
	}

	// Use Kavenegar Verify.Lookup for OTP sending with template
	// Template name defaults to "verify" but can be customized based on reason
	templateName := "verify"
	if payload.Reason != "" {
		templateName = payload.Reason
	}

	// Use Verify.Lookup API for OTP
	// API signature: (receptor, template, token, params)
	params := &kavenegar.VerifyLookupParam{}
	res, err := c.api.Verify.Lookup(payload.Phone, templateName, payload.Code, params)
	if err != nil {
		// If template lookup fails (e.g., template not configured), fall back to regular SMS
		log.Printf("Warning: Kavenegar Verify.Lookup failed for template '%s': %v, falling back to regular SMS", templateName, err)

		// Fallback to regular SMS with Persian message
		message := fmt.Sprintf("کد تأیید شما: %s", payload.Code)
		return c.SendSMS(ctx, models.SMSPayload{
			Phone:   payload.Phone,
			Message: message,
		})
	}

	return fmt.Sprintf("%d", res.MessageID), nil
}
