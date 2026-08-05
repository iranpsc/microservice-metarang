package service

import (
	"context"
	"fmt"
	"log"

	"metarang/notifications-service/internal/models"

	"github.com/kavenegar/kavenegar-go"
)

const kavenegarOTPTemplate = "verify"

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

func (c *kavenegarSMSChannel) verifyLookup(receptor, template, token string) (kavenegar.Message, error) {
	// Pass nil for params so the SDK does not add empty Token2/Token3/Type fields.
	return c.api.Verify.Lookup(receptor, template, token, nil)
}

func (c *kavenegarSMSChannel) SendSMS(ctx context.Context, payload models.SMSPayload) (string, error) {
	if payload.Phone == "" {
		return "", fmt.Errorf("phone number is required")
	}

	if payload.Template != "" {
		token := extractTemplateToken(payload.Tokens)
		res, err := c.verifyLookup(payload.Phone, payload.Template, token)
		if err != nil {
			return "", mapKavenegarError(err)
		}
		return fmt.Sprintf("%d", res.MessageID), nil
	}

	if payload.Message == "" {
		return "", fmt.Errorf("message is required when template is not provided")
	}

	res, err := c.api.Message.Send(c.sender, []string{payload.Phone}, payload.Message, nil)
	if err != nil {
		return "", mapKavenegarError(err)
	}

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

	res, err := c.verifyLookup(payload.Phone, kavenegarOTPTemplate, payload.Code)
	if err != nil {
		return "", mapKavenegarError(err)
	}

	return fmt.Sprintf("%d", res.MessageID), nil
}

func extractTemplateToken(tokens map[string]string) string {
	if tokens == nil {
		return ""
	}
	if val, ok := tokens["token"]; ok && val != "" {
		return val
	}
	if val, ok := tokens["code"]; ok && val != "" {
		return val
	}
	return ""
}

func mapKavenegarError(err error) error {
	switch e := err.(type) {
	case *kavenegar.APIError:
		if hint := kavenegarAPIErrorHint(e.Status); hint != "" {
			return fmt.Errorf("kavenegar API error: %w (%s)", e, hint)
		}
		return fmt.Errorf("kavenegar API error: %w", e)
	case *kavenegar.HTTPError:
		return fmt.Errorf("kavenegar HTTP error: %w", e)
	default:
		return fmt.Errorf("kavenegar request failed: %w", err)
	}
}

func kavenegarAPIErrorHint(status int) string {
	switch status {
	case 403:
		return "invalid API key; set SMS_API_KEY in notifications-service/config.env"
	case 416:
		return "request IP is not allowed in Kavenegar panel; add your server/Docker egress IP to API access list or disable IP restriction"
	case 424:
		return "verification template not found or not approved; ensure template \"verify\" exists in Kavenegar console"
	default:
		return ""
	}
}
