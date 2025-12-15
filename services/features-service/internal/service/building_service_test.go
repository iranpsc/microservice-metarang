package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/features-service/internal/models"
	pb "metargb/shared/pb/features"
)

// Mock repositories for testing
type mockBuildingRepository struct {
	upsertModelFunc                   func(ctx context.Context, modelID, name, sku, images, attributes, file string, requiredSatisfaction float64) error
	findModelFunc                     func(ctx context.Context, modelID uint64) (*pb.BuildingModel, error)
	hasBuildingFunc                   func(ctx context.Context, featureID uint64) (bool, error)
	createBuildingFunc                func(ctx context.Context, featureID, buildingModelID uint64, launchedSatisfaction, rotation, position, information string, startDate, endDate time.Time, bubbleDiameter float64) error
	findByFeatureIDFunc               func(ctx context.Context, featureID uint64) ([]*pb.Building, error)
	updateBuildingFunc                func(ctx context.Context, featureID, buildingModelID uint64, launchedSatisfaction, rotation, position, information string, endDate time.Time, bubbleDiameter float64) (*pb.Building, error)
	findBuildingByFeatureAndModelFunc func(ctx context.Context, featureID, buildingModelID uint64) (*pb.Building, error)
	deleteBuildingFunc                func(ctx context.Context, featureID, buildingModelID uint64) error
	firstOrCreateIsicCodeFunc         func(ctx context.Context, activityLine string) (uint64, error)
}

func (m *mockBuildingRepository) UpsertBuildingModel(ctx context.Context, modelID, name, sku, images, attributes, file string, requiredSatisfaction float64) error {
	if m.upsertModelFunc != nil {
		return m.upsertModelFunc(ctx, modelID, name, sku, images, attributes, file, requiredSatisfaction)
	}
	return errors.New("not implemented")
}

func (m *mockBuildingRepository) FindBuildingModelByModelID(ctx context.Context, modelID uint64) (*pb.BuildingModel, error) {
	if m.findModelFunc != nil {
		return m.findModelFunc(ctx, modelID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBuildingRepository) HasBuilding(ctx context.Context, featureID uint64) (bool, error) {
	if m.hasBuildingFunc != nil {
		return m.hasBuildingFunc(ctx, featureID)
	}
	return false, errors.New("not implemented")
}

func (m *mockBuildingRepository) CreateBuilding(ctx context.Context, featureID, buildingModelID uint64, launchedSatisfaction, rotation, position, information string, startDate, endDate time.Time, bubbleDiameter float64) error {
	if m.createBuildingFunc != nil {
		return m.createBuildingFunc(ctx, featureID, buildingModelID, launchedSatisfaction, rotation, position, information, startDate, endDate, bubbleDiameter)
	}
	return errors.New("not implemented")
}

func (m *mockBuildingRepository) FindByFeatureID(ctx context.Context, featureID uint64) ([]*pb.Building, error) {
	if m.findByFeatureIDFunc != nil {
		return m.findByFeatureIDFunc(ctx, featureID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBuildingRepository) UpdateBuilding(ctx context.Context, featureID, buildingModelID uint64, launchedSatisfaction, rotation, position, information string, endDate time.Time, bubbleDiameter float64) (*pb.Building, error) {
	if m.updateBuildingFunc != nil {
		return m.updateBuildingFunc(ctx, featureID, buildingModelID, launchedSatisfaction, rotation, position, information, endDate, bubbleDiameter)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBuildingRepository) FindBuildingByFeatureAndModel(ctx context.Context, featureID, buildingModelID uint64) (*pb.Building, error) {
	if m.findBuildingByFeatureAndModelFunc != nil {
		return m.findBuildingByFeatureAndModelFunc(ctx, featureID, buildingModelID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBuildingRepository) DeleteBuilding(ctx context.Context, featureID, buildingModelID uint64) error {
	if m.deleteBuildingFunc != nil {
		return m.deleteBuildingFunc(ctx, featureID, buildingModelID)
	}
	return errors.New("not implemented")
}

func (m *mockBuildingRepository) FirstOrCreateIsicCode(ctx context.Context, activityLine string) (uint64, error) {
	if m.firstOrCreateIsicCodeFunc != nil {
		return m.firstOrCreateIsicCodeFunc(ctx, activityLine)
	}
	return 0, errors.New("not implemented")
}

// Mock other repositories
type mockFeatureRepository struct {
	findByIDFunc func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error)
}

func (m *mockFeatureRepository) FindByID(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil, errors.New("not implemented")
}

type mockGeometryRepository struct {
	getCoordinatesFunc func(ctx context.Context, featureID uint64) ([]string, error)
}

func (m *mockGeometryRepository) GetCoordinatesByFeatureID(ctx context.Context, featureID uint64) ([]string, error) {
	if m.getCoordinatesFunc != nil {
		return m.getCoordinatesFunc(ctx, featureID)
	}
	return nil, errors.New("not implemented")
}

type mockHourlyProfitRepository struct {
	deactivateFunc func(ctx context.Context, featureID uint64) error
	activateFunc   func(ctx context.Context, featureID uint64) error
}

func (m *mockHourlyProfitRepository) DeactivateProfitsForFeature(ctx context.Context, featureID uint64) error {
	if m.deactivateFunc != nil {
		return m.deactivateFunc(ctx, featureID)
	}
	return errors.New("not implemented")
}

func (m *mockHourlyProfitRepository) ActivateProfitsForFeature(ctx context.Context, featureID uint64) error {
	if m.activateFunc != nil {
		return m.activateFunc(ctx, featureID)
	}
	return errors.New("not implemented")
}

// Mock 3D client
type mockThreeDClient struct {
	getBuildPackageFunc func(req interface{}) (interface{}, error)
}

func (m *mockThreeDClient) GetBuildPackage(req interface{}) (interface{}, error) {
	if m.getBuildPackageFunc != nil {
		return m.getBuildPackageFunc(req)
	}
	return nil, errors.New("not implemented")
}

func TestBuildingService_GetBuildPackage(t *testing.T) {
	ctx := context.Background()

	t.Run("unauthorized user", func(t *testing.T) {
		mockBuildingRepo := &mockBuildingRepository{}
		mockFeatureRepo := &mockFeatureRepository{}
		mockGeometryRepo := &mockGeometryRepository{}
		mockProfitRepo := &mockHourlyProfitRepository{}

		mockFeatureRepo.findByIDFunc = func(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
			return &models.Feature{
				ID:      1,
				OwnerID: 100, // Different owner
			}, &models.FeatureProperties{}, nil
		}

		// Note: 3D client is required but we can't easily mock it
		// This test focuses on authorization logic
		// For full testing, integration tests or a 3D client interface would be needed
		service := NewBuildingService(mockBuildingRepo, mockFeatureRepo, mockGeometryRepo, mockProfitRepo, nil)

		_, _, err := service.GetBuildPackage(ctx, 1, 1, 200) // Different user ID
		if err == nil {
			t.Error("Expected error for unauthorized user")
		}
		if err != nil && !contains(err.Error(), "unauthorized") && !contains(err.Error(), "does not own") {
			t.Errorf("Expected authorization error, got: %v", err)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Additional test cases would go here for BuildFeature, GetBuildings, UpdateBuilding, DestroyBuilding
// Due to complexity, these would require more comprehensive mocking setup
