package service

import (
	"context"
	"errors"
	"testing"

	"metargb/auth-service/internal/models"
)

type mockSettingsRepository struct {
	findByUserIDFunc func(context.Context, uint64) (*models.Settings, error)
	findByIDFunc     func(context.Context, uint64) (*models.Settings, error)
	updateFunc       func(context.Context, *models.Settings) error
	createFunc       func(context.Context, *models.Settings) error
}

func (m *mockSettingsRepository) FindByUserID(ctx context.Context, userID uint64) (*models.Settings, error) {
	if m.findByUserIDFunc != nil {
		return m.findByUserIDFunc(ctx, userID)
	}
	return &models.Settings{
		ID:                1,
		UserID:            userID,
		Status:            true,
		Level:             true,
		Details:           true,
		CheckoutDaysCount: 3,
		AutomaticLogout:   55,
		Privacy:           models.DefaultPrivacySettings(),
		Notifications:     models.DefaultNotificationSettings(),
	}, nil
}

func (m *mockSettingsRepository) FindByID(ctx context.Context, id uint64) (*models.Settings, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return &models.Settings{
		ID:                id,
		UserID:            1,
		Status:            true,
		Level:             true,
		Details:           true,
		CheckoutDaysCount: 3,
		AutomaticLogout:   55,
		Privacy:           models.DefaultPrivacySettings(),
		Notifications:     models.DefaultNotificationSettings(),
	}, nil
}

func (m *mockSettingsRepository) Update(ctx context.Context, settings *models.Settings) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, settings)
	}
	return nil
}

func (m *mockSettingsRepository) Create(ctx context.Context, settings *models.Settings) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, settings)
	}
	settings.ID = 1
	return nil
}

func TestSettingsService_GetSettings(t *testing.T) {
	mockRepo := &mockSettingsRepository{}
	service := NewSettingsService(mockRepo)
	ctx := context.Background()

	t.Run("returns settings successfully", func(t *testing.T) {
		settings, err := service.GetSettings(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if settings == nil {
			t.Fatal("expected settings, got nil")
		}
		if settings.UserID != 1 {
			t.Errorf("expected userID 1, got %d", settings.UserID)
		}
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo.findByUserIDFunc = func(context.Context, uint64) (*models.Settings, error) {
			return nil, errors.New("database error")
		}
		_, err := service.GetSettings(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSettingsService_UpdateSettings(t *testing.T) {
	mockRepo := &mockSettingsRepository{}
	service := NewSettingsService(mockRepo)
	ctx := context.Background()

	t.Run("updates checkout cadence successfully", func(t *testing.T) {
		checkoutDays := uint32(10)
		automaticLogout := int32(30)

		err := service.UpdateSettings(ctx, 1, &checkoutDays, &automaticLogout, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validates checkout days range", func(t *testing.T) {
		checkoutDays := uint32(2) // Invalid: must be >= 3
		automaticLogout := int32(30)

		err := service.UpdateSettings(ctx, 1, &checkoutDays, &automaticLogout, nil, nil)
		if err != ErrInvalidCheckoutDays {
			t.Errorf("expected ErrInvalidCheckoutDays, got %v", err)
		}
	})

	t.Run("validates automatic logout range - too low", func(t *testing.T) {
		checkoutDays := uint32(10)
		automaticLogout := int32(0) // Invalid: must be >= 1

		err := service.UpdateSettings(ctx, 1, &checkoutDays, &automaticLogout, nil, nil)
		if err != ErrInvalidAutomaticLogout {
			t.Errorf("expected ErrInvalidAutomaticLogout, got %v", err)
		}
	})

	t.Run("validates automatic logout range - too high", func(t *testing.T) {
		checkoutDays := uint32(10)
		automaticLogout := int32(60) // Invalid: must be <= 55

		err := service.UpdateSettings(ctx, 1, &checkoutDays, &automaticLogout, nil, nil)
		if err != ErrInvalidAutomaticLogout {
			t.Errorf("expected ErrInvalidAutomaticLogout, got %v", err)
		}
	})

	t.Run("validates checkout days range - too high", func(t *testing.T) {
		checkoutDays := uint32(1001) // Invalid: must be <= 1000
		automaticLogout := int32(30)

		err := service.UpdateSettings(ctx, 1, &checkoutDays, &automaticLogout, nil, nil)
		if err != ErrInvalidCheckoutDays {
			t.Errorf("expected ErrInvalidCheckoutDays, got %v", err)
		}
	})

	t.Run("updates profile exposure successfully", func(t *testing.T) {
		setting := "status"
		status := false

		err := service.UpdateSettings(ctx, 1, nil, nil, &setting, &status)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validates profile setting name", func(t *testing.T) {
		setting := "invalid"
		status := false

		err := service.UpdateSettings(ctx, 1, nil, nil, &setting, &status)
		if err != ErrInvalidProfileSetting {
			t.Errorf("expected ErrInvalidProfileSetting, got %v", err)
		}
	})

	t.Run("requires both checkout fields", func(t *testing.T) {
		checkoutDays := uint32(10)

		err := service.UpdateSettings(ctx, 1, &checkoutDays, nil, nil, nil)
		if err != ErrMissingRequiredFields {
			t.Errorf("expected ErrMissingRequiredFields, got %v", err)
		}
	})

	t.Run("requires both profile exposure fields", func(t *testing.T) {
		setting := "status"

		err := service.UpdateSettings(ctx, 1, nil, nil, &setting, nil)
		if err != ErrMissingRequiredFields {
			t.Errorf("expected ErrMissingRequiredFields, got %v", err)
		}
	})

	t.Run("creates settings if not exists", func(t *testing.T) {
		mockRepo.findByUserIDFunc = func(context.Context, uint64) (*models.Settings, error) {
			return &models.Settings{
				ID:                0, // No ID means not in DB
				UserID:            1,
				Status:            true,
				Level:             true,
				Details:           true,
				CheckoutDaysCount: 3,
				AutomaticLogout:   55,
				Privacy:           models.DefaultPrivacySettings(),
				Notifications:     models.DefaultNotificationSettings(),
			}, nil
		}
		created := false
		mockRepo.createFunc = func(context.Context, *models.Settings) error {
			created = true
			return nil
		}

		checkoutDays := uint32(10)
		automaticLogout := int32(30)

		err := service.UpdateSettings(ctx, 1, &checkoutDays, &automaticLogout, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Error("expected settings to be created")
		}
	})
}

func TestSettingsService_GetGeneralSettings(t *testing.T) {
	mockRepo := &mockSettingsRepository{}
	service := NewSettingsService(mockRepo)
	ctx := context.Background()

	t.Run("returns notification settings", func(t *testing.T) {
		notifications, err := service.GetGeneralSettings(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notifications == nil {
			t.Fatal("expected notifications map")
		}
		if len(notifications) == 0 {
			t.Error("expected non-empty notifications map")
		}
	})
}

func TestSettingsService_UpdateGeneralSettings(t *testing.T) {
	mockRepo := &mockSettingsRepository{}
	service := NewSettingsService(mockRepo)
	ctx := context.Background()

	t.Run("updates general settings successfully", func(t *testing.T) {
		notifications := map[string]bool{
			"announcements_sms":        false,
			"announcements_email":      true,
			"reports_sms":              true,
			"reports_email":            true,
			"login_verification_sms":   true,
			"login_verification_email": true,
			"transactions_sms":         false,
			"transactions_email":       true,
			"trades_sms":               true,
			"trades_email":             true,
		}

		updated, err := service.UpdateGeneralSettings(ctx, 1, 1, notifications)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated["announcements_sms"] != false {
			t.Error("expected announcements_sms to be false")
		}
	})

	t.Run("validates ownership", func(t *testing.T) {
		mockRepo.findByIDFunc = func(context.Context, uint64) (*models.Settings, error) {
			return &models.Settings{
				ID:     1,
				UserID: 2, // Different user
			}, nil
		}

		notifications := models.DefaultNotificationSettings()
		_, err := service.UpdateGeneralSettings(ctx, 1, 1, notifications)
		if err == nil {
			t.Fatal("expected error for ownership check")
		}
		if err.Error() != "settings do not belong to user" {
			t.Errorf("expected ownership error, got %v", err)
		}
	})

	t.Run("handles missing settings", func(t *testing.T) {
		mockRepo.findByIDFunc = func(context.Context, uint64) (*models.Settings, error) {
			return nil, nil
		}

		notifications := models.DefaultNotificationSettings()
		_, err := service.UpdateGeneralSettings(ctx, 1, 1, notifications)
		if err != ErrSettingsNotFound {
			t.Errorf("expected ErrSettingsNotFound, got %v", err)
		}
	})

	t.Run("validates all required channels", func(t *testing.T) {
		notifications := map[string]bool{
			"announcements_sms": true,
			// Missing other channels
		}

		_, err := service.UpdateGeneralSettings(ctx, 1, 1, notifications)
		if err == nil {
			t.Fatal("expected error for missing channels")
		}
	})
}

func TestSettingsService_GetPrivacySettings(t *testing.T) {
	mockRepo := &mockSettingsRepository{}
	service := NewSettingsService(mockRepo)
	ctx := context.Background()

	t.Run("returns privacy settings", func(t *testing.T) {
		privacy, err := service.GetPrivacySettings(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if privacy == nil {
			t.Fatal("expected privacy map")
		}
		if len(privacy) == 0 {
			t.Error("expected non-empty privacy map")
		}
	})
}

func TestSettingsService_UpdatePrivacySettings(t *testing.T) {
	mockRepo := &mockSettingsRepository{}
	service := NewSettingsService(mockRepo)
	ctx := context.Background()

	t.Run("updates privacy setting successfully", func(t *testing.T) {
		err := service.UpdatePrivacySettings(ctx, 1, "phone", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validates privacy key", func(t *testing.T) {
		err := service.UpdatePrivacySettings(ctx, 1, "invalid_key", 1)
		if err != ErrInvalidPrivacyKey {
			t.Errorf("expected ErrInvalidPrivacyKey, got %v", err)
		}
	})

	t.Run("validates privacy value", func(t *testing.T) {
		err := service.UpdatePrivacySettings(ctx, 1, "phone", 2)
		if err != ErrInvalidPrivacyValue {
			t.Errorf("expected ErrInvalidPrivacyValue, got %v", err)
		}
	})

	t.Run("creates settings if not exists", func(t *testing.T) {
		mockRepo.findByUserIDFunc = func(context.Context, uint64) (*models.Settings, error) {
			return &models.Settings{
				ID:      0,
				UserID:  1,
				Privacy: models.DefaultPrivacySettings(),
			}, nil
		}
		created := false
		mockRepo.createFunc = func(context.Context, *models.Settings) error {
			created = true
			return nil
		}

		err := service.UpdatePrivacySettings(ctx, 1, "phone", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Error("expected settings to be created")
		}
	})
}
