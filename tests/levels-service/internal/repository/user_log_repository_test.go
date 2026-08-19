package repository_test

import (
	"context"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/levels-service/internal/repository"
)

func TestUserLogRepository_GetUserScore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	t.Run("ValidScore", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT score FROM users WHERE id = ?")).
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow("150.5"))

		score, err := repo.GetUserScore(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, int32(150), score)
	})

	t.Run("NullScore", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT score FROM users WHERE id = ?")).
			WithArgs(uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(nil))

		score, err := repo.GetUserScore(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, int32(0), score)
	})

	t.Run("InvalidScore", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT score FROM users WHERE id = ?")).
			WithArgs(uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow("invalid"))

		_, err := repo.GetUserScore(ctx, 3)
		assert.Error(t, err)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_GetUserLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id").
		WithArgs(uint64(5)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "transactions_count", "followers_count",
			"deposit_amount", "activity_hours", "score",
		}).AddRow(1, 5, "10", "2.5", "1.0000", "0.5", "14"))

	log, err := repo.GetUserLog(ctx, 5)
	require.NoError(t, err)
	require.NotNil(t, log)
	assert.Equal(t, uint64(5), log.UserId)
	assert.Equal(t, "10", log.TransactionsCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_UpdateScore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_logs SET score = ?, updated_at = NOW() WHERE user_id = ?")).
		WithArgs("200", uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET score = ?, updated_at = NOW() WHERE id = ?")).
		WithArgs("200", uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.UpdateScore(ctx, 7, 200)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_UpdateTransactionsCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_logs SET transactions_count = ?, updated_at = NOW() WHERE user_id = ?")).
		WithArgs("20", uint64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateTransactionsCount(ctx, 4, "20")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_IncrementDeposit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE user_logs SET deposit_amount = deposit_amount + ?, updated_at = NOW() WHERE user_id = ?")).
			WithArgs("10.0000", uint64(6)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.IncrementDeposit(ctx, 6, "100000")
		require.NoError(t, err)
	})

	t.Run("InvalidAmount", func(t *testing.T) {
		err = repo.IncrementDeposit(ctx, 6, "not-a-number")
		assert.Error(t, err)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_UpdateFollowersCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_logs SET followers_count = ?, updated_at = NOW() WHERE user_id = ?")).
		WithArgs("5.0", uint64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateFollowersCount(ctx, 9, 50)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_UpdateActivityHours(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_logs SET activity_hours = ?, updated_at = NOW() WHERE user_id = ?")).
		WithArgs("0.2", uint64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateActivityHours(ctx, 11, 90)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_GetTotalFollowers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM followers WHERE followed_id = ?")).
		WithArgs(uint64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	count, err := repo.GetTotalFollowers(ctx, 12)
	require.NoError(t, err)
	assert.Equal(t, int32(25), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_CalculateScore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id").
		WithArgs(uint64(15)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "transactions_count", "followers_count",
			"deposit_amount", "activity_hours", "score",
		}).AddRow(1, 15, "10", "2.0", "1.5", "0.5", "14"))

	score, err := repo.CalculateScore(ctx, 15)
	require.NoError(t, err)
	assert.Equal(t, int32(14), score)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserLogRepository_UpdateScore_BeginError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserLogRepository(db)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err = repo.UpdateScore(context.Background(), 1, 100)
	assert.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
