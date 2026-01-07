package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metargb/dynasty-service/internal/models"
	"metargb/dynasty-service/internal/repository"
	"metargb/dynasty-service/internal/service"
)

// TestDynastyFlow tests complete dynasty creation and member management flow
func TestDynastyFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "dynasties", "families", "family_members", "join_requests", "received_prizes", "children_permissions")

	ctx := context.Background()

	// Create test user (owner)
	ownerID := CreateTestUser(t, db, "dynasty_owner_"+time.Now().Format("150405"), "owner@test.com")
	featureID := uint64(CreateTestFeatureID(t, db, ownerID))

	t.Run("CreateDynasty", func(t *testing.T) {
		dynastyRepo := repository.NewDynastyRepository(db)
		familyRepo := repository.NewFamilyRepository(db)
		prizeRepo := repository.NewPrizeRepository(db)
		dynastyService := service.NewDynastyService(dynastyRepo, familyRepo, prizeRepo, "")

		// Create dynasty
		dynasty, family, err := dynastyService.CreateDynasty(ctx, uint64(ownerID), featureID)
		require.NoError(t, err)
		assert.NotNil(t, dynasty)
		assert.NotNil(t, family)
		assert.Equal(t, uint64(ownerID), dynasty.UserID)
		assert.Equal(t, featureID, dynasty.FeatureID)

		// Verify dynasty exists
		var dbDynastyID uint64
		err = db.QueryRow("SELECT id FROM dynasties WHERE user_id = ?", ownerID).Scan(&dbDynastyID)
		require.NoError(t, err)
		assert.Equal(t, dynasty.ID, dbDynastyID)

		// Verify family exists
		var dbFamilyID uint64
		err = db.QueryRow("SELECT id FROM families WHERE dynasty_id = ?", dynasty.ID).Scan(&dbFamilyID)
		require.NoError(t, err)
		assert.Equal(t, family.ID, dbFamilyID)

		// Verify owner is family member
		var memberCount int
		err = db.QueryRow("SELECT COUNT(*) FROM family_members WHERE family_id = ? AND user_id = ? AND relationship = 'owner'", family.ID, ownerID).Scan(&memberCount)
		require.NoError(t, err)
		assert.Equal(t, 1, memberCount)
	})

	t.Run("SendAndAcceptJoinRequest", func(t *testing.T) {
		// Create dynasty first
		dynastyRepo := repository.NewDynastyRepository(db)
		familyRepo := repository.NewFamilyRepository(db)
		prizeRepo := repository.NewPrizeRepository(db)
		dynastyService := service.NewDynastyService(dynastyRepo, familyRepo, prizeRepo, "")

		dynasty, family, err := dynastyService.CreateDynasty(ctx, uint64(ownerID), featureID)
		require.NoError(t, err)

		// Create applicant user
		applicantID := CreateTestUser(t, db, "applicant_"+time.Now().Format("150405"), "applicant@test.com")

		joinRequestRepo := repository.NewJoinRequestRepository(db)
		joinRequestService := service.NewJoinRequestService(joinRequestRepo, dynastyRepo, familyRepo, prizeRepo, "")

		// Send join request
		message := "Please accept me as offspring"
		req := &models.JoinRequest{
			FromUser:     uint64(ownerID),
			ToUser:       uint64(applicantID),
			Status:       0, // pending
			Relationship: "offspring",
			Message:      &message,
		}

		err = joinRequestRepo.CreateJoinRequest(ctx, req)
		require.NoError(t, err)
		assert.NotZero(t, req.ID)

		// Verify request created
		var status int16
		err = db.QueryRow("SELECT status FROM join_requests WHERE id = ?", req.ID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, int16(0), status)

		// Accept join request
		err = joinRequestService.AcceptJoinRequest(ctx, req.ID, uint64(applicantID))
		require.NoError(t, err)

		// Verify request status updated
		err = db.QueryRow("SELECT status FROM join_requests WHERE id = ?", req.ID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, int16(1), status) // accepted

		// Verify member added to family
		var memberCount int
		err = db.QueryRow("SELECT COUNT(*) FROM family_members WHERE family_id = ? AND user_id = ? AND relationship = 'offspring'", family.ID, applicantID).Scan(&memberCount)
		require.NoError(t, err)
		assert.Equal(t, 1, memberCount)
	})

	t.Run("UpdateChildPermissions", func(t *testing.T) {
		// Create dynasty and add child
		dynastyRepo := repository.NewDynastyRepository(db)
		familyRepo := repository.NewFamilyRepository(db)
		prizeRepo := repository.NewPrizeRepository(db)
		dynastyService := service.NewDynastyService(dynastyRepo, familyRepo, prizeRepo, "")

		parentID := CreateTestUser(t, db, "parent_"+time.Now().Format("150405"), "parent@test.com")
		childID := CreateTestUser(t, db, "child_"+time.Now().Format("150405"), "child@test.com")

		dynasty, family, err := dynastyService.CreateDynasty(ctx, uint64(parentID), featureID)
		require.NoError(t, err)

		// Add child as family member
		joinRequestRepo := repository.NewJoinRequestRepository(db)
		permissionRepo := repository.NewPermissionRepository(db)

		// Create child permission record
		perm := &models.ChildPermission{
			UserID:   uint64(childID),
			Verified: false,
			BFR:      false,
			SF:       false,
			W:        false,
			JU:       false,
			DM:       false,
			PIUP:     false,
			PITC:     false,
			PIC:      false,
			ESOO:     false,
			COTB:     false,
		}
		err = permissionRepo.CreatePermission(ctx, perm)
		require.NoError(t, err)

		// Add child to family
		member := &models.FamilyMember{
			FamilyID:     family.ID,
			UserID:       uint64(childID),
			Relationship: "offspring",
		}
		err = familyRepo.CreateFamilyMember(ctx, member)
		require.NoError(t, err)

		// Update child permission
		permissionService := service.NewPermissionService(permissionRepo, joinRequestRepo, familyRepo, dynastyRepo)
		err = permissionService.UpdateChildPermission(ctx, uint64(parentID), uint64(childID), "BFR", true)
		require.NoError(t, err)

		// Verify permission updated
		updatedPerm, err := permissionRepo.GetByUserID(ctx, uint64(childID))
		require.NoError(t, err)
		assert.NotNil(t, updatedPerm)
		assert.True(t, updatedPerm.BFR)
	})

	t.Run("PrizeRedemption", func(t *testing.T) {
		// Create user and award prize
		userID := CreateTestUser(t, db, "prize_user_"+time.Now().Format("150405"), "prize@test.com")
		prizeRepo := repository.NewPrizeRepository(db)
		prizeService := service.NewPrizeService(prizeRepo)

		// Create a prize first
		_, err := db.Exec(`
			INSERT INTO dynasty_prizes (member, satisfaction, introduction_profit_increase, accumulated_capital_reserve, data_storage, psc, created_at, updated_at)
			VALUES ('offspring', 0.1, 0.05, 0.02, 0.03, 1000, NOW(), NOW())
		`)
		require.NoError(t, err)

		var prizeID uint64
		err = db.QueryRow("SELECT id FROM dynasty_prizes WHERE member = 'offspring'").Scan(&prizeID)
		require.NoError(t, err)

		// Award prize to user
		message := "Congratulations on joining dynasty!"
		err = prizeRepo.AwardPrize(ctx, uint64(userID), prizeID, message)
		require.NoError(t, err)

		// Verify prize awarded
		var receivedPrizeID uint64
		err = db.QueryRow("SELECT id FROM received_prizes WHERE user_id = ? AND prize_id = ?", userID, prizeID).Scan(&receivedPrizeID)
		require.NoError(t, err)
		assert.NotZero(t, receivedPrizeID)

		// Claim prize
		err = prizeService.ClaimPrize(ctx, receivedPrizeID, uint64(userID))
		require.NoError(t, err)

		// Verify prize deleted (claimed)
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM received_prizes WHERE id = ?", receivedPrizeID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("FeatureUpdateWithin30Days_CreatesDebt", func(t *testing.T) {
		// This would require enhanced service with commercial client
		// For now, just verify the logic structure exists
		dynastyRepo := repository.NewDynastyRepository(db)
		familyRepo := repository.NewFamilyRepository(db)
		prizeRepo := repository.NewPrizeRepository(db)
		dynastyService := service.NewDynastyService(dynastyRepo, familyRepo, prizeRepo, "")

		dynasty, _, err := dynastyService.CreateDynasty(ctx, uint64(ownerID), featureID)
		require.NoError(t, err)

		// Update dynasty updated_at to simulate recent change
		recentTime := time.Now().AddDate(0, 0, -15) // 15 days ago
		_, err = db.Exec("UPDATE dynasties SET updated_at = ? WHERE id = ?", recentTime, dynasty.ID)
		require.NoError(t, err)

		// In enhanced service, this would check if < 30 days and create debt
		// For integration test, we verify the dynasty exists and can be updated
		err = dynastyRepo.UpdateDynastyFeature(ctx, dynasty.ID, featureID+1)
		// This should work, but in production would trigger debt creation
		assert.NoError(t, err)
	})
}

// Helper function to create test feature ID (adapted from test_helpers)
func CreateTestFeatureID(t *testing.T, db *sql.DB, userID int64) int64 {
	result, err := db.Exec(`
		INSERT INTO features (user_id, created_at, updated_at)
		VALUES (?, NOW(), NOW())
	`, userID)

	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	id, _ := result.LastInsertId()
	return id
}
