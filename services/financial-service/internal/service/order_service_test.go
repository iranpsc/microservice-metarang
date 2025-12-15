package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"metargb/financial-service/internal/models"
	"metargb/financial-service/internal/parsian"
)

// Mock repositories
type mockOrderRepo struct {
	orders map[uint64]*models.Order
}

func (m *mockOrderRepo) Create(ctx context.Context, order *models.Order) error {
	if m.orders == nil {
		m.orders = make(map[uint64]*models.Order)
	}
	order.ID = uint64(len(m.orders) + 1)
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) FindByID(ctx context.Context, id uint64) (*models.Order, error) {
	if order, ok := m.orders[id]; ok {
		return order, nil
	}
	return nil, nil
}

func (m *mockOrderRepo) FindByIDWithUser(ctx context.Context, id uint64) (*models.Order, *models.User, error) {
	order, ok := m.orders[id]
	if !ok {
		return nil, nil, nil
	}
	user := &models.User{
		ID:   order.UserID,
		Name: "Test User",
	}
	return order, user, nil
}

func (m *mockOrderRepo) Update(ctx context.Context, order *models.Order) error {
	if _, ok := m.orders[order.ID]; !ok {
		return sql.ErrNoRows
	}
	m.orders[order.ID] = order
	return nil
}

type mockTransactionRepo struct {
	transactions map[string]*models.Transaction
}

func (m *mockTransactionRepo) Create(ctx context.Context, transaction *models.Transaction) error {
	if m.transactions == nil {
		m.transactions = make(map[string]*models.Transaction)
	}
	m.transactions[transaction.ID] = transaction
	return nil
}

func (m *mockTransactionRepo) Update(ctx context.Context, transaction *models.Transaction) error {
	if _, ok := m.transactions[transaction.ID]; !ok {
		return sql.ErrNoRows
	}
	m.transactions[transaction.ID] = transaction
	return nil
}

func (m *mockTransactionRepo) FindByID(ctx context.Context, id string) (*models.Transaction, error) {
	if t, ok := m.transactions[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockTransactionRepo) FindByPayable(ctx context.Context, payableType string, payableID uint64) (*models.Transaction, error) {
	for _, t := range m.transactions {
		if t.PayableType != nil && *t.PayableType == payableType &&
			t.PayableID != nil && *t.PayableID == payableID {
			return t, nil
		}
	}
	return nil, nil
}

type mockPaymentRepo struct{}

func (m *mockPaymentRepo) Create(ctx context.Context, payment *models.Payment) error {
	return nil
}

type mockVariableRepo struct {
	rates map[string]float64
}

func (m *mockVariableRepo) GetRate(ctx context.Context, asset string) (float64, error) {
	if rate, ok := m.rates[asset]; ok {
		return rate, nil
	}
	return 0, sql.ErrNoRows
}

type mockFirstOrderRepo struct {
	count int
}

func (m *mockFirstOrderRepo) Create(ctx context.Context, firstOrder *models.FirstOrder) error {
	m.count++
	return nil
}

func (m *mockFirstOrderRepo) Count(ctx context.Context, userID uint64) (int, error) {
	return m.count, nil
}

type mockParsianClient struct {
	requestResponse *parsian.RequestResponse
	verifyResponse  *parsian.VerificationResponse
	requestError    error
	verifyError     error
}

func (m *mockParsianClient) RequestPayment(params parsian.RequestParams) (*parsian.RequestResponse, error) {
	if m.requestError != nil {
		return nil, m.requestError
	}
	return m.requestResponse, nil
}

func (m *mockParsianClient) VerifyPayment(params parsian.VerificationParams) (*parsian.VerificationResponse, error) {
	if m.verifyError != nil {
		return nil, m.verifyError
	}
	return m.verifyResponse, nil
}

type mockOrderPolicy struct {
	canBuy      bool
	canGetBonus bool
}

func (m *mockOrderPolicy) CanBuyFromStore(ctx context.Context, userID uint64) (bool, error) {
	return m.canBuy, nil
}

func (m *mockOrderPolicy) CanGetBonus(ctx context.Context, userID uint64, asset string) (bool, error) {
	return m.canGetBonus, nil
}

type mockJalaliConverter struct{}

func (m *mockJalaliConverter) NowJalali() string {
	return "1403/01/01"
}

func (m *mockJalaliConverter) FormatJalaliDate(t time.Time) string {
	return "1403/01/01"
}

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name          string
		userID        uint64
		amount        int32
		asset         string
		canBuy        bool
		rate          float64
		parsianStatus int32
		parsianToken  int64
		expectError   bool
		errorType     error
	}{
		{
			name:          "successful order creation",
			userID:        1,
			amount:        10,
			asset:         "psc",
			canBuy:        true,
			rate:          1000.0,
			parsianStatus: 0,
			parsianToken:  12345,
			expectError:   false,
		},
		{
			name:        "invalid amount",
			userID:      1,
			amount:      0,
			asset:       "psc",
			canBuy:      true,
			expectError: true,
			errorType:   ErrInvalidAmount,
		},
		{
			name:        "invalid asset",
			userID:      1,
			amount:      10,
			asset:       "invalid",
			canBuy:      true,
			expectError: true,
			errorType:   ErrInvalidAsset,
		},
		{
			name:        "user not eligible",
			userID:      1,
			amount:      10,
			asset:       "psc",
			canBuy:      false,
			expectError: true,
			errorType:   ErrUserNotEligible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &mockOrderRepo{}
			transactionRepo := &mockTransactionRepo{}
			paymentRepo := &mockPaymentRepo{}
			variableRepo := &mockVariableRepo{
				rates: map[string]float64{"psc": tt.rate},
			}
			firstOrderRepo := &mockFirstOrderRepo{}
			parsianClient := &mockParsianClient{
				requestResponse: &parsian.RequestResponse{
					Status: tt.parsianStatus,
					Token:  tt.parsianToken,
				},
			}
			orderPolicy := &mockOrderPolicy{canBuy: tt.canBuy}
			jalaliConverter := &mockJalaliConverter{}

			config := OrderConfig{
				ParsianMerchantID:            "test_merchant",
				ParsianLoanAccountMerchantID: "test_loan_merchant",
				ParsianCallbackURL:           "http://localhost/callback",
				FrontendURL:                  "http://localhost",
			}

			service := NewOrderService(
				orderRepo,
				transactionRepo,
				paymentRepo,
				variableRepo,
				firstOrderRepo,
				parsianClient, // mockParsianClient implements ParsianClient interface
				orderPolicy,
				jalaliConverter,
				config,
			)

			ctx := context.Background()
			link, err := service.CreateOrder(ctx, tt.userID, tt.amount, tt.asset)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if link == "" {
					t.Errorf("expected payment link but got empty")
				}
			}
		})
	}
}
