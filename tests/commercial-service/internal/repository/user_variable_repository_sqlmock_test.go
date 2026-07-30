package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"metarang/commercial-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserVariableRepo(t *testing.T) (repository.UserVariableRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewUserVariableRepository(db), mock
}

func TestUserVariableRepository_GetReferralProfitLimit(t *testing.T) {
	repo, mock := newUserVariableRepo(t)

	mock.ExpectQuery("SELECT referral_profit").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"referral_profit"}).AddRow(15000000.0))

	limit, err := repo.GetReferralProfitLimit(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 15000000.0, limit)

	mock.ExpectQuery("SELECT referral_profit").
		WithArgs(uint64(2)).
		WillReturnError(sql.ErrNoRows)
	limit, err = repo.GetReferralProfitLimit(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 0.0, limit)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserVariableRepository_GetWithdrawProfit(t *testing.T) {
	repo, mock := newUserVariableRepo(t)

	mock.ExpectQuery("SELECT withdraw_profit").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"withdraw_profit"}).AddRow(10))

	days, err := repo.GetWithdrawProfit(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 10, days)

	mock.ExpectQuery("SELECT withdraw_profit").
		WithArgs(uint64(2)).
		WillReturnError(sql.ErrNoRows)
	days, err = repo.GetWithdrawProfit(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 7, days)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserVariableRepository_Create(t *testing.T) {
	repo, mock := newUserVariableRepo(t)

	mock.ExpectQuery("SELECT id FROM user_variables").
		WithArgs(uint64(1)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO user_variables").
		WithArgs(uint64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.Create(context.Background(), 1))

	mock.ExpectQuery("SELECT id FROM user_variables").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
	require.NoError(t, repo.Create(context.Background(), 2))

	require.NoError(t, mock.ExpectationsWereMet())
}
