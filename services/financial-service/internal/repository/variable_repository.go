package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type VariableRepository interface {
	GetRate(ctx context.Context, asset string) (float64, error)
}

type variableRepository struct {
	db *sql.DB
}

func NewVariableRepository(db *sql.DB) VariableRepository {
	return &variableRepository{db: db}
}

// GetRate retrieves the rate for a specific asset
// Laravel equivalent: Variable::getRate('psc')
// Note: The actual table structure uses 'asset' and 'price' columns
// but some code references 'key' and 'value'. We'll use asset/price
// to match the schema.
func (r *variableRepository) GetRate(ctx context.Context, asset string) (float64, error) {
	query := `
		SELECT price
		FROM variables
		WHERE asset = ?
		LIMIT 1
	`

	var price int64
	err := r.db.QueryRowContext(ctx, query, asset).Scan(&price)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("variable not found: %s", asset)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get variable rate: %w", err)
	}

	// Convert from bigint (price in smallest unit) to float64
	return float64(price), nil
}
