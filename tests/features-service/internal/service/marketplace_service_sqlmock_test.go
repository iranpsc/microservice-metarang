package service_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"metarang/features-service/internal/repository"
	"metarang/features-service/internal/service"
	"metarang/features-service/tests/internal/testutil"
	pb "metarang/shared/pb/features"
	"metarang/shared/pkg/logger"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMarketplaceSQLMock(t *testing.T) (*service.MarketplaceService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := testutil.NewSQLMock(t)
	log := logger.NewLogger("features-test")
	svc := service.NewMarketplaceService(
		repository.NewFeatureRepository(db),
		repository.NewPropertiesRepository(db),
		repository.NewGeometryRepository(db),
		repository.NewTradeRepository(db),
		repository.NewBuyRequestRepository(db),
		repository.NewSellRequestRepository(db),
		repository.NewLockedAssetRepository(db),
		repository.NewHourlyProfitRepository(db),
		repository.NewFeatureLimitRepository(db),
		repository.NewVariableRepository(db),
		nil, nil, nil,
		nil,
		db,
		log,
	)
	return svc, mock
}

func expectFeatureFindByID(mock sqlmock.Sqlmock, ownerID uint64, karbari string, stability float64, minPct int) {
	now := time.Now()
	mock.ExpectQuery("FROM features f").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "map_id", "type", "created_at", "updated_at",
			"prop_id", "feature_id", "karbari", "rgb", "owner", "label", "address",
			"area", "density", "stability", "price_psc", "price_irr", "minimum_price_percentage",
			"prop_created_at", "prop_updated_at",
		}).AddRow(1, ownerID, 1, "polygon", now, now,
			"p1", 1, karbari, "d", "o", "l", "addr",
			10.0, 1, stability, "0", "0", minPct, now, now))
}

func buyRequestFindRows(sellerID, buyerID uint64, status int) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"id", "buyer_id", "seller_id", "feature_id", "note", "price_psc", "price_irr",
		"status", "requested_grace_period", "created_at", "updated_at",
	}).AddRow(9, buyerID, sellerID, 1, "n", 10.0, 20.0, status, nil, now, now)
}

func TestMarketplaceService_SendBuyRequest_BothPricesZero(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)
	expectFeatureFindByID(mock, 3, "m", 1000, 80)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(2), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))

	_, err := svc.SendBuyRequest(context.Background(), &pb.SendBuyRequestRequest{
		BuyerId: 2, FeatureId: 1, PricePsc: "0", PriceIrr: "0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot both be zero")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_SendBuyRequest_DuplicatePending(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)
	expectFeatureFindByID(mock, 3, "m", 1000, 80)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(2), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))

	_, err := svc.SendBuyRequest(context.Background(), &pb.SendBuyRequestRequest{
		BuyerId: 2, FeatureId: 1, PricePsc: "10", PriceIrr: "10",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pending buy request")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_SendBuyRequest_PriceBelowFloor(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)
	expectFeatureFindByID(mock, 3, "m", 1000, 90)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(2), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	mock.ExpectQuery("SELECT value FROM variables").
		WithArgs("psc").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.0))
	mock.ExpectQuery("SELECT value FROM variables").
		WithArgs("yellow").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.0))

	_, err := svc.SendBuyRequest(context.Background(), &pb.SendBuyRequestRequest{
		BuyerId: 2, FeatureId: 1, PricePsc: "0", PriceIrr: "100",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "مجاز"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_UpdateGracePeriod(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)

	err := svc.UpdateGracePeriod(context.Background(), 9, 3, 0)
	require.Error(t, err)
	err = svc.UpdateGracePeriod(context.Background(), 9, 3, 31)
	require.Error(t, err)

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnError(sql.ErrNoRows)
	err = svc.UpdateGracePeriod(context.Background(), 9, 3, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnRows(buyRequestFindRows(99, 2, 0))
	err = svc.UpdateGracePeriod(context.Background(), 9, 3, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the seller")

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnRows(buyRequestFindRows(3, 2, 1))
	err = svc.UpdateGracePeriod(context.Background(), 9, 3, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not pending")

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnRows(buyRequestFindRows(3, 2, 0))
	mock.ExpectExec("SET requested_grace_period").
		WithArgs(sqlmock.AnyArg(), uint64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, svc.UpdateGracePeriod(context.Background(), 9, 3, 5))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_RequestGracePeriod(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)

	err := svc.RequestGracePeriod(context.Background(), 9, 3, "abc")
	require.Error(t, err)

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnRows(buyRequestFindRows(3, 2, 0))
	mock.ExpectExec("SET requested_grace_period").
		WithArgs(sqlmock.AnyArg(), uint64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, svc.RequestGracePeriod(context.Background(), 9, 3, "7"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_RejectBuyRequest(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnError(sql.ErrNoRows)
	err := svc.RejectBuyRequest(context.Background(), 9, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnRows(buyRequestFindRows(99, 2, 0))
	err = svc.RejectBuyRequest(context.Background(), 9, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the seller")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_DeleteBuyRequest(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnError(sql.ErrNoRows)
	err := svc.DeleteBuyRequest(context.Background(), 9, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	mock.ExpectQuery("FROM buy_feature_requests").
		WithArgs(uint64(9)).
		WillReturnRows(buyRequestFindRows(3, 99, 0))
	err = svc.DeleteBuyRequest(context.Background(), 9, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the buyer")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_CreateSellRequest_Validation(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)

	_, err := svc.CreateSellRequest(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	expectFeatureFindByID(mock, 3, "m", 1000, 80)
	_, err = svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
		SellerId: 99, FeatureId: 1, MinimumPricePercentage: 90,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the owner")

	expectFeatureFindByID(mock, 3, "m", 1000, 80)
	mock.ExpectQuery("FROM system_variables").
		WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("80", "110"))
	mock.ExpectQuery("SELECT birthdate FROM kycs").
		WithArgs(uint64(3)).
		WillReturnError(sql.ErrNoRows)
	_, err = svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
		SellerId: 3, FeatureId: 1, PricePsc: "10", MinimumPricePercentage: 90,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")

	expectFeatureFindByID(mock, 3, "m", 1000, 80)
	mock.ExpectQuery("FROM system_variables").
		WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("80", "110"))
	mock.ExpectQuery("SELECT birthdate FROM kycs").
		WithArgs(uint64(3)).
		WillReturnError(sql.ErrNoRows)
	_, err = svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
		SellerId: 3, FeatureId: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	expectFeatureFindByID(mock, 3, "m", 1000, 80)
	mock.ExpectQuery("FROM system_variables").
		WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("80", "110"))
	mock.ExpectQuery("SELECT birthdate FROM kycs").
		WithArgs(uint64(3)).
		WillReturnError(sql.ErrNoRows)
	_, err = svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
		SellerId: 3, FeatureId: 1, MinimumPricePercentage: 79,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 80")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_NotFound(t *testing.T) {
	svc, mock := newMarketplaceSQLMock(t)
	mock.ExpectQuery("FROM features f").
		WithArgs(uint64(1)).
		WillReturnError(sql.ErrNoRows)

	_, err := svc.BuyFeature(context.Background(), 1, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feature not found")
	require.NoError(t, mock.ExpectationsWereMet())
}
