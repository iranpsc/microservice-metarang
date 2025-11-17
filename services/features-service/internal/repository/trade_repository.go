package repository

import (
	"context"
	"database/sql"
	"time"

	"metargb/features-service/internal/models"
)

type TradeRepository struct {
	db *sql.DB
}

func NewTradeRepository(db *sql.DB) *TradeRepository {
	return &TradeRepository{db: db}
}

// Create creates a new trade record
func (r *TradeRepository) Create(ctx context.Context, featureID, buyerID, sellerID uint64, irrAmount, pscAmount float64) (uint64, error) {
	query := `
		INSERT INTO trades (feature_id, buyer_id, seller_id, irr_amount, psc_amount, date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW(), NOW())
	`

	result, err := r.db.ExecContext(ctx, query, featureID, buyerID, sellerID, irrAmount, pscAmount)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return uint64(id), err
}

// GetLatestForFeature gets the most recent trade for a feature
func (r *TradeRepository) GetLatestForFeature(ctx context.Context, featureID uint64) (*models.Trade, error) {
	trade := &models.Trade{}

	query := `
		SELECT id, feature_id, buyer_id, seller_id, irr_amount, psc_amount, date, created_at, updated_at
		FROM trades
		WHERE feature_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.QueryRowContext(ctx, query, featureID).Scan(
		&trade.ID, &trade.FeatureID, &trade.BuyerID, &trade.SellerID,
		&trade.IRRAmount, &trade.PSCAmount, &trade.Date,
		&trade.CreatedAt, &trade.UpdatedAt,
	)

	return trade, err
}

// GetLatestForSeller gets the most recent underpriced trade for a seller
func (r *TradeRepository) GetLatestUnderpricedForSeller(ctx context.Context, sellerID, featureID uint64) (*models.Trade, error) {
	trade := &models.Trade{}

	// Get latest trade where seller sold feature that was underpriced (< 100%)
	query := `
		SELECT t.id, t.feature_id, t.buyer_id, t.seller_id, t.irr_amount, t.psc_amount, t.date, t.created_at, t.updated_at
		FROM trades t
		INNER JOIN sell_feature_requests sfr ON t.feature_id = sfr.feature_id AND t.seller_id = sfr.seller_id
		WHERE t.seller_id = ? AND t.feature_id = ? AND sfr.limit < 100
		ORDER BY t.created_at DESC
		LIMIT 1
	`

	err := r.db.QueryRowContext(ctx, query, sellerID, featureID).Scan(
		&trade.ID, &trade.FeatureID, &trade.BuyerID, &trade.SellerID,
		&trade.IRRAmount, &trade.PSCAmount, &trade.Date,
		&trade.CreatedAt, &trade.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No underpriced trade found
	}

	return trade, err
}

// IsWithin24Hours checks if trade was created within last 24 hours
func (r *TradeRepository) IsWithin24Hours(trade *models.Trade) bool {
	if trade == nil {
		return false
	}
	return time.Since(trade.CreatedAt).Hours() < 24
}

// GetTimeRemaining returns remaining time until 24-hour lock expires
func (r *TradeRepository) GetTimeRemaining(trade *models.Trade) (hours int, minutes int) {
	if trade == nil {
		return 0, 0
	}

	lockExpiry := trade.CreatedAt.Add(24 * time.Hour)
	remaining := time.Until(lockExpiry)

	if remaining < 0 {
		return 0, 0
	}

	hours = int(remaining.Hours())
	minutes = int(remaining.Minutes()) % 60
	return hours, minutes
}

