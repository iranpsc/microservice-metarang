package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kavenegar/kavenegar-go"

	"metarang/notifications-service/internal/errs"
	"metarang/notifications-service/internal/models"
)

func TestResolveSMSAPIKeyPrefersValidSMSKey(t *testing.T) {
	t.Setenv("SMS_API_KEY", "real-key")
	t.Setenv("KAVENEGAR_API_KEY", "fallback-key")

	if got := ResolveSMSAPIKey(); got != "real-key" {
		t.Fatalf("expected real-key, got %q", got)
	}
}

func TestResolveSMSAPIKeyFallsBackFromPlaceholder(t *testing.T) {
	t.Setenv("SMS_API_KEY", "change-me")
	t.Setenv("KAVENEGAR_API_KEY", "fallback-key")

	if got := ResolveSMSAPIKey(); got != "fallback-key" {
		t.Fatalf("expected fallback-key, got %q", got)
	}
}

func TestSMSAPIKeySource(t *testing.T) {
	t.Setenv("SMS_API_KEY", "real-key")
	t.Setenv("KAVENEGAR_API_KEY", "other-key")
	if got := SMSAPIKeySource(); got != "SMS_API_KEY" {
		t.Fatalf("expected SMS_API_KEY, got %q", got)
	}
}

func TestIsPlaceholderSMSAPIKey(t *testing.T) {
	if !IsPlaceholderSMSAPIKey("change-me") {
		t.Fatal("expected change-me to be a placeholder")
	}
	if IsPlaceholderSMSAPIKey("693835337A3377547771646A327733396D6D79393539744E6A5372487644456F3448434C773974337234733D") {
		t.Fatal("expected real key not to be a placeholder")
	}
}

func TestSMSAPIKeySourceFallbackAndNone(t *testing.T) {
	t.Setenv("SMS_API_KEY", "change-me")
	t.Setenv("KAVENEGAR_API_KEY", "legacy-key")
	if got := SMSAPIKeySource(); got != "KAVENEGAR_API_KEY" {
		t.Fatalf("expected KAVENEGAR_API_KEY, got %q", got)
	}

	t.Setenv("SMS_API_KEY", "")
	t.Setenv("KAVENEGAR_API_KEY", "")
	if got := SMSAPIKeySource(); got != "none" {
		t.Fatalf("expected none, got %q", got)
	}
}

func TestMaskAPIKey(t *testing.T) {
	if got := MaskAPIKey("short"); got != "***" {
		t.Fatalf("expected ***, got %q", got)
	}
	if got := MaskAPIKey("693835337A3377547771646A327733396D6D79393539744E6A5372487644456F3448434C773974337234733D"); got != "6938...733D" {
		t.Fatalf("unexpected mask: %q", got)
	}
}

func TestResolveSMSSender(t *testing.T) {
	t.Setenv("SMS_SENDER", "")
	t.Setenv("KAVENEGAR_SENDER", "20001")
	if got := ResolveSMSSender("default"); got != "20001" {
		t.Fatalf("expected KAVENEGAR sender, got %q", got)
	}

	t.Setenv("KAVENEGAR_SENDER", "")
	if got := ResolveSMSSender("default"); got != "default" {
		t.Fatalf("expected default sender, got %q", got)
	}
}

func TestExtractTemplateToken(t *testing.T) {
	if got := extractTemplateToken(nil); got != "" {
		t.Fatalf("expected empty token, got %q", got)
	}
	if got := extractTemplateToken(map[string]string{"token": "abc"}); got != "abc" {
		t.Fatalf("expected abc, got %q", got)
	}
	if got := extractTemplateToken(map[string]string{"code": "9999"}); got != "9999" {
		t.Fatalf("expected 9999, got %q", got)
	}
}

func TestKavenegarAPIErrorHint(t *testing.T) {
	tests := map[int]string{
		403: "invalid API key",
		416: "request IP is not allowed",
		424: "verification template not found",
		500: "",
	}
	for status, want := range tests {
		got := kavenegarAPIErrorHint(status)
		if want == "" && got != "" {
			t.Fatalf("status %d: expected empty hint, got %q", status, got)
		}
		if want != "" && !strings.Contains(got, want) {
			t.Fatalf("status %d: hint %q should contain %q", status, got, want)
		}
	}
}

func TestMapKavenegarError(t *testing.T) {
	apiErr := &kavenegar.APIError{Status: 403, Message: "invalid key"}
	err := mapKavenegarError(apiErr)
	if err == nil || !strings.Contains(err.Error(), "invalid API key") {
		t.Fatalf("unexpected api error mapping: %v", err)
	}

	httpErr := &kavenegar.HTTPError{Status: 500}
	err = mapKavenegarError(httpErr)
	if err == nil || !strings.Contains(err.Error(), "HTTP error") {
		t.Fatalf("unexpected http error mapping: %v", err)
	}

	generic := errors.New("network down")
	err = mapKavenegarError(generic)
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Fatalf("unexpected generic error mapping: %v", err)
	}
}

func TestNewKavenegarSMSChannel_EmptyKeyUsesNoop(t *testing.T) {
	ch := NewKavenegarSMSChannel("", "10008663")
	_, err := ch.SendSMS(context.Background(), models.SMSPayload{Phone: "09120000000", Message: "hi"})
	if !errors.Is(err, errs.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestNewSMSChannel_KavenegarConfigured(t *testing.T) {
	ch := NewSMSChannel(SMSChannelConfig{
		Provider: "kavenegar",
		APIKey:   "693835337A3377547771646A327733396D6D79393539744E6A5372487644456F3448434C773974337234733D",
		Sender:   "",
	})
	_, err := ch.SendSMS(context.Background(), models.SMSPayload{})
	if err == nil || !strings.Contains(err.Error(), "phone number is required") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestKavenegarSendSMSValidation(t *testing.T) {
	ch := NewKavenegarSMSChannel("valid-api-key", "10008663")

	_, err := ch.SendSMS(context.Background(), models.SMSPayload{})
	if err == nil || !strings.Contains(err.Error(), "phone number is required") {
		t.Fatalf("expected phone validation error, got %v", err)
	}

	_, err = ch.SendSMS(context.Background(), models.SMSPayload{Phone: "09120000000"})
	if err == nil || !strings.Contains(err.Error(), "message is required") {
		t.Fatalf("expected message validation error, got %v", err)
	}
}

func TestKavenegarSendOTPValidation(t *testing.T) {
	ch := NewKavenegarSMSChannel("valid-api-key", "10008663")

	_, err := ch.SendOTP(context.Background(), models.OTPPayload{})
	if err == nil || !strings.Contains(err.Error(), "phone number is required") {
		t.Fatalf("expected phone validation error, got %v", err)
	}

	_, err = ch.SendOTP(context.Background(), models.OTPPayload{Phone: "09120000000"})
	if err == nil || !strings.Contains(err.Error(), "OTP code is required") {
		t.Fatalf("expected code validation error, got %v", err)
	}
}
