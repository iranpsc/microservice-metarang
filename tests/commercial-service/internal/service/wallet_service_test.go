package service_test

import (
	"context"
	"errors"
	"testing"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/service"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWalletRepo struct {
	wallet    *models.Wallet
	findErr   error
	createErr error
	deductErr error
	addErr    error
	lockErr   error
	unlockErr error
}

func (m *mockWalletRepo) FindByUserID(ctx context.Context, userID uint64) (*models.Wallet, error) {
	return m.wallet, m.findErr
}

func (m *mockWalletRepo) Create(ctx context.Context, userID uint64) (*models.Wallet, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.wallet != nil {
		return m.wallet, nil
	}
	return &models.Wallet{
		UserID: userID, PSC: decimal.NewFromInt(10), Satisfaction: decimal.NewFromFloat(0.10),
	}, nil
}

func (m *mockWalletRepo) Update(ctx context.Context, wallet *models.Wallet) error { return nil }

func (m *mockWalletRepo) DeductBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal) error {
	return m.deductErr
}

func (m *mockWalletRepo) AddBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal) error {
	return m.addErr
}

func (m *mockWalletRepo) LockBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal, reason string) error {
	return m.lockErr
}

func (m *mockWalletRepo) UnlockBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal) error {
	return m.unlockErr
}

func TestWalletService_GetWallet(t *testing.T) {
	svc := service.NewWalletService(&mockWalletRepo{
		wallet: &models.Wallet{
			PSC: decimal.NewFromInt(5), IRR: decimal.NewFromInt(1), Effect: decimal.NewFromFloat(2),
		},
	})
	got, err := svc.GetWallet(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "5", got["psc"])

	svc = service.NewWalletService(&mockWalletRepo{findErr: errors.New("db")})
	_, err = svc.GetWallet(context.Background(), 1)
	require.Error(t, err)

	svc = service.NewWalletService(&mockWalletRepo{})
	_, err = svc.GetWallet(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWalletService_CreateWallet(t *testing.T) {
	svc := service.NewWalletService(&mockWalletRepo{})
	_, err := svc.CreateWallet(context.Background(), 0)
	require.Error(t, err)

	got, err := svc.CreateWallet(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "10", got["psc"])

	svc = service.NewWalletService(&mockWalletRepo{createErr: errors.New("db")})
	_, err = svc.CreateWallet(context.Background(), 1)
	require.Error(t, err)
}

func TestWalletService_DeductAndAddBalance(t *testing.T) {
	repo := &mockWalletRepo{
		wallet: &models.Wallet{PSC: decimal.NewFromInt(10)},
	}
	svc := service.NewWalletService(repo)

	got, err := svc.DeductBalance(context.Background(), 1, "psc", 2)
	require.NoError(t, err)
	assert.Equal(t, "10", got["psc"])

	repo.deductErr = errors.New("insufficient")
	_, err = svc.DeductBalance(context.Background(), 1, "psc", 99)
	require.Error(t, err)

	repo.deductErr = nil
	got, err = svc.AddBalance(context.Background(), 1, "psc", 3)
	require.NoError(t, err)
	assert.Equal(t, "10", got["psc"])

	repo.addErr = errors.New("fail")
	_, err = svc.AddBalance(context.Background(), 1, "psc", 1)
	require.Error(t, err)
}

func TestWalletService_LockUnlockBalance(t *testing.T) {
	svc := service.NewWalletService(&mockWalletRepo{})
	require.NoError(t, svc.LockBalance(context.Background(), 1, "psc", 1, "hold"))
	require.NoError(t, svc.UnlockBalance(context.Background(), 1, "psc", 1))

	svc = service.NewWalletService(&mockWalletRepo{lockErr: errors.New("fail")})
	require.Error(t, svc.LockBalance(context.Background(), 1, "psc", 1, "hold"))

	svc = service.NewWalletService(&mockWalletRepo{unlockErr: errors.New("fail")})
	require.Error(t, svc.UnlockBalance(context.Background(), 1, "psc", 1))
}
