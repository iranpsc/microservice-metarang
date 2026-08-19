package main

import (
	"os"
	"strings"
	"testing"
)

func TestGetEnv(t *testing.T) {
	const key = "FINANCIAL_TEST_GETENV_KEY"
	t.Setenv(key, "custom")
	if got := getEnv(key, "default"); got != "custom" {
		t.Fatalf("got %q", got)
	}

	_ = os.Unsetenv(key)
	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
}

func TestParseBoolEnv(t *testing.T) {
	const key = "FINANCIAL_TEST_BOOL_KEY"
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", true},
	}
	for _, tt := range tests {
		if tt.value == "" {
			_ = os.Unsetenv(key)
		} else {
			t.Setenv(key, tt.value)
		}
		if got := parseBoolEnv(key, true); got != tt.want {
			t.Fatalf("value=%q got=%v want=%v", tt.value, got, tt.want)
		}
	}

	t.Setenv(key, "maybe")
	if got := parseBoolEnv(key, false); got != false {
		t.Fatalf("invalid bool should use default, got %v", got)
	}
}

func TestParseBoolEnvCaseInsensitive(t *testing.T) {
	const key = "FINANCIAL_TEST_BOOL_CASE"
	for _, v := range []string{"TRUE", "True", "ON"} {
		t.Setenv(key, v)
		if !parseBoolEnv(key, false) {
			t.Fatalf("expected true for %q", v)
		}
	}
	if !strings.HasPrefix("True", "T") {
		t.Fatal("sanity")
	}
}
