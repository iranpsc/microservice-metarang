package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metarang/financial-service/internal/service"
)

type mockEligibilityRepo struct {
	birthdate *time.Time
	birthErr  error
	verified  bool
	bfr       bool
	found     bool
	permErr   error
}

func (m *mockEligibilityRepo) GetUserBirthdate(ctx context.Context, userID uint64) (*time.Time, error) {
	if m.birthErr != nil {
		return nil, m.birthErr
	}
	return m.birthdate, nil
}

func (m *mockEligibilityRepo) GetChildPermissions(ctx context.Context, userID uint64) (bool, bool, bool, error) {
	if m.permErr != nil {
		return false, false, false, m.permErr
	}
	return m.verified, m.bfr, m.found, nil
}

func yearsAgo(years int) *time.Time {
	t := time.Now().AddDate(-years, 0, 0)
	return &t
}

func TestOrderPolicy_CanBuyFromStore(t *testing.T) {
	ctx := context.Background()
	firstOrders := &mockFirstOrderRepo{}

	t.Run("nil birthdate allows purchase", func(t *testing.T) {
		p := service.NewOrderPolicy(&mockEligibilityRepo{}, firstOrders)
		ok, err := p.CanBuyFromStore(ctx, 1)
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("adult allows purchase", func(t *testing.T) {
		p := service.NewOrderPolicy(&mockEligibilityRepo{birthdate: yearsAgo(30)}, firstOrders)
		ok, err := p.CanBuyFromStore(ctx, 1)
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("minor with verified BFR allows purchase", func(t *testing.T) {
		p := service.NewOrderPolicy(&mockEligibilityRepo{
			birthdate: yearsAgo(10),
			verified:  true,
			bfr:       true,
			found:     true,
		}, firstOrders)
		ok, err := p.CanBuyFromStore(ctx, 1)
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("minor without permissions is blocked", func(t *testing.T) {
		p := service.NewOrderPolicy(&mockEligibilityRepo{birthdate: yearsAgo(10)}, firstOrders)
		ok, err := p.CanBuyFromStore(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatal("expected minor without BFR to be blocked")
		}
	})

	t.Run("birthdate error", func(t *testing.T) {
		p := service.NewOrderPolicy(&mockEligibilityRepo{birthErr: errors.New("db down")}, firstOrders)
		ok, err := p.CanBuyFromStore(ctx, 1)
		if err == nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("permissions error for minor", func(t *testing.T) {
		p := service.NewOrderPolicy(&mockEligibilityRepo{
			birthdate: yearsAgo(12),
			permErr:   errors.New("perm down"),
		}, firstOrders)
		ok, err := p.CanBuyFromStore(ctx, 1)
		if err == nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})
}

func TestOrderPolicy_CanGetBonus(t *testing.T) {
	ctx := context.Background()
	elig := &mockEligibilityRepo{}

	t.Run("irr never gets bonus", func(t *testing.T) {
		p := service.NewOrderPolicy(elig, &mockFirstOrderRepo{count: 0})
		ok, err := p.CanGetBonus(ctx, 1, "irr")
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("first order of non-irr gets bonus", func(t *testing.T) {
		p := service.NewOrderPolicy(elig, &mockFirstOrderRepo{count: 0})
		ok, err := p.CanGetBonus(ctx, 1, "psc")
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("existing first order record denies bonus", func(t *testing.T) {
		p := service.NewOrderPolicy(elig, &mockFirstOrderRepo{count: 2})
		ok, err := p.CanGetBonus(ctx, 1, "psc")
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("count error", func(t *testing.T) {
		p := service.NewOrderPolicy(elig, &mockFirstOrderRepo{countErr: errors.New("count failed")})
		ok, err := p.CanGetBonus(ctx, 1, "yellow")
		if err == nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})
}
