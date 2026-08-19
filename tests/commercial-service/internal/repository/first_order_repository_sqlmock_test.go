package repository_test

import (
	"context"
	"testing"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFirstOrderRepo(t *testing.T) (repository.FirstOrderRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewFirstOrderRepository(db), mock
}

func TestFirstOrderRepository_CreateAndHasFirstOrder(t *testing.T) {
	repo, mock := newFirstOrderRepo(t)

	mock.ExpectExec("INSERT INTO first_orders").
		WillReturnResult(sqlmock.NewResult(7, 1))
	fo := &models.FirstOrder{UserID: 1, Type: "psc", Amount: 10, Date: "1404/01/01", Bonus: 5}
	require.NoError(t, repo.Create(context.Background(), fo))
	assert.Equal(t, uint64(7), fo.ID)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) > 0").
		WithArgs(uint64(1), "psc").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	has, err := repo.HasFirstOrder(context.Background(), 1, "psc")
	require.NoError(t, err)
	assert.True(t, has)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	count, err := repo.Count(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	require.NoError(t, mock.ExpectationsWereMet())
}
