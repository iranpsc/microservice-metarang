package main

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	const key = "STORAGE_TEST_SERVER_GETENV"
	t.Setenv(key, "custom")
	if got := getEnv(key, "default"); got != "custom" {
		t.Fatalf("got %q", got)
	}
	_ = os.Unsetenv(key)
	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
}
