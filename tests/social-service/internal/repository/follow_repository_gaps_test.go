package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/social-service/internal/repository"
)

func TestFollowRepository_GetFollowers_ScanAndRowsErr(t *testing.T) {
	t.Run("scan error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT follower_id FROM follows`).
			WithArgs(uint64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"follower_id"}).AddRow(nil))

		repo := repository.NewFollowRepository(db)
		_, err = repo.GetFollowers(context.Background(), 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan follower")
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"follower_id"}).
			AddRow(uint64(1)).
			RowError(0, errors.New("iter"))
		mock.ExpectQuery(`SELECT follower_id FROM follows`).
			WithArgs(uint64(10)).
			WillReturnRows(rows)

		repo := repository.NewFollowRepository(db)
		_, err = repo.GetFollowers(context.Background(), 10)
		assert.Error(t, err)
	})
}

func TestFollowRepository_GetFollowing_ScanAndRowsErr(t *testing.T) {
	t.Run("scan error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT following_id FROM follows`).
			WithArgs(uint64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"following_id"}).AddRow(nil))

		repo := repository.NewFollowRepository(db)
		_, err = repo.GetFollowing(context.Background(), 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan following")
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"following_id"}).
			AddRow(uint64(3)).
			RowError(0, errors.New("iter"))
		mock.ExpectQuery(`SELECT following_id FROM follows`).
			WithArgs(uint64(10)).
			WillReturnRows(rows)

		repo := repository.NewFollowRepository(db)
		_, err = repo.GetFollowing(context.Background(), 10)
		assert.Error(t, err)
	})
}
