package repository_test

import (
	"context"
	"testing"

	"metarang/features-service/internal/repository"
	"metarang/features-service/tests/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPropertiesRepository_Update(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewPropertiesRepository(db)

	require.NoError(t, repo.Update(context.Background(), 1, map[string]interface{}{}))

	mock.ExpectExec("UPDATE feature_properties SET").
		WithArgs("new-label", uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Update(context.Background(), 1, map[string]interface{}{"label": "new-label"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPropertiesRepository_UpdateStatus(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewPropertiesRepository(db)

	mock.ExpectExec("SET rgb = \\?, owner = \\?, label = \\?, minimum_price_percentage").
		WithArgs("a", "owner", "label", 80, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateStatus(context.Background(), 1, "a", "owner", "label", 80))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPropertiesRepository_UpdateStatusWithTx(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewPropertiesRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("SET rgb = \\?, owner = \\?, label = \\?, minimum_price_percentage").
		WithArgs("a", "owner", "label", 80, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, repo.UpdateStatusWithTx(context.Background(), tx, 1, "a", "owner", "label", 80))
	require.NoError(t, mock.ExpectationsWereMet())
}
