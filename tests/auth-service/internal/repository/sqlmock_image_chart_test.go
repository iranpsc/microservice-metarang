package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"metarang/auth-service/internal/repository"
)

func TestUserRepo_FormatImageURLViaLevels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	repo := repository.NewUserRepository(db, "https://admin")
	mock.ExpectQuery("FROM level_user").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "image_url"}).
			AddRow(uint64(2), "L", "l", int32(10), "lvl.png"))
	lvl, err := repo.GetUserLatestLevel(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "https://admin/uploads/lvl.png", lvl.Image)

	repo2 := repository.NewUserRepository(db, "")
	mock.ExpectQuery("FROM levels").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "image_url"}).
			AddRow(uint64(1), "P", "p", int32(0), "https://cdn/p.png"))
	levels, err := repo2.GetLevelsBelowScore(ctx, 10)
	require.NoError(t, err)
	require.Len(t, levels, 1)
	require.Equal(t, "https://cdn/p.png", levels[0].Image)
}

func TestCitizenReferralChartRanges_SQLMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := repository.NewCitizenRepository(db)
	ctx := context.Background()

	for _, rangeType := range []string{"daily", "weekly", "monthly", "yearly"} {
		mock.ExpectQuery("FROM users u").WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(9)))
		mock.ExpectQuery("FROM referral_order_histories").
			WillReturnRows(sqlmock.NewRows([]string{"total_count", "total_amount"}).AddRow(1, int64(10)))
		mock.ExpectQuery("FROM referral_order_histories").
			WillReturnRows(sqlmock.NewRows([]string{"label", "count", "total_amount"}).AddRow("lbl", int32(1), int64(10)))
		chart, err := repo.GetCitizenReferralChartData(ctx, 1, rangeType)
		require.NoError(t, err)
		require.Equal(t, "1", chart.TotalReferralsCount)
		require.Equal(t, "lbl", chart.ChartData[0].Label)
	}

	mock.ExpectQuery("FROM users u").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(9)))
	mock.ExpectQuery("FROM referral_order_histories").
		WillReturnRows(sqlmock.NewRows([]string{"total_count", "total_amount"}).AddRow(1, int64(21111)))
	mock.ExpectQuery("FROM referral_order_histories").
		WillReturnRows(sqlmock.NewRows([]string{"label", "count", "total_amount"}).AddRow("2023/01", int32(1), int64(21111)))
	chart, err := repo.GetCitizenReferralChartData(ctx, 1, "yearly")
	require.NoError(t, err)
	require.Equal(t, "1", chart.TotalReferralsCount)
	require.Equal(t, "21111", chart.TotalReferralOrdersAmount)
	require.Regexp(t, `^\d{4}/\d{2}$`, chart.ChartData[0].Label)
	require.NotEqual(t, "2023/01", chart.ChartData[0].Label)

	mock.ExpectQuery("FROM users u").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(9)))
	mock.ExpectQuery("FROM referral_order_histories").
		WillReturnRows(sqlmock.NewRows([]string{"total_count", "total_amount"}).AddRow(1, int64(5)))
	mock.ExpectQuery("FROM referral_order_histories").
		WillReturnRows(sqlmock.NewRows([]string{"label", "count", "total_amount"}).AddRow("2023/01/15", int32(1), int64(5)))
	chart, err = repo.GetCitizenReferralChartData(ctx, 1, "daily")
	require.NoError(t, err)
	require.Regexp(t, `^\d{4}/\d{2}/\d{2}$`, chart.ChartData[0].Label)
	require.NotEqual(t, "2023/01/15", chart.ChartData[0].Label)

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(25))
	mock.ExpectQuery("FROM users u").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "created_at"}).
			AddRow(uint64(2), "hm-2", "n", time.Now()))
	mock.ExpectQuery("FROM kycs").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("FROM images").WillReturnError(sql.ErrNoRows)
	refs, meta, err := repo.GetCitizenReferrals(ctx, 1, "n", 2, 10)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	require.Contains(t, meta.NextPageURL, "page=3")
	require.Contains(t, meta.PrevPageURL, "page=1")
}
