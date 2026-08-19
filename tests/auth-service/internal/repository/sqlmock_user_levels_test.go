package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/repository"
)

func TestProfileLimitationRepository_SQLMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := repository.NewProfileLimitationRepository(db)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("FROM profile_limitations").WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
	got, err := repo.FindByID(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, got)

	mock.ExpectExec("INSERT INTO profile_limitations").WillReturnResult(sqlmock.NewResult(5, 1))
	lim := &models.ProfileLimitation{
		LimiterUserID: 1, LimitedUserID: 2,
		Options: models.ProfileLimitationOptions{Follow: true},
	}
	require.NoError(t, repo.Create(ctx, lim))
	require.Equal(t, uint64(5), lim.ID)

	opts := `{"follow":true,"send_message":false,"share":false,"send_ticket":false,"view_profile_images":false,"view_features_locations":false}`
	cols := []string{"id", "limiter_user_id", "limited_user_id", "options", "note", "created_at", "updated_at"}
	mock.ExpectQuery("FROM profile_limitations").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(uint64(5), uint64(1), uint64(2), opts, nil, now, now))
	pair, err := repo.FindByLimiterAndLimited(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, pair)

	mock.ExpectQuery("FROM profile_limitations").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(uint64(5), uint64(1), uint64(2), opts, nil, now, now))
	between, err := repo.FindBetweenUsers(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, between)

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	exists, err := repo.ExistsForLimiterAndLimited(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, exists)

	mock.ExpectExec("UPDATE profile_limitations").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(ctx, lim))

	mock.ExpectExec("DELETE FROM profile_limitations").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Delete(ctx, 5))
}

func TestUserRepository_ListAndLevels_SQLMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := repository.NewUserRepository(db, "https://admin.example")
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("FROM users u").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "email", "phone", "password", "code", "referrer_id", "score", "ip",
			"last_seen", "email_verified_at", "phone_verified_at", "access_token",
			"refresh_token", "token_type", "expires_in", "created_at", "updated_at",
			"fname", "lname", "profile_photo_url",
		}).AddRow(
			uint64(1), "n", "e@x.com", nil, "h", "hm-1", nil, int32(5), "ip",
			nil, nil, nil, nil, nil, nil, nil, now, now,
			"F", "L", "/p.jpg",
		))
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	users, total, err := repo.ListUsers(ctx, "", "", 1, 20)
	require.NoError(t, err)
	require.Equal(t, int32(1), total)
	require.Len(t, users, 1)

	mock.ExpectQuery("FROM level_user").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "id", "name", "slug", "score", "image_url"}).
			AddRow(uint64(1), uint64(2), "L", "l", int32(10), "/l.png"))
	levels, err := repo.GetUsersLevelsForList(ctx, []uint64{1})
	require.NoError(t, err)
	require.NotNil(t, levels[1])

	empty, err := repo.GetUsersLevelsForList(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	mock.ExpectQuery("FROM level_user").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "image_url", "gem_png_file"}).
			AddRow(uint64(2), "L", "l", int32(10), "/l.png", "/gems/l.png"))
	lvl, err := repo.GetUserLatestLevel(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, lvl)
	require.Equal(t, "https://admin.example/uploads/gems/l.png", lvl.GemPngFile)

	mock.ExpectQuery("FROM levels").
		WithArgs(int32(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "image_url", "gem_png_file"}).
			AddRow(uint64(1), "P", "p", int32(1), "/p.png", "/gems/p.png"))
	below, err := repo.GetLevelsBelowScore(ctx, 10)
	require.NoError(t, err)
	require.Len(t, below, 1)
	require.Equal(t, "https://admin.example/uploads/gems/p.png", below[0].GemPngFile)

	mock.ExpectQuery("FROM levels").
		WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(int64(100)))
	next, err := repo.GetNextLevelScore(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, int32(100), next)

	mock.ExpectQuery("FROM features").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"m", "t", "a"}).AddRow(int64(1), int64(2), int64(3)))
	m, tj, a, err := repo.GetFeatureCounts(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int32(1), m)
	require.Equal(t, int32(2), tj)
	require.Equal(t, int32(3), a)

	mock.ExpectQuery("FROM images").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("/a.jpg").AddRow("/b.jpg"))
	urls, err := repo.GetAllProfilePhotoURLs(ctx, 1)
	require.NoError(t, err)
	require.Len(t, urls, 2)

	mock.ExpectQuery("SELECT id FROM users WHERE code").
		WithArgs("hm-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(1)))
	userCols := []string{
		"id", "name", "email", "phone", "password", "code", "referrer_id", "score", "ip",
		"last_seen", "email_verified_at", "phone_verified_at", "access_token",
		"refresh_token", "token_type", "expires_in", "wallet_address", "created_at", "updated_at",
	}
	mock.ExpectQuery("FROM users").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			uint64(1), "n", "e@x.com", nil, "h", "hm-1", nil, int32(5), "ip",
			nil, nil, nil, nil, nil, nil, nil, nil, now, now,
		))
	u, err := repo.FindByCode(ctx, "hm-1")
	require.NoError(t, err)
	require.Equal(t, "hm-1", u.Code)
}
