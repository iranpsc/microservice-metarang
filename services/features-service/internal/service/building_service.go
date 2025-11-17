package service

import (
	"context"
	"encoding/json"
	"fmt"

	"metargb/features-service/internal/repository"
	"metargb/features-service/pkg/threed_client"
	pb "metargb/shared/pb/features"
)

type BuildingService struct {
	buildingRepo     *repository.BuildingRepository
	featureRepo      *repository.FeatureRepository
	geometryRepo     *repository.GeometryRepository
	hourlyProfitRepo *repository.HourlyProfitRepository
	threeDClient     *threed_client.Client
}

func NewBuildingService(
	buildingRepo *repository.BuildingRepository,
	featureRepo *repository.FeatureRepository,
	geometryRepo *repository.GeometryRepository,
	hourlyProfitRepo *repository.HourlyProfitRepository,
	threeDClient *threed_client.Client,
) *BuildingService {
	return &BuildingService{
		buildingRepo:     buildingRepo,
		featureRepo:      featureRepo,
		geometryRepo:     geometryRepo,
		hourlyProfitRepo: hourlyProfitRepo,
		threeDClient:     threeDClient,
	}
}

// GetBuildPackage retrieves building models from 3D Meta API
func (s *BuildingService) GetBuildPackage(ctx context.Context, featureID uint64, page int32) ([]*pb.BuildingModel, []string, error) {
	// Get feature properties
	_, properties, err := s.featureRepo.FindByID(ctx, featureID)
	if err != nil {
		return nil, nil, fmt.Errorf("feature not found: %w", err)
	}

	// Get coordinates
	coordinates, err := s.geometryRepo.GetCoordinatesByFeatureID(ctx, featureID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get coordinates: %w", err)
	}

	// Call 3D Meta API
	apiResp, err := s.threeDClient.GetBuildPackage(threed_client.BuildPackageRequest{
		FeatureID: featureID,
		Area:      fmt.Sprintf("%.2f", properties.Area),
		Density:   "1", // Default density
		Karbari:   properties.Karbari,
		Page:      page,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("3D API call failed: %w", err)
	}

	// Convert API response to protobuf models
	models := make([]*pb.BuildingModel, 0, len(apiResp.Data))
	for _, item := range apiResp.Data {
		imagesJSON, _ := json.Marshal(item.Images)
		attrsJSON, _ := json.Marshal(item.Attributes)
		fileJSON, _ := json.Marshal(item.File)

		models = append(models, &pb.BuildingModel{
			ModelId:    item.ID,
			Name:       item.Name,
			Sku:        item.SKU,
			Images:     string(imagesJSON),
			Attributes: string(attrsJSON),
			File:       string(fileJSON),
		})
	}

	return models, coordinates, nil
}

// BuildFeature starts construction of a building
func (s *BuildingService) BuildFeature(ctx context.Context, req *pb.BuildFeatureRequest) error {
	// TODO: Implement full building logic:
	// 1. Validate ownership
	// 2. Calculate construction_end_date
	// 3. Attach building model to feature
	// 4. Deactivate hourly profits
	// 5. Calculate bubble diameter
	// 6. If has activity_line, create ISIC code

	return s.buildingRepo.CreateBuilding(ctx, req)
}

// GetBuildings retrieves all buildings on a feature
func (s *BuildingService) GetBuildings(ctx context.Context, featureID uint64) ([]*pb.Building, error) {
	buildings, err := s.buildingRepo.FindByFeatureID(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get buildings: %w", err)
	}

	return buildings, nil
}

// UpdateBuilding updates a building
func (s *BuildingService) UpdateBuilding(ctx context.Context, req *pb.UpdateBuildingRequest) (*pb.Building, error) {
	// TODO: Implement update logic
	return s.buildingRepo.UpdateBuilding(ctx, req)
}

// DestroyBuilding removes a building from a feature
func (s *BuildingService) DestroyBuilding(ctx context.Context, featureID, buildingModelID uint64) error {
	// Reactivate hourly profits when building is destroyed
	if err := s.hourlyProfitRepo.ActivateProfitsForFeature(ctx, featureID); err != nil {
		return fmt.Errorf("failed to reactivate profits: %w", err)
	}

	return s.buildingRepo.DeleteBuilding(ctx, featureID, buildingModelID)
}
