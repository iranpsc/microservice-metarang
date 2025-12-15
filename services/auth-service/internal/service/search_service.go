package service

import (
	"context"
	"fmt"
	"strings"

	"metargb/auth-service/internal/repository"
)

type SearchService interface {
	SearchUsers(ctx context.Context, searchTerm string) ([]*SearchUserResult, error)
	SearchFeatures(ctx context.Context, searchTerm string) ([]*SearchFeatureResult, error)
	SearchIsicCodes(ctx context.Context, searchTerm string) ([]*IsicCodeResult, error)
}

type searchService struct {
	searchRepo repository.SearchRepository
}

func NewSearchService(searchRepo repository.SearchRepository) SearchService {
	return &searchService{
		searchRepo: searchRepo,
	}
}

// SearchUserResult represents a user search result
type SearchUserResult struct {
	ID        uint64
	Code      string
	Name      string
	Followers int32
	Level     *string // nullable
	Photo     *string // nullable
}

// SearchFeatureResult represents a feature search result
type SearchFeatureResult struct {
	ID                  uint64
	FeaturePropertiesID string
	Address             string
	Karbari             string
	PricePsc            string
	PriceIrr            string
	OwnerCode           string
	Coordinates         []*FeatureCoordinate
}

// FeatureCoordinate represents a feature coordinate
type FeatureCoordinate struct {
	ID uint64
	X  float64
	Y  float64
}

// IsicCodeResult represents an ISIC code search result
type IsicCodeResult struct {
	ID   uint64
	Name string
	Code uint64
}

// SearchUsers searches users by name, code, and KYC fields
func (s *searchService) SearchUsers(ctx context.Context, searchTerm string) ([]*SearchUserResult, error) {
	// Validate search term is not empty
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return []*SearchUserResult{}, nil
	}

	// Call repository
	repoResults, err := s.searchRepo.SearchUsers(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// Convert repository results to service results
	results := make([]*SearchUserResult, 0, len(repoResults))
	for _, repoResult := range repoResults {
		result := &SearchUserResult{
			ID:        repoResult.User.ID,
			Code:      strings.ToUpper(repoResult.User.Code), // Uppercase code
			Followers: repoResult.Followers,
		}

		// Determine name: use KYC if verified (status = 1), otherwise use user.name
		if repoResult.KYC != nil && repoResult.KYC.Status == 1 {
			result.Name = repoResult.KYC.Fname + " " + repoResult.KYC.Lname
		} else {
			result.Name = repoResult.User.Name
		}

		// Get latest profile photo URL (last one in the array)
		if len(repoResult.ProfilePhotos) > 0 {
			lastPhoto := repoResult.ProfilePhotos[len(repoResult.ProfilePhotos)-1]
			result.Photo = &lastPhoto.URL
		}

		// Get latest level name
		if repoResult.LatestLevel != nil {
			result.Level = &repoResult.LatestLevel.Name
		}

		results = append(results, result)
	}

	return results, nil
}

// SearchFeatures searches feature properties by id and address
func (s *searchService) SearchFeatures(ctx context.Context, searchTerm string) ([]*SearchFeatureResult, error) {
	// Validate search term is not empty
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return []*SearchFeatureResult{}, nil
	}

	// Call repository
	repoResults, err := s.searchRepo.SearchFeatures(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search features: %w", err)
	}

	// Convert repository results to service results
	results := make([]*SearchFeatureResult, 0, len(repoResults))
	for _, repoResult := range repoResults {
		result := &SearchFeatureResult{
			ID:                  repoResult.FeatureID,
			FeaturePropertiesID: strings.ToUpper(repoResult.FeaturePropertiesID), // Uppercase ID
			Address:             repoResult.Address,
			PricePsc:            repoResult.PricePsc,
			PriceIrr:            repoResult.PriceIrr,
			OwnerCode:           strings.ToUpper(repoResult.OwnerCode), // Uppercase owner code
		}

		// Map karbari to Persian title (getApplicationTitle equivalent)
		result.Karbari = mapKarbariToTitle(repoResult.Karbari)

		// Convert coordinates
		result.Coordinates = make([]*FeatureCoordinate, 0, len(repoResult.Coordinates))
		for _, coord := range repoResult.Coordinates {
			result.Coordinates = append(result.Coordinates, &FeatureCoordinate{
				ID: coord.ID,
				X:  coord.X,
				Y:  coord.Y,
			})
		}

		results = append(results, result)
	}

	return results, nil
}

// mapKarbariToTitle maps karbari code to Persian title
// This implements Laravel's getApplicationTitle() method
func mapKarbariToTitle(karbari string) string {
	// Map single-letter codes to Persian titles
	switch strings.ToLower(karbari) {
	case "m":
		return "مسکونی"
	case "t":
		return "تجاری"
	case "a":
		return "آموزشی"
	case "e":
		return "اداری"
	case "b":
		return "بهداشتی"
	case "f":
		return "فضای سبز"
	case "c":
		return "فرهنگی"
	case "p":
		return "پارکینگ"
	case "z":
		return "مذهبی"
	case "n":
		return "نمایشگاه"
	case "g":
		return "گردشگری"
	default:
		// If karbari is already a title or unknown, return as-is
		return karbari
	}
}

// SearchIsicCodes searches ISIC codes by name
func (s *searchService) SearchIsicCodes(ctx context.Context, searchTerm string) ([]*IsicCodeResult, error) {
	// Validate search term is not empty
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return []*IsicCodeResult{}, nil
	}

	// Call repository
	repoResults, err := s.searchRepo.SearchIsicCodes(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search isic codes: %w", err)
	}

	// Convert repository results to service results
	results := make([]*IsicCodeResult, 0, len(repoResults))
	for _, repoResult := range repoResults {
		results = append(results, &IsicCodeResult{
			ID:   repoResult.ID,
			Name: repoResult.Name,
			Code: repoResult.Code,
		})
	}

	return results, nil
}
