package main

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	const key = "TRAINING_SERVICE_TEST_ENV_KEY"
	t.Setenv(key, "custom")
	if got := getEnv(key, "default"); got != "custom" {
		t.Fatalf("got=%q", got)
	}
	_ = os.Unsetenv(key)
	if got := getEnv(key, "default"); got != "default" {
		t.Fatalf("got=%q", got)
	}
}
