package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"metarang/features-service/internal/repository"
	"metarang/features-service/tests/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableRepository_GetRateWithCache(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewVariableRepository(db)

	mock.ExpectQuery("SELECT value FROM variables").
		WithArgs("psc").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(2.5))

	assert.Equal(t, 2.5, repo.GetRateWithCache(context.Background(), "psc"))
	assert.Equal(t, 2.5, repo.GetRateWithCache(context.Background(), "psc"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVariableRepository_GetRate_SQLErrorDefaultsToOne(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewVariableRepository(db)

	mock.ExpectQuery("SELECT value FROM variables").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	assert.Equal(t, 1.0, repo.GetRateWithCache(context.Background(), "missing"))
	require.NoError(t, mock.ExpectationsWereMet())
}
