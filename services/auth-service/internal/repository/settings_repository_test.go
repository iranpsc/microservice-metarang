package repository

import (
	"context"
	"database/sql"
	"testing"

	"metargb/auth-service/internal/models"
)

func TestSettingsRepository_FindByUserID(t *testing.T) {
	// This is an integration test - requires a real database
	// Skip if no database available
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	t.Run("returns default settings when not found", func(t *testing.T) {
		settings, err := repo.FindByUserID(ctx, 99999)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if settings == nil {
			t.Fatal("expected settings, got nil")
		}
		if settings.UserID != 99999 {
			t.Errorf("expected userID 99999, got %d", settings.UserID)
		}
		if settings.CheckoutDaysCount != 3 {
			t.Errorf("expected checkout_days_count 3, got %d", settings.CheckoutDaysCount)
		}
		if settings.AutomaticLogout != 55 {
			t.Errorf("expected automatic_logout 55, got %d", settings.AutomaticLogout)
		}
		if !settings.Status {
			t.Error("expected status true")
		}
		if !settings.Level {
			t.Error("expected level true")
		}
		if !settings.Details {
			t.Error("expected details true")
		}
		if settings.Privacy == nil {
			t.Error("expected privacy map")
		}
		if settings.Notifications == nil {
			t.Error("expected notifications map")
		}
	})

	t.Run("creates and retrieves settings", func(t *testing.T) {
		settings := &models.Settings{
			UserID:            1,
			Status:            true,
			Level:             false,
			Details:           true,
			CheckoutDaysCount: 5,
			AutomaticLogout:   30,
			Privacy:           models.DefaultPrivacySettings(),
			Notifications:     models.DefaultNotificationSettings(),
		}

		err := repo.Create(ctx, settings)
		if err != nil {
			t.Fatalf("failed to create settings: %v", err)
		}

		retrieved, err := repo.FindByUserID(ctx, 1)
		if err != nil {
			t.Fatalf("failed to retrieve settings: %v", err)
		}
		if retrieved.ID != settings.ID {
			t.Errorf("expected ID %d, got %d", settings.ID, retrieved.ID)
		}
		if retrieved.UserID != 1 {
			t.Errorf("expected userID 1, got %d", retrieved.UserID)
		}
		if retrieved.Status != true {
			t.Error("expected status true")
		}
		if retrieved.Level != false {
			t.Error("expected level false")
		}
		if retrieved.CheckoutDaysCount != 5 {
			t.Errorf("expected checkout_days_count 5, got %d", retrieved.CheckoutDaysCount)
		}
		if retrieved.AutomaticLogout != 30 {
			t.Errorf("expected automatic_logout 30, got %d", retrieved.AutomaticLogout)
		}
	})
}

func TestSettingsRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Create settings first
	settings := &models.Settings{
		UserID:            2,
		Status:            true,
		Level:             true,
		Details:           true,
		CheckoutDaysCount: 3,
		AutomaticLogout:   55,
		Privacy:           models.DefaultPrivacySettings(),
		Notifications:     models.DefaultNotificationSettings(),
	}

	err := repo.Create(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create settings: %v", err)
	}

	// Update settings
	settings.Status = false
	settings.Level = false
	settings.CheckoutDaysCount = 10
	settings.AutomaticLogout = 45

	privacy := models.DefaultPrivacySettings()
	privacy["phone"] = 1
	settings.Privacy = privacy

	err = repo.Update(ctx, settings)
	if err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.FindByID(ctx, settings.ID)
	if err != nil {
		t.Fatalf("failed to retrieve settings: %v", err)
	}
	if retrieved.Status != false {
		t.Error("expected status false")
	}
	if retrieved.Level != false {
		t.Error("expected level false")
	}
	if retrieved.CheckoutDaysCount != 10 {
		t.Errorf("expected checkout_days_count 10, got %d", retrieved.CheckoutDaysCount)
	}
	if retrieved.AutomaticLogout != 45 {
		t.Errorf("expected automatic_logout 45, got %d", retrieved.AutomaticLogout)
	}
	if retrieved.Privacy["phone"] != 1 {
		t.Error("expected privacy phone to be 1")
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	// This should use a test database connection
	// For now, we'll skip if DB is not available
	db, err := sql.Open("mysql", "root@tcp(localhost:3306)/metargb_test?parseTime=true")
	if err != nil {
		t.Skipf("skipping test - database not available: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("skipping test - database ping failed: %v", err)
	}

	return db
}
