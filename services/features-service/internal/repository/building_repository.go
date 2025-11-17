package repository

import (
	"context"
	"database/sql"

	pb "metargb/shared/pb/features"
)

type BuildingRepository struct {
	db *sql.DB
}

func NewBuildingRepository(db *sql.DB) *BuildingRepository {
	return &BuildingRepository{db: db}
}

// CreateBuilding creates a building record
func (r *BuildingRepository) CreateBuilding(ctx context.Context, req *pb.BuildFeatureRequest) error {
	// Calculate construction end date if needed (placeholder logic)
	// In real implementation, this would be calculated based on building properties
	query := `
		INSERT INTO buildings (feature_id, model_id, launched_satisfaction, rotation, position, information, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
	`

	_, err := r.db.ExecContext(ctx, query, req.FeatureId, req.BuildingModelId, req.LaunchedSatisfaction, req.Rotation, req.Position, "")
	return err
}

// FindByFeatureID retrieves all buildings for a feature
func (r *BuildingRepository) FindByFeatureID(ctx context.Context, featureID uint64) ([]*pb.Building, error) {
	query := `
		SELECT id, construction_start_date, construction_end_date, launched_satisfaction,
		       rotation, position, bubble_diameter, information
		FROM buildings
		WHERE feature_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buildings := []*pb.Building{}
	for rows.Next() {
		building := &pb.Building{}
		var constructionStartDate, constructionEndDate, launchedSatisfaction sql.NullString
		var rotation, position, bubbleDiameter, information sql.NullString
		var id uint64

		if err := rows.Scan(
			&id,
			&constructionStartDate,
			&constructionEndDate,
			&launchedSatisfaction,
			&rotation,
			&position,
			&bubbleDiameter,
			&information,
		); err != nil {
			continue
		}

		building.Id = id
		if constructionStartDate.Valid {
			building.ConstructionStartDate = constructionStartDate.String
		}
		if constructionEndDate.Valid {
			building.ConstructionEndDate = constructionEndDate.String
		}
		if launchedSatisfaction.Valid {
			building.LaunchedSatisfaction = launchedSatisfaction.String
		}
		if rotation.Valid {
			building.Rotation = rotation.String
		}
		if position.Valid {
			building.Position = position.String
		}
		if bubbleDiameter.Valid {
			building.BubbleDiameter = bubbleDiameter.String
		}
		if information.Valid {
			building.Information = information.String
		}

		buildings = append(buildings, building)
	}

	return buildings, nil
}

// UpdateBuilding updates a building
func (r *BuildingRepository) UpdateBuilding(ctx context.Context, req *pb.UpdateBuildingRequest) (*pb.Building, error) {
	query := `
		UPDATE buildings
		SET launched_satisfaction = ?, rotation = ?, position = ?, updated_at = NOW()
		WHERE feature_id = ? AND model_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, req.LaunchedSatisfaction, req.Rotation, req.Position, req.FeatureId, req.BuildingModelId)
	if err != nil {
		return nil, err
	}

	// Return updated building
	return &pb.Building{
		Id:                   0, // Would need to query to get actual ID
		LaunchedSatisfaction: req.LaunchedSatisfaction,
		Rotation:             req.Rotation,
		Position:             req.Position,
	}, nil
}

// DeleteBuilding removes a building
func (r *BuildingRepository) DeleteBuilding(ctx context.Context, featureID, buildingModelID uint64) error {
	query := "DELETE FROM buildings WHERE feature_id = ? AND model_id = ?"
	_, err := r.db.ExecContext(ctx, query, featureID, buildingModelID)
	return err
}
