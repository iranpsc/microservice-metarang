package service_test

import (
	"context"
	"errors"
	"testing"

	"metarang/commercial-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserVariableRepoOnly struct {
	createErr error
}

func (m *mockUserVariableRepoOnly) GetReferralProfitLimit(ctx context.Context, userID uint64) (float64, error) {
	return 0, nil
}

func (m *mockUserVariableRepoOnly) GetWithdrawProfit(ctx context.Context, userID uint64) (int, error) {
	return 7, nil
}

func (m *mockUserVariableRepoOnly) Create(ctx context.Context, userID uint64) error {
	return m.createErr
}

func TestUserVariableService_CreateUserVariables(t *testing.T) {
	svc := service.NewUserVariableService(&mockUserVariableRepoOnly{})
	require.Error(t, svc.CreateUserVariables(context.Background(), 0))

	require.NoError(t, svc.CreateUserVariables(context.Background(), 1))

	svc = service.NewUserVariableService(&mockUserVariableRepoOnly{createErr: errors.New("db")})
	err := svc.CreateUserVariables(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user variables")
}
