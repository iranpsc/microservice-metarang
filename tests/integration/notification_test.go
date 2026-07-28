package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotificationFlow tests notification creation and delivery
func TestNotificationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "notifications", "users")

	t.Run("CreateNotification", func(t *testing.T) {
		// Create user
		user := CreateTestUser(t, db, "notif_user_"+time.Now().Format("150405"), "notif@test.com")

		// Create notification
		notifID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO notifications (id, user_id, type, title, message, read_at, created_at, updated_at)
			VALUES (?, ?, 'system', 'Test Notification', 'Test message', NULL, NOW(), NOW())
		`, notifID, user)
		require.NoError(t, err)

		// Verify notification created
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM notifications WHERE id = ?", notifID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("MarkAsRead", func(t *testing.T) {
		// Create user and notification
		user := CreateTestUser(t, db, "read_user_"+time.Now().Format("150405"), "read@test.com")

		notifID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO notifications (id, user_id, type, title, message, read_at, created_at, updated_at)
			VALUES (?, ?, 'system', 'Test', 'Test', NULL, NOW(), NOW())
		`, notifID, user)
		require.NoError(t, err)

		// Mark as read
		_, err = db.Exec("UPDATE notifications SET read_at = NOW() WHERE id = ?", notifID)
		require.NoError(t, err)

		// Verify read_at is set
		var readAt *time.Time
		err = db.QueryRow("SELECT read_at FROM notifications WHERE id = ?", notifID).Scan(&readAt)
		require.NoError(t, err)
		assert.NotNil(t, readAt)
	})

	t.Run("MarkAllAsRead", func(t *testing.T) {
		// Create user with multiple notifications
		user := CreateTestUser(t, db, "multi_user_"+time.Now().Format("150405"), "multi@test.com")

		for i := 0; i < 5; i++ {
			_, err := db.Exec(`
				INSERT INTO notifications (id, user_id, type, title, message, read_at, created_at, updated_at)
				VALUES (?, ?, 'system', ?, 'Test', NULL, NOW(), NOW())
			`, time.Now().UnixNano()+int64(i), user, "Test "+string(rune(i)))
			require.NoError(t, err)
		}

		// Mark all as read
		_, err := db.Exec("UPDATE notifications SET read_at = NOW() WHERE user_id = ? AND read_at IS NULL", user)
		require.NoError(t, err)

		// Verify all marked as read
		var unreadCount int
		err = db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL", user).Scan(&unreadCount)
		require.NoError(t, err)
		assert.Equal(t, 0, unreadCount)
	})
}

