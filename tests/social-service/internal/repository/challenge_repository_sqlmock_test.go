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

func questionColumns() []string {
	return []string{
		"id", "code", "title", "image", "creator_code", "prize", "views", "participants", "created_at", "updated_at",
	}
}

func TestChallengeRepository_GetRandomUnansweredQuestion(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM questions q`).
			WithArgs(uint64(5)).
			WillReturnRows(sqlmock.NewRows(questionColumns()).
				AddRow(1, "q1", "Title", "img", "c1", 10, 2, 3, now, now))

		repo := repository.NewChallengeRepository(db)
		q, err := repo.GetRandomUnansweredQuestion(context.Background(), 5)
		require.NoError(t, err)
		require.NotNil(t, q)
		assert.Equal(t, uint64(1), q.ID)
		assert.Equal(t, "Title", q.Title)
	})

	t.Run("no rows", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM questions q`).
			WithArgs(uint64(5)).
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewChallengeRepository(db)
		q, err := repo.GetRandomUnansweredQuestion(context.Background(), 5)
		require.NoError(t, err)
		assert.Nil(t, q)
	})

	t.Run("error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM questions q`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetRandomUnansweredQuestion(context.Background(), 5)
		assert.Error(t, err)
	})
}

func TestChallengeRepository_GetQuestionByID(t *testing.T) {
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM questions`).
			WithArgs(uint64(9)).
			WillReturnRows(sqlmock.NewRows(questionColumns()).
				AddRow(9, "q9", "Q", "i", "c", 5, 1, 1, now, now))

		repo := repository.NewChallengeRepository(db)
		q, err := repo.GetQuestionByID(context.Background(), 9)
		require.NoError(t, err)
		require.NotNil(t, q)
		assert.Equal(t, uint64(9), q.ID)
	})

	t.Run("no rows", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM questions`).
			WithArgs(uint64(9)).
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewChallengeRepository(db)
		q, err := repo.GetQuestionByID(context.Background(), 9)
		require.NoError(t, err)
		assert.Nil(t, q)
	})

	t.Run("error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM questions`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetQuestionByID(context.Background(), 9)
		assert.Error(t, err)
	})
}

func TestChallengeRepository_GetAnswersByQuestionID(t *testing.T) {
	now := time.Now()

	t.Run("ok", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM answers`).
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "question_id", "title", "image", "is_correct", "created_at", "updated_at",
			}).AddRow(10, 1, "A", "", true, now, now).AddRow(11, 1, "B", "", false, now, now))

		repo := repository.NewChallengeRepository(db)
		answers, err := repo.GetAnswersByQuestionID(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, answers, 2)
		assert.True(t, answers[0].IsCorrect)
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM answers`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetAnswersByQuestionID(context.Background(), 1)
		assert.Error(t, err)
	})
}

func TestChallengeRepository_IncrementAndCreate(t *testing.T) {
	t.Run("increment views", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectExec(`UPDATE questions SET views`).
			WithArgs(uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		repo := repository.NewChallengeRepository(db)
		require.NoError(t, repo.IncrementQuestionViews(context.Background(), 1))
	})

	t.Run("increment views error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectExec(`UPDATE questions SET views`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		assert.Error(t, repo.IncrementQuestionViews(context.Background(), 1))
	})

	t.Run("increment participants", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectExec(`UPDATE questions SET participants`).
			WithArgs(uint64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		repo := repository.NewChallengeRepository(db)
		require.NoError(t, repo.IncrementQuestionParticipants(context.Background(), 1))
	})

	t.Run("increment participants error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectExec(`UPDATE questions SET participants`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		assert.Error(t, repo.IncrementQuestionParticipants(context.Background(), 1))
	})

	t.Run("create user answer", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectExec(`INSERT INTO user_question_answers`).
			WithArgs(uint64(1), uint64(2), uint64(3)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := repository.NewChallengeRepository(db)
		require.NoError(t, repo.CreateUserAnswer(context.Background(), 1, 2, 3))
	})

	t.Run("create user answer error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectExec(`INSERT INTO user_question_answers`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		assert.Error(t, repo.CreateUserAnswer(context.Background(), 1, 2, 3))
	})
}

func TestChallengeRepository_CountsAndVariables(t *testing.T) {
	t.Run("has answered true", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_question_answers`).
			WithArgs(uint64(1), uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

		repo := repository.NewChallengeRepository(db)
		ok, err := repo.HasUserAnswered(context.Background(), 1, 2)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("has answered error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_question_answers`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.HasUserAnswered(context.Background(), 1, 2)
		assert.Error(t, err)
	})

	t.Run("user answer count", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`INNER JOIN answers`).
			WithArgs(uint64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(4))

		repo := repository.NewChallengeRepository(db)
		n, err := repo.GetUserAnswerCount(context.Background(), 1, true)
		require.NoError(t, err)
		assert.Equal(t, int32(4), n)
	})

	t.Run("user answer count error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`INNER JOIN answers`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetUserAnswerCount(context.Background(), 1, false)
		assert.Error(t, err)
	})

	t.Run("total participants", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(DISTINCT user_id\)`).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT user_id)"}).AddRow(7))

		repo := repository.NewChallengeRepository(db)
		n, err := repo.GetTotalParticipantsCount(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int32(7), n)
	})

	t.Run("total participants error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`SELECT COUNT\(DISTINCT user_id\)`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetTotalParticipantsCount(context.Background())
		assert.Error(t, err)
	})

	t.Run("system variable found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM system_variables`).
			WithArgs("slug").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(20.5))

		repo := repository.NewChallengeRepository(db)
		v, err := repo.GetSystemVariable(context.Background(), "slug")
		require.NoError(t, err)
		assert.Equal(t, 20.5, v)
	})

	t.Run("system variable default", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM system_variables`).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		repo := repository.NewChallengeRepository(db)
		v, err := repo.GetSystemVariable(context.Background(), "missing")
		require.NoError(t, err)
		assert.Equal(t, 15.0, v)
	})

	t.Run("system variable error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`FROM system_variables`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetSystemVariable(context.Background(), "x")
		assert.Error(t, err)
	})

	t.Run("answer vote count", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`WHERE answer_id`).
			WithArgs(uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(8))

		repo := repository.NewChallengeRepository(db)
		n, err := repo.GetAnswerVoteCount(context.Background(), 3)
		require.NoError(t, err)
		assert.Equal(t, int32(8), n)
	})

	t.Run("answer vote count error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`WHERE answer_id`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetAnswerVoteCount(context.Background(), 3)
		assert.Error(t, err)
	})

	t.Run("question total answers", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`WHERE question_id`).
			WithArgs(uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(12))

		repo := repository.NewChallengeRepository(db)
		n, err := repo.GetQuestionTotalAnswers(context.Background(), 2)
		require.NoError(t, err)
		assert.Equal(t, int32(12), n)
	})

	t.Run("question total answers error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(`WHERE question_id`).WillReturnError(sql.ErrConnDone)

		repo := repository.NewChallengeRepository(db)
		_, err = repo.GetQuestionTotalAnswers(context.Background(), 2)
		assert.Error(t, err)
	})
}
