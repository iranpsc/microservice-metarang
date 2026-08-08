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

func TestDynastyServiceEnhanced_UpdateAndProfit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewDynastyServiceEnhanced(
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
	)
	assert.Equal(t, "1.500", svc.CalculateFeatureProfitIncrease(25000))
	assert.Equal(t, "0", svc.CalculateFeatureProfitIncrease(5000))

	now := time.Now()
	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
			AddRow(1, 10, 100, now, now.Add(-48*time.Hour)))
	mock.ExpectExec("UPDATE dynasties SET feature_id").
		WithArgs(uint64(200), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, svc.UpdateDynastyFeature(context.Background(), 1, 200, 10))

	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
			AddRow(2, 10, 100, now, now))
	err = svc.UpdateDynastyFeature(context.Background(), 2, 100, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already the dynasty feature")

	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
			AddRow(3, 99, 100, now, now))
	err = svc.UpdateDynastyFeature(context.Background(), 3, 200, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")

	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(uint64(4)).
		WillReturnError(sql.ErrNoRows)
	err = svc.UpdateDynastyFeature(context.Background(), 4, 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dynasty not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrizeServiceEnhanced_AwardClaimAndLists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewPrizeServiceEnhanced(repository.NewPrizeRepository(db))
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs("brother").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}).AddRow(7, "brother", 1.0, 0.1, 0.2, 0.3, 10, now, now))
	mock.ExpectExec("INSERT INTO received_prizes").
		WithArgs(uint64(1), uint64(7), "msg").
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, svc.AwardPrize(ctx, 1, "brother", "msg"))

	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs("cousin").
		WillReturnError(sql.ErrNoRows)
	require.NoError(t, svc.AwardPrize(ctx, 1, "cousin", "none"))

	receivedRows := func(uid uint64) *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"rp.id", "rp.user_id", "rp.prize_id", "rp.message", "rp.created_at", "rp.updated_at",
			"dp.member", "dp.satisfaction", "dp.introduction_profit_increase",
			"dp.accumulated_capital_reserve", "dp.data_storage", "dp.psc",
		}).AddRow(5, uid, 7, "m", now, now, "brother", 1.0, 0.1, 0.2, 0.3, 10)
	}

	mock.ExpectQuery("SELECT rp.id, rp.user_id").
		WithArgs(uint64(5)).
		WillReturnRows(receivedRows(1))
	mock.ExpectExec("DELETE FROM received_prizes WHERE id").
		WithArgs(uint64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, svc.ClaimPrize(ctx, 1, 5))

	mock.ExpectQuery("SELECT rp.id, rp.user_id").
		WithArgs(uint64(6)).
		WillReturnError(sql.ErrNoRows)
	err = svc.ClaimPrize(ctx, 1, 6)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	mock.ExpectQuery("SELECT rp.id, rp.user_id").
		WithArgs(uint64(8)).
		WillReturnRows(receivedRows(99))
	err = svc.ClaimPrize(ctx, 1, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")

	mock.ExpectQuery("FROM received_prizes rp").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"rp.id", "rp.user_id", "rp.prize_id", "rp.message", "rp.created_at", "rp.updated_at",
			"dp.id", "dp.member", "dp.satisfaction", "dp.introduction_profit_increase",
			"dp.accumulated_capital_reserve", "dp.data_storage", "dp.psc",
		}).AddRow(3, 1, 7, "m", now, now, 7, "brother", 1.0, 0.1, 0.2, 0.3, 10))
	prizes, err := svc.GetUserPrizes(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, prizes, 1)

	mock.ExpectQuery("FROM dynasty_prizes").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}).AddRow(7, "brother", 1.0, 0.1, 0.2, 0.3, 10, now, now))
	all, err := svc.GetIntroductionPrizes(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJoinRequestService_ListLookupPrizeAndPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewJoinRequestService(
		repository.NewJoinRequestRepository(db),
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
		repository.NewPrizeRepository(db),
		nil,
		"",
	)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(uint64(1), int32(10), int32(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}))
	list, total, err := svc.GetSentRequests(ctx, 1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int32(0), total)
	assert.Empty(t, list)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(uint64(1), int32(10), int32(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}))
	list, total, err = svc.GetReceivedRequests(ctx, 1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int32(0), total)
	assert.Empty(t, list)

	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}).
			AddRow(9, 1, 2, 0, "brother", nil, now, now))
	req, err := svc.GetJoinRequest(ctx, 9, 1)
	require.NoError(t, err)
	require.NotNil(t, req)

	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}).
			AddRow(9, 1, 2, 0, "brother", nil, now, now))
	_, err = svc.GetJoinRequest(ctx, 9, 99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")

	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs("brother").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}).AddRow(1, "brother", 1.0, 0.1, 0.2, 0.3, 10, now, now))
	prize, err := svc.GetPrizeByRelationship(ctx, "brother")
	require.NoError(t, err)
	require.NotNil(t, prize)

	nilPrizeSvc := service.NewJoinRequestService(
		repository.NewJoinRequestRepository(db),
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
		nil,
		nil,
		"",
	)
	p, err := nilPrizeSvc.GetPrizeByRelationship(ctx, "brother")
	require.NoError(t, err)
	assert.Nil(t, p)

	mock.ExpectQuery("FROM dynasty_permissions").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "BFR", "SF", "W", "JU", "DM", "PIUP", "PITC", "PIC", "ESOO", "COTB", "created_at", "updated_at",
		}).AddRow(1, true, false, true, false, false, false, false, false, false, false, now, now))
	perms, err := svc.GetDefaultPermissions(ctx)
	require.NoError(t, err)
	require.NotNil(t, perms)
	assert.True(t, perms.BFR)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrizeService_GetAllAndGetPrize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewPrizeService(db, repository.NewPrizeRepository(db), nil, nil, nil)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs(int32(10), int32(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}).AddRow(1, "brother", 1.0, 0.1, 0.2, 0.3, 10, now, now))
	items, total, err := svc.GetAllPrizes(ctx, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int32(1), total)
	assert.Len(t, items, 1)

	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}).AddRow(1, "brother", 1.0, 0.1, 0.2, 0.3, 10, now, now))
	prize, err := svc.GetPrize(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, prize)

	mock.ExpectQuery("FROM dynasty_prizes").
		WithArgs(uint64(99)).
		WillReturnError(sql.ErrNoRows)
	_, err = svc.GetPrize(ctx, 99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFamilyService_GetUserBasicInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewFamilyService(repository.NewFamilyRepository(db), repository.NewDynastyRepository(db))
	mock.ExpectQuery("SELECT id, code, name FROM users").
		WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name"}).AddRow(3, "C3", "Name"))
	mock.ExpectQuery("SELECT url FROM images").
		WithArgs(uint64(3)).
		WillReturnError(sql.ErrNoRows)
	info, err := svc.GetUserBasicInfo(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "C3", info.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDynastyService_GetByUserIDAndHelpers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewDynastyService(
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
		repository.NewPrizeRepository(db),
		"",
	)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(uint64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
			AddRow(5, 5, 100, now, now))
	d, err := svc.GetDynastyByUserID(ctx, 5)
	require.NoError(t, err)
	require.NotNil(t, d)

	mock.ExpectQuery("FROM dynasty_prizes").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "member", "satisfaction", "introduction_profit_increase",
			"accumulated_capital_reserve", "data_storage", "psc", "created_at", "updated_at",
		}))
	prizes, err := svc.GetIntroductionPrizes(ctx)
	require.NoError(t, err)
	assert.Empty(t, prizes)

	mock.ExpectQuery("SELECT price FROM variables").
		WithArgs("psc").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(100.0))
	rate, err := svc.GetVariableRate(ctx, "psc")
	require.NoError(t, err)
	assert.Equal(t, 100.0, rate)

	require.NoError(t, mock.ExpectationsWereMet())
}
