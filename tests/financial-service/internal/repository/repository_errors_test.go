package repository_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"metarang/financial-service/internal/models"
	"metarang/financial-service/internal/repository"
)

type lastInsertIDErrorResult struct{}

func (lastInsertIDErrorResult) LastInsertId() (int64, error) { return 0, errors.New("no insert id") }
func (lastInsertIDErrorResult) RowsAffected() (int64, error) { return 1, nil }

var _ driver.Result = lastInsertIDErrorResult{}

func TestOrderRepository_ErrorPaths(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()
	order := &models.Order{UserID: 1, Asset: "psc", Amount: 1, Status: -138}

	mock.ExpectExec("INSERT INTO orders").WillReturnError(errors.New("insert fail"))
	require.Error(t, repo.Create(ctx, order))

	mock.ExpectExec("INSERT INTO orders").WillReturnResult(lastInsertIDErrorResult{})
	require.Error(t, repo.Create(ctx, order))

	mock.ExpectQuery("SELECT id, user_id").WithArgs(uint64(1)).WillReturnError(errors.New("scan fail"))
	_, err = repo.FindByID(ctx, 1)
	require.Error(t, err)

	mock.ExpectQuery("FROM orders o").WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
	o, u, err := repo.FindByIDWithUser(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, o)
	require.Nil(t, u)

	mock.ExpectQuery("FROM orders o").WithArgs(uint64(2)).WillReturnError(errors.New("join fail"))
	_, _, err = repo.FindByIDWithUser(ctx, 2)
	require.Error(t, err)

	mock.ExpectExec("UPDATE orders").WillReturnError(errors.New("update fail"))
	require.Error(t, repo.Update(ctx, &models.Order{ID: 1}))

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectExec("UPDATE orders").WillReturnError(errors.New("tx update fail"))
	require.Error(t, repo.UpdateWithTx(ctx, tx, &models.Order{ID: 1}))

	mock.ExpectExec("DELETE FROM orders").WillReturnError(errors.New("delete fail"))
	require.Error(t, repo.Delete(ctx, 1))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepository_ErrorPaths(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := repository.NewTransactionRepository(db)
	ctx := context.Background()
	txModel := &models.Transaction{ID: "tx-1", UserID: 1, Asset: "psc", Amount: 1, Action: "deposit", Status: 1}

	mock.ExpectExec("INSERT INTO transactions").WillReturnError(errors.New("insert fail"))
	require.Error(t, repo.Create(ctx, txModel))

	mock.ExpectExec("UPDATE transactions").WillReturnError(errors.New("update fail"))
	require.Error(t, repo.Update(ctx, txModel))

	mock.ExpectBegin()
	sqlTx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectExec("UPDATE transactions").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.UpdateWithTx(ctx, sqlTx, txModel))

	mock.ExpectExec("DELETE FROM transactions").WillReturnError(errors.New("delete fail"))
	require.Error(t, repo.Delete(ctx, "tx-1"))

	mock.ExpectQuery("FROM transactions WHERE id").WithArgs("tx-1").WillReturnError(sql.ErrNoRows)
	found, err := repo.FindByID(ctx, "tx-1")
	require.NoError(t, err)
	require.Nil(t, found)

	mock.ExpectQuery("FROM transactions WHERE id").WithArgs("tx-2").WillReturnError(errors.New("find fail"))
	_, err = repo.FindByID(ctx, "tx-2")
	require.Error(t, err)

	mock.ExpectQuery("FROM transactions WHERE payable_type").WithArgs("order", uint64(5)).WillReturnError(sql.ErrNoRows)
	found, err = repo.FindByPayable(ctx, "order", 5)
	require.NoError(t, err)
	require.Nil(t, found)

	mock.ExpectQuery("FROM transactions WHERE payable_type").WithArgs("order", uint64(6)).WillReturnError(errors.New("payable fail"))
	_, err = repo.FindByPayable(ctx, "order", 6)
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPaymentAndFirstOrderAndOthers_ErrorPaths(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	pay := repository.NewPaymentRepository(db)
	mock.ExpectExec("INSERT INTO payments").WillReturnError(errors.New("pay fail"))
	require.Error(t, pay.Create(ctx, &models.Payment{UserID: 1}))

	mock.ExpectExec("INSERT INTO payments").WillReturnResult(lastInsertIDErrorResult{})
	require.Error(t, pay.Create(ctx, &models.Payment{UserID: 1}))

	fo := repository.NewFirstOrderRepository(db)
	mock.ExpectBegin()
	sqlTx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectExec("INSERT INTO first_orders").WillReturnResult(sqlmock.NewResult(9, 1))
	record := &models.FirstOrder{UserID: 1, Type: "psc", Amount: 1, Date: "1403/01/01"}
	require.NoError(t, fo.CreateWithTx(ctx, sqlTx, record))
	require.Equal(t, uint64(9), record.ID)

	mock.ExpectExec("INSERT INTO first_orders").WillReturnError(errors.New("fo fail"))
	require.Error(t, fo.Create(ctx, record))

	mock.ExpectQuery("SELECT COUNT").WithArgs(uint64(1)).WillReturnError(errors.New("count fail"))
	_, err = fo.Count(ctx, 1)
	require.Error(t, err)

	vars := repository.NewVariableRepository(db)
	mock.ExpectQuery("SELECT price FROM variables").WithArgs("psc").WillReturnError(errors.New("rate fail"))
	_, err = vars.GetRate(ctx, "psc")
	require.Error(t, err)

	img := repository.NewImageRepository(db)
	mock.ExpectQuery("SELECT url FROM images").WithArgs("option", uint64(1)).WillReturnError(errors.New("img fail"))
	_, err = img.FindImageURLByImageable(ctx, "option", 1)
	require.Error(t, err)

	opt := repository.NewOptionRepository(db)
	mock.ExpectQuery("FROM options WHERE code IN").WithArgs("aa").WillReturnError(errors.New("opt fail"))
	_, err = opt.FindByCodes(ctx, []string{"aa"})
	require.Error(t, err)

	elig := repository.NewEligibilityRepository(db)
	mock.ExpectQuery("SELECT birthdate FROM kycs").WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
	bd, err := elig.GetUserBirthdate(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, bd)

	mock.ExpectQuery("SELECT birthdate FROM kycs").WithArgs(uint64(2)).WillReturnError(errors.New("kyc fail"))
	_, err = elig.GetUserBirthdate(ctx, 2)
	require.Error(t, err)

	nullRows := sqlmock.NewRows([]string{"birthdate"}).AddRow(nil)
	mock.ExpectQuery("SELECT birthdate FROM kycs").WithArgs(uint64(3)).WillReturnRows(nullRows)
	bd, err = elig.GetUserBirthdate(ctx, 3)
	require.NoError(t, err)
	require.Nil(t, bd)

	mock.ExpectQuery("SELECT verified, BFR").WithArgs(uint64(4)).WillReturnError(errors.New("perm fail"))
	_, _, _, err = elig.GetChildPermissions(ctx, 4)
	require.Error(t, err)

	_ = time.Now()
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByIDWithUser_NullPhone(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := repository.NewOrderRepository(db)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "asset", "amount", "status", "created_at", "updated_at",
		"id", "name", "email", "phone", "birthdate",
	}).AddRow(1, 5, "psc", 100.0, 1, time.Now(), time.Now(), 5, "User", "u@x.com", nil, nil)
	mock.ExpectQuery("FROM orders o").WithArgs(uint64(1)).WillReturnRows(rows)

	order, user, err := repo.FindByIDWithUser(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, order)
	require.NotNil(t, user)
	require.Empty(t, user.Phone)
	require.Nil(t, user.Birthdate)
	require.NoError(t, mock.ExpectationsWereMet())
}
