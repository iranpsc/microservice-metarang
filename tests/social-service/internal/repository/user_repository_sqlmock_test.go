package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/social-service/internal/repository"
)

func TestUserRepository_GetUserBasicInfo(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "profile_photo"}).
				AddRow(7, "Alice", "a1", "https://cdn.example.com/p.jpg"))

		repo := repository.NewUserRepository(db)
		info, err := repo.GetUserBasicInfo(context.Background(), 7)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "Alice", info.Name)
		assert.Equal(t, "a1", info.Code)
		assert.Equal(t, "https://cdn.example.com/p.jpg", info.ProfilePhoto)
	})

	t.Run("null profile photo", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "profile_photo"}).
				AddRow(7, "Alice", "a1", nil))

		repo := repository.NewUserRepository(db)
		info, err := repo.GetUserBasicInfo(context.Background(), 7)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "", info.ProfilePhoto)
	})

	t.Run("no rows", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM users`).
			WithArgs(uint64(7)).
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewUserRepository(db)
		info, err := repo.GetUserBasicInfo(context.Background(), 7)
		require.NoError(t, err)
		assert.Nil(t, info)
	})

	t.Run("error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM users`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewUserRepository(db)
		_, err = repo.GetUserBasicInfo(context.Background(), 7)
		assert.Error(t, err)
	})
}

func TestUserRepository_GetUserLevel(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT score FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(100))
		mock.ExpectQuery(`FROM levels`).
			WithArgs(int32(100)).
			WillReturnRows(sqlmock.NewRows([]string{"slug"}).AddRow("gold"))

		repo := repository.NewUserRepository(db)
		level, err := repo.GetUserLevel(context.Background(), 7)
		require.NoError(t, err)
		assert.Equal(t, "gold", level)
	})

	t.Run("user missing", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT score FROM users`).
			WithArgs(uint64(7)).
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewUserRepository(db)
		level, err := repo.GetUserLevel(context.Background(), 7)
		require.NoError(t, err)
		assert.Equal(t, "", level)
	})

	t.Run("score error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT score FROM users`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewUserRepository(db)
		_, err = repo.GetUserLevel(context.Background(), 7)
		assert.Error(t, err)
	})

	t.Run("level missing", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT score FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(10))
		mock.ExpectQuery(`FROM levels`).
			WithArgs(int32(10)).
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewUserRepository(db)
		level, err := repo.GetUserLevel(context.Background(), 7)
		require.NoError(t, err)
		assert.Equal(t, "", level)
	})

	t.Run("level error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT score FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(10))
		mock.ExpectQuery(`FROM levels`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewUserRepository(db)
		_, err = repo.GetUserLevel(context.Background(), 7)
		assert.Error(t, err)
	})
}

func TestUserRepository_IsUserOnline(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT last_seen FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"last_seen"}).AddRow(time.Now()))

		repo := repository.NewUserRepository(db)
		ok, err := repo.IsUserOnline(context.Background(), 7)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("offline", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT last_seen FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"last_seen"}).AddRow(time.Now().Add(-10 * time.Minute)))

		repo := repository.NewUserRepository(db)
		ok, err := repo.IsUserOnline(context.Background(), 7)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("null last_seen", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT last_seen FROM users`).
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"last_seen"}).AddRow(nil))

		repo := repository.NewUserRepository(db)
		ok, err := repo.IsUserOnline(context.Background(), 7)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("no rows", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT last_seen FROM users`).
			WithArgs(uint64(7)).
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewUserRepository(db)
		ok, err := repo.IsUserOnline(context.Background(), 7)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT last_seen FROM users`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewUserRepository(db)
		_, err = repo.IsUserOnline(context.Background(), 7)
		assert.Error(t, err)
	})
}
