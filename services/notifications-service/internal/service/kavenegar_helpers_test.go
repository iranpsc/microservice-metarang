package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/kavenegar/kavenegar-go"
)

func TestExtractTemplateToken(t *testing.T) {
	if got := extractTemplateToken(nil); got != "" {
		t.Fatalf("nil tokens: got %q", got)
	}
	if got := extractTemplateToken(map[string]string{"token": "abc"}); got != "abc" {
		t.Fatalf("token key: got %q", got)
	}
	if got := extractTemplateToken(map[string]string{"token": "", "code": "1234"}); got != "1234" {
		t.Fatalf("code fallback: got %q", got)
	}
	if got := extractTemplateToken(map[string]string{"other": "x"}); got != "" {
		t.Fatalf("unknown keys: got %q", got)
	}
}

func TestKavenegarAPIErrorHint(t *testing.T) {
	if hint := kavenegarAPIErrorHint(403); hint == "" || !strings.Contains(hint, "invalid API key") {
		t.Fatalf("403 hint=%q", hint)
	}
	if hint := kavenegarAPIErrorHint(416); hint == "" || !strings.Contains(hint, "IP") {
		t.Fatalf("416 hint=%q", hint)
	}
	if hint := kavenegarAPIErrorHint(424); hint == "" || !strings.Contains(hint, "template") {
		t.Fatalf("424 hint=%q", hint)
	}
	if hint := kavenegarAPIErrorHint(500); hint != "" {
		t.Fatalf("500 hint=%q", hint)
	}
}

func TestMapKavenegarError(t *testing.T) {
	err := mapKavenegarError(&kavenegar.APIError{Status: 403, Message: "forbidden"})
	if err == nil || !strings.Contains(err.Error(), "invalid API key") {
		t.Fatalf("403 mapped=%v", err)
	}

	err = mapKavenegarError(&kavenegar.APIError{Status: 500, Message: "boom"})
	if err == nil || !strings.Contains(err.Error(), "kavenegar API error") {
		t.Fatalf("500 mapped=%v", err)
	}
	if strings.Contains(err.Error(), "invalid API key") {
		t.Fatalf("500 should not include 403 hint: %v", err)
	}

	err = mapKavenegarError(&kavenegar.HTTPError{Status: 502, Message: "bad gateway"})
	if err == nil || !strings.Contains(err.Error(), "kavenegar HTTP error") {
		t.Fatalf("http mapped=%v", err)
	}

	err = mapKavenegarError(errors.New("network down"))
	if err == nil || !strings.Contains(err.Error(), "kavenegar request failed") {
		t.Fatalf("generic mapped=%v", err)
	}
}
