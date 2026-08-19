package service_test

import (
	"context"
	"testing"
	"time"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletHistoryService_GetSummary_PrivacyRestricted(t *testing.T) {
	repo := &mockWalletHistoryRepo{
		deposits: 10,
		balance:  &models.WalletBalance{PSC: 100, IRR: 200},
	}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)

	result, err := svc.GetSummary(context.Background(), 1, "daily", []string{"psc", "irr"}, map[string]int32{
		"psc_transactions": 1,
		"irr_transactions": 0,
	})
	require.NoError(t, err)
	require.Len(t, result.Cards, 2)
	assert.Equal(t, "psc", result.Cards[0].Asset)
	assert.False(t, result.Cards[0].PrivacyRestricted)
	assert.Equal(t, "irr", result.Cards[1].Asset)
	assert.True(t, result.Cards[1].PrivacyRestricted)
}

func TestWalletHistoryService_GetSummary_CurrentBalanceIndependentOfPeriod(t *testing.T) {
	repo := &mockWalletHistoryRepo{
		deposits: 10,
		balance:  &models.WalletBalance{PSC: 250, IRR: 500},
	}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)

	for _, period := range []string{"daily", "weekly", "monthly", "yearly"} {
		result, err := svc.GetSummary(context.Background(), 1, period, []string{"psc", "irr"}, nil)
		require.NoError(t, err)
		require.Len(t, result.Cards, 2)
		assert.Equal(t, 250.0, result.Cards[0].CurrentBalance, "psc balance for period %s", period)
		assert.Equal(t, 500.0, result.Cards[1].CurrentBalance, "irr balance for period %s", period)
	}
}

func TestWalletHistoryService_GetSummary_Growth(t *testing.T) {
	repo := &mockWalletHistoryRepo{
		deposits:  50,
		tradeBuys: 10,
		balance:   &models.WalletBalance{PSC: 99},
	}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetSummary(context.Background(), 1, "daily", []string{"psc"}, nil)
	require.NoError(t, err)
	require.Len(t, result.Cards, 1)
	assert.Equal(t, 99.0, result.Cards[0].CurrentBalance)
	assert.Equal(t, 50.0, result.Cards[0].PeriodIncome)
	assert.Equal(t, 10.0, result.Cards[0].PeriodSpending)
	// previousIncome also returns deposits=50 so growth = (40/50)*100 = 80
	assert.Equal(t, 80.0, result.Cards[0].GrowthPercent)
	assert.Equal(t, "up", result.Cards[0].Direction)
}

func TestWalletHistoryService_GetChart_OmitsRestricted(t *testing.T) {
	repo := &mockWalletHistoryRepo{deposits: 1}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetChart(context.Background(), 1, "weekly", []string{"psc", "irr"}, map[string]int32{
		"psc_transactions": 1,
		"irr_transactions": 0,
	})
	require.NoError(t, err)
	_, hasPSC := result.Charts["psc"]
	_, hasIRR := result.Charts["irr"]
	assert.True(t, hasPSC)
	assert.False(t, hasIRR)
	assert.Len(t, result.Charts["psc"].Income, 4)
}

func TestWalletHistoryService_GetSummary_GrowthWhenPreviousZero(t *testing.T) {
	repo := &mockWalletHistoryRepo{
		deposits:  0,
		tradeBuys: 0,
		balance:   &models.WalletBalance{PSC: 1},
	}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetSummary(context.Background(), 1, "daily", []string{"psc"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Cards[0].GrowthPercent)
	assert.Equal(t, "up", result.Cards[0].Direction)
}

func TestWalletHistoryService_GetSummary_PositiveNetWithZeroPrevious(t *testing.T) {
	repo := &countingIncomeRepo{
		mockWalletHistoryRepo: mockWalletHistoryRepo{balance: &models.WalletBalance{PSC: 5}},
		incomeValues:          []float64{20, 0},
	}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetSummary(context.Background(), 1, "daily", []string{"psc"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 100.0, result.Cards[0].GrowthPercent)
}

func TestWalletHistoryService_GetSummary_DefaultsAllAssets(t *testing.T) {
	repo := &mockWalletHistoryRepo{balance: &models.WalletBalance{}}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetSummary(context.Background(), 1, "daily", nil, nil)
	require.NoError(t, err)
	assert.Len(t, result.Cards, len(models.AllWalletAssets))
}

func TestWalletHistoryService_GetChart_DefaultsAllVisible(t *testing.T) {
	repo := &mockWalletHistoryRepo{}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetChart(context.Background(), 1, "daily", nil, nil)
	require.NoError(t, err)
	assert.Len(t, result.Charts, len(models.AllWalletAssets))
}

type countingIncomeRepo struct {
	mockWalletHistoryRepo
	incomeCalls  int
	incomeValues []float64
}

func (m *countingIncomeRepo) SumDeposits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	if m.incomeCalls < len(m.incomeValues) {
		v := m.incomeValues[m.incomeCalls]
		m.incomeCalls++
		return v, nil
	}
	return 0, nil
}

func TestWalletHistoryService_GetSummary_BalanceError(t *testing.T) {
	repo := &errBalanceRepo{}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	_, err := svc.GetSummary(context.Background(), 1, "daily", []string{"psc"}, nil)
	require.Error(t, err)
}

func TestWalletHistoryService_GetChart_IncomeError(t *testing.T) {
	repo := &errRepo{failOn: "deposits"}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	_, err := svc.GetChart(context.Background(), 1, "daily", []string{"psc"}, nil)
	require.Error(t, err)
}

func TestWalletHistoryService_GetSummary_DownDirection(t *testing.T) {
	repo := &countingIncomeRepo{
		mockWalletHistoryRepo: mockWalletHistoryRepo{tradeBuys: 30, balance: &models.WalletBalance{PSC: 1}},
		incomeValues:          []float64{10, 50},
	}
	svc := service.NewWalletHistoryService(
		repo,
		service.NewIncomeCalculator(repo),
		service.NewSpendingCalculator(repo),
	)
	result, err := svc.GetSummary(context.Background(), 1, "daily", []string{"psc"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "down", result.Cards[0].Direction)
}

type errBalanceRepo struct {
	mockWalletHistoryRepo
}

func (m *errBalanceRepo) GetCurrentBalance(ctx context.Context, userID uint64) (*models.WalletBalance, error) {
	return nil, assert.AnError
}
