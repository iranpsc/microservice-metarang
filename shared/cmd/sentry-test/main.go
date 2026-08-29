// Command sentry-test sends a test event to Sentry to verify connectivity.
//
// Reads SENTRY_* from the environment. When repo-root .env exists, unset vars
// are loaded from it (same source as Docker Compose).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sentrygo "github.com/getsentry/sentry-go"

	"metarang/shared/pkg/sentry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	loadDotEnv(findRepoRoot())

	if err := sentry.InitFromEnv("sentry-test"); err != nil {
		return fmt.Errorf("failed to initialize Sentry: %w", err)
	}
	if !sentry.Enabled() {
		return fmt.Errorf("SENTRY_DSN is not set — copy sentry.env.sample to .env and configure SENTRY_DSN")
	}

	sentrygo.WithScope(func(scope *sentrygo.Scope) {
		scope.SetLevel(sentrygo.LevelInfo)
		scope.SetTag("source", "make sentry:test")
		sentrygo.CaptureMessage("Sentry connection test from make sentry:test")
	})

	sentry.Flush(5 * time.Second)
	fmt.Println("Test event sent to Sentry. Check your project dashboard for a message tagged source=make sentry:test.")
	return nil
}

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if fileExists(filepath.Join(dir, "go.work")) || fileExists(filepath.Join(dir, "Makefile")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadDotEnv(root string) {
	data, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
}
