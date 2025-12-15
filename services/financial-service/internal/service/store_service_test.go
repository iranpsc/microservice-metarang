package service

import (
	"context"
	"testing"

	"metargb/financial-service/internal/models"
)

type mockOptionRepo struct {
	options map[string]*models.Option
}

func (m *mockOptionRepo) FindByCodes(ctx context.Context, codes []string) ([]*models.Option, error) {
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
}

func (m *mockImageRepo) FindImageURLByImageable(ctx context.Context, imageableType string, imageableID uint64) (string, error) {
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
			expectedLen: 1, // Only PACK1 found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optionRepo := &mockOptionRepo{options: tt.options}
			variableRepo := &mockVariableRepo{rates: tt.rates}
			imageRepo := &mockImageRepo{images: tt.images}

			service := NewStoreService(optionRepo, variableRepo, imageRepo)

			ctx := context.Background()
			packages, err := service.GetStorePackages(ctx, tt.codes)

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
