package service

import (
	"context"
	"database/sql"
	"fmt"

	"metargb/features-service/internal/client"
	"metargb/features-service/internal/constants"
	"metargb/features-service/internal/repository"
	"metargb/shared/pkg/logger"
)

// ProfitService implements profit service with gRPC cross-service calls
type ProfitService struct {
	profitRepo       *repository.HourlyProfitRepository
	featureRepo      *repository.FeatureRepository
	propertiesRepo   *repository.PropertiesRepository
	commercialClient *client.CommercialClient
	db               *sql.DB
	log              *logger.Logger
}

func NewProfitService(
	profitRepo *repository.HourlyProfitRepository,
	featureRepo *repository.FeatureRepository,
	propertiesRepo *repository.PropertiesRepository,
	commercialClient *client.CommercialClient,
	db *sql.DB,
	log *logger.Logger,
) *ProfitService {
	return &ProfitService{
		profitRepo:       profitRepo,
		featureRepo:      featureRepo,
		propertiesRepo:   propertiesRepo,
		commercialClient: commercialClient,
		db:               db,
		log:              log,
	}
}

// GetSingleProfit withdraws a single profit using gRPC
func (s *ProfitService) GetSingleProfit(ctx context.Context, profitID, userID uint64) (float64, error) {
	// Get profit record
	profit, err := s.profitRepo.FindByID(ctx, profitID)
	if err != nil {
		return 0, fmt.Errorf("profit not found: %w", err)
	}

	// Verify ownership
	if profit.UserID != userID {
		return 0, fmt.Errorf("unauthorized")
	}

	// Add amount to user wallet via gRPC
	if profit.Amount > 0 {
		if err := s.commercialClient.AddBalance(ctx, userID, profit.Asset, profit.Amount); err != nil {
			return 0, fmt.Errorf("failed to update wallet: %w", err)
		}

		s.log.Info("Profit withdrawn",
			"profit_id", profitID,
			"user_id", userID,
			"asset", profit.Asset,
			"amount", profit.Amount,
		)
	}

	// Get user's withdraw_profit days
	withdrawProfitDays, err := s.getUserVariableWithdrawProfit(ctx, userID)
	if err != nil || withdrawProfitDays == 0 {
		withdrawProfitDays = 10
	}

	// Reset profit and update deadline
	if err := s.profitRepo.ResetProfitAndUpdateDeadline(ctx, profitID, withdrawProfitDays); err != nil {
		return 0, fmt.Errorf("failed to reset profit: %w", err)
	}

	// TODO: Send notification via Notifications Service
	// notificationsClient.SendNotification(ctx, &pb.SendRequest{
	//     UserId: userID,
	//     Type:   "FeatureHourlyProfitDeposit",
	//     Data:   map[string]interface{}{
	//         "asset":  profit.Asset,
	//         "amount": profit.Amount,
	//         "id":     profit.FeatureID,
	//     },
	// })

	return profit.Amount, nil
}

// GetProfitsByApplication withdraws all profits by karbari using gRPC
func (s *ProfitService) GetProfitsByApplication(ctx context.Context, userID uint64, karbari string) (float64, error) {
	// Validate karbari
	if karbari != constants.Maskoni && karbari != constants.Tejari && karbari != constants.Amozeshi {
		return 0, fmt.Errorf("invalid karbari: must be m, t, or a")
	}

	// Map karbari to asset
	asset := constants.GetColor(karbari)

	// Get user's withdraw_profit days
	withdrawProfitDays, err := s.getUserVariableWithdrawProfit(ctx, userID)
	if err != nil || withdrawProfitDays == 0 {
		withdrawProfitDays = 10
	}

	// Get all profits for this user and karbari
	profits, err := s.profitRepo.GetAllByUserAndKarbari(ctx, userID, asset)
	if err != nil {
		return 0, fmt.Errorf("failed to get profits: %w", err)
	}

	// Sum amounts and add to wallet via gRPC
	totalAmount := 0.0
	for _, profit := range profits {
		totalAmount += profit.Amount

		// Add to wallet via gRPC
		if profit.Amount > 0 {
			if err := s.commercialClient.AddBalance(ctx, userID, profit.Asset, profit.Amount); err != nil {
				s.log.Error("Failed to add profit to wallet", "profit_id", profit.ID, "error", err)
				continue
			}
		}

		// Reset profit
		if err := s.profitRepo.ResetProfitAndUpdateDeadline(ctx, profit.ID, withdrawProfitDays); err != nil {
			s.log.Error("Failed to reset profit", "profit_id", profit.ID, "error", err)
		}
	}

	// Log withdrawal
	if totalAmount > 0 {
		s.log.Info("Batch profits withdrawn by karbari",
			"user_id", userID,
			"karbari", karbari,
			"asset", asset,
			"amount", totalAmount,
			"count", len(profits),
		)

		// TODO: Send notification via Notifications Service
		// karbariTitle := constants.GetKarbariTitle(karbari)
		// notificationsClient.SendNotification(ctx, &pb.SendRequest{
		//     UserId: userID,
		//     Type:   "FeatureHourlyProfitDeposit",
		//     Data:   map[string]interface{}{
		//         "asset":   asset,
		//         "amount":  totalAmount,
		//         "karbari": karbariTitle,
		//         "id":      nil,
		//     },
		// })
	}

	return totalAmount, nil
}

// TransferProfitOnSale handles profit transfer when feature is sold
// Uses gRPC to add accumulated profit to seller's wallet
func (s *ProfitService) TransferProfitOnSale(ctx context.Context, featureID, sellerID, buyerID uint64, withdrawProfitDays int) error {
	// Get existing profit for seller
	oldProfit, err := s.profitRepo.GetByFeatureAndUser(ctx, featureID, sellerID)
	if err == nil && oldProfit != nil && oldProfit.Amount > 0 {
		// Add accumulated profit to seller's wallet via gRPC
		if err := s.commercialClient.AddBalance(ctx, sellerID, oldProfit.Asset, oldProfit.Amount); err != nil {
			s.log.Error("Failed to transfer profit to seller", "error", err)
			return err
		}

		s.log.Info("Profit transferred on sale",
			"feature_id", featureID,
			"seller_id", sellerID,
			"amount", oldProfit.Amount,
			"asset", oldProfit.Asset,
		)
	}

	// Transfer profit record to new owner
	if err := s.profitRepo.TransferProfitToNewOwner(ctx, featureID, sellerID, buyerID, withdrawProfitDays); err != nil {
		return fmt.Errorf("failed to transfer profit record: %w", err)
	}

	return nil
}

// GetHourlyProfits retrieves paginated hourly profits for a user
func (s *ProfitService) GetHourlyProfits(ctx context.Context, userID uint64, page, pageSize int32) (interface{}, string, string, string, error) {
	// Get profits with pagination
	profits, err := s.profitRepo.FindByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, "0", "0", "0", fmt.Errorf("failed to get profits: %w", err)
	}

	// Get totals by karbari
	totalMaskoni, totalTejari, totalAmozeshi, err := s.profitRepo.GetTotalsByKarbari(ctx, userID)
	if err != nil {
		return profits, "0", "0", "0", nil
	}

	return profits, totalMaskoni, totalTejari, totalAmozeshi, nil
}

// StartHourlyProfitCalculator runs the background job to calculate hourly profits
func (s *ProfitService) StartHourlyProfitCalculator(ctx context.Context, log *logger.Logger) {
	// TODO: Implement background job similar to Laravel's CalculateFeatureProfit command
	// This should run periodically and call profitRepo.CalculateAndUpdateProfits
	log.Info("Hourly profit calculator started (not yet implemented)")
}

// Utility methods
func (s *ProfitService) getUserVariableWithdrawProfit(ctx context.Context, userID uint64) (int, error) {
	var days int
	err := s.db.QueryRowContext(ctx, "SELECT withdraw_profit FROM user_variables WHERE user_id = ?", userID).Scan(&days)
	if err != nil {
		return 10, nil
	}
	return days, nil
}

