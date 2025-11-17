package repository

import (
	"context"
	"database/sql"

	"metargb/features-service/internal/models"
)

type SellRequestRepository struct {
	db *sql.DB
}

func NewSellRequestRepository(db *sql.DB) *SellRequestRepository{
	return &SellRequestRepository{db: db}
}

// Create creates a new sell feature request
func (r *SellRequestRepository) Create(ctx context.Context, sellerID, featureID uint64, pricePSC, priceIRR float64, limit int) (uint64, error) {
	query := `
		INSERT INTO sell_feature_requests (seller_id, feature_id, price_psc, price_irr, ` + "`limit`" + `, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, NOW(), NOW())
	`

	result, err := r.db.ExecContext(ctx, query, sellerID, featureID, pricePSC, priceIRR, limit)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return uint64(id), err
}

// GetLatestForSellerAndFeature gets the latest sell request for a seller's feature
func (r *SellRequestRepository) GetLatestForSellerAndFeature(ctx context.Context, sellerID, featureID uint64) (*models.SellFeatureRequest, error) {
	request := &models.SellFeatureRequest{}

	query := `
		SELECT id, seller_id, feature_id, price_psc, price_irr, ` + "`limit`" + `, status, created_at, updated_at
		FROM sell_feature_requests
		WHERE seller_id = ? AND feature_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.QueryRowContext(ctx, query, sellerID, featureID).Scan(
		&request.ID, &request.SellerID, &request.FeatureID,
		&request.PricePSC, &request.PriceIRR, &request.Limit, &request.Status,
		&request.CreatedAt, &request.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return request, err
}

// GetLatestUnderpricedForSeller gets the latest underpriced sell request for a seller
func (r *SellRequestRepository) GetLatestUnderpricedForSeller(ctx context.Context, sellerID uint64) (*models.SellFeatureRequest, error) {
	request := &models.SellFeatureRequest{}

	query := `
		SELECT id, seller_id, feature_id, price_psc, price_irr, ` + "`limit`" + `, status, created_at, updated_at
		FROM sell_feature_requests
		WHERE seller_id = ? AND ` + "`limit`" + ` < 100
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.QueryRowContext(ctx, query, sellerID).Scan(
		&request.ID, &request.SellerID, &request.FeatureID,
		&request.PricePSC, &request.PriceIRR, &request.Limit, &request.Status,
		&request.CreatedAt, &request.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return request, err
}

// UpdateStatus updates all sell requests for a feature to completed
func (r *SellRequestRepository) UpdateAllForFeatureToCompleted(ctx context.Context, featureID uint64) error {
	query := "UPDATE sell_feature_requests SET status = 1, updated_at = NOW() WHERE feature_id = ?"
	_, err := r.db.ExecContext(ctx, query, featureID)
	return err
}

// IsUnderpriced checks if a feature's latest sell request is underpriced
func (r *SellRequestRepository) IsUnderpriced(ctx context.Context, featureID uint64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM sell_feature_requests
			WHERE feature_id = ? AND ` + "`limit`" + ` < 100
			ORDER BY created_at DESC
			LIMIT 1
		)
	`

	var underpriced bool
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&underpriced)
	return underpriced, err
}

