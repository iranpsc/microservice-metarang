package repository

import (
	"context"
	"database/sql"
	"fmt"

	"metargb/features-service/internal/models"
)

type GeometryRepository struct {
	db *sql.DB
}

func NewGeometryRepository(db *sql.DB) *GeometryRepository {
	return &GeometryRepository{db: db}
}

// GetByFeatureID retrieves geometry data for a feature
func (r *GeometryRepository) GetByFeatureID(ctx context.Context, featureID uint64) (*models.Geometry, error) {
	geometry := &models.Geometry{}

	query := `
		SELECT g.id, g.type, g.created_at, g.updated_at
		FROM geometries g
		INNER JOIN features f ON f.geometry_id = g.id
		WHERE f.id = ?
	`

	err := r.db.QueryRowContext(ctx, query, featureID).Scan(
		&geometry.ID, &geometry.Type, &geometry.CreatedAt, &geometry.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return geometry, nil
}

// GetCoordinatesByFeatureID retrieves coordinates for a feature as "x,y" strings
func (r *GeometryRepository) GetCoordinatesByFeatureID(ctx context.Context, featureID uint64) ([]string, error) {
	query := `
		SELECT c.x, c.y
		FROM coordinates c
		INNER JOIN features f ON f.geometry_id = c.geometry_id
		WHERE f.id = ?
		ORDER BY c.id
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coordinates := []string{}
	for rows.Next() {
		var x, y float64
		if err := rows.Scan(&x, &y); err != nil {
			continue
		}
		// Format as "x,y" string
		coordinates = append(coordinates, formatCoordinate(x, y))
	}

	return coordinates, nil
}

func formatCoordinate(x, y float64) string {
	return fmt.Sprintf("%.6f,%.6f", x, y)
}

