package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"metarang/notifications-service/internal/service"
)

func TestResolveSMSAPIKey(t *testing.T) {
	t.Run("prefers valid SMS_API_KEY", func(t *testing.T) {
		t.Setenv("SMS_API_KEY", "real-key")
		t.Setenv("KAVENEGAR_API_KEY", "fallback-key")
		assert.Equal(t, "real-key", service.ResolveSMSAPIKey())
	})

	t.Run("falls back from change-me placeholder", func(t *testing.T) {
		t.Setenv("SMS_API_KEY", "change-me")
		t.Setenv("KAVENEGAR_API_KEY", "fallback-key")
		assert.Equal(t, "fallback-key", service.ResolveSMSAPIKey())
	})

	t.Run("falls back from your-kavenegar-api-key placeholder", func(t *testing.T) {
		t.Setenv("SMS_API_KEY", "your-kavenegar-api-key")
		t.Setenv("KAVENEGAR_API_KEY", "legacy-key")
		assert.Equal(t, "legacy-key", service.ResolveSMSAPIKey())
	})
}

func TestSMSAPIKeySource(t *testing.T) {
	t.Run("SMS_API_KEY", func(t *testing.T) {
		t.Setenv("SMS_API_KEY", "real-key")
		t.Setenv("KAVENEGAR_API_KEY", "other-key")
		assert.Equal(t, "SMS_API_KEY", service.SMSAPIKeySource())
	})

	t.Run("placeholder falls back to KAVENEGAR_API_KEY", func(t *testing.T) {
		t.Setenv("SMS_API_KEY", "change-me")
		t.Setenv("KAVENEGAR_API_KEY", "legacy-key")
		assert.Equal(t, "KAVENEGAR_API_KEY", service.SMSAPIKeySource())
	})

	t.Run("both empty is none", func(t *testing.T) {
		t.Setenv("SMS_API_KEY", "")
		t.Setenv("KAVENEGAR_API_KEY", "")
		assert.Equal(t, "none", service.SMSAPIKeySource())
	})
}

func TestIsPlaceholderSMSAPIKey(t *testing.T) {
	assert.True(t, service.IsPlaceholderSMSAPIKey("change-me"))
	assert.True(t, service.IsPlaceholderSMSAPIKey("your-kavenegar-api-key"))
	assert.True(t, service.IsPlaceholderSMSAPIKey("changeme-kavenegar-key"))
	assert.True(t, service.IsPlaceholderSMSAPIKey(""))
	assert.True(t, service.IsPlaceholderSMSAPIKey("  CHANGE-ME  "))
	assert.False(t, service.IsPlaceholderSMSAPIKey("693835337A3377547771646A327733396D6D79393539744E6A5372487644456F3448434C773974337234733D"))
}

func TestMaskAPIKey(t *testing.T) {
	assert.Equal(t, "***", service.MaskAPIKey("short"))
	assert.Equal(t, "***", service.MaskAPIKey("12345678"))
	assert.Equal(t, "***", service.MaskAPIKey(""))
	assert.Equal(t, "6938...733D", service.MaskAPIKey("693835337A3377547771646A327733396D6D79393539744E6A5372487644456F3448434C773974337234733D"))
}

func TestResolveSMSSender(t *testing.T) {
	t.Run("prefers SMS_SENDER", func(t *testing.T) {
		t.Setenv("SMS_SENDER", "10001")
		t.Setenv("KAVENEGAR_SENDER", "20001")
		assert.Equal(t, "10001", service.ResolveSMSSender("default"))
	})

	t.Run("falls back to KAVENEGAR_SENDER", func(t *testing.T) {
		t.Setenv("SMS_SENDER", "")
		t.Setenv("KAVENEGAR_SENDER", "20001")
		assert.Equal(t, "20001", service.ResolveSMSSender("default"))
	})

	t.Run("uses default when both empty", func(t *testing.T) {
		t.Setenv("SMS_SENDER", "")
		t.Setenv("KAVENEGAR_SENDER", "")
		assert.Equal(t, "default", service.ResolveSMSSender("default"))
	})
}
