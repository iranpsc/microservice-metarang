package service_test

import (
	"context"
	"testing"
	"time"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/service"
	"metarang/shared/pkg/period"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWalletHistoryRepo struct {
	deposits        float64
	withdrawals     float64
	hourly          float64
	tradeSells      float64
	tradeBuys       float64
	referrals       float64
	firstOrders     float64
	levelRewards    float64
	featurePurchase float64
	building        float64
	balance         *models.WalletBalance
	pscRate         float64
}

func (m *mockWalletHistoryRepo) SumDeposits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.deposits, nil
}
func (m *mockWalletHistoryRepo) SumWithdrawals(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.withdrawals, nil
}
func (m *mockWalletHistoryRepo) SumHourlyProfits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.hourly, nil
}
func (m *mockWalletHistoryRepo) SumTradeSells(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.tradeSells, nil
}
func (m *mockWalletHistoryRepo) SumTradeBuys(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.tradeBuys, nil
}
func (m *mockWalletHistoryRepo) SumReferralBonuses(ctx context.Context, userID uint64, start, end time.Time) (float64, error) {
	return m.referrals, nil
}
func (m *mockWalletHistoryRepo) SumFirstOrderBonuses(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.firstOrders, nil
}
func (m *mockWalletHistoryRepo) SumLevelRewards(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.levelRewards, nil
}
func (m *mockWalletHistoryRepo) SumFeaturePurchaseWithdrawals(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return m.featurePurchase, nil
}
func (m *mockWalletHistoryRepo) SumBuildingSatisfaction(ctx context.Context, userID uint64, start, end time.Time) (float64, error) {
	return m.building, nil
}
func (m *mockWalletHistoryRepo) GetCurrentBalance(ctx context.Context, userID uint64) (*models.WalletBalance, error) {
	if m.balance == nil {
		return &models.WalletBalance{}, nil
	}
	return m.balance, nil
}
func (m *mockWalletHistoryRepo) GetUserCreatedAt(ctx context.Context, userID uint64) (time.Time, error) {
	return time.Time{}, nil
}
func (m *mockWalletHistoryRepo) GetPSCRate(ctx context.Context) (float64, error) {
	if m.pscRate == 0 {
		return 1, nil
	}
	return m.pscRate, nil
}

func TestIncomeCalculator_SumsAllSourcesForPSC(t *testing.T) {
	repo := &mockWalletHistoryRepo{
		deposits: 10, hourly: 2, tradeSells: 5, referrals: 3, firstOrders: 4, levelRewards: 1,
	}
	calc := service.NewIncomeCalculator(repo)
	total, err := calc.CalcIncome(context.Background(), 1, "psc", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 25.0, total)
}

func TestIncomeCalculator_SkipsReferralForNonPSC(t *testing.T) {
	repo := &mockWalletHistoryRepo{deposits: 10, referrals: 99, tradeSells: 5}
	calc := service.NewIncomeCalculator(repo)
	total, err := calc.CalcIncome(context.Background(), 1, "irr", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 15.0, total)
}

func TestSpendingCalculator_PSC(t *testing.T) {
	repo := &mockWalletHistoryRepo{tradeBuys: 7, featurePurchase: 3, withdrawals: 100, building: 50}
	calc := service.NewSpendingCalculator(repo)
	total, err := calc.CalcSpending(context.Background(), 1, "psc", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 10.0, total)
}

func TestSpendingCalculator_ColorWithdraw(t *testing.T) {
	repo := &mockWalletHistoryRepo{withdrawals: 12, tradeBuys: 5}
	calc := service.NewSpendingCalculator(repo)
	total, err := calc.CalcSpending(context.Background(), 1, "red", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 12.0, total)
}

func TestSpendingCalculator_SatisfactionBuilding(t *testing.T) {
	repo := &mockWalletHistoryRepo{building: 40, withdrawals: 9}
	calc := service.NewSpendingCalculator(repo)
	total, err := calc.CalcSpending(context.Background(), 1, "satisfaction", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 40.0, total)
}

func TestIncomeCalculator_Buckets(t *testing.T) {
	repo := &mockWalletHistoryRepo{deposits: 1.234}
	calc := service.NewIncomeCalculator(repo)
	window, err := period.ResolvePeriod("weekly", time.Date(2026, 5, 15, 12, 0, 0, 0, time.Local), time.Time{})
	require.NoError(t, err)
	buckets, err := calc.CalcIncomeBuckets(context.Background(), 1, "psc", window.Buckets)
	require.NoError(t, err)
	require.Len(t, buckets, 4)
	assert.Equal(t, 1.23, buckets[0].Amount)
}

func TestSpendingCalculator_Buckets(t *testing.T) {
	repo := &mockWalletHistoryRepo{tradeBuys: 2.5}
	calc := service.NewSpendingCalculator(repo)
	window, err := period.ResolvePeriod("weekly", time.Date(2026, 5, 15, 12, 0, 0, 0, time.Local), time.Time{})
	require.NoError(t, err)
	buckets, err := calc.CalcSpendingBuckets(context.Background(), 1, "psc", window.Buckets)
	require.NoError(t, err)
	require.Len(t, buckets, 4)
	assert.Equal(t, 2.5, buckets[0].Amount)
}

func TestIncomeCalculator_PropagatesRepoError(t *testing.T) {
	repo := &errRepo{failOn: "deposits"}
	calc := service.NewIncomeCalculator(repo)
	_, err := calc.CalcIncome(context.Background(), 1, "psc", time.Now().Add(-time.Hour), time.Now())
	require.Error(t, err)
}

func TestSpendingCalculator_PropagatesRepoError(t *testing.T) {
	repo := &errRepo{failOn: "tradeBuys"}
	calc := service.NewSpendingCalculator(repo)
	_, err := calc.CalcSpending(context.Background(), 1, "psc", time.Now().Add(-time.Hour), time.Now())
	require.Error(t, err)
}

func TestIncomeCalculator_BucketsPropagatesError(t *testing.T) {
	repo := &errRepo{failOn: "deposits"}
	calc := service.NewIncomeCalculator(repo)
	window, err := period.ResolvePeriod("daily", time.Date(2026, 5, 15, 12, 0, 0, 0, time.Local), time.Time{})
	require.NoError(t, err)
	_, err = calc.CalcIncomeBuckets(context.Background(), 1, "psc", window.Buckets)
	require.Error(t, err)
}

func TestSpendingCalculator_BucketsPropagatesError(t *testing.T) {
	repo := &errRepo{failOn: "tradeBuys"}
	calc := service.NewSpendingCalculator(repo)
	window, err := period.ResolvePeriod("daily", time.Date(2026, 5, 15, 12, 0, 0, 0, time.Local), time.Time{})
	require.NoError(t, err)
	_, err = calc.CalcSpendingBuckets(context.Background(), 1, "psc", window.Buckets)
	require.Error(t, err)
}

type errRepo struct {
	mockWalletHistoryRepo
	failOn string
}

func (m *errRepo) SumDeposits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	if m.failOn == "deposits" {
		return 0, assert.AnError
	}
	return 0, nil
}

func (m *errRepo) SumTradeBuys(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	if m.failOn == "tradeBuys" {
		return 0, assert.AnError
	}
	return 0, nil
}
