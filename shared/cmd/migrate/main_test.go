package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	if err := run([]string{"help"}, stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "baseline") {
		t.Fatalf("usage = %s", stdout.String())
	}
}

func TestRunMake(t *testing.T) {
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	err := run([]string{"make", "-path", dir, "add_example_column"}, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), "_add_example_column.sql") {
		t.Fatalf("entries = %v", entries)
	}
	raw, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "-- migrate:down") {
		t.Fatalf("stub = %s", raw)
	}
}

func TestRunUnknown(t *testing.T) {
	err := run([]string{"nope"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunProductionGuard(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	err := run([]string{"up", "-path", t.TempDir()}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "-force") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunMakeEmptyName(t *testing.T) {
	err := run([]string{"make", "-path", t.TempDir()}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}
