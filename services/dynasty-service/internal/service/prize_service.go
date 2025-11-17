package service

import (
	"context"
	"fmt"

	"metargb/dynasty-service/internal/models"
	"metargb/dynasty-service/internal/repository"
)

type PrizeService struct {
	prizeRepo *repository.PrizeRepository
}

func NewPrizeService(prizeRepo *repository.PrizeRepository) *PrizeService {
	return &PrizeService{prizeRepo: prizeRepo}
}

// GetAllPrizes retrieves all dynasty prizes
func (s *PrizeService) GetAllPrizes(ctx context.Context, page, perPage int32) ([]*models.DynastyPrize, int32, error) {
	return s.prizeRepo.GetAllPrizes(ctx, page, perPage)
}

// GetPrize retrieves a specific prize
func (s *PrizeService) GetPrize(ctx context.Context, prizeID uint64) (*models.DynastyPrize, error) {
	prize, err := s.prizeRepo.GetPrizeByID(ctx, prizeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prize: %w", err)
	}
	if prize == nil {
		return nil, fmt.Errorf("prize not found")
	}
	return prize, nil
}

// ClaimPrize allows a user to claim a dynasty prize
func (s *PrizeService) ClaimPrize(ctx context.Context, prizeID, userID uint64) error {
	// Check if already claimed
	claimed, err := s.prizeRepo.CheckPrizeClaimed(ctx, userID, prizeID)
	if err != nil {
		return fmt.Errorf("failed to check prize status: %w", err)
	}
	if claimed {
		return fmt.Errorf("prize already claimed")
	}

	// Claim the prize
	if err := s.prizeRepo.ClaimPrize(ctx, userID, prizeID); err != nil {
		return fmt.Errorf("failed to claim prize: %w", err)
	}

	return nil
}

