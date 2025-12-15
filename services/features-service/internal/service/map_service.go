package service

import (
	"context"
	"errors"
	"fmt"

	"metargb/features-service/internal/models"
	"metargb/features-service/internal/repository"
	pb "metargb/shared/pb/features"
	"metargb/shared/pkg/helpers"
)

type MapService struct {
	mapRepo     *repository.MapRepository
	featureRepo *repository.FeatureRepository
}

func NewMapService(
	mapRepo *repository.MapRepository,
	featureRepo *repository.FeatureRepository,
) *MapService {
	return &MapService{
		mapRepo:     mapRepo,
		featureRepo: featureRepo,
	}
}

// ListMaps retrieves all maps with sold_features_percentage calculated
func (s *MapService) ListMaps(ctx context.Context) ([]*pb.Map, error) {
	maps, err := s.mapRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list maps: %w", err)
	}

	result := make([]*pb.Map, 0, len(maps))
	for _, m := range maps {
		// Load features for this map
		features, err := s.mapRepo.FindFeaturesByMapID(ctx, m.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load features for map %d: %w", m.ID, err)
		}

		// Calculate sold_features_percentage
		soldPercentage := s.calculateSoldPercentage(features)

		// Convert to protobuf (list view - no detail fields)
		pbMap := s.mapToPBList(m, soldPercentage)
		result = append(result, pbMap)
	}

	return result, nil
}

// GetMap retrieves a single map with all detail fields
func (s *MapService) GetMap(ctx context.Context, mapID uint64) (*pb.Map, error) {
	m, err := s.mapRepo.FindByID(ctx, mapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get map: %w", err)
	}
	if m == nil {
		return nil, errors.New("map not found")
	}

	// Load features for this map
	features, err := s.mapRepo.FindFeaturesByMapID(ctx, mapID)
	if err != nil {
		return nil, fmt.Errorf("failed to load features for map %d: %w", mapID, err)
	}

	// Calculate sold_features_percentage
	soldPercentage := s.calculateSoldPercentage(features)

	// Calculate feature counts by karbari
	featureCounts := s.calculateFeatureCountsByKarbari(features)

	// Convert to protobuf (detail view)
	return s.mapToPBDetail(m, soldPercentage, featureCounts), nil
}

// GetMapBorder retrieves just the border coordinates for a map
func (s *MapService) GetMapBorder(ctx context.Context, mapID uint64) (string, error) {
	m, err := s.mapRepo.FindByID(ctx, mapID)
	if err != nil {
		return "", fmt.Errorf("failed to get map: %w", err)
	}
	if m == nil {
		return "", errors.New("map not found")
	}

	if !m.BorderCoordinates.Valid {
		return "", nil
	}

	return m.BorderCoordinates.String, nil
}

// calculateSoldPercentage calculates the percentage of features sold (owner_id != 1)
func (s *MapService) calculateSoldPercentage(features []*models.MapFeature) string {
	if len(features) == 0 {
		return "0.00"
	}

	soldCount := 0
	for _, f := range features {
		if f.OwnerID != 1 {
			soldCount++
		}
	}

	percentage := float64(soldCount) / float64(len(features)) * 100.0
	return fmt.Sprintf("%.2f", percentage)
}

// calculateFeatureCountsByKarbari calculates sold feature counts by karbari type
func (s *MapService) calculateFeatureCountsByKarbari(features []*models.MapFeature) *pb.MapFeatures {
	counts := &pb.MapFeatures{
		Maskoni:   &pb.MapFeatureCount{Sold: 0},
		Tejari:    &pb.MapFeatureCount{Sold: 0},
		Amoozeshi: &pb.MapFeatureCount{Sold: 0},
	}

	for _, f := range features {
		// Only count sold features (owner_id != 1)
		if f.OwnerID == 1 {
			continue
		}

		switch f.Karbari {
		case "m":
			counts.Maskoni.Sold++
		case "t":
			counts.Tejari.Sold++
		case "a":
			counts.Amoozeshi.Sold++
		}
	}

	return counts
}

// mapToPBList converts Map model to protobuf for list view
func (s *MapService) mapToPBList(m *models.Map, soldPercentage string) *pb.Map {
	pbMap := &pb.Map{
		Id:                     m.ID,
		Name:                   m.Name,
		SoldFeaturesPercentage: soldPercentage,
	}

	if m.PolygonColor.Valid {
		pbMap.Color = m.PolygonColor.String
	}

	if m.CentralPointCoordinates.Valid {
		pbMap.CentralPointCoordinates = m.CentralPointCoordinates.String
	}

	return pbMap
}

// mapToPBDetail converts Map model to protobuf for detail view
func (s *MapService) mapToPBDetail(m *models.Map, soldPercentage string, featureCounts *pb.MapFeatures) *pb.Map {
	pbMap := s.mapToPBList(m, soldPercentage)

	// Add detail fields
	if m.BorderCoordinates.Valid {
		pbMap.BorderCoordinates = m.BorderCoordinates.String
	}

	pbMap.Area = m.PolygonArea

	if m.PolygonAddress.Valid {
		pbMap.Address = m.PolygonAddress.String
	}

	// Format publish_date as Jalali date
	pbMap.PublishedAt = helpers.FormatJalaliDate(m.PublishDate)

	// Add feature counts
	pbMap.Features = featureCounts

	return pbMap
}
