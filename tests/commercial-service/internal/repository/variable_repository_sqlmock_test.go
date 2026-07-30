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

func newVariableRepo(t *testing.T) (repository.VariableRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewVariableRepository(db), mock
}

func TestVariableRepository_GetRate(t *testing.T) {
	repo, mock := newVariableRepo(t)

	mock.ExpectQuery("SELECT price").
		WithArgs("psc").
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(100.0))

	rate, err := repo.GetRate(context.Background(), "psc")
	require.NoError(t, err)
	assert.Equal(t, 100.0, rate)

	mock.ExpectQuery("SELECT price").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	_, err = repo.GetRate(context.Background(), "missing")
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVariableRepository_GetAllRates(t *testing.T) {
	repo, mock := newVariableRepo(t)

	mock.ExpectQuery("SELECT asset, price").
		WillReturnRows(sqlmock.NewRows([]string{"asset", "price"}).
			AddRow("psc", 100.0).
			AddRow("red", 50.0))

	rates, err := repo.GetAllRates(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 100.0, rates["psc"])
	assert.Equal(t, 50.0, rates["red"])

	require.NoError(t, mock.ExpectationsWereMet())
}
