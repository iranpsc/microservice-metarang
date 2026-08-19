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

func featureCols() []string {
	return []string{"id", "owner_id", "map_id", "type", "created_at", "updated_at"}
}

func featureWithPropsCols() []string {
	return []string{
		"id", "owner_id", "map_id", "type", "created_at", "updated_at",
		"prop_id", "feature_id", "karbari", "rgb", "owner", "label", "address",
		"area", "density", "stability", "price_psc", "price_irr", "minimum_price_percentage",
		"prop_created_at", "prop_updated_at",
	}
}

func TestFeatureRepository_FindByOwner(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureRepository(db)
	now := time.Now()

	mock.ExpectQuery("FROM features").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows(featureCols()).AddRow(1, 2, 1, "polygon", now, now))

	list, err := repo.FindByOwner(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, uint64(1), list[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeatureRepository_FindByOwnerPaginated(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureRepository(db)
	now := time.Now()

	mock.ExpectQuery("FROM features f LEFT JOIN feature_properties").
		WithArgs(uint64(2), 5, 0).
		WillReturnRows(sqlmock.NewRows(featureWithPropsCols()).
			AddRow(1, 2, 1, "polygon", now, now, "p1", 1, "m", "d", "o", "l", "addr", 10.0, 1, 100.0, "0", "0", 80, now, now))

	feats, props, err := repo.FindByOwnerPaginated(context.Background(), 2, 1, "", "")
	require.NoError(t, err)
	require.Len(t, feats, 1)
	assert.Equal(t, "m", props[0].Karbari)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeatureRepository_FindByOwnerAndFeatureID(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureRepository(db)
	now := time.Now()

	mock.ExpectQuery("WHERE f.id = \\? AND f.owner_id = \\?").
		WithArgs(uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows(featureWithPropsCols()).
			AddRow(1, 2, 1, "polygon", now, now, "p1", 1, "m", "d", "o", "l", "addr", 10.0, 1, 100.0, "0", "0", 80, now, now))

	f, p, err := repo.FindByOwnerAndFeatureID(context.Background(), 2, 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), f.ID)
	assert.Equal(t, "p1", p.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeatureRepository_UpdateOwnerWithTx(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE features SET owner_id").
		WithArgs(uint64(9), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, repo.UpdateOwnerWithTx(context.Background(), tx, 1, 9))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeatureRepository_FindByBoundingBox_SQLMock(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureRepository(db)
	now := time.Now()
	points := []string{"0,1", "1,1", "0,0", "1,0"}

	mock.ExpectQuery("FROM coordinates c").
		WillReturnRows(sqlmock.NewRows([]string{"geometry_id"}))
	empty, err := repo.FindByBoundingBox(context.Background(), points, false)
	require.NoError(t, err)
	assert.Empty(t, empty)

	mock.ExpectQuery("FROM coordinates c").
		WillReturnRows(sqlmock.NewRows([]string{"geometry_id"}).AddRow(uint64(1)))
	mock.ExpectQuery("WHERE f.id IN").
		WillReturnRows(sqlmock.NewRows(featureWithPropsCols()).
			AddRow(1, 2, 1, "polygon", now, now, "p1", 1, "m", "d", "o", "l", "addr", 10.0, 1, 100.0, "0", "0", 80, now, now))
	list, err := repo.FindByBoundingBox(context.Background(), points, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
