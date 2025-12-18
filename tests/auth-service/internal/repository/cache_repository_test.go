package repository

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupTestRedis creates a test Redis client
// In a real scenario, you might use a test Redis container or mock
func setupTestRedis(t *testing.T) *redis.Client {
	opts := &redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use DB 15 for tests
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	// Clean up test database
	client.FlushDB(ctx)

	return client
}

func TestCacheRepository_SetState(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	repo := NewCacheRepository(client)
	ctx := context.Background()

	t.Run("successful set", func(t *testing.T) {
		state := "test_state_123"
		ttl := 5 * time.Minute

		err := repo.SetState(ctx, state, ttl)
		if err != nil {
			t.Fatalf("SetState failed: %v", err)
		}

		// Verify it was set
		val, err := client.Get(ctx, "oauth:state:"+state).Result()
		if err != nil {
			t.Fatalf("Failed to get state: %v", err)
		}
		if val != "1" {
			t.Errorf("Expected state value '1', got %q", val)
		}
	})
}

func TestCacheRepository_GetState(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	repo := NewCacheRepository(client)
	ctx := context.Background()

	t.Run("successful get and delete", func(t *testing.T) {
		state := "test_state_get"
		ttl := 5 * time.Minute

		// Set state first
		err := repo.SetState(ctx, state, ttl)
		if err != nil {
			t.Fatalf("SetState failed: %v", err)
		}

		// Get state (should delete it)
		exists, err := repo.GetState(ctx, state)
		if err != nil {
			t.Fatalf("GetState failed: %v", err)
		}
		if !exists {
			t.Error("Expected state to exist")
		}

		// Verify it was deleted (pull semantics)
		_, err = client.Get(ctx, "oauth:state:"+state).Result()
		if err != redis.Nil {
			t.Errorf("Expected state to be deleted, but it still exists: %v", err)
		}
	})

	t.Run("get non-existent state", func(t *testing.T) {
		exists, err := repo.GetState(ctx, "non_existent_state")
		if err != nil {
			t.Fatalf("GetState should not return error for non-existent state: %v", err)
		}
		if exists {
			t.Error("Expected state to not exist")
		}
	})
}

func TestCacheRepository_RedirectTo(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	repo := NewCacheRepository(client)
	ctx := context.Background()

	t.Run("set and get redirect_to", func(t *testing.T) {
		state := "test_state_redirect"
		redirectTo := "https://example.com/dashboard"
		ttl := 5 * time.Minute

		err := repo.SetRedirectTo(ctx, state, redirectTo, ttl)
		if err != nil {
			t.Fatalf("SetRedirectTo failed: %v", err)
		}

		// Get redirect_to (should delete it)
		val, err := repo.GetRedirectTo(ctx, state)
		if err != nil {
			t.Fatalf("GetRedirectTo failed: %v", err)
		}
		if val != redirectTo {
			t.Errorf("Expected redirect_to %q, got %q", redirectTo, val)
		}

		// Verify it was deleted
		_, err = client.Get(ctx, "oauth:redirect_to:"+state).Result()
		if err != redis.Nil {
			t.Errorf("Expected redirect_to to be deleted, but it still exists: %v", err)
		}
	})

	t.Run("get non-existent redirect_to", func(t *testing.T) {
		val, err := repo.GetRedirectTo(ctx, "non_existent_state")
		if err != nil {
			t.Fatalf("GetRedirectTo should not return error for non-existent state: %v", err)
		}
		if val != "" {
			t.Errorf("Expected empty string, got %q", val)
		}
	})
}

func TestCacheRepository_BackURL(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	repo := NewCacheRepository(client)
	ctx := context.Background()

	t.Run("set and get back_url", func(t *testing.T) {
		state := "test_state_backurl"
		backURL := "https://example.com/home"
		ttl := 5 * time.Minute

		err := repo.SetBackURL(ctx, state, backURL, ttl)
		if err != nil {
			t.Fatalf("SetBackURL failed: %v", err)
		}

		// Get back_url (should delete it)
		val, err := repo.GetBackURL(ctx, state)
		if err != nil {
			t.Fatalf("GetBackURL failed: %v", err)
		}
		if val != backURL {
			t.Errorf("Expected back_url %q, got %q", backURL, val)
		}

		// Verify it was deleted
		_, err = client.Get(ctx, "oauth:back_url:"+state).Result()
		if err != redis.Nil {
			t.Errorf("Expected back_url to be deleted, but it still exists: %v", err)
		}
	})

	t.Run("get non-existent back_url", func(t *testing.T) {
		val, err := repo.GetBackURL(ctx, "non_existent_state")
		if err != nil {
			t.Fatalf("GetBackURL should not return error for non-existent state: %v", err)
		}
		if val != "" {
			t.Errorf("Expected empty string, got %q", val)
		}
	})
}

func TestCacheRepository_TTL(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	repo := NewCacheRepository(client)
	ctx := context.Background()

	t.Run("verify TTL is set", func(t *testing.T) {
		state := "test_state_ttl"
		ttl := 5 * time.Minute

		err := repo.SetState(ctx, state, ttl)
		if err != nil {
			t.Fatalf("SetState failed: %v", err)
		}

		// Check TTL (should be approximately 5 minutes)
		remaining, err := client.TTL(ctx, "oauth:state:"+state).Result()
		if err != nil {
			t.Fatalf("Failed to get TTL: %v", err)
		}

		// TTL should be between 4.5 and 5.5 minutes (allowing for small delays)
		if remaining < 4*time.Minute+30*time.Second || remaining > 5*time.Minute+30*time.Second {
			t.Errorf("Expected TTL around 5 minutes, got %v", remaining)
		}
	})
}
