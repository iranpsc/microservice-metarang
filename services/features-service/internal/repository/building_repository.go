package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pb "metargb/shared/pb/features"
)

type BuildingRepository struct {
	db *sql.DB
}

func NewBuildingRepository(db *sql.DB) *BuildingRepository {
	return &BuildingRepository{db: db}
}

// UpsertBuildingModel upserts a building model from 3D API into building_models table
func (r *BuildingRepository) UpsertBuildingModel(ctx context.Context, modelID string, name, sku string, images, attributes, file string, requiredSatisfaction float64) error {
	// Parse modelID to uint64 for the database
	var dbModelID uint64
	_, err := fmt.Sscanf(modelID, "%d", &dbModelID)
	if err != nil {
		return fmt.Errorf("invalid model_id: %w", err)
	}

	query := `
		INSERT INTO building_models (model_id, name, sku, images, attributes, file, required_satisfaction, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			sku = VALUES(sku),
			images = VALUES(images),
			attributes = VALUES(attributes),
			file = VALUES(file),
			required_satisfaction = VALUES(required_satisfaction),
			updated_at = NOW()
	`

	_, err = r.db.ExecContext(ctx, query, dbModelID, name, sku, images, attributes, file, requiredSatisfaction)
	return err
}

// FindBuildingModelByModelID finds a building model by its model_id (from 3D API)
func (r *BuildingRepository) FindBuildingModelByModelID(ctx context.Context, modelID uint64) (*pb.BuildingModel, error) {
	query := `
		SELECT id, model_id, name, sku, images, attributes, file, required_satisfaction
		FROM building_models
		WHERE model_id = ?
	`

	var id, dbModelID uint64
	var name, sku, images, attributes, file string
	var requiredSatisfaction float64

	err := r.db.QueryRowContext(ctx, query, modelID).Scan(
		&id, &dbModelID, &name, &sku, &images, &attributes, &file, &requiredSatisfaction,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find building model: %w", err)
	}

	return &pb.BuildingModel{
		Id:                   id,
		ModelId:              fmt.Sprintf("%d", dbModelID),
		Name:                 name,
		Sku:                  sku,
		Images:               images,
		Attributes:           attributes,
		File:                 file,
		RequiredSatisfaction: fmt.Sprintf("%.4f", requiredSatisfaction),
	}, nil
}

// HasBuilding checks if a feature already has a building
func (r *BuildingRepository) HasBuilding(ctx context.Context, featureID uint64) (bool, error) {
	query := `SELECT COUNT(*) FROM buildings WHERE feature_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check building existence: %w", err)
	}
	return count > 0, nil
}

// CreateBuilding creates a building record with all required fields
func (r *BuildingRepository) CreateBuilding(ctx context.Context, featureID, buildingModelID uint64, launchedSatisfaction, rotation, position, information string, constructionStartDate, constructionEndDate time.Time, bubbleDiameter float64) error {
	query := `
		INSERT INTO buildings (
			feature_id, model_id, construction_start_date, construction_end_date,
			launched_satisfaction, information, rotation, position, bubble_diameter,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		featureID, buildingModelID, constructionStartDate, constructionEndDate,
		launchedSatisfaction, information, rotation, position, bubbleDiameter,
	)
	return err
}

// FindByFeatureID retrieves all buildings for a feature with building model data
func (r *BuildingRepository) FindByFeatureID(ctx context.Context, featureID uint64) ([]*pb.Building, error) {
	query := `
		SELECT 
			b.id, 
			b.construction_start_date, 
			b.construction_end_date, 
			b.launched_satisfaction,
			b.rotation, 
			b.position, 
			b.bubble_diameter, 
			b.information,
			bm.id as model_id,
			bm.model_id as model_model_id,
			bm.name as model_name,
			bm.sku as model_sku,
			bm.images as model_images,
			bm.attributes as model_attributes,
			bm.file as model_file,
			bm.required_satisfaction as model_required_satisfaction
		FROM buildings b
		INNER JOIN building_models bm ON b.model_id = bm.id
		WHERE b.feature_id = ?
		ORDER BY b.id ASC
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
		var modelID, modelModelID uint64
		var modelName, modelSKU, modelImages, modelAttributes, modelFile sql.NullString
		var modelRequiredSatisfaction sql.NullFloat64

		if err := rows.Scan(
			&id,
			&constructionStartDate,
			&constructionEndDate,
			&launchedSatisfaction,
			&rotation,
			&position,
			&bubbleDiameter,
			&information,
			&modelID,
			&modelModelID,
			&modelName,
			&modelSKU,
			&modelImages,
			&modelAttributes,
			&modelFile,
			&modelRequiredSatisfaction,
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

		// Build BuildingModel
		model := &pb.BuildingModel{
			Id: modelID,
		}
		if modelModelID > 0 {
			model.ModelId = fmt.Sprintf("%d", modelModelID)
		}
		if modelName.Valid {
			model.Name = modelName.String
		}
		if modelSKU.Valid {
			model.Sku = modelSKU.String
		}
		if modelImages.Valid {
			model.Images = modelImages.String
		}
		if modelAttributes.Valid {
			model.Attributes = modelAttributes.String
		}
		if modelFile.Valid {
			model.File = modelFile.String
		}
		if modelRequiredSatisfaction.Valid {
			model.RequiredSatisfaction = fmt.Sprintf("%.4f", modelRequiredSatisfaction.Float64)
		}

		building.Model = model
		buildings = append(buildings, building)
	}

	return buildings, nil
}

// UpdateBuilding updates a building and returns the updated building with model data
func (r *BuildingRepository) UpdateBuilding(ctx context.Context, featureID, buildingModelID uint64, launchedSatisfaction, rotation, position, information string, constructionEndDate time.Time, bubbleDiameter float64) (*pb.Building, error) {
	query := `
		UPDATE buildings
		SET launched_satisfaction = ?, rotation = ?, position = ?, information = ?,
		    construction_end_date = ?, bubble_diameter = ?, updated_at = NOW()
		WHERE feature_id = ? AND model_id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		launchedSatisfaction, rotation, position, information,
		constructionEndDate, bubbleDiameter, featureID, buildingModelID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update building: %w", err)
	}

	// Return the updated building by querying it back
	return r.FindBuildingByFeatureAndModel(ctx, featureID, buildingModelID)
}

// FindBuildingByFeatureAndModel finds a building by feature_id and model_id
func (r *BuildingRepository) FindBuildingByFeatureAndModel(ctx context.Context, featureID, buildingModelID uint64) (*pb.Building, error) {
	query := `
		SELECT 
			b.id, 
			b.construction_start_date, 
			b.construction_end_date, 
			b.launched_satisfaction,
			b.rotation, 
			b.position, 
			b.bubble_diameter, 
			b.information,
			bm.id as model_id,
			bm.model_id as model_model_id,
			bm.name as model_name,
			bm.sku as model_sku,
			bm.images as model_images,
			bm.attributes as model_attributes,
			bm.file as model_file,
			bm.required_satisfaction as model_required_satisfaction
		FROM buildings b
		INNER JOIN building_models bm ON b.model_id = bm.id
		WHERE b.feature_id = ? AND b.model_id = ?
		LIMIT 1
	`

	var building pb.Building
	var constructionStartDate, constructionEndDate, launchedSatisfaction sql.NullString
	var rotation, position, bubbleDiameter, information sql.NullString
	var id, modelID, modelModelID uint64
	var modelName, modelSKU, modelImages, modelAttributes, modelFile sql.NullString
	var modelRequiredSatisfaction sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, featureID, buildingModelID).Scan(
		&id,
		&constructionStartDate,
		&constructionEndDate,
		&launchedSatisfaction,
		&rotation,
		&position,
		&bubbleDiameter,
		&information,
		&modelID,
		&modelModelID,
		&modelName,
		&modelSKU,
		&modelImages,
		&modelAttributes,
		&modelFile,
		&modelRequiredSatisfaction,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find building: %w", err)
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

	// Build BuildingModel
	model := &pb.BuildingModel{
		Id: modelID,
	}
	if modelModelID > 0 {
		model.ModelId = fmt.Sprintf("%d", modelModelID)
	}
	if modelName.Valid {
		model.Name = modelName.String
	}
	if modelSKU.Valid {
		model.Sku = modelSKU.String
	}
	if modelImages.Valid {
		model.Images = modelImages.String
	}
	if modelAttributes.Valid {
		model.Attributes = modelAttributes.String
	}
	if modelFile.Valid {
		model.File = modelFile.String
	}
	if modelRequiredSatisfaction.Valid {
		model.RequiredSatisfaction = fmt.Sprintf("%.4f", modelRequiredSatisfaction.Float64)
	}

	building.Model = model
	return &building, nil
}

// DeleteBuilding removes a building
func (r *BuildingRepository) DeleteBuilding(ctx context.Context, featureID, buildingModelID uint64) error {
	query := "DELETE FROM buildings WHERE feature_id = ? AND model_id = ?"
	_, err := r.db.ExecContext(ctx, query, featureID, buildingModelID)
	if err != nil {
		return fmt.Errorf("failed to delete building: %w", err)
	}
	return nil
}

// FirstOrCreateIsicCode finds or creates an ISIC code by name (activity_line)
func (r *BuildingRepository) FirstOrCreateIsicCode(ctx context.Context, activityLine string) (uint64, error) {
	// First try to find existing
	var id uint64
	query := `SELECT id FROM isic_codes WHERE name = ? LIMIT 1`
	err := r.db.QueryRowContext(ctx, query, activityLine).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query isic code: %w", err)
	}

	// Create new
	insertQuery := `INSERT INTO isic_codes (name, verified, created_at, updated_at) VALUES (?, 0, NOW(), NOW())`
	result, err := r.db.ExecContext(ctx, insertQuery, activityLine)
	if err != nil {
		return 0, fmt.Errorf("failed to create isic code: %w", err)
	}

	insertID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get isic code id: %w", err)
	}

	return uint64(insertID), nil
}
