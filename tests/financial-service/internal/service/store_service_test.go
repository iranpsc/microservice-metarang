package service_test

import (
	"context"
	"errors"
	"testing"

	"metarang/financial-service/internal/models"
	"metarang/financial-service/internal/service"
)

type mockOptionRepo struct {
	options map[string]*models.Option
	findErr error
}

func (m *mockOptionRepo) FindByCodes(ctx context.Context, codes []string) ([]*models.Option, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*models.Option
	for _, code := range codes {
		if opt, ok := m.options[code]; ok {
			result = append(result, opt)
		}
	}
	return result, nil
}

type mockImageRepo struct {
	images map[uint64]string
	err    error
}

func (m *mockImageRepo) FindImageURLByImageable(ctx context.Context, imageableType string, imageableID uint64) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if url, ok := m.images[imageableID]; ok {
		return url, nil
	}
	return "", nil
}

func TestStoreService_GetStorePackages(t *testing.T) {
	tests := []struct {
		name        string
		codes       []string
		options     map[string]*models.Option
		rates       map[string]float64
		images      map[uint64]string
		expectError bool
		expectedLen int
	}{
		{
			name:  "successful package retrieval",
			codes: []string{"PACK1", "PACK2"},
			options: map[string]*models.Option{
				"PACK1": {ID: 1, Code: "PACK1", Asset: "psc", Amount: 100},
				"PACK2": {ID: 2, Code: "PACK2", Asset: "red", Amount: 50},
			},
			rates: map[string]float64{
				"psc": 1000.0,
				"red": 2000.0,
			},
			images: map[uint64]string{
				1: "http://example.com/image1.jpg",
			},
			expectError: false,
			expectedLen: 2,
		},
		{
			name:        "insufficient codes",
			codes:       []string{"PACK1"},
			expectError: true,
		},
		{
			name:        "invalid code length",
			codes:       []string{"A", "B"},
			expectError: true,
		},
		{
			name:  "missing options",
			codes: []string{"PACK1", "PACK2"},
			options: map[string]*models.Option{
				"PACK1": {ID: 1, Code: "PACK1", Asset: "psc", Amount: 100},
			},
			rates: map[string]float64{
				"psc": 1000.0,
			},
			expectError: false,
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optionRepo := &mockOptionRepo{options: tt.options}
			variableRepo := &mockVariableRepo{rates: tt.rates}
			imageRepo := &mockImageRepo{images: tt.images}

			svc := service.NewStoreService(optionRepo, variableRepo, imageRepo)

			ctx := context.Background()
			packages, err := svc.GetStorePackages(ctx, tt.codes)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(packages) != tt.expectedLen {
					t.Errorf("expected %d packages, got %d", tt.expectedLen, len(packages))
				}
			}
		})
	}
}

func TestStoreService_OptionLookupError(t *testing.T) {
	svc := service.NewStoreService(&mockOptionRepo{findErr: errors.New("options down")}, &mockVariableRepo{}, &mockImageRepo{})
	_, err := svc.GetStorePackages(context.Background(), []string{"AA", "BB"})
	if err == nil {
		t.Fatal("expected option lookup error")
	}
}
