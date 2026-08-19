package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type UserVariableRepository struct {
	db *sql.DB
}

func NewUserVariableRepository(db *sql.DB) *UserVariableRepository {
	return &UserVariableRepository{db: db}
}

// ApplyDynastyPrizeMultipliers adjusts user_variables prize fields:
// referral_profit += referral_profit * introduction_profit_increase;
// data_storage += data_storage * data_storage (prize multiplier);
// withdraw_profit += withdraw_profit * accumulated_capital_reserve.
func (r *UserVariableRepository) ApplyDynastyPrizeMultipliers(
	ctx context.Context,
	tx *sql.Tx,
	userID uint64,
	introductionProfitIncrease float64,
	dataStorageMultiplier float64,
	withdrawProfitMultiplier float64,
) error {
	const q = `
		UPDATE user_variables SET
			referral_profit = referral_profit + FLOOR(referral_profit * ?),
			data_storage = data_storage + FLOOR(data_storage * ?),
			withdraw_profit = withdraw_profit + (withdraw_profit * ?),
			updated_at = NOW()
		WHERE user_id = ?
	`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, q,
			introductionProfitIncrease,
			dataStorageMultiplier,
			withdrawProfitMultiplier,
			userID,
		)
	} else {
		_, err = r.db.ExecContext(ctx, q,
			introductionProfitIncrease,
			dataStorageMultiplier,
			withdrawProfitMultiplier,
			userID,
		)
	}
	if err != nil {
		return fmt.Errorf("update user_variables for prize: %w", err)
	}
	return nil
}
