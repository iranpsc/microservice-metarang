package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"metarang/features-service/internal/constants"
	"metarang/features-service/tests/internal/testutil"
	pb "metarang/shared/pb/features"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expectUnderpricedChain(mock sqlmock.Sqlmock, sellerID uint64, tradeCreatedAt time.Time, hasSell, hasTrade bool) {
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	now := time.Now()
	if !hasSell {
		mock.ExpectQuery("`limit` < 100").
			WithArgs(sellerID).
			WillReturnError(sql.ErrNoRows)
		return
	}
	mock.ExpectQuery("`limit` < 100").
		WithArgs(sellerID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "seller_id", "feature_id", "price_psc", "price_irr", "limit", "status", "created_at", "updated_at",
		}).AddRow(8, sellerID, 1, 10.0, 20.0, 90, 0, now, now))
	if !hasTrade {
		mock.ExpectQuery("INNER JOIN sell_feature_requests").
			WithArgs(sellerID, uint64(1)).
			WillReturnError(sql.ErrNoRows)
		return
	}
	mock.ExpectQuery("INNER JOIN sell_feature_requests").
		WithArgs(sellerID, uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feature_id", "buyer_id", "seller_id", "irr_amount", "psc_amount", "date", "created_at", "updated_at",
		}).AddRow(9, 1, 2, sellerID, 50.0, 10.0, tradeCreatedAt, tradeCreatedAt, now))
}

func TestMarketplaceService_BuyFeature_OwnerLookupFails(t *testing.T) {
	stub := testutil.NewCommercialStub()
	svc, mock := newMarketplaceWithWallet(t, stub)
	expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
	mock.ExpectQuery("SELECT code FROM users WHERE id").
		WithArgs(uint64(5)).
		WillReturnError(sql.ErrNoRows)

	_, err := svc.BuyFeature(context.Background(), 1, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_Limited_InvalidKarbari(t *testing.T) {
	stub := testutil.NewCommercialStub()
	svc, mock := newMarketplaceWithWallet(t, stub)
	expectFeatureFindByIDRGB(mock, 5, "x", constants.MaskoniTradingLimited, "0", "0", 10, 80)
	expectOwnerCode(mock, 5, "user-5")
	expectLimitation(mock, true)
	expectBuyerKYCLimited(mock, 2)

	_, err := svc.BuyFeature(context.Background(), 1, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid feature karbari")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_Limited_BuyerKYCFails(t *testing.T) {
	stub := testutil.NewCommercialStub()
	svc, mock := newMarketplaceWithWallet(t, stub)
	expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniTradingLimited, "0", "0", 10, 80)
	expectOwnerCode(mock, 5, "user-5")
	expectLimitation(mock, true)
	mock.ExpectQuery("SELECT u.name, u.dynasty_id, k.birthdate").
		WithArgs(uint64(2)).
		WillReturnError(sql.ErrNoRows)

	_, err := svc.BuyFeature(context.Background(), 1, 2)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_Limited_Under18NoPriceLimit(t *testing.T) {
	stub := testutil.NewCommercialStub()
	stub.Yellow = "1000"
	svc, mock := newMarketplaceWithWallet(t, stub)

	expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniTradingLimited, "0", "0", 10, 80)
	expectOwnerCode(mock, 5, "user-5")
	expectLimitation(mock, false)
	mock.ExpectQuery("SELECT u.name, u.dynasty_id, k.birthdate").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"name", "dynasty_id", "birthdate"}).
			AddRow("buyer", nil, time.Now().AddDate(-10, 0, 0)))

	newStatus := constants.ChangeStatusToSoldAndNotPriced("m")
	mock.ExpectExec("UPDATE features SET owner_id").
		WithArgs(uint64(2), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("SET rgb = \\?, owner = \\?, label").
		WithArgs(newStatus, "buyer", "", constants.DefaultUnder18PricingLimit, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO trades").
		WithArgs(uint64(1), uint64(2), uint64(5), 0.0, 0.0).
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectQuery("SELECT withdraw_profit FROM user_variables").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"withdraw_profit"}).AddRow(14))
	now := time.Now()
	mock.ExpectQuery("SELECT id, user_id, feature_id, asset, amount").
		WithArgs(uint64(1), uint64(5)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "feature_id", "asset", "amount", "dead_line", "is_active", "created_at", "updated_at",
		}).AddRow(4, 5, 1, "yellow", 3.5, now, true, now, now))
	mock.ExpectQuery("SELECT id, asset FROM feature_hourly_profits WHERE feature_id = \\? AND user_id").
		WithArgs(uint64(1), uint64(5)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, asset FROM feature_hourly_profits WHERE feature_id = \\? ORDER BY").
		WithArgs(uint64(1)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO feature_hourly_profits").
		WithArgs(uint64(2), uint64(1), "yellow", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM feature_hourly_profits WHERE feature_id").
		WithArgs(uint64(1), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO limited_feature_purchases").
		WithArgs(uint64(2), uint64(3), uint64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectReloadFeatureAndGeometry(mock, 2, constants.MaskoniSoldAndNotPriced)

	feat, err := svc.BuyFeature(context.Background(), 1, 2)
	require.NoError(t, err)
	require.NotNil(t, feat)
	require.Len(t, stub.DeductCalls, 1)
	require.GreaterOrEqual(t, len(stub.AddCalls), 2)
	assert.Equal(t, testutil.BalanceOp{UserID: 5, Asset: "yellow", Amount: 10}, stub.AddCalls[0])
	assert.Equal(t, testutil.BalanceOp{UserID: 5, Asset: "yellow", Amount: 3.5}, stub.AddCalls[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_Limited_UpdateOwnerFails(t *testing.T) {
	stub := testutil.NewCommercialStub()
	stub.Yellow = "1000"
	svc, mock := newMarketplaceWithWallet(t, stub)
	expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniTradingLimited, "0", "0", 10, 80)
	expectOwnerCode(mock, 5, "user-5")
	expectLimitation(mock, true)
	expectBuyerKYCLimited(mock, 2)
	mock.ExpectExec("UPDATE features SET owner_id").
		WithArgs(uint64(2), uint64(1)).
		WillReturnError(sql.ErrConnDone)

	_, err := svc.BuyFeature(context.Background(), 1, 2)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_RGB_InvalidKarbari(t *testing.T) {
	stub := testutil.NewCommercialStub()
	svc, mock := newMarketplaceWithWallet(t, stub)
	expectFeatureFindByIDRGB(mock, 5, "x", constants.MaskoniPriced, "0", "0", 10, 80)
	expectOwnerCode(mock, 5, constants.RGBUserCode)
	expectBuyerKYCSimple(mock, 2)

	_, err := svc.BuyFeature(context.Background(), 1, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid feature karbari")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarketplaceService_BuyFeature_RGB_InsufficientAndDeductFail(t *testing.T) {
	t.Run("insufficient", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		stub.Yellow = "1"
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniPriced, "0", "0", 10, 80)
		expectOwnerCode(mock, 5, constants.RGBUserCode)
		expectBuyerKYCSimple(mock, 2)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("deduct fails", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		stub.Yellow = "1000"
		stub.FailNthDeduct = 1
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniPriced, "0", "0", 10, 80)
		expectOwnerCode(mock, 5, constants.RGBUserCode)
		expectBuyerKYCSimple(mock, 2)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("credit rgb fails rollback", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		stub.Yellow = "1000"
		stub.FailNthAdd = 1
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniPriced, "0", "0", 10, 80)
		expectOwnerCode(mock, 5, constants.RGBUserCode)
		expectBuyerKYCSimple(mock, 2)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		require.Len(t, stub.AddCalls, 2)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMarketplaceService_BuyFeature_UserToUser_UnderpricedBlocked(t *testing.T) {
	t.Run("hours remaining", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
		expectOwnerCode(mock, 5, "user-5")
		expectUnderpricedChain(mock, 5, time.Now().Add(-2*time.Hour), true, true)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "۲۴ ساعت")
		assert.Contains(t, err.Error(), "ساعت")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("minutes remaining", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
		expectOwnerCode(mock, 5, "user-5")
		expectUnderpricedChain(mock, 5, time.Now().Add(-23*time.Hour-40*time.Minute), true, true)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "دقیقه")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMarketplaceService_BuyFeature_UserToUser_UnderpricedSkips(t *testing.T) {
	t.Run("no latest sell", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
		expectOwnerCode(mock, 5, "user-5")
		expectUnderpricedChain(mock, 5, time.Time{}, false, false)
		expectBuyerKYCSimple(mock, 2)
		mock.ExpectQuery("SELECT id FROM users WHERE code").
			WithArgs(constants.RGBUserCode).
			WillReturnError(sql.ErrNoRows)
		expectUserBuyDBTail(mock)
		expectReloadFeatureAndGeometry(mock, 2, constants.MaskoniSoldAndNotPriced)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("outside 24h", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
		expectOwnerCode(mock, 5, "user-5")
		expectUnderpricedChain(mock, 5, time.Now().Add(-30*time.Hour), true, true)
		expectBuyerKYCSimple(mock, 2)
		mock.ExpectQuery("SELECT id FROM users WHERE code").
			WithArgs(constants.RGBUserCode).
			WillReturnError(sql.ErrNoRows)
		expectUserBuyDBTail(mock)
		mock.ExpectQuery("FROM features f").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrConnDone)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reload")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMarketplaceService_BuyFeature_UserToUser_InsufficientAndGeometryNil(t *testing.T) {
	t.Run("insufficient", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		stub.Psc = "0"
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
		expectOwnerCode(mock, 5, "user-5")
		mock.ExpectQuery("FROM sell_feature_requests").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		expectBuyerKYCSimple(mock, 2)
		_, err := svc.BuyFeature(context.Background(), 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "موجودی")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("geometry missing still succeeds", func(t *testing.T) {
		stub := testutil.NewCommercialStub()
		svc, mock := newMarketplaceWithWallet(t, stub)
		expectFeatureFindByIDRGB(mock, 5, "m", constants.MaskoniSoldAndPriced, "100", "100", 10, 80)
		expectOwnerCode(mock, 5, "user-5")
		mock.ExpectQuery("FROM sell_feature_requests").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectQuery("SELECT u.name, k.birthdate FROM users u").
			WithArgs(uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"name", "birthdate"}).
				AddRow("buyer", time.Now().AddDate(-10, 0, 0)))
		mock.ExpectQuery("SELECT id FROM users WHERE code").
			WithArgs(constants.RGBUserCode).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(99)))
		mock.ExpectExec("INSERT INTO trades").
			WithArgs(uint64(1), uint64(2), uint64(5), 100.0, 100.0).
			WillReturnResult(sqlmock.NewResult(7, 1))
		mock.ExpectExec("INSERT INTO comissions").
			WithArgs(uint64(7), 10.0, 10.0).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("UPDATE features SET owner_id").
			WithArgs(uint64(2), uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("SET rgb = \\?, owner = \\?, label").
			WithArgs(constants.ChangeStatusToSoldAndNotPriced("m"), "buyer", "", constants.DefaultUnder18PricingLimit, uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("SELECT withdraw_profit FROM user_variables").
			WithArgs(uint64(2)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, user_id, feature_id, asset, amount").
			WithArgs(uint64(1), uint64(5)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, asset FROM feature_hourly_profits WHERE feature_id = \\? AND user_id").
			WithArgs(uint64(1), uint64(5)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, asset FROM feature_hourly_profits WHERE feature_id = \\? ORDER BY").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("INSERT INTO feature_hourly_profits").
			WithArgs(uint64(2), uint64(1), "yellow", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("DELETE FROM feature_hourly_profits WHERE feature_id").
			WithArgs(uint64(1), uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("SET deleted_at = NOW\\(\\) WHERE feature_id").
			WithArgs(uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("SET status = 1, updated_at = NOW\\(\\) WHERE feature_id").
			WithArgs(uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		expectFeatureFindByIDRGB(mock, 2, "m", constants.MaskoniSoldAndNotPriced, "0", "0", 10, 80)
		mock.ExpectQuery("FROM geometries g").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		feat, err := svc.BuyFeature(context.Background(), 1, 2)
		require.NoError(t, err)
		require.NotNil(t, feat)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func expectUserBuyDBTail(mock sqlmock.Sqlmock) {
	mock.ExpectExec("INSERT INTO trades").
		WithArgs(uint64(1), uint64(2), uint64(5), 100.0, 100.0).
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectExec("INSERT INTO comissions").
		WithArgs(uint64(7), 10.0, 10.0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE features SET owner_id").
		WithArgs(uint64(2), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("SET rgb = \\?, owner = \\?, label").
		WithArgs(constants.ChangeStatusToSoldAndNotPriced("m"), "buyer", "", constants.DefaultPublicPricingLimit, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT withdraw_profit FROM user_variables").
		WithArgs(uint64(2)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, user_id, feature_id, asset, amount").
		WithArgs(uint64(1), uint64(5)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, asset FROM feature_hourly_profits WHERE feature_id = \\? AND user_id").
		WithArgs(uint64(1), uint64(5)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, asset FROM feature_hourly_profits WHERE feature_id = \\? ORDER BY").
		WithArgs(uint64(1)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO feature_hourly_profits").
		WithArgs(uint64(2), uint64(1), "yellow", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM feature_hourly_profits WHERE feature_id").
		WithArgs(uint64(1), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SET deleted_at = NOW\\(\\) WHERE feature_id").
		WithArgs(uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SET status = 1, updated_at = NOW\\(\\) WHERE feature_id").
		WithArgs(uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestMarketplaceService_CreateSellRequest_Under18AndZeroStability(t *testing.T) {
	t.Run("under18 percentage below floor", func(t *testing.T) {
		svc, mock := newMarketplaceSQLMock(t)
		expectFeatureFindByID(mock, 3, "m", 1000, 80)
		mock.ExpectQuery("FROM system_variables").
			WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("80", "110"))
		mock.ExpectQuery("SELECT birthdate FROM kycs").
			WithArgs(uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"birthdate"}).AddRow(time.Now().AddDate(-10, 0, 0)))
		_, err := svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
			SellerId: 3, FeatureId: 1, MinimumPricePercentage: 90,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "110")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("explicit prices below public floor", func(t *testing.T) {
		svc, mock := newMarketplaceSQLMock(t)
		expectFeatureFindByID(mock, 3, "m", 1000, 80)
		mock.ExpectQuery("FROM system_variables").
			WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("80", "110"))
		mock.ExpectQuery("SELECT birthdate FROM kycs").
			WithArgs(uint64(3)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT value FROM variables").
			WithArgs("psc").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.0))
		mock.ExpectQuery("SELECT value FROM variables").
			WithArgs("yellow").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.0))
		_, err := svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
			SellerId: 3, FeatureId: 1, PricePsc: "10", PriceIrr: "10",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "80")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("zero stability uses 100 percent", func(t *testing.T) {
		svc, mock := newMarketplaceSQLMock(t)
		expectFeatureFindByID(mock, 3, "m", 0, 80)
		mock.ExpectQuery("FROM system_variables").
			WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("80", "110"))
		mock.ExpectQuery("SELECT birthdate FROM kycs").
			WithArgs(uint64(3)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT value FROM variables").
			WithArgs("psc").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.0))
		mock.ExpectQuery("SELECT value FROM variables").
			WithArgs("yellow").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1.0))
		mock.ExpectExec("INSERT INTO sell_feature_requests").
			WillReturnResult(sqlmock.NewResult(8, 1))
		mock.ExpectExec("UPDATE feature_properties SET").
			WillReturnResult(sqlmock.NewResult(0, 1))
		now := time.Now()
		mock.ExpectQuery("FROM sell_feature_requests").
			WithArgs(uint64(8)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "seller_id", "feature_id", "price_psc", "price_irr", "limit", "status", "created_at", "updated_at",
			}).AddRow(8, 3, 1, 1.0, 1.0, 100, 0, now, now))
		out, err := svc.CreateSellRequest(context.Background(), &pb.CreateSellRequestRequest{
			SellerId: 3, FeatureId: 1, PricePsc: "1", PriceIrr: "1",
		})
		require.NoError(t, err)
		assert.Equal(t, uint64(8), out.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
