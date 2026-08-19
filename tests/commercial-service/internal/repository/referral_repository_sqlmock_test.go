package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReferralRepo(t *testing.T) (repository.ReferralRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewReferralRepository(db), mock
}

func TestReferralRepository_GetReferrerID(t *testing.T) {
	repo, mock := newReferralRepo(t)

	mock.ExpectQuery("SELECT referrer_id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"referrer_id"}).AddRow(2))

	id, err := repo.GetReferrerID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, uint64(2), *id)

	mock.ExpectQuery("SELECT referrer_id").
		WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"referrer_id"}).AddRow(nil))
	id, err = repo.GetReferrerID(context.Background(), 3)
	require.NoError(t, err)
	assert.Nil(t, id)

	mock.ExpectQuery("SELECT referrer_id").
		WithArgs(uint64(4)).
		WillReturnError(sql.ErrNoRows)
	id, err = repo.GetReferrerID(context.Background(), 4)
	require.NoError(t, err)
	assert.Nil(t, id)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferralRepository_TotalAndCreate(t *testing.T) {
	repo, mock := newReferralRepo(t)

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount\\), 0\\)").
		WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(50.0))

	total, err := repo.GetTotalReferredAmount(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 50.0, total)

	mock.ExpectExec("INSERT INTO referral_order_histories").
		WillReturnResult(sqlmock.NewResult(10, 1))
	history := &models.ReferralOrderHistory{UserID: 2, ReferralID: 1, Amount: 5}
	require.NoError(t, repo.CreateReferralOrder(context.Background(), history))
	assert.Equal(t, uint64(10), history.ID)

	require.NoError(t, mock.ExpectationsWereMet())
}
