package repository_test

import (
	"context"
	"testing"

	"metarang/features-service/internal/repository"
	"metarang/features-service/tests/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageRepository_GetImagesByFeatureID(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewImageRepository(db)

	mock.ExpectQuery("FROM images").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "url"}).AddRow(uint64(4), "https://img/1.jpg"))

	imgs, err := repo.GetImagesByFeatureID(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, imgs, 1)
	assert.Equal(t, "https://img/1.jpg", imgs[0].URL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImageRepository_CreateImage(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewImageRepository(db)

	mock.ExpectExec("INSERT INTO images").
		WithArgs(uint64(1), "https://img/1.jpg").
		WillReturnResult(sqlmock.NewResult(4, 1))

	img, err := repo.CreateImage(context.Background(), 1, "https://img/1.jpg")
	require.NoError(t, err)
	assert.Equal(t, uint64(4), img.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImageRepository_DeleteImage(t *testing.T) {
	db, mock := testutil.NewSQLMock(t)
	repo := repository.NewImageRepository(db)

	mock.ExpectExec("DELETE FROM images").
		WithArgs(uint64(4), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.DeleteImage(context.Background(), 1, 4))

	mock.ExpectExec("DELETE FROM images").
		WithArgs(uint64(4), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err := repo.DeleteImage(context.Background(), 1, 4)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	require.NoError(t, mock.ExpectationsWereMet())
}
