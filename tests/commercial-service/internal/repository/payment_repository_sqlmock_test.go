package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPaymentRepo(t *testing.T) (repository.PaymentRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewPaymentRepository(db), mock
}

func TestPaymentRepository_Create(t *testing.T) {
	repo, mock := newPaymentRepo(t)

	mock.ExpectExec("INSERT INTO payments").
		WillReturnResult(sqlmock.NewResult(21, 1))

	payment := &models.Payment{
		UserID: 1, RefID: 100, CardPan: "1234", Gateway: "sadad", Amount: 50000, Product: "psc",
	}
	require.NoError(t, repo.Create(context.Background(), payment))
	assert.Equal(t, uint64(21), payment.ID)

	mock.ExpectExec("INSERT INTO payments").
		WillReturnError(assert.AnError)
	require.Error(t, repo.Create(context.Background(), &models.Payment{UserID: 2}))

	mock.ExpectExec("INSERT INTO payments").
		WillReturnResult(sqlmock.NewErrorResult(assert.AnError))
	require.Error(t, repo.Create(context.Background(), &models.Payment{UserID: 3}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPaymentRepository_FindLatestByUserID(t *testing.T) {
	repo, mock := newPaymentRepo(t)
	now := time.Now()

	mock.ExpectQuery("SELECT id, user_id, ref_id, card_pan, gateway, amount, product, created_at, updated_at").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "ref_id", "card_pan", "gateway", "amount", "product", "created_at", "updated_at",
		}).AddRow(21, 1, 100, "1234", "sadad", 50000.0, "psc", now, now))

	payment, err := repo.FindLatestByUserID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, int64(100), payment.RefID)
	assert.Equal(t, "sadad", payment.Gateway)

	mock.ExpectQuery("SELECT id, user_id, ref_id, card_pan, gateway, amount, product, created_at, updated_at").
		WithArgs(uint64(99)).
		WillReturnError(sql.ErrNoRows)
	payment, err = repo.FindLatestByUserID(context.Background(), 99)
	require.NoError(t, err)
	assert.Nil(t, payment)

	mock.ExpectQuery("SELECT id, user_id, ref_id, card_pan, gateway, amount, product, created_at, updated_at").
		WithArgs(uint64(2)).
		WillReturnError(assert.AnError)
	_, err = repo.FindLatestByUserID(context.Background(), 2)
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
