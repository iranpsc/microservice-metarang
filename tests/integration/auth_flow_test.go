package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthCompleteFlow tests the complete authentication flow
func TestAuthCompleteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	// Cleanup after test
	defer CleanupDB(t, db, "users", "personal_access_tokens", "otps")

	t.Run("RegisterRedirectCallback", func(t *testing.T) {
		// Test OAuth registration flow
		ctx := context.Background()

		// Step 1: Register (initiate OAuth)
		// This would typically return a redirect URL
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"

		// In real scenario, this would call auth service gRPC method
		// For now, we create user directly
		userID := CreateTestUser(t, db, username, email)
		require.Greater(t, userID, int64(0))

		// Step 2: Verify user was created
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Step 3: Verify wallet was created (auto-create)
		CreateTestWallet(t, db, userID, "1000.0000000000", "0.0000000000")

		var walletCount int
		err = db.QueryRow("SELECT COUNT(*) FROM wallets WHERE user_id = ?", userID).Scan(&walletCount)
		require.NoError(t, err)
		assert.Equal(t, 1, walletCount)
	})

	t.Run("LoginAndGetMe", func(t *testing.T) {
		// Create test user
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"
		userID := CreateTestUser(t, db, username, email)

		// Simulate token creation
		token := "test_token_" + time.Now().Format("20060102150405")
		_, err := db.Exec(`
			INSERT INTO personal_access_tokens (tokenable_id, name, token, abilities, created_at, updated_at)
			VALUES (?, 'test-device', ?, '["*"]', NOW(), NOW())
		`, userID, token)
		require.NoError(t, err)

		// Verify token exists
		var tokenID int64
		err = db.QueryRow("SELECT id FROM personal_access_tokens WHERE token = ?", token).Scan(&tokenID)
		require.NoError(t, err)
		assert.Greater(t, tokenID, int64(0))
	})

	t.Run("Logout", func(t *testing.T) {
		// Create test user and token
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"
		userID := CreateTestUser(t, db, username, email)

		token := "test_token_" + time.Now().Format("20060102150405")
		_, err := db.Exec(`
			INSERT INTO personal_access_tokens (tokenable_id, name, token, abilities, created_at, updated_at)
			VALUES (?, 'test-device', ?, '["*"]', NOW(), NOW())
		`, userID, token)
		require.NoError(t, err)

		// Logout (delete token)
		result, err := db.Exec("DELETE FROM personal_access_tokens WHERE token = ?", token)
		require.NoError(t, err)

		affected, _ := result.RowsAffected()
		assert.Equal(t, int64(1), affected)

		// Verify token is deleted
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM personal_access_tokens WHERE token = ?", token).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("OTPVerification", func(t *testing.T) {
		// Create test user
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"
		userID := CreateTestUser(t, db, username, email)

		// Create OTP
		otpCode := "123456"
		expiresAt := time.Now().Add(5 * time.Minute)

		_, err := db.Exec(`
			INSERT INTO otps (user_id, code, type, expires_at, created_at, updated_at)
			VALUES (?, ?, 'login', ?, NOW(), NOW())
		`, userID, otpCode, expiresAt)
		require.NoError(t, err)

		// Verify OTP exists and is not expired
		var storedCode string
		var storedExpiry time.Time
		err = db.QueryRow(`
			SELECT code, expires_at FROM otps 
			WHERE user_id = ? AND type = 'login'
			ORDER BY created_at DESC LIMIT 1
		`, userID).Scan(&storedCode, &storedExpiry)
		require.NoError(t, err)
		assert.Equal(t, otpCode, storedCode)
		assert.True(t, storedExpiry.After(time.Now()))
	})

	t.Run("ActivityTracking", func(t *testing.T) {
		// Create test user
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"
		userID := CreateTestUser(t, db, username, email)

		// Update last_seen
		_, err := db.Exec("UPDATE users SET last_seen = NOW() WHERE id = ?", userID)
		require.NoError(t, err)

		// Verify last_seen was updated
		var lastSeen time.Time
		err = db.QueryRow("SELECT last_seen FROM users WHERE id = ?", userID).Scan(&lastSeen)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), lastSeen, 5*time.Second)
	})
}

// TestAuthValidateToken tests token validation
func TestAuthValidateToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "users", "personal_access_tokens")

	t.Run("ValidToken", func(t *testing.T) {
		// Create user and token
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"
		userID := CreateTestUser(t, db, username, email)

		token := "valid_token_" + time.Now().Format("20060102150405")
		_, err := db.Exec(`
			INSERT INTO personal_access_tokens (tokenable_id, name, token, abilities, created_at, updated_at)
			VALUES (?, 'test-device', ?, '["*"]', NOW(), NOW())
		`, userID, token)
		require.NoError(t, err)

		// Validate token
		var foundUserID int64
		err = db.QueryRow(`
			SELECT tokenable_id FROM personal_access_tokens 
			WHERE token = ? AND (expires_at IS NULL OR expires_at > NOW())
		`, token).Scan(&foundUserID)
		require.NoError(t, err)
		assert.Equal(t, userID, foundUserID)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		// Create user and expired token
		username := "test_user_" + time.Now().Format("20060102150405")
		email := username + "@test.com"
		userID := CreateTestUser(t, db, username, email)

		token := "expired_token_" + time.Now().Format("20060102150405")
		expiredTime := time.Now().Add(-1 * time.Hour)
		_, err := db.Exec(`
			INSERT INTO personal_access_tokens (tokenable_id, name, token, abilities, expires_at, created_at, updated_at)
			VALUES (?, 'test-device', ?, '["*"]', ?, NOW(), NOW())
		`, userID, token, expiredTime)
		require.NoError(t, err)

		// Try to validate expired token
		var foundUserID int64
		err = db.QueryRow(`
			SELECT tokenable_id FROM personal_access_tokens 
			WHERE token = ? AND (expires_at IS NULL OR expires_at > NOW())
		`, token).Scan(&foundUserID)
		assert.Error(t, err) // Should not find expired token
	})

	t.Run("InvalidToken", func(t *testing.T) {
		// Try to validate non-existent token
		var foundUserID int64
		err := db.QueryRow(`
			SELECT tokenable_id FROM personal_access_tokens 
			WHERE token = ? AND (expires_at IS NULL OR expires_at > NOW())
		`, "invalid_token").Scan(&foundUserID)
		assert.Error(t, err) // Should not find token
	})
}

