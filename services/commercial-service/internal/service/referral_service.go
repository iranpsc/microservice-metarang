package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"metargb/commercial-service/internal/models"
	"metargb/commercial-service/internal/repository"
)

type ReferralService interface {
	ProcessReferralCommission(ctx context.Context, userID uint64, order *models.Order) error
}

type referralService struct {
	referralRepo     repository.ReferralRepository
	variableRepo     repository.VariableRepository
	userVariableRepo repository.UserVariableRepository
	walletRepo       repository.WalletRepository
}

func NewReferralService(
	referralRepo repository.ReferralRepository,
	variableRepo repository.VariableRepository,
	userVariableRepo repository.UserVariableRepository,
	walletRepo repository.WalletRepository,
) ReferralService {
	return &referralService{
		referralRepo:     referralRepo,
		variableRepo:     variableRepo,
		userVariableRepo: userVariableRepo,
		walletRepo:       walletRepo,
	}
}

// ProcessReferralCommission implements the referral commission logic from Laravel
// Laravel: App\Services\ReferralService::referral(User $user, Order $order)
func (s *referralService) ProcessReferralCommission(ctx context.Context, userID uint64, order *models.Order) error {
	// If the asset is 'irr', do not proceed with referral
	if order.Asset == "irr" {
		return nil
	}

	// Get the referrer (the user who referred this buyer)
	referrerID, err := s.referralRepo.GetReferrerID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get referrer: %w", err)
	}

	// If user has no referrer, skip
	if referrerID == nil {
		return nil
	}

	// Get PSC rate
	pscRate, err := s.variableRepo.GetRate(ctx, "psc")
	if err != nil {
		return fmt.Errorf("failed to get PSC rate: %w", err)
	}

	// Calculate the total amount already referred
	referredAmount, err := s.referralRepo.GetTotalReferredAmount(ctx, *referrerID)
	if err != nil {
		return fmt.Errorf("failed to get referred amount: %w", err)
	}

	// Multiply by PSC rate to get total in PSC equivalent
	totalReferredPSC := referredAmount * pscRate

	// Get referral profit limit for the referrer
	referralLimit, err := s.userVariableRepo.GetReferralProfitLimit(ctx, *referrerID)
	if err != nil {
		return fmt.Errorf("failed to get referral limit: %w", err)
	}

	// Check if limit exceeded
	if totalReferredPSC >= referralLimit {
		return nil // Limit reached, no commission
	}

	// Calculate referral commission (50% of order amount)
	var referrerAmount float64

	// If order asset is a color (blue, red, yellow), convert to PSC equivalent first
	if order.Asset == "blue" || order.Asset == "red" || order.Asset == "yellow" {
		// Get the color rate
		colorRate, err := s.variableRepo.GetRate(ctx, order.Asset)
		if err != nil {
			return fmt.Errorf("failed to get color rate: %w", err)
		}

		// Convert: (order.amount * colorRate) / pscRate * 0.5
		referrerAmount = ((order.Amount * colorRate) / pscRate) * 0.5
	} else {
		// For PSC orders, direct 50% multiplier
		referrerAmount = order.Amount * 0.5
	}

	// Increment referrer's PSC wallet
	referrerAmountDec := decimal.NewFromFloat(referrerAmount)
	err = s.walletRepo.AddBalance(ctx, *referrerID, "psc", referrerAmountDec)
	if err != nil {
		return fmt.Errorf("failed to add referral commission to wallet: %w", err)
	}

	// Create referral order history
	history := &models.ReferralOrderHistory{
		UserID:     *referrerID, // The referrer who receives commission
		ReferralID: userID,      // The user who was referred
		Amount:     referrerAmount,
	}

	err = s.referralRepo.CreateReferralOrder(ctx, history)
	if err != nil {
		return fmt.Errorf("failed to create referral order history: %w", err)
	}

	return nil
}
