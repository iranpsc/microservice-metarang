package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

// MockSearchRepository is a mock implementation of SearchRepository
type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) SearchUsers(ctx context.Context, searchTerm string) ([]*repository.SearchUserResult, error) {
	args := m.Called(ctx, searchTerm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.SearchUserResult), args.Error(1)
}

func (m *MockSearchRepository) SearchFeatures(ctx context.Context, searchTerm string) ([]*repository.SearchFeatureResult, error) {
	args := m.Called(ctx, searchTerm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.SearchFeatureResult), args.Error(1)
}

func (m *MockSearchRepository) SearchIsicCodes(ctx context.Context, searchTerm string) ([]*repository.IsicCodeResult, error) {
	args := m.Called(ctx, searchTerm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.IsicCodeResult), args.Error(1)
}

func TestSearchService_SearchUsers(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		searchTerm    string
		repoResults   []*repository.SearchUserResult
		repoError     error
		wantResults   int
		wantName      string
		wantCode      string
		wantFollowers int32
		wantLevel     *string
		wantPhoto     *string
		wantError     bool
	}{
		{
			name:       "successful search with KYC verified user",
			searchTerm: "john",
			repoResults: []*repository.SearchUserResult{
				{
					User: &models.User{
						ID:   1,
						Name: "john",
						Code: "usr001",
					},
					KYC: &models.KYC{
						Fname:  "John",
						Lname:  "Smith",
						Status: 1, // Verified
					},
					ProfilePhotos: []*models.Image{
						{URL: "http://example.com/photo1.jpg"},
					},
					Followers:   5,
					LatestLevel: &repository.UserLevel{Name: "Level 3"},
				},
			},
			wantResults:   1,
			wantName:      "John Smith", // Should use KYC name
			wantCode:      "USR001",     // Should be uppercased
			wantFollowers: 5,
			wantLevel:     stringPtr("Level 3"),
			wantPhoto:     stringPtr("http://example.com/photo1.jpg"),
			wantError:     false,
		},
		{
			name:       "successful search with non-verified user",
			searchTerm: "jane",
			repoResults: []*repository.SearchUserResult{
				{
					User: &models.User{
						ID:   2,
						Name: "jane doe",
						Code: "usr002",
					},
					KYC: &models.KYC{
						Fname:  "Jane",
						Lname:  "Doe",
						Status: 0, // Not verified
					},
					Followers: 0,
				},
			},
			wantResults:   1,
			wantName:      "jane doe", // Should use user name
			wantCode:      "USR002",
			wantFollowers: 0,
			wantError:     false,
		},
		{
			name:        "empty search term",
			searchTerm:  "",
			wantResults: 0,
			wantError:   false,
		},
		{
			name:       "repository error",
			searchTerm: "error",
			repoError:  assert.AnError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSearchRepository)
			if tt.searchTerm != "" && tt.repoError == nil {
				mockRepo.On("SearchUsers", ctx, tt.searchTerm).Return(tt.repoResults, tt.repoError)
			}

			service := NewSearchService(mockRepo)
			results, err := service.SearchUsers(ctx, tt.searchTerm)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.wantResults)

				if tt.wantResults > 0 && len(results) > 0 {
					assert.Equal(t, tt.wantName, results[0].Name)
					assert.Equal(t, tt.wantCode, results[0].Code)
					assert.Equal(t, tt.wantFollowers, results[0].Followers)
					if tt.wantLevel != nil {
						assert.Equal(t, *tt.wantLevel, *results[0].Level)
					}
					if tt.wantPhoto != nil {
						assert.Equal(t, *tt.wantPhoto, *results[0].Photo)
					}
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSearchService_SearchFeatures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		searchTerm    string
		repoResults   []*repository.SearchFeatureResult
		repoError     error
		wantResults   int
		wantKarbari   string
		wantOwnerCode string
		wantError     bool
	}{
		{
			name:       "successful search",
			searchTerm: "TEH-",
			repoResults: []*repository.SearchFeatureResult{
				{
					FeatureID:           1,
					FeaturePropertiesID: "prop-123",
					Address:             "Tehran, District 1",
					Karbari:             "m", // residential
					PricePsc:            "2.5",
					PriceIrr:            "3500000000",
					OwnerCode:           "cit998",
					Coordinates: []*repository.Coordinate{
						{ID: 1, X: 51.1234, Y: 35.6789},
					},
				},
			},
			wantResults:   1,
			wantKarbari:   "مسکونی", // Should be mapped to Persian
			wantOwnerCode: "CIT998", // Should be uppercased
			wantError:     false,
		},
		{
			name:        "empty search term",
			searchTerm:  "",
			wantResults: 0,
			wantError:   false,
		},
		{
			name:       "repository error",
			searchTerm: "error",
			repoError:  assert.AnError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSearchRepository)
			if tt.searchTerm != "" && tt.repoError == nil {
				mockRepo.On("SearchFeatures", ctx, tt.searchTerm).Return(tt.repoResults, tt.repoError)
			}

			service := NewSearchService(mockRepo)
			results, err := service.SearchFeatures(ctx, tt.searchTerm)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.wantResults)

				if tt.wantResults > 0 && len(results) > 0 {
					assert.Equal(t, tt.wantKarbari, results[0].Karbari)
					assert.Equal(t, tt.wantOwnerCode, results[0].OwnerCode)
					assert.Equal(t, "PROP-123", results[0].FeaturePropertiesID) // Should be uppercased
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSearchService_SearchIsicCodes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		searchTerm  string
		repoResults []*repository.IsicCodeResult
		repoError   error
		wantResults int
		wantError   bool
	}{
		{
			name:       "successful search",
			searchTerm: "manufacturing",
			repoResults: []*repository.IsicCodeResult{
				{ID: 1, Name: "Manufacture of textiles", Code: 1311},
				{ID: 2, Name: "Manufacture of beverages", Code: 1104},
			},
			wantResults: 2,
			wantError:   false,
		},
		{
			name:        "empty search term",
			searchTerm:  "",
			wantResults: 0,
			wantError:   false,
		},
		{
			name:       "repository error",
			searchTerm: "error",
			repoError:  assert.AnError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSearchRepository)
			if tt.searchTerm != "" && tt.repoError == nil {
				mockRepo.On("SearchIsicCodes", ctx, tt.searchTerm).Return(tt.repoResults, tt.repoError)
			}

			service := NewSearchService(mockRepo)
			results, err := service.SearchIsicCodes(ctx, tt.searchTerm)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.wantResults)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMapKarbariToTitle(t *testing.T) {
	tests := []struct {
		name     string
		karbari  string
		expected string
	}{
		{"residential", "m", "مسکونی"},
		{"commercial", "t", "تجاری"},
		{"educational", "a", "آموزشی"},
		{"administrative", "e", "اداری"},
		{"health", "b", "بهداشتی"},
		{"green space", "f", "فضای سبز"},
		{"cultural", "c", "فرهنگی"},
		{"parking", "p", "پارکینگ"},
		{"religious", "z", "مذهبی"},
		{"exhibition", "n", "نمایشگاه"},
		{"tourism", "g", "گردشگری"},
		{"unknown code", "x", "x"},            // Should return as-is
		{"already title", "مسکونی", "مسکونی"}, // Should return as-is
		{"uppercase", "M", "مسکونی"},          // Should handle uppercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Access the private function through a test helper
			// Since it's private, we'll test it indirectly through SearchFeatures
			// Or we can make it public for testing
			result := mapKarbariToTitle(tt.karbari)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
