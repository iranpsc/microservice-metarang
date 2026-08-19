package service

import (
	"context"
	"fmt"
	"time"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/repository"
	periodpkg "metarang/shared/pkg/period"
)

// WalletHistoryService orchestrates privacy, period resolution, and income/spending aggregation.
type WalletHistoryService struct {
	repo     repository.WalletHistoryRepository
	income   *IncomeCalculator
	spending *SpendingCalculator
	now      func() time.Time
}

func NewWalletHistoryService(
	repo repository.WalletHistoryRepository,
	income *IncomeCalculator,
	spending *SpendingCalculator,
) *WalletHistoryService {
	return &WalletHistoryService{
		repo:     repo,
		income:   income,
		spending: spending,
		now:      time.Now,
	}
}

// WalletHistorySummaryResult is the service-layer summary payload.
type WalletHistorySummaryResult struct {
	Cards  []models.WalletHistorySummaryCard
	Period string
}

// WalletHistoryChartResult is the service-layer chart payload.
type WalletHistoryChartResult struct {
	Charts map[string]models.WalletAssetChart
	Period string
}

func (s *WalletHistoryService) GetSummary(
	ctx context.Context,
	userID uint64,
	periodStr string,
	assets []string,
	privacy map[string]int32,
) (*WalletHistorySummaryResult, error) {
	registeredAt, err := s.repo.GetUserCreatedAt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user registration date: %w", err)
	}

	now := s.now()
	window, err := periodpkg.ResolvePeriod(periodStr, now, registeredAt)
	if err != nil {
		return nil, err
	}
	previous, err := periodpkg.ResolvePrevious(periodStr, now, registeredAt)
	if err != nil {
		return nil, err
	}

	if len(assets) == 0 {
		assets = models.AllWalletAssets
	}

	balance, err := s.repo.GetCurrentBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("current balance: %w", err)
	}

	cards := make([]models.WalletHistorySummaryCard, 0, len(assets))
	for _, asset := range assets {
		if !IsAssetVisible(privacy, asset) {
			cards = append(cards, models.WalletHistorySummaryCard{
				Asset:             asset,
				PrivacyRestricted: true,
			})
			continue
		}

		periodIncome, err := s.income.CalcIncome(ctx, userID, asset, window.Start, window.End)
		if err != nil {
			return nil, err
		}
		periodSpending, err := s.spending.CalcSpending(ctx, userID, asset, window.Start, window.End)
		if err != nil {
			return nil, err
		}
		previousIncome, err := s.income.CalcIncome(ctx, userID, asset, previous.Start, previous.End)
		if err != nil {
			return nil, err
		}

		netChange := periodIncome - periodSpending
		growth := calculateGrowthPercent(netChange, previousIncome)
		direction := "down"
		if growth >= 0 {
			direction = "up"
		}

		cards = append(cards, models.WalletHistorySummaryCard{
			Asset:             asset,
			CurrentBalance:    round2(balance.BalanceFor(asset)),
			PeriodIncome:      round2(periodIncome),
			PeriodSpending:    round2(periodSpending),
			GrowthPercent:     growth,
			Direction:         direction,
			PrivacyRestricted: false,
		})
	}

	return &WalletHistorySummaryResult{Cards: cards, Period: periodStr}, nil
}

func (s *WalletHistoryService) GetChart(
	ctx context.Context,
	userID uint64,
	periodStr string,
	assets []string,
	privacy map[string]int32,
) (*WalletHistoryChartResult, error) {
	registeredAt, err := s.repo.GetUserCreatedAt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user registration date: %w", err)
	}

	window, err := periodpkg.ResolvePeriod(periodStr, s.now(), registeredAt)
	if err != nil {
		return nil, err
	}

	if len(assets) == 0 {
		assets = models.AllWalletAssets
	}

	charts := make(map[string]models.WalletAssetChart)
	for _, asset := range assets {
		if !IsAssetVisible(privacy, asset) {
			continue
		}
		incomeBuckets, err := s.income.CalcIncomeBuckets(ctx, userID, asset, window.Buckets)
		if err != nil {
			return nil, err
		}
		spendingBuckets, err := s.spending.CalcSpendingBuckets(ctx, userID, asset, window.Buckets)
		if err != nil {
			return nil, err
		}
		charts[asset] = models.WalletAssetChart{
			Income:   toChartPoints(incomeBuckets),
			Spending: toChartPoints(spendingBuckets),
		}
	}

	return &WalletHistoryChartResult{Charts: charts, Period: periodStr}, nil
}

func calculateGrowthPercent(netChange, previousIncome float64) float64 {
	if previousIncome <= 0 {
		if netChange > 0 {
			return 100
		}
		return 0
	}
	return round2((netChange / previousIncome) * 100)
}

func toChartPoints(buckets []BucketAmount) []models.WalletChartPoint {
	out := make([]models.WalletChartPoint, len(buckets))
	for i, b := range buckets {
		out[i] = models.WalletChartPoint{Label: b.Label, Amount: b.Amount}
	}
	return out
}
