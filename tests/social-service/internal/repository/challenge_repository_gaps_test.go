package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/social-service/internal/repository"
)

func TestChallengeRepository_GetAnswersByQuestionID_ScanAndRowsErr(t *testing.T) {
	t.Run("scan error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM answers`).
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "question_id", "title", "image", "is_correct", "created_at", "updated_at",
			}).AddRow(nil, 1, "A", "", true, time.Now(), time.Now()))

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetAnswersByQuestionID(context.Background(), 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan answer")
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "question_id", "title", "image", "is_correct", "created_at", "updated_at",
		}).AddRow(10, 1, "A", "", true, now, now).RowError(0, errors.New("iter"))
		mock.ExpectQuery(`FROM answers`).
			WithArgs(uint64(1)).
			WillReturnRows(rows)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetAnswersByQuestionID(context.Background(), 1)
		assert.Error(t, err)
	})
}

func TestChallengeRepository_HasUserAnswered_False(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_question_answers`).
		WithArgs(uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

	repo := repository.NewChallengeRepository(db)
	ok, err := repo.HasUserAnswered(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.False(t, ok)
}
