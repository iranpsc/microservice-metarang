package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"metarang/features-service/internal/repository"
	"metarang/features-service/tests/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureLimitRepository_GetLimitationByPropertyID(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureLimitRepository(db)
	now := time.Now()

	cols := []string{
		"id", "title", "start_date", "end_date", "start_id", "end_id",
		"price_limit", "verified_kyc_limit", "under_18_limit", "more_than_18_limit",
		"dynasty_owner_limit", "individual_buy_limit", "individual_buy_count", "expired",
		"created_at", "updated_at",
	}
	mock.ExpectQuery("FROM feature_limits").
		WithArgs("p1", "p1").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			uint64(3), "campaign", now, now, "a", "z",
			true, false, false, true, false, true, 2, false, now, now,
		))

	limit, err := repo.GetLimitationByPropertyID(context.Background(), "p1")
	require.NoError(t, err)
	assert.Equal(t, "campaign", limit.Title)

	mock.ExpectQuery("FROM feature_limits").
		WithArgs("missing", "missing").
		WillReturnError(sql.ErrNoRows)
	limit, err = repo.GetLimitationByPropertyID(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, limit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeatureLimitRepository_TrackLimitedPurchase(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewFeatureLimitRepository(db)

	mock.ExpectExec("INSERT INTO limited_feature_purchases").
		WithArgs(uint64(2), uint64(3), uint64(5)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.TrackLimitedPurchase(context.Background(), 2, 3, 5))
	require.NoError(t, mock.ExpectationsWereMet())
}
