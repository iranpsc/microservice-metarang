package service_test

import (
	"context"
	"errors"
	"testing"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFirstOrderRepo struct {
	has    bool
	hasErr error
}

func (m *mockFirstOrderRepo) Create(ctx context.Context, firstOrder *models.FirstOrder) error {
	return nil
}

func (m *mockFirstOrderRepo) HasFirstOrder(ctx context.Context, userID uint64, orderType string) (bool, error) {
	return m.has, m.hasErr
}

func (m *mockFirstOrderRepo) Count(ctx context.Context, userID uint64) (int, error) {
	return 0, nil
}

func TestOrderPolicy_CanGetBonus(t *testing.T) {
	policy := service.NewOrderPolicy(&mockFirstOrderRepo{has: false})
	ok, err := policy.CanGetBonus(context.Background(), 1, "psc")
	require.NoError(t, err)
	assert.True(t, ok)

	policy = service.NewOrderPolicy(&mockFirstOrderRepo{has: true})
	ok, err = policy.CanGetBonus(context.Background(), 1, "psc")
	require.NoError(t, err)
	assert.False(t, ok)

	policy = service.NewOrderPolicy(&mockFirstOrderRepo{hasErr: errors.New("db")})
	_, err = policy.CanGetBonus(context.Background(), 1, "psc")
	require.Error(t, err)
}
