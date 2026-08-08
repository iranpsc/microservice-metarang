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

func TestJoinRequestServiceEnhanced_AcceptJoinRequest(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewJoinRequestServiceEnhanced(
		repository.NewJoinRequestRepository(db),
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
		repository.NewValidationRepository(db),
		repository.NewMessageRepository(db),
		repository.NewPrizeRepository(db),
	)
	ctx := context.Background()
	now := time.Now()
	requestID, fromUserID, toUserID := uint64(1), uint64(10), uint64(20)

	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(requestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}).
			AddRow(requestID, fromUserID, toUserID, 0, "brother", nil, now, now))
	mock.ExpectExec("UPDATE join_requests SET status").
		WithArgs(1, requestID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
			AddRow(1, fromUserID, 100, now, now))
	mock.ExpectQuery("SELECT id, dynasty_id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "dynasty_id", "created_at", "updated_at"}).
			AddRow(1, 1, now, now))
	mock.ExpectExec("INSERT INTO family_members").
		WithArgs(sqlmock.AnyArg(), toUserID, "brother").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(false))
	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(false))
	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs("brother").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}).AddRow(7, "brother", 1.0, 0.1, 0.2, 0.3, 10, now, now))
	mock.ExpectQuery("SELECT message FROM dynasty_messages").
		WithArgs("requester_accept_message").
		WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("ok [sender-name]"))
	mock.ExpectQuery("SELECT message FROM dynasty_messages").
		WithArgs("reciever_accept_message").
		WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("ok [reciever-name]"))
	mock.ExpectExec("INSERT INTO received_prizes").
		WithArgs(fromUserID, uint64(7), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(requestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}).
			AddRow(requestID, fromUserID, toUserID, 1, "brother", nil, now, now))

	req, err := svc.AcceptJoinRequest(ctx, requestID, toUserID)
	require.NoError(t, err)
	require.NotNil(t, req)
	require.NoError(t, mock.ExpectationsWereMet())
}
