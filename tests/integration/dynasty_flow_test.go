package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDynastyFlow tests dynasty creation and member management
func TestDynastyFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "dynasties", "families", "family_members", "join_requests", "users", "features")

	t.Run("CreateDynasty", func(t *testing.T) {
		// Create user and feature
		owner := CreateTestUser(t, db, "dynasty_owner_"+time.Now().Format("150405"), "dynasty_owner@test.com")
		featureID := CreateTestFeature(t, db, &owner)

		// Create dynasty
		dynastyID := time.Now().UnixNano()
		dynastyName := "Test Dynasty " + time.Now().Format("150405")
		
		_, err := db.Exec(`
			INSERT INTO dynasties (id, name, feature_id, founder_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, NOW(), NOW())
		`, dynastyID, dynastyName, featureID, owner)
		require.NoError(t, err)

		// Create family for founder
		familyID := time.Now().UnixNano()
		_, err = db.Exec(`
			INSERT INTO families (id, dynasty_id, name, created_at, updated_at)
			VALUES (?, ?, 'Founder Family', NOW(), NOW())
		`, familyID, dynastyID)
		require.NoError(t, err)

		// Add founder as family member
		_, err = db.Exec(`
			INSERT INTO family_members (id, family_id, user_id, role, relationship, created_at, updated_at)
			VALUES (?, ?, ?, 'founder', 'self', NOW(), NOW())
		`, time.Now().UnixNano(), familyID, owner)
		require.NoError(t, err)

		// Verify dynasty created
		var foundName string
		err = db.QueryRow("SELECT name FROM dynasties WHERE id = ?", dynastyID).Scan(&foundName)
		require.NoError(t, err)
		assert.Equal(t, dynastyName, foundName)

		// Verify family created
		var familyCount int
		err = db.QueryRow("SELECT COUNT(*) FROM families WHERE dynasty_id = ?", dynastyID).Scan(&familyCount)
		require.NoError(t, err)
		assert.Equal(t, 1, familyCount)

		// Verify founder is member
		var memberCount int
		err = db.QueryRow("SELECT COUNT(*) FROM family_members WHERE family_id = ? AND user_id = ?", familyID, owner).Scan(&memberCount)
		require.NoError(t, err)
		assert.Equal(t, 1, memberCount)
	})

	t.Run("SendAndAcceptJoinRequest", func(t *testing.T) {
		// Create dynasty
		owner := CreateTestUser(t, db, "dynasty_owner2_"+time.Now().Format("150405"), "dynasty_owner2@test.com")
		featureID := CreateTestFeature(t, db, &owner)

		dynastyID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO dynasties (id, name, feature_id, founder_id, created_at, updated_at)
			VALUES (?, 'Test Dynasty 2', ?, ?, NOW(), NOW())
		`, dynastyID, featureID, owner)
		require.NoError(t, err)

		familyID := time.Now().UnixNano()
		_, err = db.Exec(`
			INSERT INTO families (id, dynasty_id, name, created_at, updated_at)
			VALUES (?, ?, 'Main Family', NOW(), NOW())
		`, familyID, dynastyID)
		require.NoError(t, err)

		// Create applicant
		applicant := CreateTestUser(t, db, "applicant_"+time.Now().Format("150405"), "applicant@test.com")

		// Send join request
		requestID := time.Now().UnixNano()
		_, err = db.Exec(`
			INSERT INTO join_requests (id, dynasty_id, user_id, family_id, status, relationship, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'pending', 'child', NOW(), NOW())
		`, requestID, dynastyID, applicant, familyID)
		require.NoError(t, err)

		// Verify request created
		var status string
		err = db.QueryRow("SELECT status FROM join_requests WHERE id = ?", requestID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "pending", status)

		// Accept request
		tx, err := db.Begin()
		require.NoError(t, err)

		_, err = tx.Exec("UPDATE join_requests SET status = 'accepted' WHERE id = ?", requestID)
		require.NoError(t, err)

		// Add as family member
		_, err = tx.Exec(`
			INSERT INTO family_members (id, family_id, user_id, role, relationship, created_at, updated_at)
			VALUES (?, ?, ?, 'member', 'child', NOW(), NOW())
		`, time.Now().UnixNano(), familyID, applicant)
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Verify request accepted
		err = db.QueryRow("SELECT status FROM join_requests WHERE id = ?", requestID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "accepted", status)

		// Verify member added
		var memberCount int
		err = db.QueryRow("SELECT COUNT(*) FROM family_members WHERE user_id = ?", applicant).Scan(&memberCount)
		require.NoError(t, err)
		assert.Equal(t, 1, memberCount)
	})

	t.Run("SetChildPermissions", func(t *testing.T) {
		// Create family
		owner := CreateTestUser(t, db, "parent_"+time.Now().Format("150405"), "parent@test.com")
		child := CreateTestUser(t, db, "child_"+time.Now().Format("150405"), "child@test.com")

		featureID := CreateTestFeature(t, db, &owner)
		dynastyID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO dynasties (id, name, feature_id, founder_id, created_at, updated_at)
			VALUES (?, 'Test Dynasty 3', ?, ?, NOW(), NOW())
		`, dynastyID, featureID, owner)
		require.NoError(t, err)

		familyID := time.Now().UnixNano()
		_, err = db.Exec(`
			INSERT INTO families (id, dynasty_id, name, created_at, updated_at)
			VALUES (?, ?, 'Family 3', NOW(), NOW())
		`, familyID, dynastyID)
		require.NoError(t, err)

		// Add child as member
		_, err = db.Exec(`
			INSERT INTO family_members (id, family_id, user_id, role, relationship, created_at, updated_at)
			VALUES (?, ?, ?, 'member', 'child', NOW(), NOW())
		`, time.Now().UnixNano(), familyID, child)
		require.NoError(t, err)

		// Set permissions
		permissions := `{"can_trade": true, "can_build": false, "can_withdraw": false}`
		_, err = db.Exec(`
			INSERT INTO children_permissions (id, parent_id, child_id, permissions, created_at, updated_at)
			VALUES (?, ?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE permissions = VALUES(permissions)
		`, time.Now().UnixNano(), owner, child, permissions)
		require.NoError(t, err)

		// Verify permissions
		var storedPerms string
		err = db.QueryRow("SELECT permissions FROM children_permissions WHERE parent_id = ? AND child_id = ?", owner, child).Scan(&storedPerms)
		require.NoError(t, err)
		assert.Contains(t, storedPerms, "can_trade")
	})
}

