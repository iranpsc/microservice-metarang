package service

import (
	"os"
	"strings"
)

var placeholderSMSAPIKeys = map[string]struct{}{
	"":                       {},
	"change-me":              {},
	"your-kavenegar-api-key": {},
	"changeme-kavenegar-key": {},
}

// ResolveSMSAPIKey returns the Kavenegar API key.
func ResolveSMSAPIKey() string {
	smsKey := strings.TrimSpace(os.Getenv("SMS_API_KEY"))
	if smsKey != "" {
		if _, placeholder := placeholderSMSAPIKeys[strings.ToLower(smsKey)]; !placeholder {
			return smsKey
		}
	}
	return strings.TrimSpace(os.Getenv("KAVENEGAR_API_KEY"))
}

// SMSAPIKeySource reports which env var supplied the resolved API key (for startup logs).
func SMSAPIKeySource() string {
	smsKey := strings.TrimSpace(os.Getenv("SMS_API_KEY"))
	if smsKey != "" {
		if _, placeholder := placeholderSMSAPIKeys[strings.ToLower(smsKey)]; !placeholder {
			return "SMS_API_KEY"
		}
	}
	if k := strings.TrimSpace(os.Getenv("KAVENEGAR_API_KEY")); k != "" {
		return "KAVENEGAR_API_KEY"
	}
	return "none"
}

// MaskAPIKey returns a redacted API key for logs.
func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// ResolveSMSSender returns the sender line from SMS_SENDER or KAVENEGAR_SENDER.
func ResolveSMSSender(defaultSender string) string {
	if sender := strings.TrimSpace(os.Getenv("SMS_SENDER")); sender != "" {
		return sender
	}
	if sender := strings.TrimSpace(os.Getenv("KAVENEGAR_SENDER")); sender != "" {
		return sender
	}
	return defaultSender
}

// IsPlaceholderSMSAPIKey reports whether the key is missing or a documented placeholder.
func IsPlaceholderSMSAPIKey(key string) bool {
	_, ok := placeholderSMSAPIKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}
