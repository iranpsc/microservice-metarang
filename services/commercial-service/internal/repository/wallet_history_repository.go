package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metarang/commercial-service/internal/models"
)

const buyFeaturePayableType = `App\Models\BuyFeatureRequest`

// WalletHistoryRepository aggregates wallet income/spending from financial tables.
type WalletHistoryRepository interface {
	SumDeposits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumWithdrawals(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumHourlyProfits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumTradeSells(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumTradeBuys(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumReferralBonuses(ctx context.Context, userID uint64, start, end time.Time) (float64, error)
	SumFirstOrderBonuses(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumLevelRewards(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumFeaturePurchaseWithdrawals(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error)
	SumBuildingSatisfaction(ctx context.Context, userID uint64, start, end time.Time) (float64, error)
	GetCurrentBalance(ctx context.Context, userID uint64) (*models.WalletBalance, error)
	GetUserCreatedAt(ctx context.Context, userID uint64) (time.Time, error)
	GetPSCRate(ctx context.Context) (float64, error)
}

type walletHistoryRepository struct {
	db *sql.DB
}

// NewWalletHistoryRepository creates a MySQL-backed wallet history repository.
func NewWalletHistoryRepository(db *sql.DB) WalletHistoryRepository {
	return &walletHistoryRepository{db: db}
}

func (r *walletHistoryRepository) SumDeposits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return r.sumTransactionAmount(ctx, userID, asset, "deposit", "", true, start, end)
}

func (r *walletHistoryRepository) SumWithdrawals(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return r.sumTransactionAmount(ctx, userID, asset, "withdraw", "", false, start, end)
}

func (r *walletHistoryRepository) SumFeaturePurchaseWithdrawals(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	return r.sumTransactionAmount(ctx, userID, asset, "withdraw", buyFeaturePayableType, false, start, end)
}

func (r *walletHistoryRepository) sumTransactionAmount(
	ctx context.Context,
	userID uint64,
	asset, action, payableType string,
	requireCompleted bool,
	start, end time.Time,
) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = ?
		  AND asset = ?
		  AND action = ?
		  AND created_at BETWEEN ? AND ?
	`
	args := []interface{}{userID, asset, action, start, end}

	if requireCompleted {
		query = `
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE user_id = ?
			  AND asset = ?
			  AND action = ?
			  AND status = 1
			  AND created_at BETWEEN ? AND ?
		`
		args = []interface{}{userID, asset, action, start, end}
	}

	if payableType != "" {
		query = `
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE user_id = ?
			  AND asset = ?
			  AND action = ?
			  AND payable_type = ?
			  AND created_at BETWEEN ? AND ?
		`
		args = []interface{}{userID, asset, action, payableType, start, end}
	}

	var total float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum transactions: %w", err)
	}
	return total, nil
}

func (r *walletHistoryRepository) SumHourlyProfits(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM feature_hourly_profits
		WHERE user_id = ?
		  AND asset = ?
		  AND is_active = 1
		  AND updated_at BETWEEN ? AND ?
	`
	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, asset, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum hourly profits: %w", err)
	}
	return total, nil
}

func (r *walletHistoryRepository) SumTradeSells(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	column, ok := tradeAmountColumn(asset)
	if !ok {
		return 0, nil
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(%s), 0)
		FROM trades
		WHERE seller_id = ?
		  AND created_at BETWEEN ? AND ?
	`, column)
	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum trade sells: %w", err)
	}
	return total, nil
}

func (r *walletHistoryRepository) SumTradeBuys(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	column, ok := tradeAmountColumn(asset)
	if !ok {
		return 0, nil
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(%s), 0)
		FROM trades
		WHERE buyer_id = ?
		  AND created_at BETWEEN ? AND ?
	`, column)
	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum trade buys: %w", err)
	}
	return total, nil
}

func tradeAmountColumn(asset string) (string, bool) {
	switch asset {
	case "psc":
		return "psc_amount", true
	case "irr":
		return "irr_amount", true
	default:
		return "", false
	}
}

func (r *walletHistoryRepository) SumReferralBonuses(ctx context.Context, userID uint64, start, end time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM referral_order_histories
		WHERE user_id = ?
		  AND created_at BETWEEN ? AND ?
	`
	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum referral bonuses: %w", err)
	}
	return total, nil
}

func (r *walletHistoryRepository) SumFirstOrderBonuses(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(bonus), 0)
		FROM first_orders
		WHERE user_id = ?
		  AND type = ?
		  AND created_at BETWEEN ? AND ?
	`
	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, asset, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum first order bonuses: %w", err)
	}
	return total, nil
}

func (r *walletHistoryRepository) SumLevelRewards(ctx context.Context, userID uint64, asset string, start, end time.Time) (float64, error) {
	column, ok := levelPrizeColumn(asset)
	if !ok {
		return 0, nil
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(level_prizes.%s), 0)
		FROM recieved_level_prizes
		INNER JOIN level_prizes ON level_prizes.id = recieved_level_prizes.level_prize_id
		WHERE recieved_level_prizes.user_id = ?
		  AND recieved_level_prizes.created_at BETWEEN ? AND ?
	`, column)

	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum level rewards: %w", err)
	}

	if asset == "psc" {
		rate, err := r.GetPSCRate(ctx)
		if err != nil {
			return 0, err
		}
		if rate <= 0 {
			rate = 1
		}
		return total / rate, nil
	}
	return total, nil
}

func levelPrizeColumn(asset string) (string, bool) {
	switch asset {
	case "psc", "blue", "red", "yellow", "satisfaction", "effect":
		return asset, true
	default:
		return "", false
	}
}

func (r *walletHistoryRepository) SumBuildingSatisfaction(ctx context.Context, userID uint64, start, end time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(launched_satisfaction), 0)
		FROM buildings
		WHERE user_id = ?
		  AND construction_start_date BETWEEN ? AND ?
	`
	var total float64
	if err := r.db.QueryRowContext(ctx, query, userID, start, end).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum building satisfaction: %w", err)
	}
	return total, nil
}

func (r *walletHistoryRepository) GetUserCreatedAt(ctx context.Context, userID uint64) (time.Time, error) {
	query := `SELECT created_at FROM users WHERE id = ? LIMIT 1`
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&createdAt)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get user created_at: %w", err)
	}
	return createdAt, nil
}

func (r *walletHistoryRepository) GetCurrentBalance(ctx context.Context, userID uint64) (*models.WalletBalance, error) {
	query := `
		SELECT COALESCE(psc, 0), COALESCE(irr, 0), COALESCE(red, 0), COALESCE(blue, 0),
		       COALESCE(yellow, 0), COALESCE(satisfaction, 0), COALESCE(effect, 0)
		FROM wallets
		WHERE user_id = ?
		LIMIT 1
	`
	bal := &models.WalletBalance{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&bal.PSC, &bal.IRR, &bal.Red, &bal.Blue, &bal.Yellow, &bal.Satisfaction, &bal.Effect,
	)
	if err == sql.ErrNoRows {
		return &models.WalletBalance{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get wallet balance: %w", err)
	}
	return bal, nil
}

func (r *walletHistoryRepository) GetPSCRate(ctx context.Context) (float64, error) {
	query := `SELECT price FROM variables WHERE asset = 'psc' LIMIT 1`
	var price float64
	err := r.db.QueryRowContext(ctx, query).Scan(&price)
	if err == sql.ErrNoRows {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get psc rate: %w", err)
	}
	if price <= 0 {
		return 1, nil
	}
	return price, nil
}
