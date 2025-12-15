package models

import (
	"database/sql"
	"time"
)

// NewNullString creates a sql.NullString from a string
func NewNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// Map represents the maps table
type Map struct {
	ID                      uint64         `db:"id"`
	Name                    string         `db:"name"`
	Karbari                 string         `db:"karbari"`
	PublishDate             time.Time      `db:"publish_date"`
	PublisherName           string         `db:"publisher_name"`
	PolygonCount            int64          `db:"polygon_count"`
	TotalArea               int64          `db:"total_area"`
	FirstID                 string         `db:"first_id"`
	LastID                  string         `db:"last_id"`
	Status                  int            `db:"status"`
	FileName                string         `db:"fileName"`
	CentralPointCoordinates sql.NullString `db:"central_point_coordinates"` // JSON string
	BorderCoordinates       sql.NullString `db:"border_coordinates"`        // JSON string
	PolygonArea             uint64         `db:"polygon_area"`
	PolygonAddress          sql.NullString `db:"polygon_address"`
	PolygonColor            sql.NullString `db:"polygon_color"`
}

// MapFeature represents a feature with minimal info needed for map calculations
type MapFeature struct {
	ID      uint64 `db:"id"`
	OwnerID uint64 `db:"owner_id"`
	Karbari string `db:"karbari"`
}
