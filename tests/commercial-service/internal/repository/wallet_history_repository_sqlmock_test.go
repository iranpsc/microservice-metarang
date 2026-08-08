package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"metarang/commercial-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRepo(t *testing.T) (repository.WalletHistoryRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewWalletHistoryRepository(db), mock
}

func TestWalletHistoryRepository_SumDeposits(t *testing.T) {
	repo, mock := newMockRepo(t)
	start, end := time.Now().Add(-time.Hour), time.Now()
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount\\), 0\\)").
		WithArgs(uint64(1), "psc", "deposit", start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(12.5))

	total, err := repo.SumDeposits(context.Background(), 1, "psc", start, end)
	require.NoError(t, err)
	assert.Equal(t, 12.5, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletHistoryRepository_SumWithdrawalsAndFeaturePurchases(t *testing.T) {
	repo, mock := newMockRepo(t)
	start, end := time.Now().Add(-time.Hour), time.Now()

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount\\), 0\\)").
		WithArgs(uint64(1), "red", "withdraw", start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(3.0))
	total, err := repo.SumWithdrawals(context.Background(), 1, "red", start, end)
	require.NoError(t, err)
	assert.Equal(t, 3.0, total)

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount\\), 0\\)").
		WithArgs(uint64(1), "psc", "withdraw", `App\Models\BuyFeatureRequest`, start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(8.0))
	total, err = repo.SumFeaturePurchaseWithdrawals(context.Background(), 1, "psc", start, end)
	require.NoError(t, err)
	assert.Equal(t, 8.0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletHistoryRepository_HourlyTradesReferrals(t *testing.T) {
	repo, mock := newMockRepo(t)
	start, end := time.Now().Add(-time.Hour), time.Now()

	mock.ExpectQuery("FROM feature_hourly_profits").
		WithArgs(uint64(1), "blue", start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(1.5))
	total, err := repo.SumHourlyProfits(context.Background(), 1, "blue", start, end)
	require.NoError(t, err)
	assert.Equal(t, 1.5, total)

	mock.ExpectQuery("FROM trades").
		WithArgs(uint64(1), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(4.0))
	total, err = repo.SumTradeSells(context.Background(), 1, "psc", start, end)
	require.NoError(t, err)
	assert.Equal(t, 4.0, total)

	total, err = repo.SumTradeBuys(context.Background(), 1, "red", start, end)
	require.NoError(t, err)
	assert.Equal(t, 0.0, total)

	mock.ExpectQuery("FROM trades").
		WithArgs(uint64(1), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(2.0))
	total, err = repo.SumTradeBuys(context.Background(), 1, "irr", start, end)
	require.NoError(t, err)
	assert.Equal(t, 2.0, total)

	mock.ExpectQuery("FROM referral_order_histories").
		WithArgs(uint64(1), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(9.0))
	total, err = repo.SumReferralBonuses(context.Background(), 1, start, end)
	require.NoError(t, err)
	assert.Equal(t, 9.0, total)

	mock.ExpectQuery("FROM first_orders").
		WithArgs(uint64(1), "psc", start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(6.0))
	total, err = repo.SumFirstOrderBonuses(context.Background(), 1, "psc", start, end)
	require.NoError(t, err)
	assert.Equal(t, 6.0, total)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletHistoryRepository_LevelRewardsAndBuilding(t *testing.T) {
	repo, mock := newMockRepo(t)
	start, end := time.Now().Add(-time.Hour), time.Now()

	mock.ExpectQuery("FROM recieved_level_prizes").
		WithArgs(uint64(1), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(100.0))
	mock.ExpectQuery("FROM variables").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(10.0))
	total, err := repo.SumLevelRewards(context.Background(), 1, "psc", start, end)
	require.NoError(t, err)
	assert.Equal(t, 10.0, total)

	total, err = repo.SumLevelRewards(context.Background(), 1, "irr", start, end)
	require.NoError(t, err)
	assert.Equal(t, 0.0, total)

	mock.ExpectQuery("FROM recieved_level_prizes").
		WithArgs(uint64(1), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(7.0))
	total, err = repo.SumLevelRewards(context.Background(), 1, "effect", start, end)
	require.NoError(t, err)
	assert.Equal(t, 7.0, total)

	mock.ExpectQuery("FROM buildings").
		WithArgs(uint64(1), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(55.0))
	total, err = repo.SumBuildingSatisfaction(context.Background(), 1, start, end)
	require.NoError(t, err)
	assert.Equal(t, 55.0, total)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletHistoryRepository_QueryErrors(t *testing.T) {
	repo, mock := newMockRepo(t)
	start, end := time.Now().Add(-time.Hour), time.Now()

	mock.ExpectQuery("FROM feature_hourly_profits").
		WithArgs(uint64(1), "blue", start, end).
		WillReturnError(assert.AnError)
	_, err := repo.SumHourlyProfits(context.Background(), 1, "blue", start, end)
	require.Error(t, err)

	mock.ExpectQuery("FROM trades").
		WithArgs(uint64(1), start, end).
		WillReturnError(assert.AnError)
	_, err = repo.SumTradeSells(context.Background(), 1, "psc", start, end)
	require.Error(t, err)

	mock.ExpectQuery("FROM referral_order_histories").
		WithArgs(uint64(1), start, end).
		WillReturnError(assert.AnError)
	_, err = repo.SumReferralBonuses(context.Background(), 1, start, end)
	require.Error(t, err)

	mock.ExpectQuery("FROM first_orders").
		WithArgs(uint64(1), "psc", start, end).
		WillReturnError(assert.AnError)
	_, err = repo.SumFirstOrderBonuses(context.Background(), 1, "psc", start, end)
	require.Error(t, err)

	mock.ExpectQuery("FROM buildings").
		WithArgs(uint64(1), start, end).
		WillReturnError(assert.AnError)
	_, err = repo.SumBuildingSatisfaction(context.Background(), 1, start, end)
	require.Error(t, err)

	mock.ExpectQuery("FROM variables").
		WillReturnError(assert.AnError)
	_, err = repo.GetPSCRate(context.Background())
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletHistoryRepository_GetCurrentBalanceAndRate(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery("FROM wallets").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"psc", "irr", "red", "blue", "yellow", "satisfaction", "effect",
		}).AddRow(1, 2, 3, 4, 5, 6, 7))
	bal, err := repo.GetCurrentBalance(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1.0, bal.PSC)
	assert.Equal(t, 7.0, bal.Effect)

	repo2, mock2 := newMockRepo(t)
	mock2.ExpectQuery("FROM wallets").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"psc", "irr", "red", "blue", "yellow", "satisfaction", "effect",
		}))
	bal, err = repo2.GetCurrentBalance(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 0.0, bal.PSC)

	mock.ExpectQuery("FROM variables").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(0.0))
	rate, err := repo.GetPSCRate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1.0, rate)

	mock.ExpectQuery("FROM variables").
		WillReturnError(sql.ErrNoRows)
	rate, err = repo.GetPSCRate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1.0, rate)

	mock.ExpectQuery("FROM variables").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(1.25))
	rate, err = repo.GetPSCRate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1.25, rate)

	require.NoError(t, mock.ExpectationsWereMet())
	require.NoError(t, mock2.ExpectationsWereMet())
}

func TestWalletHistoryRepository_GetUserCreatedAt(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	mock.ExpectQuery("SELECT created_at FROM users").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))
	got, err := repo.GetUserCreatedAt(context.Background(), 1)
	require.NoError(t, err)
	assert.WithinDuration(t, now, got, time.Second)

	mock.ExpectQuery("SELECT created_at FROM users").
		WithArgs(uint64(99)).
		WillReturnError(sql.ErrNoRows)
	got, err = repo.GetUserCreatedAt(context.Background(), 99)
	require.NoError(t, err)
	assert.True(t, got.IsZero())

	mock.ExpectQuery("SELECT created_at FROM users").
		WithArgs(uint64(2)).
		WillReturnError(assert.AnError)
	_, err = repo.GetUserCreatedAt(context.Background(), 2)
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
