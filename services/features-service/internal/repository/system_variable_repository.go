package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

type SystemVariableRepository struct {
	db *sql.DB
}

func NewSystemVariableRepository(db *sql.DB) *SystemVariableRepository {
	return &SystemVariableRepository{db: db}
}

// GetByKey retrieves a system variable value by key
// Implements Laravel: SystemVariable::getByKey('public_pricing_limit') ?? 80
func (r *SystemVariableRepository) GetByKey(ctx context.Context, key string) (int, error) {
	query := `
		SELECT value
		FROM system_variables
		WHERE key_name = ?
		LIMIT 1
	`

	var valueStr string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&valueStr)
	if err == sql.ErrNoRows {
		return 0, nil // Return 0 if not found (caller will use default)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get system variable: %w", err)
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse system variable value: %w", err)
	}

	return value, nil
}

// GetPricingLimits retrieves both pricing limits at once
func (r *SystemVariableRepository) GetPricingLimits(ctx context.Context) (publicLimit int, under18Limit int, err error) {
	query := `
		SELECT 
			COALESCE(MAX(CASE WHEN key_name = 'public_pricing_limit' THEN value END), '80') as public_limit,
			COALESCE(MAX(CASE WHEN key_name = 'under_18_pricing_limit' THEN value END), '110') as under_18_limit
		FROM system_variables
		WHERE key_name IN ('public_pricing_limit', 'under_18_pricing_limit')
	`

	var publicLimitStr, under18LimitStr string
	err = r.db.QueryRowContext(ctx, query).Scan(&publicLimitStr, &under18LimitStr)
	if err != nil && err != sql.ErrNoRows {
		return 80, 110, nil // Return defaults on error
	}

	if err == sql.ErrNoRows {
		return 80, 110, nil // Return defaults if no rows
	}

	publicLimit, err = strconv.Atoi(publicLimitStr)
	if err != nil {
		publicLimit = 80 // Default on parse error
	}

	under18Limit, err = strconv.Atoi(under18LimitStr)
	if err != nil {
		under18Limit = 110 // Default on parse error
	}

	return publicLimit, under18Limit, nil
}
