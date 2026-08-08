package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/dynasty-service/internal/repository"
	"metarang/dynasty-service/internal/service"
)

func TestPermissionService_GetDefaultAndCheckPermission(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewPermissionService(
		repository.NewPermissionRepository(db),
		repository.NewJoinRequestRepository(db),
		repository.NewFamilyRepository(db),
		repository.NewDynastyRepository(db),
	)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("SELECT id, BFR, SF, W, JU, DM").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at",
		}).AddRow(1, true, true, true, true, true, true, true, true, true, true, now, now))
	def, err := svc.GetDefaultPermissions(ctx)
	require.NoError(t, err)
	require.NotNil(t, def)

	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(false))
	ok, err := svc.CheckPermission(ctx, 9, "BFR")
	require.NoError(t, err)
	assert.True(t, ok)

	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
	mock.ExpectQuery("SELECT id, user_id").
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "verified", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at",
		}).AddRow(1, 10, true, true, false, true, false, true, false, false, false, false, false, now, now))
	ok, err = svc.CheckPermission(ctx, 10, "BFR")
	require.NoError(t, err)
	assert.True(t, ok)
	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
	mock.ExpectQuery("SELECT id, user_id").
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "verified", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at",
		}).AddRow(1, 10, true, true, false, true, false, true, false, false, false, false, false, now, now))
	ok, err = svc.CheckPermission(ctx, 10, "SF")
	require.NoError(t, err)
	assert.False(t, ok)

	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
	mock.ExpectQuery("SELECT id, user_id").
		WithArgs(uint64(11)).
		WillReturnError(sql.ErrNoRows)
	ok, err = svc.CheckPermission(ctx, 11, "DM")
	require.NoError(t, err)
	assert.False(t, ok)

	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(uint64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(true))
	mock.ExpectQuery("SELECT id, user_id").
		WithArgs(uint64(12)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "verified", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at",
		}).AddRow(1, 12, true, true, true, true, true, true, true, true, true, true, true, now, now))
	_, err = svc.CheckPermission(ctx, 12, "NOPE")
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
