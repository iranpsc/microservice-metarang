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
	// Get received prize (this is the ID of received_prizes table)
	receivedPrize, err := s.prizeRepo.GetReceivedPrize(ctx, prizeID)
	if err != nil {
		return fmt.Errorf("failed to get received prize: %w", err)
	}
	if receivedPrize == nil {
		return fmt.Errorf("prize not found")
	}

	// Verify ownership
	if receivedPrize.UserID != userID {
		return fmt.Errorf("unauthorized: prize does not belong to user")
	}

	// TODO: Update wallet and variables via commercial service
	// For now, just delete the received prize record
	if err := s.prizeRepo.DeleteReceivedPrize(ctx, prizeID); err != nil {
		return fmt.Errorf("failed to delete received prize: %w", err)
	}

	return nil
}

// GetUserReceivedPrizes retrieves all received prizes for a user
func (s *PrizeService) GetUserReceivedPrizes(ctx context.Context, userID uint64, page, perPage int32) ([]*models.ReceivedPrize, int32, error) {
	// Get all prizes for user
	prizes, err := s.prizeRepo.GetUserReceivedPrizes(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user prizes: %w", err)
	}

	total := int32(len(prizes))

	// Simple pagination
	offset := (page - 1) * perPage
	if offset >= total {
		return []*models.ReceivedPrize{}, total, nil
	}

	end := offset + perPage
	if end > total {
		end = total
	}

	return prizes[offset:end], total, nil
}

// GetReceivedPrize retrieves a received prize by ID
func (s *PrizeService) GetReceivedPrize(ctx context.Context, receivedPrizeID uint64) (*models.ReceivedPrize, error) {
	return s.prizeRepo.GetReceivedPrize(ctx, receivedPrizeID)
}
