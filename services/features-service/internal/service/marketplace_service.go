package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/features-service/internal/client"
	"metargb/features-service/internal/constants"
	"metargb/features-service/internal/models"
	"metargb/features-service/internal/repository"
	"metargb/shared/pkg/logger"
)

// MarketplaceService implements marketplace logic with gRPC cross-service calls
// This version uses CommercialClient instead of direct SQL for wallet operations
type MarketplaceService struct {
	featureRepo       *repository.FeatureRepository
	propertiesRepo    *repository.PropertiesRepository
	tradeRepo         *repository.TradeRepository
	buyRequestRepo    *repository.BuyRequestRepository
	sellRequestRepo   *repository.SellRequestRepository
	lockedAssetRepo   *repository.LockedAssetRepository
	hourlyProfitRepo  *repository.HourlyProfitRepository
	featureLimitRepo  *repository.FeatureLimitRepository
	commercialClient  *client.CommercialClient
	db                *sql.DB
	log               *logger.Logger
}

func NewMarketplaceService(
	featureRepo *repository.FeatureRepository,
	propertiesRepo *repository.PropertiesRepository,
	tradeRepo *repository.TradeRepository,
	buyRequestRepo *repository.BuyRequestRepository,
	sellRequestRepo *repository.SellRequestRepository,
	lockedAssetRepo *repository.LockedAssetRepository,
	hourlyProfitRepo *repository.HourlyProfitRepository,
	featureLimitRepo *repository.FeatureLimitRepository,
	commercialClient *client.CommercialClient,
	db *sql.DB,
	log *logger.Logger,
) *MarketplaceService {
	return &MarketplaceService{
		featureRepo:      featureRepo,
		propertiesRepo:   propertiesRepo,
		tradeRepo:        tradeRepo,
		buyRequestRepo:   buyRequestRepo,
		sellRequestRepo:  sellRequestRepo,
		lockedAssetRepo:  lockedAssetRepo,
		hourlyProfitRepo: hourlyProfitRepo,
		featureLimitRepo: featureLimitRepo,
		commercialClient: commercialClient,
		db:               db,
		log:              log,
	}
}

// BuyFeature implements the three-path buy logic using gRPC
func (s *MarketplaceService) BuyFeature(ctx context.Context, featureID, buyerID uint64) error {
	// Load feature with properties and owner
	feature, properties, err := s.featureRepo.FindByID(ctx, featureID)
	if err != nil {
		return fmt.Errorf("feature not found: %w", err)
	}

	// Get owner code
	var ownerCode string
	err = s.db.QueryRowContext(ctx, "SELECT code FROM users WHERE id = ?", feature.OwnerID).Scan(&ownerCode)
	if err != nil {
		return fmt.Errorf("failed to get owner: %w", err)
	}

	// Route to appropriate buy path
	if constants.IsLimitedFeature(properties.RGB) {
		return s.handleLimitedFeature(ctx, feature, properties, buyerID)
	} else if ownerCode == constants.RGBUserCode {
		return s.buyFromRGB(ctx, feature, properties, buyerID)
	} else {
		return s.buyFromUser(ctx, feature, properties, buyerID)
	}
}

// handleLimitedFeature - Path A with gRPC wallet operations
func (s *MarketplaceService) handleLimitedFeature(ctx context.Context, feature *models.Feature, properties *models.FeatureProperties, buyerID uint64) error {
	// Get feature limitation
	limitation, err := s.featureLimitRepo.GetLimitationByPropertyID(ctx, properties.ID)
	if err != nil || limitation == nil {
		return fmt.Errorf("خطایی رخ داده است. لطفا با پشتیبانی تماس بگیرید")
	}

	// Get buyer info
	var buyerName string
	var buyerDynastyID sql.NullInt64
	var buyerBirthdate sql.NullTime
	err = s.db.QueryRowContext(ctx,
		"SELECT u.name, u.dynasty_id, k.birthdate FROM users u LEFT JOIN kycs k ON u.id = k.user_id WHERE u.id = ?",
		buyerID,
	).Scan(&buyerName, &buyerDynastyID, &buyerBirthdate)
	if err != nil {
		return err
	}

	isUnder18 := false
	if buyerBirthdate.Valid {
		age := time.Since(buyerBirthdate.Time).Hours() / 24 / 365
		isUnder18 = age < 18
	}

	// Check buyer balance for color using gRPC
	color := constants.GetColor(properties.Karbari)
	if limitation.PriceLimit {
		hasBalance, err := s.commercialClient.CheckBalance(ctx, buyerID, color, properties.Stability)
		if err != nil || !hasBalance {
			return fmt.Errorf("برای خرید این ملک شما نیاز به %.2f لیتر رنگ %s دارید!",
				properties.Stability, constants.GetColorPersian(properties.Karbari))
		}
	}

	// Deduct buyer's color wallet via gRPC
	if err := s.commercialClient.DeductBalance(ctx, buyerID, color, properties.Stability); err != nil {
		return fmt.Errorf("failed to deduct buyer wallet: %w", err)
	}

	// Credit seller's color wallet via gRPC
	if err := s.commercialClient.AddBalance(ctx, feature.OwnerID, color, properties.Stability); err != nil {
		// Rollback buyer deduction
		s.commercialClient.AddBalance(ctx, buyerID, color, properties.Stability)
		return fmt.Errorf("failed to credit seller wallet: %w", err)
	}

	// Transfer ownership
	if err := s.featureRepo.UpdateOwner(ctx, feature.ID, buyerID); err != nil {
		return err
	}

	// Update properties
	pricingLimit := constants.DefaultPublicPricingLimit
	if isUnder18 {
		pricingLimit = constants.DefaultUnder18PricingLimit
	}

	newStatus := constants.ChangeStatusToSoldAndNotPriced(properties.Karbari)
	if err := s.propertiesRepo.UpdateStatus(ctx, feature.ID, newStatus, buyerName, "", pricingLimit); err != nil {
		return err
	}

	// Create trade
	tradeID, err := s.tradeRepo.Create(ctx, feature.ID, buyerID, feature.OwnerID, 0, 0)
	if err != nil {
		return err
	}

	s.log.Info("Limited feature purchased", "trade_id", tradeID, "feature_id", feature.ID, "buyer_id", buyerID)

	// Create hourly profit
	withdrawProfitDays, err := s.getUserVariableWithdrawProfit(ctx, buyerID)
	if err != nil {
		withdrawProfitDays = 10
	}

	_, err = s.hourlyProfitRepo.Create(ctx, buyerID, feature.ID, color, withdrawProfitDays)
	if err != nil {
		s.log.Error("Failed to create hourly profit", "error", err)
	}

	// Track limited feature purchase
	if err := s.featureLimitRepo.TrackLimitedPurchase(ctx, buyerID, limitation.ID, feature.ID); err != nil {
		s.log.Error("Failed to track limited purchase", "error", err)
	}

	return nil
}

// buyFromRGB - Path B with gRPC wallet operations
func (s *MarketplaceService) buyFromRGB(ctx context.Context, feature *models.Feature, properties *models.FeatureProperties, buyerID uint64) error {
	// Get buyer info
	var buyerName string
	var buyerBirthdate sql.NullTime
	err := s.db.QueryRowContext(ctx,
		"SELECT u.name, k.birthdate FROM users u LEFT JOIN kycs k ON u.id = k.user_id WHERE u.id = ?",
		buyerID,
	).Scan(&buyerName, &buyerBirthdate)
	if err != nil {
		return err
	}

	isUnder18 := false
	if buyerBirthdate.Valid {
		age := time.Since(buyerBirthdate.Time).Hours() / 24 / 365
		isUnder18 = age < 18
	}

	color := constants.GetColor(properties.Karbari)

	// Check buyer balance via gRPC
	hasBalance, err := s.commercialClient.CheckBalance(ctx, buyerID, color, properties.Stability)
	if err != nil || !hasBalance {
		return fmt.Errorf("برای خرید این ملک شما نیاز به %.2f لیتر رنگ %s دارید!",
			properties.Stability, constants.GetColorPersian(properties.Karbari))
	}

	// Deduct buyer's wallet via gRPC
	if err := s.commercialClient.DeductBalance(ctx, buyerID, color, properties.Stability); err != nil {
		return err
	}

	// Credit RGB's wallet via gRPC
	if err := s.commercialClient.AddBalance(ctx, feature.OwnerID, color, properties.Stability); err != nil {
		// Rollback
		s.commercialClient.AddBalance(ctx, buyerID, color, properties.Stability)
		return err
	}

	// Transfer ownership
	if err := s.featureRepo.UpdateOwner(ctx, feature.ID, buyerID); err != nil {
		return err
	}

	// Update properties
	pricingLimit := constants.DefaultPublicPricingLimit
	if isUnder18 {
		pricingLimit = constants.DefaultUnder18PricingLimit
	}

	newStatus := constants.ChangeStatusToSoldAndNotPriced(properties.Karbari)
	if err := s.propertiesRepo.UpdateStatus(ctx, feature.ID, newStatus, buyerName, "", pricingLimit); err != nil {
		return err
	}

	// Create trade
	_, err = s.tradeRepo.Create(ctx, feature.ID, buyerID, feature.OwnerID, 0, 0)
	if err != nil {
		return err
	}

	// Create hourly profit
	withdrawProfitDays, _ := s.getUserVariableWithdrawProfit(ctx, buyerID)
	if withdrawProfitDays == 0 {
		withdrawProfitDays = 10
	}

	_, err = s.hourlyProfitRepo.Create(ctx, buyerID, feature.ID, color, withdrawProfitDays)
	if err != nil {
		s.log.Error("Failed to create hourly profit", "error", err)
	}

	return nil
}

// buyFromUser - Path C with gRPC wallet operations and transactions
func (s *MarketplaceService) buyFromUser(ctx context.Context, feature *models.Feature, properties *models.FeatureProperties, buyerID uint64) error {
	// Check underpriced restriction
	if err := s.checkUnderpricedRestriction(ctx, feature, properties); err != nil {
		return err
	}

	// Get buyer info
	var buyerName string
	var buyerBirthdate sql.NullTime
	err := s.db.QueryRowContext(ctx,
		"SELECT u.name, k.birthdate FROM users u LEFT JOIN kycs k ON u.id = k.user_id WHERE u.id = ?",
		buyerID,
	).Scan(&buyerName, &buyerBirthdate)
	if err != nil {
		return err
	}

	isUnder18 := false
	if buyerBirthdate.Valid {
		age := time.Since(buyerBirthdate.Time).Hours() / 24 / 365
		isUnder18 = age < 18
	}

	// Parse prices
	pricePSC := parseFloat(properties.PricePSC)
	priceIRR := parseFloat(properties.PriceIRR)

	// Calculate amounts with fees
	buyerChargePSC := constants.CalculateBuyerCharge(pricePSC)
	buyerChargeIRR := constants.CalculateBuyerCharge(priceIRR)
	sellerPayPSC := constants.CalculateSellerPayment(pricePSC)
	sellerPayIRR := constants.CalculateSellerPayment(priceIRR)
	platformFeePSC := constants.CalculatePlatformFee(pricePSC)
	platformFeeIRR := constants.CalculatePlatformFee(priceIRR)

	// Check buyer balance via gRPC
	hasPSC, _ := s.commercialClient.CheckBalance(ctx, buyerID, "psc", buyerChargePSC)
	hasIRR, _ := s.commercialClient.CheckBalance(ctx, buyerID, "irr", buyerChargeIRR)
	if !hasPSC || !hasIRR {
		return fmt.Errorf("موجودی شما کافی نمی باشد")
	}

	// Deduct from buyer via gRPC
	if err := s.commercialClient.DeductBalance(ctx, buyerID, "psc", buyerChargePSC); err != nil {
		return err
	}
	if err := s.commercialClient.DeductBalance(ctx, buyerID, "irr", buyerChargeIRR); err != nil {
		// Rollback PSC
		s.commercialClient.AddBalance(ctx, buyerID, "psc", buyerChargePSC)
		return err
	}

	// Pay seller via gRPC
	if err := s.commercialClient.AddBalance(ctx, feature.OwnerID, "psc", sellerPayPSC); err != nil {
		return err
	}
	if err := s.commercialClient.AddBalance(ctx, feature.OwnerID, "irr", sellerPayIRR); err != nil {
		return err
	}

	// Pay RGB platform via gRPC
	rgbUserID, err := s.getRGBUserID(ctx)
	if err == nil {
		s.commercialClient.AddBalance(ctx, rgbUserID, "psc", platformFeePSC)
		s.commercialClient.AddBalance(ctx, rgbUserID, "irr", platformFeeIRR)
	}

	// Create trade
	tradeID, err := s.tradeRepo.Create(ctx, feature.ID, buyerID, feature.OwnerID, priceIRR, pricePSC)
	if err != nil {
		return err
	}

	// Create commission via direct SQL (Commercial service doesn't have commission endpoint yet)
	s.createCommission(ctx, tradeID, platformFeePSC, platformFeeIRR)

	// Transfer ownership
	if err := s.featureRepo.UpdateOwner(ctx, feature.ID, buyerID); err != nil {
		return err
	}

	// Update properties
	pricingLimit := constants.DefaultPublicPricingLimit
	if isUnder18 {
		pricingLimit = constants.DefaultUnder18PricingLimit
	}

	newStatus := constants.ChangeStatusToSoldAndNotPriced(properties.Karbari)
	if err := s.propertiesRepo.UpdateStatus(ctx, feature.ID, newStatus, buyerName, "", pricingLimit); err != nil {
		return err
	}

	// Transfer hourly profit
	withdrawProfitDays, _ := s.getUserVariableWithdrawProfit(ctx, buyerID)
	if withdrawProfitDays == 0 {
		withdrawProfitDays = 10
	}

	oldProfit, err := s.hourlyProfitRepo.GetByFeatureAndUser(ctx, feature.ID, feature.OwnerID)
	if err == nil && oldProfit != nil && oldProfit.Amount > 0 {
		// Add accumulated profit to seller's wallet via gRPC
		if err := s.commercialClient.AddBalance(ctx, feature.OwnerID, oldProfit.Asset, oldProfit.Amount); err != nil {
			s.log.Error("Failed to transfer profit to seller", "error", err)
		}
	}

	// Transfer profit to new owner
	_ = constants.GetColor(properties.Karbari) // Color for potential future use
	if err := s.hourlyProfitRepo.TransferProfitToNewOwner(ctx, feature.ID, feature.OwnerID, buyerID, withdrawProfitDays); err != nil {
		s.log.Error("Failed to transfer hourly profit", "error", err)
	}

	// Cancel all pending buy requests
	if err := s.buyRequestRepo.CancelAllForFeature(ctx, feature.ID); err != nil {
		s.log.Error("Failed to cancel buy requests", "error", err)
	}

	// Update sell requests
	if err := s.sellRequestRepo.UpdateAllForFeatureToCompleted(ctx, feature.ID); err != nil {
		s.log.Error("Failed to update sell requests", "error", err)
	}

	s.log.Info("Feature purchased from user",
		"trade_id", tradeID,
		"feature_id", feature.ID,
		"buyer_id", buyerID,
		"seller_id", feature.OwnerID,
	)

	return nil
}

// Helper methods
func (s *MarketplaceService) checkUnderpricedRestriction(ctx context.Context, feature *models.Feature, properties *models.FeatureProperties) error {
	isUnderpriced, err := s.sellRequestRepo.IsUnderpriced(ctx, feature.ID)
	if err != nil || !isUnderpriced {
		return nil
	}

	latestSellReq, err := s.sellRequestRepo.GetLatestUnderpricedForSeller(ctx, feature.OwnerID)
	if err != nil || latestSellReq == nil {
		return nil
	}

	latestTrade, err := s.tradeRepo.GetLatestUnderpricedForSeller(ctx, feature.OwnerID, latestSellReq.FeatureID)
	if err != nil || latestTrade == nil {
		return nil
	}

	if !s.tradeRepo.IsWithin24Hours(latestTrade) {
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

func (s *MarketplaceService) getUserVariableWithdrawProfit(ctx context.Context, userID uint64) (int, error) {
	var days int
	err := s.db.QueryRowContext(ctx, "SELECT withdraw_profit FROM user_variables WHERE user_id = ?", userID).Scan(&days)
	if err != nil {
		return 10, nil
	}
	return days, nil
}

func (s *MarketplaceService) getRGBUserID(ctx context.Context) (uint64, error) {
	var rgbID uint64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM users WHERE code = ?", constants.RGBUserCode).Scan(&rgbID)
	return rgbID, err
}

func (s *MarketplaceService) createCommission(ctx context.Context, tradeID uint64, psc, irr float64) error {
	query := "INSERT INTO comissions (trade_id, psc, irr, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())"
	_, err := s.db.ExecContext(ctx, query, tradeID, psc, irr)
	return err
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// SendBuyRequest creates a buy request for a feature
func (s *MarketplaceService) SendBuyRequest(ctx context.Context, req interface{}) (interface{}, error) {
	// TODO: Implement buy request creation
	return nil, fmt.Errorf("not implemented")
}

// AcceptBuyRequest accepts a buy request
func (s *MarketplaceService) AcceptBuyRequest(ctx context.Context, requestID, sellerID uint64) (interface{}, error) {
	// TODO: Implement buy request acceptance
	return nil, fmt.Errorf("not implemented")
}

// CreateSellRequest creates a sell request for a feature
func (s *MarketplaceService) CreateSellRequest(ctx context.Context, req interface{}) (interface{}, error) {
	// TODO: Implement sell request creation
	return nil, fmt.Errorf("not implemented")
}

// RequestGracePeriod adds grace period to a buy request
func (s *MarketplaceService) RequestGracePeriod(ctx context.Context, requestID, buyerID uint64, gracePeriod string) error {
	// TODO: Implement grace period request
	return fmt.Errorf("not implemented")
}

