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

func TestCitizenFeaturesRepository_SQLMockPaths(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewCitizenFeaturesRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(2), "m").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(int32(4)))
	n, err := repo.CountOwnedByKarbari(context.Background(), 2, "m")
	require.NoError(t, err)
	assert.Equal(t, int32(4), n)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(2), "m", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(int32(1)))
	bought, err := repo.CountTradesByKarbari(context.Background(), 2, "buyer", "m", now.Add(-time.Hour), now)
	require.NoError(t, err)
	assert.Equal(t, int32(1), bought)

	mock.ExpectQuery("SELECT t.id, t.created_at").
		WithArgs(uint64(2), "m", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uint64(9), now))
	ts, err := repo.ListTradeTimestamps(context.Background(), 2, "seller", []string{"m"}, now.Add(-time.Hour), now)
	require.NoError(t, err)
	require.Len(t, ts, 1)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(2), "m").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery("SELECT f.id, fp.id").
		WithArgs(uint64(2), "m", 15, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "vod", "address", "area", "density", "karbari", "owner_code", "price_psc", "price_irr", "label",
		}).AddRow(uint64(1), "p1", "addr", 10.0, 1, "m", "hm-2", "0", "0", "l"))
	mock.ExpectQuery("GROUP BY g.feature_id").
		WillReturnRows(sqlmock.NewRows([]string{"feature_id", "cx", "cy"}).AddRow(uint64(1), 1.5, 2.5))
	mock.ExpectQuery("FROM images").
		WillReturnRows(sqlmock.NewRows([]string{"id", "imageable_id", "url"}).AddRow(uint64(4), uint64(1), "https://x"))

	items, total, err := repo.ListOwnedFeatures(context.Background(), 2, []string{"m"}, "", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "https://x", items[0].Images[0].URL)
	require.NotNil(t, items[0].Center)

	mock.ExpectQuery("SELECT f.id, COALESCE\\(fp.karbari").
		WithArgs(uint64(2), "m").
		WillReturnRows(sqlmock.NewRows([]string{"id", "karbari"}).AddRow(uint64(1), "m"))
	mock.ExpectQuery("GROUP BY g.feature_id").
		WillReturnRows(sqlmock.NewRows([]string{"feature_id", "cx", "cy"}).AddRow(uint64(1), 3.0, 4.0))
	markers, err := repo.ListMapMarkers(context.Background(), 2, []string{"m"})
	require.NoError(t, err)
	require.Len(t, markers, 1)
	require.NotNil(t, markers[0].Center)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildingRepository_CompletedSQLMock(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewBuildingRepository(db)
	now := time.Now()

	empty, err := repo.CountCompletedByKarbari(context.Background(), 2, nil, now)
	require.NoError(t, err)
	assert.Empty(t, empty)
	dates, err := repo.ListCompletedEndDates(context.Background(), 2, nil, now, now, now)
	require.NoError(t, err)
	assert.Empty(t, dates)
	n, err := repo.CountUserCompletedBuildings(context.Background(), 2, nil, now)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	rows, err := repo.ListUserCompletedBuildings(context.Background(), 2, nil, now, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, rows)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(3))
	count, err := repo.CountCompleted(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	mock.ExpectQuery("GROUP BY fp.karbari").
		WithArgs(now, uint64(2), "m").
		WillReturnRows(sqlmock.NewRows([]string{"karbari", "count"}).AddRow("m", int32(2)))
	byK, err := repo.CountCompletedByKarbari(context.Background(), 2, []string{"m"}, now)
	require.NoError(t, err)
	assert.Equal(t, int32(2), byK["m"])

	mock.ExpectQuery("SELECT b.construction_end_date").
		WithArgs(now, now.Add(-time.Hour), now, uint64(2), "m").
		WillReturnRows(sqlmock.NewRows([]string{"d"}).AddRow(now))
	dates, err = repo.ListCompletedEndDates(context.Background(), 2, []string{"m"}, now.Add(-time.Hour), now, now)
	require.NoError(t, err)
	require.Len(t, dates, 1)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(now, uint64(2), "m").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	n, err = repo.CountUserCompletedBuildings(context.Background(), 2, []string{"m"}, now)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	mock.ExpectQuery("ORDER BY b.construction_end_date DESC").
		WithArgs(now, uint64(2), "m", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"sku", "karbari", "attributes", "images", "end"}).
			AddRow("sku", "m", "[]", "[]", now))
	list, err := repo.ListUserCompletedBuildings(context.Background(), 2, []string{"m"}, now, 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildingRepository_FindByFeatureIDsAndInfo(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewBuildingRepository(db)

	got, err := repo.FindByFeatureIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	cols := []string{
		"feature_id", "id", "construction_start_date", "construction_end_date", "launched_satisfaction",
		"rotation", "position", "bubble_diameter", "information",
		"model_id", "model_model_id", "model_name", "model_sku", "model_images",
		"model_attributes", "model_file", "model_required_satisfaction",
	}
	mock.ExpectQuery("WHERE b.feature_id IN").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			uint64(1), uint64(9), "2024-01-01", "2024-02-01", "10",
			"0", "1,2", "3.5", "{}",
			uint64(7), uint64(101), "tower", "sku", "[]",
			"[]", "{}", 1.25,
		))
	byID, err := repo.FindByFeatureIDs(context.Background(), []uint64{1})
	require.NoError(t, err)
	require.Len(t, byID[1], 1)
	assert.Equal(t, uint64(9), byID[1][0].Id)
	require.NotNil(t, byID[1][0].Model)
	assert.Equal(t, "tower", byID[1][0].Model.Name)

	findCols := cols[1:]
	mock.ExpectQuery("WHERE b.feature_id = \\?").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(findCols).AddRow(
			uint64(9), "2024-01-01", "2024-02-01", "10",
			"0", "1,2", "3.5", "{}",
			uint64(7), uint64(101), "tower", "sku", "[]",
			"[]", "{}", 1.25,
		))
	one, err := repo.FindByFeatureID(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, one, 1)

	mock.ExpectQuery("FROM building_models").
		WithArgs(uint64(101), uint64(101)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "name", "sku", "images", "attributes", "file", "required_satisfaction",
		}).AddRow(uint64(7), uint64(101), "n", "s", "[]", "[]", "{}", 1.0))
	mock.ExpectExec("SET information").
		WithArgs(`{"name":"n"}`, uint64(1), uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.UpdateBuildingInformation(context.Background(), 1, "101", `{"name":"n"}`))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGeometryRepository_EmptyIDs(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewGeometryRepository(db)
	got, err := repo.GetCoordinatesByFeatureIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsicCodeRepository_SQLMock(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewIsicCodeRepository(db)

	mock.ExpectQuery("FROM isic_codes").
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "verified"}).
			AddRow(uint64(1), "shop", int64(11), true).
			AddRow(uint64(2), "empty", nil, false))
	items, err := repo.FindPaginated(context.Background(), "", 10, 0)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.NotNil(t, items[0].Code)

	mock.ExpectQuery("FROM isic_codes").
		WithArgs("%ab%", "%ab%", 5, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "verified"}))
	items, err = repo.FindPaginated(context.Background(), "ab", 5, 0)
	require.NoError(t, err)
	assert.Empty(t, items)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM isic_codes").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(3))
	n, err := repo.Count(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	require.NoError(t, mock.ExpectationsWereMet())
}
