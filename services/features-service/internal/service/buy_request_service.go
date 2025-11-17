package service

import (
	"context"
	"database/sql"
	"fmt"

	"metargb/features-service/internal/client"
	"metargb/features-service/internal/constants"
	"metargb/features-service/internal/models"
	"metargb/features-service/internal/repository"
	"metargb/shared/pkg/logger"
)

// BuyRequestService handles buy requests with gRPC cross-service calls
type BuyRequestService struct {
	featureRepo      *repository.FeatureRepository
	propertiesRepo   *repository.PropertiesRepository
	tradeRepo        *repository.TradeRepository
	buyRequestRepo   *repository.BuyRequestRepository
	sellRequestRepo  *repository.SellRequestRepository
	lockedAssetRepo  *repository.LockedAssetRepository
	hourlyProfitRepo *repository.HourlyProfitRepository
	commercialClient *client.CommercialClient
	db               *sql.DB
	log              *logger.Logger
}

func NewBuyRequestService(
	featureRepo *repository.FeatureRepository,
	propertiesRepo *repository.PropertiesRepository,
	tradeRepo *repository.TradeRepository,
	buyRequestRepo *repository.BuyRequestRepository,
	sellRequestRepo *repository.SellRequestRepository,
	lockedAssetRepo *repository.LockedAssetRepository,
	hourlyProfitRepo *repository.HourlyProfitRepository,
	commercialClient *client.CommercialClient,
	db *sql.DB,
	log *logger.Logger,
) *BuyRequestService {
	return &BuyRequestService{
		featureRepo:      featureRepo,
		propertiesRepo:   propertiesRepo,
		tradeRepo:        tradeRepo,
		buyRequestRepo:   buyRequestRepo,
		sellRequestRepo:  sellRequestRepo,
		lockedAssetRepo:  lockedAssetRepo,
		hourlyProfitRepo: hourlyProfitRepo,
		commercialClient: commercialClient,
		db:               db,
		log:              log,
	}
}

// SendBuyRequest creates a buy request with locked assets using gRPC
func (s *BuyRequestService) SendBuyRequest(ctx context.Context, buyerID, featureID uint64, pricePSC, priceIRR float64, note string) (uint64, error) {
	// Get feature and seller
	feature, properties, err := s.featureRepo.FindByID(ctx, featureID)
	if err != nil {
		return 0, fmt.Errorf("feature not found: %w", err)
	}

	sellerID := feature.OwnerID

	// Validate price against minimum_price_percentage
	totalRequestedPrice := priceIRR + (pricePSC * s.getVariableRate(ctx, "psc"))
	color := constants.GetColor(properties.Karbari)
	colorRate := s.getVariableRate(ctx, color)
	totalFeaturePrice := properties.Stability * colorRate

	floorPercentage := float64(properties.MinimumPricePercentage)
	actualPercentage := (totalRequestedPrice / totalFeaturePrice) * 100

	if actualPercentage < floorPercentage {
		return 0, fmt.Errorf("شما مجاز به ارسال درخواست خرید به کمتر از %.0f%% قیمت ملک نمی باشید!", floorPercentage)
	}

	// Calculate amounts with fees
	buyerChargePSC := constants.CalculateBuyerCharge(pricePSC)
	buyerChargeIRR := constants.CalculateBuyerCharge(priceIRR)

	// Check buyer balance via gRPC
	hasPSC, _ := s.commercialClient.CheckBalance(ctx, buyerID, "psc", buyerChargePSC)
	hasIRR, _ := s.commercialClient.CheckBalance(ctx, buyerID, "irr", buyerChargeIRR)
	if !hasPSC || !hasIRR {
		return 0, fmt.Errorf("موجودی شما کافی نمی باشد")
	}

	// Create buy request
	requestID, err := s.buyRequestRepo.Create(ctx, buyerID, sellerID, featureID, note, pricePSC, priceIRR)
	if err != nil {
		return 0, err
	}

	// Deduct buyer's wallet via gRPC (lock funds)
	if err := s.commercialClient.DeductBalance(ctx, buyerID, "psc", buyerChargePSC); err != nil {
		return 0, fmt.Errorf("failed to lock PSC: %w", err)
	}
	if err := s.commercialClient.DeductBalance(ctx, buyerID, "irr", buyerChargeIRR); err != nil {
		// Rollback PSC
		s.commercialClient.AddBalance(ctx, buyerID, "psc", buyerChargePSC)
		return 0, fmt.Errorf("failed to lock IRR: %w", err)
	}

	// Create locked asset record
	if _, err := s.lockedAssetRepo.Create(ctx, requestID, featureID, buyerChargePSC, buyerChargeIRR); err != nil {
		s.log.Error("Failed to create locked asset", "error", err)
	}

	// Create transactions via gRPC
	s.commercialClient.CreateTransaction(ctx, buyerID, "psc", buyerChargePSC, "withdraw", 0, "App\\Models\\BuyFeatureRequest", requestID)
	s.commercialClient.CreateTransaction(ctx, buyerID, "irr", buyerChargeIRR, "withdraw", 0, "App\\Models\\BuyFeatureRequest", requestID)

	s.log.Info("Buy request created with locked assets",
		"request_id", requestID,
		"buyer_id", buyerID,
		"feature_id", featureID,
		"psc_locked", buyerChargePSC,
		"irr_locked", buyerChargeIRR,
	)

	// TODO: Send notifications via Notifications Service

	return requestID, nil
}

// AcceptBuyRequest accepts a request and releases locked assets using gRPC
func (s *BuyRequestService) AcceptBuyRequest(ctx context.Context, requestID, sellerID uint64) error {
	// Get buy request
	buyRequest, err := s.buyRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("buy request not found: %w", err)
	}

	// Verify seller
	if buyRequest.SellerID != sellerID {
		return fmt.Errorf("unauthorized: not the seller")
	}

	// Get feature
	feature, properties, err := s.featureRepo.FindByID(ctx, buyRequest.FeatureID)
	if err != nil {
		return fmt.Errorf("feature not found: %w", err)
	}

	// Check underpriced restriction
	if err := s.checkUnderpricedRestriction(ctx, feature, properties); err != nil {
		return err
	}

	// Get locked assets to verify they exist
	_, err = s.lockedAssetRepo.GetByBuyRequestID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("locked assets not found: %w", err)
	}

	pscAmount := buyRequest.PricePSC
	irrAmount := buyRequest.PriceIRR
	pscFee := constants.CalculateFee(pscAmount)
	irrFee := constants.CalculateFee(irrAmount)

	// Pay seller via gRPC (price - fee)
	if err := s.commercialClient.AddBalance(ctx, sellerID, "psc", pscAmount-pscFee); err != nil {
		return err
	}
	if err := s.commercialClient.AddBalance(ctx, sellerID, "irr", irrAmount-irrFee); err != nil {
		return err
	}

	// Pay RGB platform via gRPC (fee × 2)
	rgbUserID, err := s.getRGBUserID(ctx)
	if err == nil {
		s.commercialClient.AddBalance(ctx, rgbUserID, "psc", pscFee*2)
		s.commercialClient.AddBalance(ctx, rgbUserID, "irr", irrFee*2)
	}

	// Create trade
	tradeID, err := s.tradeRepo.Create(ctx, buyRequest.FeatureID, buyRequest.BuyerID, sellerID, irrAmount, pscAmount)
	if err != nil {
		return err
	}

	// Create commission
	s.createCommission(ctx, tradeID, pscFee*2, irrFee*2)

	// Create transactions for seller via gRPC
	s.commercialClient.CreateTransaction(ctx, sellerID, "psc", pscAmount-pscFee, "deposit", 1, "App\\Models\\Trade", tradeID)
	s.commercialClient.CreateTransaction(ctx, sellerID, "irr", irrAmount-irrFee, "deposit", 1, "App\\Models\\Trade", tradeID)

	// Transfer ownership
	if err := s.featureRepo.UpdateOwner(ctx, feature.ID, buyRequest.BuyerID); err != nil {
		return err
	}

	// Update properties
	buyerName := s.getUserName(ctx, buyRequest.BuyerID)
	isUnder18 := s.isUserUnder18(ctx, buyRequest.BuyerID)
	pricingLimit := constants.DefaultPublicPricingLimit
	if isUnder18 {
		pricingLimit = constants.DefaultUnder18PricingLimit
	}

	newStatus := constants.ChangeStatusToSoldAndNotPriced(properties.Karbari)
	if err := s.propertiesRepo.UpdateStatus(ctx, feature.ID, newStatus, buyerName, "", pricingLimit); err != nil {
		return err
	}

	// Transfer hourly profit
	withdrawProfitDays, _ := s.getUserVariableWithdrawProfit(ctx, buyRequest.BuyerID)
	if withdrawProfitDays == 0 {
		withdrawProfitDays = 10
	}

	oldProfit, _ := s.hourlyProfitRepo.GetByFeatureAndUser(ctx, feature.ID, sellerID)
	if oldProfit != nil && oldProfit.Amount > 0 {
		s.commercialClient.AddBalance(ctx, sellerID, oldProfit.Asset, oldProfit.Amount)
	}

	s.hourlyProfitRepo.TransferProfitToNewOwner(ctx, feature.ID, sellerID, buyRequest.BuyerID, withdrawProfitDays)

	// Update request and delete locked asset
	s.buyRequestRepo.UpdateStatus(ctx, requestID, 1)
	s.buyRequestRepo.SoftDelete(ctx, requestID)
	s.lockedAssetRepo.Delete(ctx, requestID)

	// Cancel other requests and refund
	allRequests, _ := s.buyRequestRepo.GetAllForFeature(ctx, buyRequest.FeatureID)
	for _, req := range allRequests {
		if req.ID != requestID {
			s.refundBuyRequest(ctx, req.ID)
		}
	}

	// Update sell requests
	s.sellRequestRepo.UpdateAllForFeatureToCompleted(ctx, buyRequest.FeatureID)

	s.log.Info("Buy request accepted",
		"request_id", requestID,
		"feature_id", buyRequest.FeatureID,
		"buyer_id", buyRequest.BuyerID,
		"seller_id", sellerID,
		"trade_id", tradeID,
	)

	return nil
}

// refundBuyRequest refunds a cancelled request using gRPC
func (s *BuyRequestService) refundBuyRequest(ctx context.Context, requestID uint64) {
	buyRequest, err := s.buyRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		return
	}

	lockedAsset, err := s.lockedAssetRepo.GetByBuyRequestID(ctx, requestID)
	if err != nil {
		return
	}

	// Refund buyer via gRPC
	s.commercialClient.AddBalance(ctx, buyRequest.BuyerID, "psc", lockedAsset.PSC)
	s.commercialClient.AddBalance(ctx, buyRequest.BuyerID, "irr", lockedAsset.IRR)

	// Delete locked asset and soft delete request
	s.lockedAssetRepo.Delete(ctx, requestID)
	s.buyRequestRepo.SoftDelete(ctx, requestID)

	s.log.Info("Buy request refunded", "request_id", requestID, "buyer_id", buyRequest.BuyerID)
}

// Helper methods
func (s *BuyRequestService) checkUnderpricedRestriction(ctx context.Context, feature *models.Feature, properties *models.FeatureProperties) error {
	// Reuse from marketplace_service_grpc.go logic
	isUnderpriced, _ := s.sellRequestRepo.IsUnderpriced(ctx, feature.ID)
	if !isUnderpriced {
		return nil
	}

	latestSellReq, _ := s.sellRequestRepo.GetLatestUnderpricedForSeller(ctx, feature.OwnerID)
	if latestSellReq == nil {
		return nil
	}

	latestTrade, _ := s.tradeRepo.GetLatestUnderpricedForSeller(ctx, feature.OwnerID, latestSellReq.FeatureID)
	if latestTrade == nil || !s.tradeRepo.IsWithin24Hours(latestTrade) {
		return nil
	}

	hours, minutes := s.tradeRepo.GetTimeRemaining(latestTrade)
	var elapsedTime string
	if hours < 1 {
		elapsedTime = fmt.Sprintf("%d دقیقه", minutes)
	} else {
		elapsedTime = fmt.Sprintf("%d ساعت", hours)
	}

	return fmt.Errorf("شما در ۲۴ ساعت گذشته ملکی با زیر قیمت ۱۰۰٪ بفروش رسانده اید. برای پذیرش این درخواست باید %s صبر کنید", elapsedTime)
}

func (s *BuyRequestService) getVariableRate(ctx context.Context, asset string) float64 {
	var rate float64
	query := "SELECT value FROM variables WHERE `key` = ?"
	if err := s.db.QueryRowContext(ctx, query, asset).Scan(&rate); err != nil {
		return 1.0
	}
	return rate
}

func (s *BuyRequestService) getRGBUserID(ctx context.Context) (uint64, error) {
	var rgbID uint64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM users WHERE code = ?", constants.RGBUserCode).Scan(&rgbID)
	return rgbID, err
}

func (s *BuyRequestService) getUserName(ctx context.Context, userID uint64) string {
	var name string
	s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", userID).Scan(&name)
	return name
}

func (s *BuyRequestService) isUserUnder18(ctx context.Context, userID uint64) bool {
	var birthdate sql.NullTime
	s.db.QueryRowContext(ctx, "SELECT birthdate FROM kycs WHERE user_id = ?", userID).Scan(&birthdate)
	if !birthdate.Valid {
		return false
	}
	// Simplified age check
	return false
}

func (s *BuyRequestService) getUserVariableWithdrawProfit(ctx context.Context, userID uint64) (int, error) {
	var days int
	err := s.db.QueryRowContext(ctx, "SELECT withdraw_profit FROM user_variables WHERE user_id = ?", userID).Scan(&days)
	return days, err
}

func (s *BuyRequestService) createCommission(ctx context.Context, tradeID uint64, psc, irr float64) {
	query := "INSERT INTO comissions (trade_id, psc, irr, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())"
	s.db.ExecContext(ctx, query, tradeID, psc, irr)
}

