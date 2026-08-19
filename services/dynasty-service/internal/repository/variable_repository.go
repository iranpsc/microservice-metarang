package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type VariableRepository struct {
	db *sql.DB
}

func NewVariableRepository(db *sql.DB) *VariableRepository {
	return &VariableRepository{db: db}
}

// GetPriceByAsset returns the price column for an asset slug (e.g. "psc").
func (r *VariableRepository) GetPriceByAsset(ctx context.Context, asset string) (float64, error) {
	const q = `SELECT price FROM variables WHERE asset = ? LIMIT 1`
	var price int64
	err := r.db.QueryRowContext(ctx, q, asset).Scan(&price)
	if err == sql.ErrNoRows {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get asset price: %w", err)
	}
	if price <= 0 {
		return 1, nil
	}
	return float64(price), nil
}
