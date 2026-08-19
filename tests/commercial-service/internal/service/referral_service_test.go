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

type mockReferralRepo struct {
	referrerID    *uint64
	referrerErr   error
	totalReferred float64
	totalErr      error
	createErr     error
}

func (m *mockReferralRepo) GetReferrerID(ctx context.Context, userID uint64) (*uint64, error) {
	return m.referrerID, m.referrerErr
}

func (m *mockReferralRepo) GetTotalReferredAmount(ctx context.Context, referrerID uint64) (float64, error) {
	return m.totalReferred, m.totalErr
}

func (m *mockReferralRepo) CreateReferralOrder(ctx context.Context, history *models.ReferralOrderHistory) error {
	return m.createErr
}

type mockVariableRepo struct {
	rates map[string]float64
	err   error
}

func (m *mockVariableRepo) GetRate(ctx context.Context, asset string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.rates[asset], nil
}

func (m *mockVariableRepo) GetAllRates(ctx context.Context) (map[string]float64, error) {
	return m.rates, m.err
}

type mockUserVariableRepo struct {
	limit float64
	err   error
}

func (m *mockUserVariableRepo) GetReferralProfitLimit(ctx context.Context, userID uint64) (float64, error) {
	return m.limit, m.err
}

func (m *mockUserVariableRepo) GetWithdrawProfit(ctx context.Context, userID uint64) (int, error) {
	return 7, nil
}

func (m *mockUserVariableRepo) Create(ctx context.Context, userID uint64) error {
	return nil
}

type mockReferralWalletRepo struct {
	addErr error
}

func (m *mockReferralWalletRepo) FindByUserID(ctx context.Context, userID uint64) (*models.Wallet, error) {
	return nil, nil
}

func (m *mockReferralWalletRepo) Create(ctx context.Context, userID uint64) (*models.Wallet, error) {
	return nil, nil
}

func (m *mockReferralWalletRepo) Update(ctx context.Context, wallet *models.Wallet) error { return nil }

func (m *mockReferralWalletRepo) DeductBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal) error {
	return nil
}

func (m *mockReferralWalletRepo) AddBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal) error {
	return m.addErr
}

func (m *mockReferralWalletRepo) LockBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal, reason string) error {
	return nil
}

func (m *mockReferralWalletRepo) UnlockBalance(ctx context.Context, userID uint64, asset string, amount decimal.Decimal) error {
	return nil
}

func newReferralSvc(referral *mockReferralRepo, variable *mockVariableRepo, userVar *mockUserVariableRepo, wallet *mockReferralWalletRepo) service.ReferralService {
	return service.NewReferralService(referral, variable, userVar, wallet)
}

func TestReferralService_SkipsIRR(t *testing.T) {
	svc := newReferralSvc(&mockReferralRepo{}, &mockVariableRepo{}, &mockUserVariableRepo{}, &mockReferralWalletRepo{})
	err := svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "irr", Amount: 100})
	require.NoError(t, err)
}

func TestReferralService_NoReferrer(t *testing.T) {
	svc := newReferralSvc(&mockReferralRepo{}, &mockVariableRepo{}, &mockUserVariableRepo{}, &mockReferralWalletRepo{})
	err := svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 100})
	require.NoError(t, err)
}

func TestReferralService_LimitReached(t *testing.T) {
	referrer := uint64(2)
	svc := newReferralSvc(
		&mockReferralRepo{referrerID: &referrer, totalReferred: 100},
		&mockVariableRepo{rates: map[string]float64{"psc": 2}},
		&mockUserVariableRepo{limit: 100},
		&mockReferralWalletRepo{},
	)
	err := svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 10})
	require.NoError(t, err)
}

func TestReferralService_PSCCommission(t *testing.T) {
	referrer := uint64(2)
	svc := newReferralSvc(
		&mockReferralRepo{referrerID: &referrer},
		&mockVariableRepo{rates: map[string]float64{"psc": 1}},
		&mockUserVariableRepo{limit: 1000},
		&mockReferralWalletRepo{},
	)
	err := svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 20})
	require.NoError(t, err)
}

func TestReferralService_ColorCommission(t *testing.T) {
	referrer := uint64(2)
	svc := newReferralSvc(
		&mockReferralRepo{referrerID: &referrer},
		&mockVariableRepo{rates: map[string]float64{"psc": 10, "blue": 5}},
		&mockUserVariableRepo{limit: 1000},
		&mockReferralWalletRepo{},
	)
	err := svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "blue", Amount: 20})
	require.NoError(t, err)
}

func TestReferralService_Errors(t *testing.T) {
	referrer := uint64(2)
	svc := newReferralSvc(
		&mockReferralRepo{referrerErr: errors.New("db")},
		&mockVariableRepo{}, &mockUserVariableRepo{}, &mockReferralWalletRepo{},
	)
	err := svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 1})
	require.Error(t, err)

	svc = newReferralSvc(
		&mockReferralRepo{referrerID: &referrer},
		&mockVariableRepo{err: errors.New("rate")},
		&mockUserVariableRepo{}, &mockReferralWalletRepo{},
	)
	err = svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 1})
	require.Error(t, err)

	svc = newReferralSvc(
		&mockReferralRepo{referrerID: &referrer, totalErr: errors.New("total")},
		&mockVariableRepo{rates: map[string]float64{"psc": 1}},
		&mockUserVariableRepo{}, &mockReferralWalletRepo{},
	)
	err = svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 1})
	require.Error(t, err)

	svc = newReferralSvc(
		&mockReferralRepo{referrerID: &referrer},
		&mockVariableRepo{rates: map[string]float64{"psc": 1}},
		&mockUserVariableRepo{err: errors.New("limit")},
		&mockReferralWalletRepo{},
	)
	err = svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 1})
	require.Error(t, err)

	svc = newReferralSvc(
		&mockReferralRepo{referrerID: &referrer},
		&mockVariableRepo{rates: map[string]float64{"psc": 1, "red": 1}},
		&mockUserVariableRepo{limit: 1000},
		&mockReferralWalletRepo{addErr: errors.New("wallet")},
	)
	err = svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "red", Amount: 10})
	require.Error(t, err)

	svc = newReferralSvc(
		&mockReferralRepo{referrerID: &referrer, createErr: errors.New("history")},
		&mockVariableRepo{rates: map[string]float64{"psc": 1}},
		&mockUserVariableRepo{limit: 1000},
		&mockReferralWalletRepo{},
	)
	err = svc.ProcessReferralCommission(context.Background(), 1, &models.Order{Asset: "psc", Amount: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "referral order history")
}
