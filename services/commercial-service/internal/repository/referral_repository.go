package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/commercial-service/internal/models"
)

type ReferralRepository interface {
	GetReferrerID(ctx context.Context, userID uint64) (*uint64, error)
	GetTotalReferredAmount(ctx context.Context, referrerID uint64) (float64, error)
	CreateReferralOrder(ctx context.Context, history *models.ReferralOrderHistory) error
}

type referralRepository struct {
	db *sql.DB
}

func NewReferralRepository(db *sql.DB) ReferralRepository {
	return &referralRepository{db: db}
}

// GetReferrerID gets the user's referrer ID (who referred them)
// Laravel: $user->referred->id (user who referred this user)
func (r *referralRepository) GetReferrerID(ctx context.Context, userID uint64) (*uint64, error) {
	query := `
		SELECT referrer_id
		FROM users
		WHERE id = ?
		LIMIT 1
	`
	
	var referrerID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&referrerID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get referrer ID: %w", err)
	}

	if !referrerID.Valid {
		return nil, nil
	}

	id := uint64(referrerID.Int64)
	return &id, nil
}

// GetTotalReferredAmount calculates total referral amount for a referrer
// Laravel: $referred->referalOrders()->sum('amount')
func (r *referralRepository) GetTotalReferredAmount(ctx context.Context, referrerID uint64) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM referral_order_histories
		WHERE user_id = ?
	`
	
	var total float64
	err := r.db.QueryRowContext(ctx, query, referrerID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total referred amount: %w", err)
	}

	return total, nil
}

// CreateReferralOrder creates a referral order history record
// Laravel: $referred->referralOrders()->create([...])
func (r *referralRepository) CreateReferralOrder(ctx context.Context, history *models.ReferralOrderHistory) error {
	query := `
		INSERT INTO referral_order_histories (user_id, referral_id, amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		history.UserID,
		history.ReferralID,
		history.Amount,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create referral order: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		history.ID = uint64(id)
	}

	return nil
}

