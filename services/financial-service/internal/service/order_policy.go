package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/financial-service/internal/repository"
)

type OrderPolicy interface {
	CanBuyFromStore(ctx context.Context, userID uint64) (bool, error)
	CanGetBonus(ctx context.Context, userID uint64, asset string) (bool, error)
}

type orderPolicy struct {
	db             *sql.DB
	firstOrderRepo repository.FirstOrderRepository
}

func NewOrderPolicy(db *sql.DB, firstOrderRepo repository.FirstOrderRepository) OrderPolicy {
	return &orderPolicy{
		db:             db,
		firstOrderRepo: firstOrderRepo,
	}
}

// CanBuyFromStore checks if user can buy from store
// Laravel: UserPolicy::buyFromStore
// Rule: Blocks users under 18 unless permissions are verified and BFR flag is set; adults pass automatically
func (p *orderPolicy) CanBuyFromStore(ctx context.Context, userID uint64) (bool, error) {
	// Get user birthdate
	var birthdate sql.NullTime
	err := p.db.QueryRowContext(ctx,
		"SELECT birthdate FROM kycs WHERE user_id = ?",
		userID,
	).Scan(&birthdate)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("failed to check user age: %w", err)
	}

	// If no birthdate, assume adult (pass)
	if !birthdate.Valid {
		return true, nil
	}

	// Calculate age
	age := time.Since(birthdate.Time).Hours() / (365.25 * 24)
	if age >= 18 {
		// Adults pass automatically
		return true, nil
	}

	// User is under 18 - check permissions
	// Need verified permissions with BFR flag set
	var verified sql.NullBool
	var bfr sql.NullBool
	err = p.db.QueryRowContext(ctx,
		"SELECT verified, BFR FROM child_permissions WHERE user_id = ?",
		userID,
	).Scan(&verified, &bfr)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("failed to check permissions: %w", err)
	}

	// Must have verified permissions AND BFR flag set
	if verified.Valid && verified.Bool && bfr.Valid && bfr.Bool {
		return true, nil
	}

	return false, nil
}

// CanGetBonus checks if user can get first order bonus
// Laravel: OrderPolicy::canGetBonus
// Rule: Returns true only when user has never logged a firstOrder record and asset is not 'irr'
func (p *orderPolicy) CanGetBonus(ctx context.Context, userID uint64, asset string) (bool, error) {
	// Asset must not be 'irr'
	if asset == "irr" {
		return false, nil
	}

	// Check if user has any first order
	count, err := p.firstOrderRepo.Count(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check first order: %w", err)
	}

	// User can get bonus only if they have NO first orders
	return count == 0, nil
}
