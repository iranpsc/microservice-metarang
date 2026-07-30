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

func newTransactionRepo(t *testing.T) (repository.TransactionRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewTransactionRepository(db), mock
}

func TestTransactionRepository_CreateAndFind(t *testing.T) {
	repo, mock := newTransactionRepo(t)
	now := time.Now()
	payableType := `App\Models\Order`
	payableID := uint64(5)

	mock.ExpectExec("INSERT INTO transactions").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Create(context.Background(), &models.Transaction{
		ID: "TR-1", UserID: 1, Asset: "psc", Amount: 10, Action: "deposit", Status: 1,
	}))

	mock.ExpectQuery("SELECT id, user_id, asset").
		WithArgs("TR-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "asset", "amount", "action", "status", "token", "ref_id", "payable_type", "payable_id", "created_at", "updated_at",
		}).AddRow("TR-1", 1, "psc", 10.0, "deposit", 1, nil, nil, &payableType, &payableID, now, now))

	tx, err := repo.FindByID(context.Background(), "TR-1")
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, "TR-1", tx.ID)

	mock.ExpectQuery("SELECT id, user_id, asset").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	tx, err = repo.FindByID(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, tx)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_UpdateAndLatest(t *testing.T) {
	repo, mock := newTransactionRepo(t)
	now := time.Now()

	mock.ExpectExec("UPDATE transactions").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), &models.Transaction{
		ID: "TR-1", UserID: 1, Asset: "psc", Amount: 10, Action: "deposit", Status: 1,
	}))

	mock.ExpectQuery("SELECT id, user_id, asset").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "asset", "amount", "action", "status", "token", "ref_id", "payable_type", "payable_id", "created_at", "updated_at",
		}).AddRow("TR-1", 1, "psc", 10.0, "deposit", 1, nil, nil, nil, nil, now, now))

	tx, err := repo.FindLatestByUserID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, tx)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_FindByUserID(t *testing.T) {
	repo, mock := newTransactionRepo(t)
	now := time.Now()

	mock.ExpectQuery("SELECT id, user_id, asset").
		WithArgs(uint64(1), "TR-1", int32(1), "deposit", "psc", `App\Models\Order`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "asset", "amount", "action", "status", "token", "ref_id", "payable_type", "payable_id", "created_at", "updated_at",
		}).AddRow("TR-1", 1, "psc", 10.0, "deposit", 1, nil, nil, nil, nil, now, now).
			AddRow("TR-2", 1, "psc", 5.0, "deposit", 1, nil, nil, nil, nil, now, now))

	txs, err := repo.FindByUserID(context.Background(), 1, map[string]interface{}{
		"search":   "TR-1",
		"status":   []int32{1},
		"action":   "deposit",
		"asset":    "psc",
		"type":     "order",
		"per_page": 15,
		"page":     1,
	})
	require.NoError(t, err)
	assert.Len(t, txs, 2)

	mock.ExpectQuery("SELECT id, user_id, asset").
		WithArgs(uint64(2), "psc", "irr", `App\Models\Trade`, `App\Models\Order`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "asset", "amount", "action", "status", "token", "ref_id", "payable_type", "payable_id", "created_at", "updated_at",
		}))

	_, err = repo.FindByUserID(context.Background(), 2, map[string]interface{}{
		"asset": "psc, irr",
		"type":  "trade,order",
	})
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
