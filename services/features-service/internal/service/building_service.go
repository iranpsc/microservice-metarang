package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"metargb/features-service/internal/client"
	"metargb/features-service/internal/constants"
	"metargb/features-service/internal/repository"
	"metargb/features-service/pkg/threed_client"
	pb "metargb/shared/pb/features"
	"metargb/shared/pkg/auth"
	"metargb/shared/pkg/helpers"
)

type BuildingService struct {
	buildingRepo     *repository.BuildingRepository
	featureRepo      *repository.FeatureRepository
	geometryRepo     *repository.GeometryRepository
	hourlyProfitRepo *repository.HourlyProfitRepository
	threeDClient     *threed_client.Client
	commercialClient *client.CommercialClient
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

// SetCommercialClient sets the commercial client for wallet operations
func (s *BuildingService) SetCommercialClient(client *client.CommercialClient) {
	s.commercialClient = client
}

// GetBuildPackage retrieves building models from 3D Meta API
// Checks ownership, calls 3D API, calculates required_satisfaction, upserts models, and returns with coordinates
func (s *BuildingService) GetBuildPackage(ctx context.Context, featureID uint64, page int32) ([]*pb.BuildingModel, []string, error) {
	// Get feature with properties
	feature, properties, err := s.featureRepo.FindByID(ctx, featureID)
	if err != nil {
		return nil, nil, fmt.Errorf("feature not found: %w", err)
	}

	// Get user from context for ownership check
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unauthorized: authentication required")
	}

	// Ownership check: user must own the feature
	if feature.OwnerID != user.UserID {
		return nil, nil, fmt.Errorf("unauthorized: user does not own this feature")
	}

	// Get coordinates for feature
	coordinates, err := s.geometryRepo.GetCoordinatesByFeatureID(ctx, featureID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get coordinates: %w", err)
	}

	// Get density from properties (default to 1 if not set)
	density := properties.Density
	if density == 0 {
		density = 1 // Default to 1 if density is 0
	}

	// Call 3D Meta API
	apiResp, err := s.threeDClient.GetBuildPackage(threed_client.BuildPackageRequest{
		FeatureID: featureID,
		Area:      fmt.Sprintf("%.2f", properties.Area),
		Density:   fmt.Sprintf("%d", density),
		Karbari:   properties.Karbari,
		Page:      page,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("3D API call failed: %w", err)
	}

	// Get karbari coefficient
	karbariCoeff := constants.GetKarbariCoefficient(properties.Karbari)

	// Convert API response to protobuf models and calculate required_satisfaction
	models := make([]*pb.BuildingModel, 0, len(apiResp.Data))
	for _, item := range apiResp.Data {
		imagesJSON, _ := json.Marshal(item.Images)
		attrsJSON, _ := json.Marshal(item.Attributes)
		fileJSON, _ := json.Marshal(item.File)

		// Calculate required_satisfaction: area * karbariCoefficient * density * 0.1 / 100
		requiredSatisfaction := properties.Area * karbariCoeff * float64(density) * 0.1 / 100.0

		// Upsert building model locally
		err = s.buildingRepo.UpsertBuildingModel(ctx, item.ID, item.Name, item.SKU,
			string(imagesJSON), string(attrsJSON), string(fileJSON), requiredSatisfaction)
		if err != nil {
			// Log error but continue processing other models
			fmt.Printf("failed to upsert building model %s: %v\n", item.ID, err)
		}

		models = append(models, &pb.BuildingModel{
			ModelId:              item.ID,
			Name:                 item.Name,
			Sku:                  item.SKU,
			Images:               string(imagesJSON),
			Attributes:           string(attrsJSON),
			File:                 string(fileJSON),
			RequiredSatisfaction: fmt.Sprintf("%.4f", requiredSatisfaction),
		})
	}

	return models, coordinates, nil
}

// BuildFeature starts construction of a building on a feature
func (s *BuildingService) BuildFeature(ctx context.Context, req *pb.BuildFeatureRequest) error {
	// 1. Get feature and validate ownership
	feature, _, err := s.featureRepo.FindByID(ctx, req.FeatureId)
	if err != nil {
		return fmt.Errorf("feature not found: %w", err)
	}

	// Get user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthorized: authentication required")
	}

	if feature.OwnerID != user.UserID {
		return fmt.Errorf("unauthorized: user does not own this feature")
	}

	// 2. Check if feature already has a building
	hasBuilding, err := s.buildingRepo.HasBuilding(ctx, req.FeatureId)
	if err != nil {
		return fmt.Errorf("failed to check building existence: %w", err)
	}
	if hasBuilding {
		return fmt.Errorf("feature already has a building")
	}

	// 3. Get building model
	buildingModel, err := s.buildingRepo.FindBuildingModelByModelID(ctx, req.BuildingModelId)
	if err != nil {
		return fmt.Errorf("failed to find building model: %w", err)
	}
	if buildingModel == nil {
		return fmt.Errorf("building model not found")
	}

	// 4. Validate launched_satisfaction
	launchedSatisfaction, err := strconv.ParseFloat(req.LaunchedSatisfaction, 64)
	if err != nil {
		return fmt.Errorf("invalid launched_satisfaction: %w", err)
	}

	requiredSatisfaction, err := strconv.ParseFloat(buildingModel.RequiredSatisfaction, 64)
	if err != nil {
		return fmt.Errorf("invalid required_satisfaction: %w", err)
	}

	if launchedSatisfaction < requiredSatisfaction {
		return fmt.Errorf("launched_satisfaction must be at least %f", requiredSatisfaction)
	}

	// Get user wallet satisfaction
	if s.commercialClient == nil {
		return fmt.Errorf("commercial client not available")
	}
	wallet, err := s.commercialClient.GetWallet(ctx, user.UserID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}
	walletSatisfaction, err := strconv.ParseFloat(wallet.Satisfaction, 64)
	if err != nil {
		return fmt.Errorf("invalid wallet satisfaction: %w", err)
	}

	if launchedSatisfaction > walletSatisfaction {
		return fmt.Errorf("insufficient satisfaction: required %f, available %f", launchedSatisfaction, walletSatisfaction)
	}

	// 5. Validate rotation
	_, err = strconv.ParseFloat(req.Rotation, 64)
	if err != nil {
		return fmt.Errorf("invalid rotation: %w", err)
	}

	// 6. Validate position format (regex: ^(-?\d+(\.\d+)?),\s*(-?\d+(\.\d+)?)$)
	positionRegex := regexp.MustCompile(`^(-?\d+(\.\d+)?),\s*(-?\d+(\.\d+)?)$`)
	if !positionRegex.MatchString(req.Position) {
		return fmt.Errorf("invalid position format: expected 'x,y'")
	}

	// 7. Build information JSON if provided
	var informationJSON string
	if req.Information != nil {
		infoMap := make(map[string]interface{})
		if req.Information.ActivityLine != "" {
			infoMap["activity_line"] = req.Information.ActivityLine
		}
		if req.Information.Name != "" {
			infoMap["name"] = req.Information.Name
		}
		if req.Information.Address != "" {
			infoMap["address"] = req.Information.Address
		}
		if req.Information.PostalCode != "" {
			infoMap["postal_code"] = req.Information.PostalCode
		}
		if req.Information.Website != "" {
			infoMap["website"] = req.Information.Website
		}
		if req.Information.Description != "" {
			infoMap["description"] = req.Information.Description
		}

		if len(infoMap) > 0 {
			infoBytes, err := json.Marshal(infoMap)
			if err != nil {
				return fmt.Errorf("failed to marshal information: %w", err)
			}
			informationJSON = string(infoBytes)
		}

		// Create ISIC code if activity_line is provided
		if req.Information.ActivityLine != "" {
			_, err = s.buildingRepo.FirstOrCreateIsicCode(ctx, req.Information.ActivityLine)
			if err != nil {
				return fmt.Errorf("failed to create ISIC code: %w", err)
			}
		}
	}

	// 8. Calculate construction end date
	// Duration: buildingModel.required_satisfaction * 288000 / launched_satisfaction
	constructionDuration := requiredSatisfaction * 288000.0 / launchedSatisfaction
	constructionStartDate := time.Now()
	constructionEndDate := constructionStartDate.Add(time.Duration(constructionDuration) * time.Second)

	// 9. Deactivate hourly profits for this feature
	if err := s.hourlyProfitRepo.DeactivateProfitsForFeature(ctx, req.FeatureId); err != nil {
		return fmt.Errorf("failed to deactivate profits: %w", err)
	}

	// 10. Calculate bubble diameter from model attributes
	// Parse attributes JSON to extract width, length, and density (from attributes, not properties)
	var bubbleDiameter float64
	var attributes map[string]interface{}
	if err := json.Unmarshal([]byte(buildingModel.Attributes), &attributes); err == nil {
		bubbleDiameter = s.calculateBubbleDiameter(attributes)
	}

	// 12. Create building record
	err = s.buildingRepo.CreateBuilding(ctx, req.FeatureId, req.BuildingModelId,
		req.LaunchedSatisfaction, req.Rotation, req.Position, informationJSON,
		constructionStartDate, constructionEndDate, bubbleDiameter)
	if err != nil {
		// Reactivate profits on error
		s.hourlyProfitRepo.ActivateProfitsForFeature(ctx, req.FeatureId)
		return fmt.Errorf("failed to create building: %w", err)
	}

	return nil
}

// calculateBubbleDiameter calculates bubble diameter from model attributes
// Expects attributes to have 'width', 'length', and 'density'
func (s *BuildingService) calculateBubbleDiameter(attributes map[string]interface{}) float64 {
	width, widthOk := attributes["width"].(float64)
	length, lengthOk := attributes["length"].(float64)
	density, densityOk := attributes["density"].(float64)

	if !widthOk || !lengthOk || !densityOk {
		return 0.0
	}

	// Calculate based on width, length, and density
	// Formula: sqrt((width * length * density) / PI)
	area := width * length * density
	if area <= 0 {
		return 0.0
	}

	diameter := math.Sqrt(area / math.Pi)
	return diameter
}

// GetBuildings retrieves all buildings on a feature with Jalali formatted dates
func (s *BuildingService) GetBuildings(ctx context.Context, featureID uint64) ([]*pb.Building, error) {
	buildings, err := s.buildingRepo.FindByFeatureID(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get buildings: %w", err)
	}

	// Format dates to Jalali format
	for _, building := range buildings {
		if building.ConstructionStartDate != "" {
			// Try multiple date formats that MySQL might return
			dateFormats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05",
				time.RFC3339,
			}
			var t time.Time
			var err error
			for _, format := range dateFormats {
				if t, err = time.Parse(format, building.ConstructionStartDate); err == nil {
					building.ConstructionStartDate = helpers.FormatJalaliDateTime(t)
					break
				}
			}
		}
		if building.ConstructionEndDate != "" {
			dateFormats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05",
				time.RFC3339,
			}
			var t time.Time
			var err error
			for _, format := range dateFormats {
				if t, err = time.Parse(format, building.ConstructionEndDate); err == nil {
					building.ConstructionEndDate = helpers.FormatJalaliDateTime(t)
					break
				}
			}
		}
		// Format launched_satisfaction to 4 decimals
		if building.LaunchedSatisfaction != "" {
			if sat, err := strconv.ParseFloat(building.LaunchedSatisfaction, 64); err == nil {
				building.LaunchedSatisfaction = fmt.Sprintf("%.4f", sat)
			}
		}
	}

	return buildings, nil
}

// UpdateBuilding updates an existing building
func (s *BuildingService) UpdateBuilding(ctx context.Context, req *pb.UpdateBuildingRequest) (*pb.Building, error) {
	// 1. Get feature and validate ownership
	feature, _, err := s.featureRepo.FindByID(ctx, req.FeatureId)
	if err != nil {
		return nil, fmt.Errorf("feature not found: %w", err)
	}

	// Get user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: authentication required")
	}

	if feature.OwnerID != user.UserID {
		return nil, fmt.Errorf("unauthorized: user does not own this feature")
	}

	// 2. Get building model
	buildingModel, err := s.buildingRepo.FindBuildingModelByModelID(ctx, req.BuildingModelId)
	if err != nil {
		return nil, fmt.Errorf("failed to find building model: %w", err)
	}
	if buildingModel == nil {
		return nil, fmt.Errorf("building model not found")
	}

	// 3. Validate launched_satisfaction
	launchedSatisfaction, err := strconv.ParseFloat(req.LaunchedSatisfaction, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid launched_satisfaction: %w", err)
	}

	requiredSatisfaction, err := strconv.ParseFloat(buildingModel.RequiredSatisfaction, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid required_satisfaction: %w", err)
	}

	if launchedSatisfaction < requiredSatisfaction {
		return nil, fmt.Errorf("launched_satisfaction must be at least %f", requiredSatisfaction)
	}

	// Get user wallet satisfaction
	if s.commercialClient == nil {
		return nil, fmt.Errorf("commercial client not available")
	}
	wallet, err := s.commercialClient.GetWallet(ctx, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	walletSatisfaction, err := strconv.ParseFloat(wallet.Satisfaction, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet satisfaction: %w", err)
	}

	if launchedSatisfaction > walletSatisfaction {
		return nil, fmt.Errorf("insufficient satisfaction: required %f, available %f", launchedSatisfaction, walletSatisfaction)
	}

	// 4. Validate rotation
	_, err = strconv.ParseFloat(req.Rotation, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rotation: %w", err)
	}

	// 5. Validate position format
	positionRegex := regexp.MustCompile(`^(-?\d+(\.\d+)?),\s*(-?\d+(\.\d+)?)$`)
	if !positionRegex.MatchString(req.Position) {
		return nil, fmt.Errorf("invalid position format: expected 'x,y'")
	}

	// 6. Build information JSON if provided
	var informationJSON string
	if req.Information != nil {
		infoMap := make(map[string]interface{})
		if req.Information.ActivityLine != "" {
			infoMap["activity_line"] = req.Information.ActivityLine
		}
		if req.Information.Name != "" {
			infoMap["name"] = req.Information.Name
		}
		if req.Information.Address != "" {
			infoMap["address"] = req.Information.Address
		}
		if req.Information.PostalCode != "" {
			infoMap["postal_code"] = req.Information.PostalCode
		}
		if req.Information.Website != "" {
			infoMap["website"] = req.Information.Website
		}
		if req.Information.Description != "" {
			infoMap["description"] = req.Information.Description
		}

		if len(infoMap) > 0 {
			infoBytes, err := json.Marshal(infoMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal information: %w", err)
			}
			informationJSON = string(infoBytes)
		}
	}

	// 7. Recalculate construction end date using updated satisfaction
	constructionDuration := requiredSatisfaction * 288000.0 / launchedSatisfaction
	// Get existing building to preserve start date
	existingBuilding, err := s.buildingRepo.FindBuildingByFeatureAndModel(ctx, req.FeatureId, req.BuildingModelId)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing building: %w", err)
	}
	if existingBuilding == nil {
		return nil, fmt.Errorf("building not found")
	}

	// Parse start date from existing building
	// The date comes from database in MySQL datetime format: "2006-01-02 15:04:05"
	var constructionStartDate time.Time
	if existingBuilding.ConstructionStartDate != "" {
		dateFormats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			time.RFC3339,
		}
		parsed := false
		for _, format := range dateFormats {
			if t, err := time.Parse(format, existingBuilding.ConstructionStartDate); err == nil {
				constructionStartDate = t
				parsed = true
				break
			}
		}
		if !parsed {
			// If we can't parse, use current time (fallback)
			constructionStartDate = time.Now()
		}
	} else {
		constructionStartDate = time.Now()
	}

	constructionEndDate := constructionStartDate.Add(time.Duration(constructionDuration) * time.Second)

	// 8. Calculate bubble diameter
	var bubbleDiameter float64
	var attributes map[string]interface{}
	if err := json.Unmarshal([]byte(buildingModel.Attributes), &attributes); err == nil {
		bubbleDiameter = s.calculateBubbleDiameter(attributes)
	}

	// 9. Update building
	updatedBuilding, err := s.buildingRepo.UpdateBuilding(ctx, req.FeatureId, req.BuildingModelId,
		req.LaunchedSatisfaction, req.Rotation, req.Position, informationJSON,
		constructionEndDate, bubbleDiameter)
	if err != nil {
		return nil, fmt.Errorf("failed to update building: %w", err)
	}

	// Format dates to Jalali
	dateFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	if updatedBuilding.ConstructionStartDate != "" {
		for _, format := range dateFormats {
			if t, err := time.Parse(format, updatedBuilding.ConstructionStartDate); err == nil {
				updatedBuilding.ConstructionStartDate = helpers.FormatJalaliDateTime(t)
				break
			}
		}
	}
	if updatedBuilding.ConstructionEndDate != "" {
		for _, format := range dateFormats {
			if t, err := time.Parse(format, updatedBuilding.ConstructionEndDate); err == nil {
				updatedBuilding.ConstructionEndDate = helpers.FormatJalaliDateTime(t)
				break
			}
		}
	}
	if updatedBuilding.LaunchedSatisfaction != "" {
		if sat, err := strconv.ParseFloat(updatedBuilding.LaunchedSatisfaction, 64); err == nil {
			updatedBuilding.LaunchedSatisfaction = fmt.Sprintf("%.4f", sat)
		}
	}

	return updatedBuilding, nil
}

// DestroyBuilding removes a building from a feature
func (s *BuildingService) DestroyBuilding(ctx context.Context, featureID, buildingModelID uint64) error {
	// Check ownership
	feature, _, err := s.featureRepo.FindByID(ctx, featureID)
	if err != nil {
		return fmt.Errorf("feature not found: %w", err)
	}

	// Get user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthorized: authentication required")
	}

	if feature.OwnerID != user.UserID {
		return fmt.Errorf("unauthorized: user does not own this feature")
	}

	// Reactivate hourly profits when building is destroyed
	if err := s.hourlyProfitRepo.ActivateProfitsForFeature(ctx, featureID); err != nil {
		return fmt.Errorf("failed to reactivate profits: %w", err)
	}

	return s.buildingRepo.DeleteBuilding(ctx, featureID, buildingModelID)
}
