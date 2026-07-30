package main

import (
	"os"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestGetEnv(t *testing.T) {
	const key = "NOTIFICATIONS_TEST_GETENV_KEY"
	t.Setenv(key, "custom")
	if got := getEnv(key, "default"); got != "custom" {
		t.Fatalf("got %q", got)
	}

	_ = os.Unsetenv(key)
	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	const key = "NOTIFICATIONS_TEST_INT_KEY"

	t.Setenv(key, "42")
	if got := getEnvAsInt(key, 10); got != 42 {
		t.Fatalf("got %d", got)
	}

	t.Setenv(key, "invalid")
	if got := getEnvAsInt(key, 10); got != 10 {
		t.Fatalf("got %d", got)
	}

	_ = os.Unsetenv(key)
	if got := getEnvAsInt(key, 10); got != 10 {
		t.Fatalf("got %d", got)
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	const key = "NOTIFICATIONS_TEST_DURATION_KEY"

	t.Setenv(key, "2h")
	if got := getEnvAsDuration(key, time.Minute); got != 2*time.Hour {
		t.Fatalf("got %v", got)
	}

	t.Setenv(key, "bad")
	if got := getEnvAsDuration(key, time.Minute); got != time.Minute {
		t.Fatalf("got %v", got)
	}

	_ = os.Unsetenv(key)
	if got := getEnvAsDuration(key, time.Minute); got != time.Minute {
		t.Fatalf("got %v", got)
	}
}

func TestSetupDatabase_InvalidPort(t *testing.T) {
	t.Setenv("DB_PORT", "not-a-number")
	_, err := setupDatabase()
	if err == nil {
		t.Fatal("expected error for invalid DB_PORT")
	}
}

func TestSetupDatabase_Success(t *testing.T) {
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_DATABASE", "testdb")
	t.Setenv("DB_MAX_OPEN_CONNS", "10")
	t.Setenv("DB_MAX_IDLE_CONNS", "2")
	t.Setenv("DB_CONN_MAX_LIFETIME", "1m")

	db, err := setupDatabase()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	if db == nil {
		t.Fatal("expected db")
	}
}

func TestPingDatabase(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectPing()
	if err := pingDatabase(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
