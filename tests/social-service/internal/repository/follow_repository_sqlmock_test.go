package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/social-service/internal/repository"
)

func TestFollowRepository_Create_OK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`INSERT INTO follows`).
		WithArgs(uint64(1), uint64(2), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := repository.NewFollowRepository(db)
	require.NoError(t, repo.Create(context.Background(), 1, 2))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`INSERT INTO follows`).
		WillReturnError(sql.ErrConnDone)

	repo := repository.NewFollowRepository(db)
	assert.Error(t, repo.Create(context.Background(), 1, 2))
}

func TestFollowRepository_Delete_OK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`DELETE FROM follows`).
		WithArgs(uint64(1), uint64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := repository.NewFollowRepository(db)
	require.NoError(t, repo.Delete(context.Background(), 1, 2))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_Delete_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`DELETE FROM follows`).WillReturnError(sql.ErrConnDone)

	repo := repository.NewFollowRepository(db)
	assert.Error(t, repo.Delete(context.Background(), 1, 2))
}

func TestFollowRepository_Exists(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM follows`).
			WithArgs(uint64(1), uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

		repo := repository.NewFollowRepository(db)
		ok, err := repo.Exists(context.Background(), 1, 2)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("false", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM follows`).
			WithArgs(uint64(1), uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

		repo := repository.NewFollowRepository(db)
		ok, err := repo.Exists(context.Background(), 1, 2)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM follows`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewFollowRepository(db)
		_, err = repo.Exists(context.Background(), 1, 2)
		assert.Error(t, err)
	})
}

func TestFollowRepository_GetFollowers(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT follower_id FROM follows`).
			WithArgs(uint64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"follower_id"}).AddRow(1).AddRow(2))

		repo := repository.NewFollowRepository(db)
		ids, err := repo.GetFollowers(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, []uint64{1, 2}, ids)
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT follower_id FROM follows`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewFollowRepository(db)
		_, err = repo.GetFollowers(context.Background(), 10)
		assert.Error(t, err)
	})
}

func TestFollowRepository_GetFollowing(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT following_id FROM follows`).
			WithArgs(uint64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"following_id"}).AddRow(3).AddRow(4))

		repo := repository.NewFollowRepository(db)
		ids, err := repo.GetFollowing(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, []uint64{3, 4}, ids)
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT following_id FROM follows`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewFollowRepository(db)
		_, err = repo.GetFollowing(context.Background(), 10)
		assert.Error(t, err)
	})
}
