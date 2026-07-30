package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTransactionRepo struct {
	transactions []*models.Transaction
	latest       *models.Transaction
	findErr      error
	createErr    error
}

func (m *mockTransactionRepo) Create(ctx context.Context, transaction *models.Transaction) error {
	return m.createErr
}

func (m *mockTransactionRepo) Update(ctx context.Context, transaction *models.Transaction) error {
	return nil
}

func (m *mockTransactionRepo) FindByID(ctx context.Context, id string) (*models.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionRepo) FindLatestByUserID(ctx context.Context, userID uint64) (*models.Transaction, error) {
	return m.latest, m.findErr
}

func (m *mockTransactionRepo) FindByUserID(ctx context.Context, userID uint64, filters map[string]interface{}) ([]*models.Transaction, error) {
	return m.transactions, m.findErr
}

type stubJalaliConverter struct{}

func (stubJalaliConverter) NowJalali() string { return "1404/01/01" }

func (stubJalaliConverter) FormatJalaliDate(t time.Time) string {
	return "1404/05/15"
}

func (stubJalaliConverter) FormatJalaliTime(t time.Time) string {
	return "14:30:05"
}

func TestTransactionService_ListTransactions(t *testing.T) {
	payableType := `App\Models\Trade`
	now := time.Now()
	repo := &mockTransactionRepo{
		transactions: []*models.Transaction{
			{ID: "TR-1", Asset: "psc", Amount: 10.5, Action: "deposit", Status: 1, PayableType: &payableType, CreatedAt: now},
			{ID: "TR-2", Asset: "psc", Amount: 5, Action: "withdraw", Status: 1, CreatedAt: now},
		},
	}
	svc := service.NewTransactionService(repo, stubJalaliConverter{})

	dtos, err := svc.ListTransactions(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, dtos, 2)
	assert.Equal(t, "trade", dtos[0].Type)
	assert.Equal(t, "10.5", dtos[0].Amount)
	assert.Equal(t, "", dtos[1].Type)

	repo.findErr = errors.New("db")
	_, err = svc.ListTransactions(context.Background(), 1, nil)
	require.Error(t, err)
}

func TestTransactionService_GetLatestTransaction(t *testing.T) {
	repo := &mockTransactionRepo{latest: &models.Transaction{ID: "TR-1"}}
	svc := service.NewTransactionService(repo, stubJalaliConverter{})
	got, err := svc.GetLatestTransaction(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "TR-1", got.ID)

	repo.findErr = errors.New("db")
	_, err = svc.GetLatestTransaction(context.Background(), 1)
	require.Error(t, err)
}

func TestTransactionService_CreateTransaction(t *testing.T) {
	repo := &mockTransactionRepo{}
	svc := service.NewTransactionService(repo, stubJalaliConverter{})

	tx := &models.Transaction{UserID: 1, Asset: "psc", Amount: 10, Action: "deposit", Status: 1}
	require.NoError(t, svc.CreateTransaction(context.Background(), tx))
	assert.NotEmpty(t, tx.ID)

	tx2 := &models.Transaction{ID: "TR-fixed", UserID: 1, Asset: "psc", Amount: 1, Action: "deposit", Status: 1}
	require.NoError(t, svc.CreateTransaction(context.Background(), tx2))
	assert.Equal(t, "TR-fixed", tx2.ID)

	repo.createErr = errors.New("db")
	require.Error(t, svc.CreateTransaction(context.Background(), &models.Transaction{UserID: 1, Asset: "psc", Amount: 1, Action: "deposit", Status: 1}))
}
