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

func newOrderRepo(t *testing.T) (repository.OrderRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewOrderRepository(db), mock
}

func TestOrderRepository_Create(t *testing.T) {
	repo, mock := newOrderRepo(t)

	mock.ExpectExec("INSERT INTO orders").
		WillReturnResult(sqlmock.NewResult(11, 1))

	order := &models.Order{UserID: 1, Asset: "psc", Amount: 10.5, Status: 1}
	require.NoError(t, repo.Create(context.Background(), order))
	assert.Equal(t, uint64(11), order.ID)

	mock.ExpectExec("INSERT INTO orders").
		WillReturnError(assert.AnError)
	require.Error(t, repo.Create(context.Background(), &models.Order{UserID: 2, Asset: "irr", Amount: 1, Status: 0}))

	mock.ExpectExec("INSERT INTO orders").
		WillReturnResult(sqlmock.NewErrorResult(assert.AnError))
	require.Error(t, repo.Create(context.Background(), &models.Order{UserID: 3, Asset: "psc", Amount: 1, Status: 0}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_FindByID(t *testing.T) {
	repo, mock := newOrderRepo(t)
	now := time.Now()

	mock.ExpectQuery("SELECT id, user_id, asset, amount, status, created_at, updated_at").
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "asset", "amount", "status", "created_at", "updated_at",
		}).AddRow(11, 1, "psc", 10.5, 1, now, now))

	order, err := repo.FindByID(context.Background(), 11)
	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, uint64(11), order.ID)
	assert.Equal(t, "psc", order.Asset)
	assert.Equal(t, 10.5, order.Amount)

	mock.ExpectQuery("SELECT id, user_id, asset, amount, status, created_at, updated_at").
		WithArgs(uint64(99)).
		WillReturnError(sql.ErrNoRows)
	order, err = repo.FindByID(context.Background(), 99)
	require.NoError(t, err)
	assert.Nil(t, order)

	mock.ExpectQuery("SELECT id, user_id, asset, amount, status, created_at, updated_at").
		WithArgs(uint64(1)).
		WillReturnError(assert.AnError)
	_, err = repo.FindByID(context.Background(), 1)
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_Update(t *testing.T) {
	repo, mock := newOrderRepo(t)

	mock.ExpectExec("UPDATE orders").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), &models.Order{ID: 11, Status: 2}))

	mock.ExpectExec("UPDATE orders").
		WillReturnError(assert.AnError)
	require.Error(t, repo.Update(context.Background(), &models.Order{ID: 11, Status: 3}))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_FindLatestByUserID(t *testing.T) {
	repo, mock := newOrderRepo(t)
	now := time.Now()

	mock.ExpectQuery("SELECT id, user_id, asset, amount, status, created_at, updated_at").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "asset", "amount", "status", "created_at", "updated_at",
		}).AddRow(11, 1, "irr", 20, 1, now, now))

	order, err := repo.FindLatestByUserID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, "irr", order.Asset)

	mock.ExpectQuery("SELECT id, user_id, asset, amount, status, created_at, updated_at").
		WithArgs(uint64(99)).
		WillReturnError(sql.ErrNoRows)
	order, err = repo.FindLatestByUserID(context.Background(), 99)
	require.NoError(t, err)
	assert.Nil(t, order)

	mock.ExpectQuery("SELECT id, user_id, asset, amount, status, created_at, updated_at").
		WithArgs(uint64(2)).
		WillReturnError(assert.AnError)
	_, err = repo.FindLatestByUserID(context.Background(), 2)
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
