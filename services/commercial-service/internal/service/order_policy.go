package service

import (
	"context"
	"fmt"

	"metargb/commercial-service/internal/repository"
)

type OrderPolicy interface {
	CanGetBonus(ctx context.Context, userID uint64, orderType string) (bool, error)
}

type orderPolicy struct {
	firstOrderRepo repository.FirstOrderRepository
}

func NewOrderPolicy(firstOrderRepo repository.FirstOrderRepository) OrderPolicy {
	return &orderPolicy{
		firstOrderRepo: firstOrderRepo,
	}
}

// CanGetBonus checks if user can get first order bonus
// Laravel: OrderPolicy::canGetBonus
// Rule: User can get bonus only if they haven't received first order bonus for this asset type
func (p *orderPolicy) CanGetBonus(ctx context.Context, userID uint64, orderType string) (bool, error) {
	// Check if user already has a first order for this type
	hasFirstOrder, err := p.firstOrderRepo.HasFirstOrder(ctx, userID, orderType)
	if err != nil {
		return false, fmt.Errorf("failed to check first order: %w", err)
	}

	// User can get bonus only if they DON'T have a first order for this type yet
	return !hasFirstOrder, nil
}
