package service

import (
	"context"
	"errors"
	"fmt"

	"metargb/financial-service/internal/repository"
)

var (
	ErrInvalidCodes      = errors.New("codes must be an array with at least 2 items")
	ErrInvalidCodeLength = errors.New("each code must be at least 2 characters")
)

type StoreService interface {
	GetStorePackages(ctx context.Context, codes []string) ([]*PackageResource, error)
}

type PackageResource struct {
	ID        uint64  `json:"id"`
	Code      string  `json:"code"`
	Asset     string  `json:"asset"`
	Amount    float64 `json:"amount"`
	UnitPrice float64 `json:"unitPrice"`
	Image     *string `json:"image"` // null if no image
}

type storeService struct {
	optionRepo   repository.OptionRepository
	variableRepo repository.VariableRepository
	imageRepo    repository.ImageRepository
}

func NewStoreService(
	optionRepo repository.OptionRepository,
	variableRepo repository.VariableRepository,
	imageRepo repository.ImageRepository,
) StoreService {
	return &storeService{
		optionRepo:   optionRepo,
		variableRepo: variableRepo,
		imageRepo:    imageRepo,
	}
}

func (s *storeService) GetStorePackages(ctx context.Context, codes []string) ([]*PackageResource, error) {
	// Validation: at least 2 codes required
	if len(codes) < 2 {
		return nil, ErrInvalidCodes
	}

	// Validate each code
	for _, code := range codes {
		if len(code) < 2 {
			return nil, ErrInvalidCodeLength
		}
	}

	// Find options by codes
	options, err := s.optionRepo.FindByCodes(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("failed to find options: %w", err)
	}

	// Convert to PackageResource with rates and images
	packages := make([]*PackageResource, 0, len(options))
	for _, option := range options {
		// Get rate for asset
		rate, err := s.variableRepo.GetRate(ctx, option.Asset)
		if err != nil {
			// If rate not found, set to null (per documentation)
			rate = 0
		}

		// Get image URL if exists
		var imageURL *string
		url, err := s.imageRepo.FindImageURLByImageable(ctx, "App\\Models\\Option", option.ID)
		if err == nil && url != "" {
			imageURL = &url
		}

		packageResource := &PackageResource{
			ID:        option.ID,
			Code:      option.Code,
			Asset:     option.Asset,
			Amount:    option.Amount,
			UnitPrice: rate,
			Image:     imageURL,
		}

		packages = append(packages, packageResource)
	}

	return packages, nil
}
