package repository_test

import (
	"context"
	"testing"
	"time"

	"metarang/features-service/internal/repository"
	"metarang/features-service/tests/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeometryRepository_GetByFeatureID(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewGeometryRepository(db)
	now := time.Now()

	mock.ExpectQuery("FROM geometries g").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "created_at", "updated_at"}).
			AddRow(3, "Polygon", now, now))

	g, err := repo.GetByFeatureID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), g.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGeometryRepository_GetCoordinatesByFeatureID(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewGeometryRepository(db)

	mock.ExpectQuery("SELECT c.x, c.y").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"x", "y"}).AddRow(1.5, 2.5))

	coords, err := repo.GetCoordinatesByFeatureID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.500000,2.500000"}, coords)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGeometryRepository_GetByFeatureIDs(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewGeometryRepository(db)
	now := time.Now()

	empty, err := repo.GetByFeatureIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	mock.ExpectQuery("g.feature_id IN").
		WillReturnRows(sqlmock.NewRows([]string{"id", "feature_id", "type", "created_at", "updated_at"}).
			AddRow(3, 1, "Polygon", now, now))

	got, err := repo.GetByFeatureIDs(context.Background(), []uint64{1})
	require.NoError(t, err)
	assert.Equal(t, uint64(3), got[1].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGeometryRepository_GetCoordinatesByFeatureIDs(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewGeometryRepository(db)

	mock.ExpectQuery("SELECT g.feature_id, c.id, c.geometry_id, c.x, c.y").
		WillReturnRows(sqlmock.NewRows([]string{"feature_id", "id", "geometry_id", "x", "y"}).
			AddRow(uint64(1), uint64(9), uint64(3), "1.5", "2.5"))

	got, err := repo.GetCoordinatesByFeatureIDs(context.Background(), []uint64{1})
	require.NoError(t, err)
	require.Len(t, got[1], 1)
	assert.Equal(t, 1.5, got[1][0].X)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGeometryRepository_GetCoordinatesWithIDs(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewGeometryRepository(db)

	mock.ExpectQuery("SELECT c.id, c.geometry_id, c.x, c.y").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "geometry_id", "x", "y"}).
			AddRow(uint64(9), uint64(3), "1.5", "2.5"))

	coords, err := repo.GetCoordinatesWithIDs(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, coords, 1)
	assert.Equal(t, uint64(9), coords[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
