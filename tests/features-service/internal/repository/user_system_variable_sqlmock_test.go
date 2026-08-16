package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"metarang/features-service/internal/repository"
	"metarang/features-service/tests/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_GetUserCreatedAt(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewUserRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT created_at FROM users").
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))
	got, err := repo.GetUserCreatedAt(context.Background(), 9)
	require.NoError(t, err)
	assert.True(t, got.Equal(now) || got.Sub(now) < time.Second)

	mock.ExpectQuery("SELECT created_at FROM users").
		WithArgs(uint64(8)).
		WillReturnError(sql.ErrNoRows)
	got, err = repo.GetUserCreatedAt(context.Background(), 8)
	require.NoError(t, err)
	assert.True(t, got.IsZero())

	mock.ExpectQuery("SELECT created_at FROM users").
		WithArgs(uint64(7)).
		WillReturnError(sql.ErrConnDone)
	_, err = repo.GetUserCreatedAt(context.Background(), 7)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemVariableRepository_GetByKey(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewSystemVariableRepository(db)

	mock.ExpectQuery("FROM system_variables").
		WithArgs("public_pricing_limit").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("80"))
	n, err := repo.GetByKey(context.Background(), "public_pricing_limit")
	require.NoError(t, err)
	assert.Equal(t, 80, n)

	mock.ExpectQuery("FROM system_variables").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	n, err = repo.GetByKey(context.Background(), "missing")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	mock.ExpectQuery("FROM system_variables").
		WithArgs("bad").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("nope"))
	_, err = repo.GetByKey(context.Background(), "bad")
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemVariableRepository_GetPricingLimits(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewSystemVariableRepository(db)

	mock.ExpectQuery("public_pricing_limit").
		WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("70", "120"))
	pub, under, err := repo.GetPricingLimits(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 70, pub)
	assert.Equal(t, 120, under)

	mock.ExpectQuery("public_pricing_limit").
		WillReturnError(sql.ErrNoRows)
	pub, under, err = repo.GetPricingLimits(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 80, pub)
	assert.Equal(t, 110, under)

	mock.ExpectQuery("public_pricing_limit").
		WillReturnRows(sqlmock.NewRows([]string{"public_limit", "under_18_limit"}).AddRow("x", "y"))
	pub, under, err = repo.GetPricingLimits(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 80, pub)
	assert.Equal(t, 110, under)
	require.NoError(t, mock.ExpectationsWereMet())
}
