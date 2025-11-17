package service

import (
	"context"
	"fmt"

	"metargb/features-service/internal/models"
	"metargb/features-service/internal/repository"
	pb "metargb/shared/pb/features"
)

type FeatureService struct {
	featureRepo    *repository.FeatureRepository
	propertiesRepo *repository.PropertiesRepository
	geometryRepo   *repository.GeometryRepository
}

func NewFeatureService(
	featureRepo *repository.FeatureRepository,
	propertiesRepo *repository.PropertiesRepository,
	geometryRepo *repository.GeometryRepository,
) *FeatureService {
	return &FeatureService{
		featureRepo:    featureRepo,
		propertiesRepo: propertiesRepo,
		geometryRepo:   geometryRepo,
	}
}

// ListFeatures retrieves features within a bounding box
// Implements Laravel's FeatureRepository@all logic
func (s *FeatureService) ListFeatures(ctx context.Context, points []string, loadBuildings bool, userFeaturesLocation bool) ([]*pb.Feature, error) {
	// Parse points into coordinates
	// points[0] = "x1,y1", points[1] = "x2,y2", etc.
	// Expected format: [minX,minY, maxX,minY, maxX,maxY, minX,maxY]
	
	features, err := s.featureRepo.FindByBoundingBox(ctx, points, loadBuildings)
	if err != nil {
		return nil, fmt.Errorf("failed to find features by bbox: %w", err)
	}

	return models.FeaturesToPB(features), nil
}

// GetFeature retrieves a single feature with all relations
func (s *FeatureService) GetFeature(ctx context.Context, featureID uint64) (*pb.Feature, error) {
	feature, properties, err := s.featureRepo.FindByID(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("feature not found: %w", err)
	}

	// Load geometry
	geometry, err := s.geometryRepo.GetByFeatureID(ctx, featureID)
	if err != nil {
		geometry = nil
	}

	return models.FeatureToPB(feature, properties, geometry), nil
}

// UpdateFeature updates feature properties
func (s *FeatureService) UpdateFeature(ctx context.Context, featureID uint64, properties *pb.FeatureProperties) (*pb.Feature, error) {
	// Convert protobuf properties to map for update
	updates := map[string]interface{}{
		"karbari": properties.Karbari,
		"rgb":     properties.Rgb,
		"owner":   properties.Owner,
		"label":   properties.Label,
	}
	
	if properties.PricePsc != "" {
		updates["price_psc"] = properties.PricePsc
	}
	if properties.PriceIrr != "" {
		updates["price_irr"] = properties.PriceIrr
	}
	if properties.MinimumPricePercentage > 0 {
		updates["minimum_price_percentage"] = properties.MinimumPricePercentage
	}

	// Update properties
	if err := s.propertiesRepo.Update(ctx, featureID, updates); err != nil {
		return nil, fmt.Errorf("failed to update properties: %w", err)
	}

	// Return updated feature
	return s.GetFeature(ctx, featureID)
}

// AddFeatureImages adds images to a feature
func (s *FeatureService) AddFeatureImages(ctx context.Context, featureID uint64, imageURLs []string) (*pb.Feature, error) {
	// TODO: Implement image addition
	// For now, just return the feature
	return s.GetFeature(ctx, featureID)
}

// GetMyFeatures retrieves all features owned by a user
func (s *FeatureService) GetMyFeatures(ctx context.Context, userID uint64) ([]*pb.Feature, error) {
	features, err := s.featureRepo.FindByOwner(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user features: %w", err)
	}

	return models.FeaturesToPB(features), nil
}

