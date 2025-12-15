package service

import (
	"context"
	"fmt"

	"metargb/dynasty-service/internal/models"
	"metargb/dynasty-service/internal/repository"
)

// PrizeServiceEnhanced provides enhanced prize operations
type PrizeServiceEnhanced struct {
	prizeRepo *repository.PrizeRepository

	// gRPC clients (to be injected)
	// commercialClient commercial.CommercialServiceClient
}

func NewPrizeServiceEnhanced(prizeRepo *repository.PrizeRepository) *PrizeServiceEnhanced {
	return &PrizeServiceEnhanced{
		prizeRepo: prizeRepo,
	}
}

// AwardPrize awards a prize to a user based on relationship
func (s *PrizeServiceEnhanced) AwardPrize(
	ctx context.Context,
	userID uint64,
	relationship string,
	message string,
) error {
	// Get prize by relationship
	prize, err := s.prizeRepo.GetPrizeByRelationship(ctx, relationship)
	if err != nil {
		return fmt.Errorf("failed to get prize: %w", err)
	}
	if prize == nil {
		// No prize defined for this relationship
		return nil
	}

	// Create received prize record
	if err := s.prizeRepo.AwardPrize(ctx, userID, prize.ID, message); err != nil {
		return fmt.Errorf("failed to award prize: %w", err)
	}

	return nil
}

// ClaimPrize claims a received prize
func (s *PrizeServiceEnhanced) ClaimPrize(
	ctx context.Context,
	userID, receivedPrizeID uint64,
) error {
	// Get received prize
	receivedPrize, err := s.prizeRepo.GetReceivedPrize(ctx, receivedPrizeID)
	if err != nil {
		return fmt.Errorf("failed to get received prize: %w", err)
	}
	if receivedPrize == nil {
		return fmt.Errorf("received prize not found")
	}

	// Verify ownership
	if receivedPrize.UserID != userID {
		return fmt.Errorf("unauthorized to claim this prize")
	}

	_ = receivedPrize.Prize // Prize is available but not currently used

	// TODO: Update wallet via Commercial Service gRPC call
	// This is the critical business logic that must be implemented:
	/*
		// 1. Get PSC rate from variables
		pscRate := getVariableRate("psc") // Would call Commercial service

		// 2. Add PSC to wallet (converted by rate)
		err = s.commercialClient.UpdateWallet(ctx, &commercial.UpdateWalletRequest{
			UserId: userID,
			Updates: []*commercial.WalletUpdate{
				{
					Asset:  "psc",
					Amount: float64(prize.PSC) / pscRate,
					Action: "increment",
				},
				{
					Asset:  "satisfaction",
					Amount: prize.Satisfaction,
					Action: "increment",
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}

		// 3. Get current user variables
		variables, err := s.commercialClient.GetUserVariables(ctx, &commercial.GetUserVariablesRequest{
			UserId: userID,
		})
		if err != nil {
			return fmt.Errorf("failed to get user variables: %w", err)
		}

		// 4. Update user variables with percentage increases
		err = s.commercialClient.UpdateUserVariables(ctx, &commercial.UpdateUserVariablesRequest{
			UserId: userID,
			Updates: map[string]float64{
				"referral_profit": variables.ReferralProfit +
					(variables.ReferralProfit * prize.IntroductionProfitIncrease),
				"data_storage": variables.DataStorage +
					(variables.DataStorage * prize.DataStorage),
				"withdraw_profit": variables.WithdrawProfit +
					(variables.WithdrawProfit * prize.AccumulatedCapitalReserve),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update user variables: %w", err)
		}
	*/

	// 5. Delete claimed prize
	if err := s.prizeRepo.DeleteReceivedPrize(ctx, receivedPrizeID); err != nil {
		return fmt.Errorf("failed to delete received prize: %w", err)
	}

	return nil
}

// GetUserPrizes retrieves all received prizes for a user
func (s *PrizeServiceEnhanced) GetUserPrizes(ctx context.Context, userID uint64) ([]*models.ReceivedPrize, error) {
	prizes, err := s.prizeRepo.GetUserReceivedPrizes(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user prizes: %w", err)
	}

	return prizes, nil
}

// GetIntroductionPrizes retrieves all dynasty prizes for display
func (s *PrizeServiceEnhanced) GetIntroductionPrizes(ctx context.Context) ([]*models.DynastyPrize, error) {
	prizes, err := s.prizeRepo.GetAllDynastyPrizes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dynasty prizes: %w", err)
	}

	return prizes, nil
}
