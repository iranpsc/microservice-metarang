package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"metargb/features-service/internal/models"
)

type FeatureRepository struct {
	db *sql.DB
}

func NewFeatureRepository(db *sql.DB) *FeatureRepository {
	return &FeatureRepository{db: db}
}

// FindByID retrieves a feature by ID with its properties
func (r *FeatureRepository) FindByID(ctx context.Context, id uint64) (*models.Feature, *models.FeatureProperties, error) {
	feature := &models.Feature{}
	properties := &models.FeatureProperties{}

	query := `
		SELECT f.id, f.owner_id, f.geometry_id, f.dynasty_id, f.created_at, f.updated_at,
		       fp.id as prop_id, fp.feature_id, fp.karbari, fp.rgb, fp.owner, fp.label,
		       fp.area, fp.stability, fp.price_psc, fp.price_irr, fp.minimum_price_percentage,
		       fp.created_at as prop_created_at, fp.updated_at as prop_updated_at
		FROM features f
		LEFT JOIN feature_properties fp ON f.id = fp.feature_id
		WHERE f.id = ?
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&feature.ID, &feature.OwnerID, &feature.GeometryID, &feature.DynastyID,
		&feature.CreatedAt, &feature.UpdatedAt,
		&properties.ID, &properties.FeatureID, &properties.Karbari, &properties.RGB,
		&properties.Owner, &properties.Label, &properties.Area, &properties.Stability,
		&properties.PricePSC, &properties.PriceIRR, &properties.MinimumPricePercentage,
		&properties.CreatedAt, &properties.UpdatedAt,
	)

	if err != nil {
		return nil, nil, err
	}

	return feature, properties, nil
}

// FindByBoundingBox implements Laravel's FeatureRepository@all logic
// Points format: ["minX,minY", "maxX,minY", "maxX,maxY", "minX,maxY"]
func (r *FeatureRepository) FindByBoundingBox(ctx context.Context, points []string, loadBuildings bool) ([]*models.Feature, error) {
	if len(points) != 4 {
		return nil, fmt.Errorf("expected 4 points, got %d", len(points))
	}

	// Parse points
	parsePoint := func(point string) (float64, float64, error) {
		parts := strings.Split(point, ",")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid point format: %s", point)
		}
		x, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, 0, err
		}
		y, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, 0, err
		}
		return x, y, nil
	}

	// Extract bounds
	minX, minY, err := parsePoint(points[0])
	if err != nil {
		return nil, err
	}
	maxX, maxY, err := parsePoint(points[2])
	if err != nil {
		return nil, err
	}

	// Query coordinates table for features within bounds
	// Matches Laravel: whereBetween('x', [minX, maxX])->whereBetween('y', [minY, maxY])
	query := `
		SELECT DISTINCT c.geometry_id
		FROM coordinates c
		WHERE c.x BETWEEN ? AND ?
		  AND c.y BETWEEN ? AND ?
	`

	rows, err := r.db.QueryContext(ctx, query, minX, maxX, minY, maxY)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	geometryIDs := []uint64{}
	for rows.Next() {
		var geoID uint64
		if err := rows.Scan(&geoID); err != nil {
			continue
		}
		geometryIDs = append(geometryIDs, geoID)
	}

	if len(geometryIDs) == 0 {
		return []*models.Feature{}, nil
	}

	// Convert to string for IN clause
	idStrs := make([]string, len(geometryIDs))
	for i, id := range geometryIDs {
		idStrs[i] = fmt.Sprintf("%d", id)
	}

	// Load features with properties
	featureQuery := `
		SELECT f.id, f.owner_id, f.geometry_id, f.dynasty_id, f.created_at, f.updated_at
		FROM features f
		WHERE f.geometry_id IN (` + strings.Join(idStrs, ",") + `)
	`

	featureRows, err := r.db.QueryContext(ctx, featureQuery)
	if err != nil {
		return nil, err
	}
	defer featureRows.Close()

	features := []*models.Feature{}
	for featureRows.Next() {
		feature := &models.Feature{}
		if err := featureRows.Scan(
			&feature.ID, &feature.OwnerID, &feature.GeometryID,
			&feature.DynastyID, &feature.CreatedAt, &feature.UpdatedAt,
		); err != nil {
			continue
		}
		features = append(features, feature)
	}

	return features, nil
}

// FindByOwner retrieves all features owned by a user
func (r *FeatureRepository) FindByOwner(ctx context.Context, ownerID uint64) ([]*models.Feature, error) {
	query := `
		SELECT id, owner_id, geometry_id, dynasty_id, created_at, updated_at
		FROM features
		WHERE owner_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	features := []*models.Feature{}
	for rows.Next() {
		feature := &models.Feature{}
		if err := rows.Scan(
			&feature.ID, &feature.OwnerID, &feature.GeometryID,
			&feature.DynastyID, &feature.CreatedAt, &feature.UpdatedAt,
		); err != nil {
			continue
		}
		features = append(features, feature)
	}

	return features, nil
}

// UpdateOwner transfers ownership
func (r *FeatureRepository) UpdateOwner(ctx context.Context, featureID, newOwnerID uint64) error {
	query := "UPDATE features SET owner_id = ?, updated_at = NOW() WHERE id = ?"
	_, err := r.db.ExecContext(ctx, query, newOwnerID, featureID)
	return err
}

// IsLocked checks if a feature is locked
func (r *FeatureRepository) IsLocked(ctx context.Context, featureID uint64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM locked_features
			WHERE feature_id = ? AND status = 0
		)
	`

	var locked bool
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&locked)
	return locked, err
}

// HasPendingBuyRequests checks if feature has pending buy requests
func (r *FeatureRepository) HasPendingBuyRequests(ctx context.Context, featureID uint64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM buy_feature_requests
			WHERE feature_id = ? AND deleted_at IS NULL
		)
	`

	var hasPending bool
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&hasPending)
	return hasPending, err
}

