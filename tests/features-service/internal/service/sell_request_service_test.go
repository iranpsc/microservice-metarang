package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/features-service/internal/constants"
	"metargb/features-service/internal/models"
	pb "metargb/shared/pb/features"
	"metargb/shared/pkg/logger"

	_ "github.com/go-sql-driver/mysql"
)

// Mock repositories
type mockSellRequestRepository struct {
	createFunc        func(ctx context.Context, sellerID, featureID uint64, pricePSC, priceIRR float64, limit int) (uint64, error)
	listFunc          func(ctx context.Context, sellerID uint64) ([]*models.SellFeatureRequest, error)
	findByIDFunc      func(ctx context.Context, id uint64) (*models.SellFeatureRequest, error)
	deleteFunc        func(ctx context.Context, id uint64) error
	isUnderpricedFunc func(ctx context.Context, featureID uint64) (bool, error)
}

func (m *mockSellRequestRepository) Create(ctx context.Context, sellerID, featureID uint64, pricePSC, priceIRR float64, limit int) (uint64, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, sellerID, featureID, pricePSC, priceIRR, limit)
	}
	return 0, errors.New("not implemented")
}

func (m *mockSellRequestRepository) ListBySellerID(ctx context.Context, sellerID uint64) ([]*models.SellFeatureRequest, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, sellerID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSellRequestRepository) FindByID(ctx context.Context, id uint64) (*models.SellFeatureRequest, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSellRequestRepository) Delete(ctx context.Context, id uint64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockSellRequestRepository) IsUnderpriced(ctx context.Context, featureID uint64) (bool, error) {
	if m.isUnderpricedFunc != nil {
		return m.isUnderpricedFunc(ctx, featureID)
	}
	return false, errors.New("not implemented")
}

type mockSystemVariableRepository struct {
	getPricingLimitsFunc func(ctx context.Context) (int, int, error)
}

func (m *mockSystemVariableRepository) GetPricingLimits(ctx context.Context) (int, int, error) {
	if m.getPricingLimitsFunc != nil {
		return m.getPricingLimitsFunc(ctx)
	}
	return 80, 110, nil
}

func TestMarketplaceService_CreateSellRequest(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLogger("test")

	t.Run("success with explicit prices", func(t *testing.T) {
		mockSellRepo := &mockSellRequestRepository{
			createFunc: func(ctx context.Context, sellerID, featureID uint64, pricePSC, priceIRR float64, limit int) (uint64, error) {
				if sellerID != 1 || featureID != 100 {
					return 0, errors.New("invalid IDs")
				}
				return 1, nil
			},
			findByIDFunc: func(ctx context.Context, id uint64) (*models.SellFeatureRequest, error) {
				return &models.SellFeatureRequest{
					ID:        1,
					SellerID:  1,
					FeatureID: 100,
					PricePSC:  12.5,
					PriceIRR:  8500000,
					Limit:     90,
					Status:    0,
					CreatedAt: time.Now(),
				}, nil
			},
		}

		mockFeatureRepo := &mockSellRequestFeatureRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
				return &models.Feature{
						ID:      100,
						OwnerID: 1,
					}, &models.FeatureProperties{
						ID:        "prop1",
						FeatureID: 100,
						Karbari:   constants.Maskoni,
						RGB:       constants.MaskoniSoldAndNotPriced,
						Stability: 10000000,
					}, nil
			},
		}

		mockPropsRepo := &mockPropertiesRepository{
			updateFunc: func(ctx context.Context, featureID uint64, updates map[string]interface{}) error {
				return nil
			},
		}

		mockSysVarRepo := &mockSystemVariableRepository{
			getPricingLimitsFunc: func(ctx context.Context) (int, int, error) {
				return 80, 110, nil
			},
		}

		service := &MarketplaceService{
			featureRepo:        mockFeatureRepo,
			propertiesRepo:     mockPropsRepo,
			sellRequestRepo:    mockSellRepo,
			systemVariableRepo: mockSysVarRepo,
			db:                 nil, // Mock DB
			log:                log,
		}

		req := &pb.CreateSellRequestRequest{
			FeatureId: 100,
			SellerId:  1,
			PricePsc:  "12.5",
			PriceIrr:  "8500000",
		}

		result, err := service.CreateSellRequest(ctx, req)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		if result.ID != 1 {
			t.Errorf("Expected ID 1, got %d", result.ID)
		}
	})

	t.Run("success with percentage", func(t *testing.T) {
		mockSellRepo := &mockSellRequestRepository{
			createFunc: func(ctx context.Context, sellerID, featureID uint64, pricePSC, priceIRR float64, limit int) (uint64, error) {
				return 1, nil
			},
			findByIDFunc: func(ctx context.Context, id uint64) (*models.SellFeatureRequest, error) {
				return &models.SellFeatureRequest{
					ID:        1,
					SellerID:  1,
					FeatureID: 100,
					PricePSC:  62500,
					PriceIRR:  5000000,
					Limit:     125,
					Status:    0,
				}, nil
			},
		}

		mockFeatureRepo := &mockSellRequestFeatureRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
				return &models.Feature{
						ID:      100,
						OwnerID: 1,
					}, &models.FeatureProperties{
						ID:        "prop1",
						FeatureID: 100,
						Karbari:   constants.Maskoni,
						RGB:       constants.MaskoniSoldAndNotPriced,
						Stability: 10000000,
					}, nil
			},
		}

		mockPropsRepo := &mockPropertiesRepository{
			updateFunc: func(ctx context.Context, featureID uint64, updates map[string]interface{}) error {
				return nil
			},
		}

		service := &MarketplaceService{
			featureRepo:        mockFeatureRepo,
			propertiesRepo:     mockPropsRepo,
			sellRequestRepo:    mockSellRepo,
			systemVariableRepo: &mockSystemVariableRepository{},
			db:                 nil,
			log:                log,
		}

		req := &pb.CreateSellRequestRequest{
			FeatureId:              100,
			SellerId:               1,
			MinimumPricePercentage: 125,
		}

		result, err := service.CreateSellRequest(ctx, req)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		if result.Limit != 125 {
			t.Errorf("Expected limit 125, got %d", result.Limit)
		}
	})

	t.Run("unauthorized - not owner", func(t *testing.T) {
		mockFeatureRepo := &mockSellRequestFeatureRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
				return &models.Feature{
					ID:      100,
					OwnerID: 2, // Different owner
				}, &models.FeatureProperties{}, nil
			},
		}

		service := &MarketplaceService{
			featureRepo: mockFeatureRepo,
			log:         log,
		}

		req := &pb.CreateSellRequestRequest{
			FeatureId: 100,
			SellerId:  1, // Trying to sell someone else's feature
		}

		_, err := service.CreateSellRequest(ctx, req)
		if err == nil || !containsSellRequest(err.Error(), "unauthorized") {
			t.Errorf("Expected unauthorized error, got: %v", err)
		}
	})

	t.Run("mutually exclusive fields", func(t *testing.T) {
		service := &MarketplaceService{
			log: log,
		}

		req := &pb.CreateSellRequestRequest{
			FeatureId:              100,
			SellerId:               1,
			PricePsc:               "12.5",
			MinimumPricePercentage: 125,
		}

		_, err := service.CreateSellRequest(ctx, req)
		if err == nil || !contains(err.Error(), "mutually exclusive") {
			t.Errorf("Expected mutually exclusive error, got: %v", err)
		}
	})

	t.Run("pricing below floor", func(t *testing.T) {
		mockFeatureRepo := &mockSellRequestFeatureRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
				return &models.Feature{
						ID:      100,
						OwnerID: 1,
					}, &models.FeatureProperties{
						ID:        "prop1",
						FeatureID: 100,
						Karbari:   constants.Maskoni,
						Stability: 10000000,
					}, nil
			},
		}

		service := &MarketplaceService{
			featureRepo:        mockFeatureRepo,
			systemVariableRepo: &mockSystemVariableRepository{},
			db:                 nil,
			log:                log,
		}

		req := &pb.CreateSellRequestRequest{
			FeatureId: 100,
			SellerId:  1,
			PricePsc:  "1.0",  // Very low price
			PriceIrr:  "1000", // Very low price
		}

		_, err := service.CreateSellRequest(ctx, req)
		if err == nil || !contains(err.Error(), "مجاز") {
			t.Errorf("Expected pricing floor error, got: %v", err)
		}
	})
}

func TestMarketplaceService_ListSellRequests(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLogger("test")

	t.Run("success", func(t *testing.T) {
		mockSellRepo := &mockSellRequestRepository{
			listFunc: func(ctx context.Context, sellerID uint64) ([]*models.SellFeatureRequest, error) {
				return []*models.SellFeatureRequest{
					{
						ID:        1,
						SellerID:  1,
						FeatureID: 100,
						PricePSC:  12.5,
						PriceIRR:  8500000,
						Limit:     90,
						Status:    0,
					},
				}, nil
			},
		}

		service := &MarketplaceService{
			sellRequestRepo: mockSellRepo,
			log:             log,
		}

		requests, err := service.ListSellRequests(ctx, 1)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		if len(requests) != 1 {
			t.Errorf("Expected 1 request, got %d", len(requests))
		}
	})
}

func TestMarketplaceService_DeleteSellRequest(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLogger("test")

	t.Run("success", func(t *testing.T) {
		mockSellRepo := &mockSellRequestRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.SellFeatureRequest, error) {
				return &models.SellFeatureRequest{
					ID:        1,
					SellerID:  1,
					FeatureID: 100,
				}, nil
			},
			deleteFunc: func(ctx context.Context, id uint64) error {
				return nil
			},
		}

		mockFeatureRepo := &mockSellRequestFeatureRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
				return &models.Feature{
						ID: 100,
					}, &models.FeatureProperties{
						Karbari: constants.Maskoni,
					}, nil
			},
		}

		mockPropsRepo := &mockPropertiesRepository{
			updateFunc: func(ctx context.Context, featureID uint64, updates map[string]interface{}) error {
				return nil
			},
		}

		service := &MarketplaceService{
			featureRepo:     mockFeatureRepo,
			propertiesRepo:  mockPropsRepo,
			sellRequestRepo: mockSellRepo,
			log:             log,
		}

		err := service.DeleteSellRequest(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockSellRepo := &mockSellRequestRepository{
			findByIDFunc: func(ctx context.Context, id uint64) (*models.SellFeatureRequest, error) {
				return &models.SellFeatureRequest{
					ID:       1,
					SellerID: 2, // Different seller
				}, nil
			},
		}

		service := &MarketplaceService{
			sellRequestRepo: mockSellRepo,
			log:             log,
		}

		err := service.DeleteSellRequest(ctx, 1, 1)
		if err == nil || !containsSellRequest(err.Error(), "unauthorized") {
			t.Errorf("Expected unauthorized error, got: %v", err)
		}
	})
}

// Helper functions and mock repositories
type mockSellRequestFeatureRepository struct {
	findByIDFunc func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error)
}

func (m *mockSellRequestFeatureRepository) FindByID(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil, errors.New("not implemented")
}

type mockPropertiesRepository struct {
	updateFunc func(ctx context.Context, featureID uint64, updates map[string]interface{}) error
}

func (m *mockPropertiesRepository) Update(ctx context.Context, featureID uint64, updates map[string]interface{}) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, featureID, updates)
	}
	return errors.New("not implemented")
}

func containsSellRequest(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSellRequestMiddle(s, substr)))
}

func containsSellRequestMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
