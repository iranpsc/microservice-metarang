package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"metarang/dynasty-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/dynasty-service/internal/repository"
)

func TestPermissionService_UpdateChildPermission(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	permissionRepo := repository.NewPermissionRepository(db)
	joinRequestRepo := repository.NewJoinRequestRepository(db)
	familyRepo := repository.NewFamilyRepository(db)
	dynastyRepo := repository.NewDynastyRepository(db)
	service := service.NewPermissionService(permissionRepo, joinRequestRepo, familyRepo, dynastyRepo)

	ctx := context.Background()
	parentUserID := uint64(1)
	childUserID := uint64(2)

	t.Run("Success", func(t *testing.T) {
		// Check user age (under 18)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))

		// Get dynasty
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, time.Now(), time.Now()))

		// Get family
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
				AddRow(1, 1, time.Now(), time.Now()))

		// Get family members (with pagination: page=1, perPage=1000)
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(2))
		mock.ExpectQuery("SELECT id, family_id, user_id").
			WithArgs(1, 1000, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "family_id", "user_id", "relationship", "created_at", "updated_at"}).
				AddRow(1, 1, parentUserID, "owner", time.Now(), time.Now()).
				AddRow(2, 1, childUserID, "offspring", time.Now(), time.Now()))

		// Get existing permissions
		mock.ExpectQuery("SELECT id, user_id").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "verified", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at"}).
				AddRow(1, childUserID, true, false, false, false, false, false, false, false, false, false, false, time.Now(), time.Now()))

		// Update specific permission
		mock.ExpectExec("UPDATE children_permissions SET BFR = \\?").
			WithArgs(true, childUserID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.UpdateChildPermission(ctx, parentUserID, childUserID, "BFR", true)
		require.NoError(t, err)
	})

	t.Run("CannotControlSelf", func(t *testing.T) {
		canControl, err := service.CanControlPermissions(ctx, parentUserID, parentUserID)
		require.NoError(t, err)
		assert.False(t, canControl)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPermissionService_UpdateChildPermission_Failures(t *testing.T) {
	ctx := context.Background()
	parentUserID, childUserID := uint64(1), uint64(2)
	now := time.Now()

	t.Run("CannotControl", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		svc := service.NewPermissionService(
			repository.NewPermissionRepository(db),
			repository.NewJoinRequestRepository(db),
			repository.NewFamilyRepository(db),
			repository.NewDynastyRepository(db),
		)
		err = svc.UpdateChildPermission(ctx, parentUserID, parentUserID, "BFR", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "مجاز")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CanControlError", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		svc := service.NewPermissionService(
			repository.NewPermissionRepository(db),
			repository.NewJoinRequestRepository(db),
			repository.NewFamilyRepository(db),
			repository.NewDynastyRepository(db),
		)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnError(errors.New("age failed"))
		err = svc.UpdateChildPermission(ctx, parentUserID, childUserID, "BFR", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check permissions")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetPermissionsError", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		svc := service.NewPermissionService(
			repository.NewPermissionRepository(db),
			repository.NewJoinRequestRepository(db),
			repository.NewFamilyRepository(db),
			repository.NewDynastyRepository(db),
		)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, now, now))
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
				AddRow(1, 1, now, now))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
		mock.ExpectQuery("SELECT id, family_id, user_id").
			WithArgs(uint64(1), int32(1000), int32(0)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "family_id", "user_id", "relationship", "created_at", "updated_at"}).
				AddRow(2, 1, childUserID, "offspring", now, now))
		mock.ExpectQuery("SELECT id, user_id").
			WithArgs(childUserID).
			WillReturnError(errors.New("perm query failed"))
		err = svc.UpdateChildPermission(ctx, parentUserID, childUserID, "BFR", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get permissions")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NoPermissionRecord", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		svc := service.NewPermissionService(
			repository.NewPermissionRepository(db),
			repository.NewJoinRequestRepository(db),
			repository.NewFamilyRepository(db),
			repository.NewDynastyRepository(db),
		)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, now, now))
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
				AddRow(1, 1, now, now))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
		mock.ExpectQuery("SELECT id, family_id, user_id").
			WithArgs(uint64(1), int32(1000), int32(0)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "family_id", "user_id", "relationship", "created_at", "updated_at"}).
				AddRow(2, 1, childUserID, "offspring", now, now))
		mock.ExpectQuery("SELECT id, user_id").
			WithArgs(childUserID).
			WillReturnError(sql.ErrNoRows)
		err = svc.UpdateChildPermission(ctx, parentUserID, childUserID, "BFR", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "child has no permission record")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdatePermissionError", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		svc := service.NewPermissionService(
			repository.NewPermissionRepository(db),
			repository.NewJoinRequestRepository(db),
			repository.NewFamilyRepository(db),
			repository.NewDynastyRepository(db),
		)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, now, now))
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
				AddRow(1, 1, now, now))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
		mock.ExpectQuery("SELECT id, family_id, user_id").
			WithArgs(uint64(1), int32(1000), int32(0)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "family_id", "user_id", "relationship", "created_at", "updated_at"}).
				AddRow(2, 1, childUserID, "offspring", now, now))
		mock.ExpectQuery("SELECT id, user_id").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "verified", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at"}).
				AddRow(1, childUserID, true, false, false, false, false, false, false, false, false, false, false, now, now))
		mock.ExpectExec("UPDATE children_permissions SET BFR").
			WithArgs(true, childUserID).
			WillReturnError(errors.New("update failed"))
		err = svc.UpdateChildPermission(ctx, parentUserID, childUserID, "BFR", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update permission")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPermissionService_CanControlPermissions_Branches(t *testing.T) {
	ctx := context.Background()
	parentUserID, childUserID := uint64(1), uint64(2)
	now := time.Now()

	newSvc := func(t *testing.T) (sqlmock.Sqlmock, *service.PermissionService) {
		t.Helper()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		return mock, service.NewPermissionService(
			repository.NewPermissionRepository(db),
			repository.NewJoinRequestRepository(db),
			repository.NewFamilyRepository(db),
			repository.NewDynastyRepository(db),
		)
	}

	t.Run("NotUnder18", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(false))
		ok, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		require.NoError(t, err)
		assert.False(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("AgeError", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnError(errors.New("age failed"))
		_, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check child age")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NoDynasty", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnError(sql.ErrNoRows)
		ok, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		require.NoError(t, err)
		assert.False(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DynastyError", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnError(errors.New("db"))
		_, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dynasty")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FamilyError", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, now, now))
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(uint64(1)).
			WillReturnError(errors.New("family failed"))
		_, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get family")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("MembersError", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, now, now))
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
				AddRow(1, 1, now, now))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(uint64(1)).
			WillReturnError(errors.New("count failed"))
		_, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get family members")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ChildNotInFamily", func(t *testing.T) {
		mock, svc := newSvc(t)
		mock.ExpectQuery("SELECT TIMESTAMPDIFF").
			WithArgs(childUserID).
			WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
		mock.ExpectQuery("SELECT id, user_id, feature_id").
			WithArgs(parentUserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
				AddRow(1, parentUserID, 100, now, now))
		mock.ExpectQuery("SELECT id, dynasty_id").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
				AddRow(1, 1, now, now))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
		mock.ExpectQuery("SELECT id, family_id, user_id").
			WithArgs(uint64(1), int32(1000), int32(0)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "family_id", "user_id", "relationship", "created_at", "updated_at"}).
				AddRow(1, 1, parentUserID, "owner", now, now))
		ok, err := svc.CanControlPermissions(ctx, parentUserID, childUserID)
		require.NoError(t, err)
		assert.False(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
