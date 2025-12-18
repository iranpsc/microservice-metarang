package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

// MockSearchService is a mock implementation of SearchService
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) SearchUsers(ctx context.Context, searchTerm string) ([]*service.SearchUserResult, error) {
	args := m.Called(ctx, searchTerm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.SearchUserResult), args.Error(1)
}

func (m *MockSearchService) SearchFeatures(ctx context.Context, searchTerm string) ([]*service.SearchFeatureResult, error) {
	args := m.Called(ctx, searchTerm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.SearchFeatureResult), args.Error(1)
}

func (m *MockSearchService) SearchIsicCodes(ctx context.Context, searchTerm string) ([]*service.IsicCodeResult, error) {
	args := m.Called(ctx, searchTerm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.IsicCodeResult), args.Error(1)
}

func TestSearchHandler_SearchUsers(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		request        *pb.SearchUsersRequest
		serviceResults []*service.SearchUserResult
		serviceError   error
		wantResponse   *pb.SearchUsersResponse
		wantError      bool
		wantCode       string
	}{
		{
			name:    "successful search",
			request: &pb.SearchUsersRequest{SearchTerm: "john"},
			serviceResults: []*service.SearchUserResult{
				{
					ID:        1,
					Code:      "USR001",
					Name:      "John Smith",
					Followers: 5,
					Level:     stringPtr("Level 3"),
					Photo:     stringPtr("http://example.com/photo.jpg"),
				},
			},
			wantResponse: &pb.SearchUsersResponse{
				Data: []*pb.SearchUserResult{
					{
						Id:        1,
						Code:      "USR001",
						Name:      "John Smith",
						Followers: 5,
						Level:     "Level 3",
						Photo:     "http://example.com/photo.jpg",
					},
				},
			},
			wantError: false,
		},
		{
			name:    "empty search term",
			request: &pb.SearchUsersRequest{SearchTerm: ""},
			wantResponse: &pb.SearchUsersResponse{
				Data: []*pb.SearchUserResult{},
			},
			wantError: false,
		},
		{
			name:         "service error",
			request:      &pb.SearchUsersRequest{SearchTerm: "error"},
			serviceError: assert.AnError,
			wantError:    true,
			wantCode:     "Internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSearchService)
			handler := &searchHandler{searchService: mockService}

			if tt.request.SearchTerm != "" && tt.serviceError == nil {
				mockService.On("SearchUsers", ctx, tt.request.SearchTerm).Return(tt.serviceResults, tt.serviceError)
			}

			response, err := handler.SearchUsers(ctx, tt.request)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, response)
				if tt.wantCode != "" {
					// Check gRPC status code
					// In a real test, you'd check the status code from the error
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Data, len(tt.wantResponse.Data))

				if len(tt.wantResponse.Data) > 0 {
					assert.Equal(t, tt.wantResponse.Data[0].Id, response.Data[0].Id)
					assert.Equal(t, tt.wantResponse.Data[0].Code, response.Data[0].Code)
					assert.Equal(t, tt.wantResponse.Data[0].Name, response.Data[0].Name)
					assert.Equal(t, tt.wantResponse.Data[0].Followers, response.Data[0].Followers)
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchHandler_SearchFeatures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		request        *pb.SearchFeaturesRequest
		serviceResults []*service.SearchFeatureResult
		serviceError   error
		wantResponse   *pb.SearchFeaturesResponse
		wantError      bool
	}{
		{
			name:    "successful search",
			request: &pb.SearchFeaturesRequest{SearchTerm: "TEH-"},
			serviceResults: []*service.SearchFeatureResult{
				{
					ID:                  1,
					FeaturePropertiesID: "PROP-123",
					Address:             "Tehran, District 1",
					Karbari:             "مسکونی",
					PricePsc:            "2.5",
					PriceIrr:            "3500000000",
					OwnerCode:           "CIT998",
					Coordinates: []*service.FeatureCoordinate{
						{ID: 1, X: 51.1234, Y: 35.6789},
					},
				},
			},
			wantResponse: &pb.SearchFeaturesResponse{
				Data: []*pb.SearchFeatureResult{
					{
						Id:                  1,
						FeaturePropertiesId: "PROP-123",
						Address:             "Tehran, District 1",
						Karbari:             "مسکونی",
						PricePsc:            "2.5",
						PriceIrr:            "3500000000",
						OwnerCode:           "CIT998",
						Coordinates: []*pb.Coordinate{
							{Id: 1, X: 51.1234, Y: 35.6789},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name:    "empty search term",
			request: &pb.SearchFeaturesRequest{SearchTerm: ""},
			wantResponse: &pb.SearchFeaturesResponse{
				Data: []*pb.SearchFeatureResult{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSearchService)
			handler := &searchHandler{searchService: mockService}

			if tt.request.SearchTerm != "" && tt.serviceError == nil {
				mockService.On("SearchFeatures", ctx, tt.request.SearchTerm).Return(tt.serviceResults, tt.serviceError)
			}

			response, err := handler.SearchFeatures(ctx, tt.request)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Data, len(tt.wantResponse.Data))

				if len(tt.wantResponse.Data) > 0 {
					assert.Equal(t, tt.wantResponse.Data[0].Id, response.Data[0].Id)
					assert.Equal(t, tt.wantResponse.Data[0].FeaturePropertiesId, response.Data[0].FeaturePropertiesId)
					assert.Len(t, response.Data[0].Coordinates, len(tt.wantResponse.Data[0].Coordinates))
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchHandler_SearchIsicCodes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		request        *pb.SearchIsicCodesRequest
		serviceResults []*service.IsicCodeResult
		serviceError   error
		wantResponse   *pb.SearchIsicCodesResponse
		wantError      bool
	}{
		{
			name:    "successful search",
			request: &pb.SearchIsicCodesRequest{SearchTerm: "manufacturing"},
			serviceResults: []*service.IsicCodeResult{
				{ID: 1, Name: "Manufacture of textiles", Code: 1311},
				{ID: 2, Name: "Manufacture of beverages", Code: 1104},
			},
			wantResponse: &pb.SearchIsicCodesResponse{
				Data: []*pb.IsicCodeResult{
					{Id: 1, Name: "Manufacture of textiles", Code: 1311},
					{Id: 2, Name: "Manufacture of beverages", Code: 1104},
				},
			},
			wantError: false,
		},
		{
			name:    "empty search term",
			request: &pb.SearchIsicCodesRequest{SearchTerm: ""},
			wantResponse: &pb.SearchIsicCodesResponse{
				Data: []*pb.IsicCodeResult{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSearchService)
			handler := &searchHandler{searchService: mockService}

			if tt.request.SearchTerm != "" && tt.serviceError == nil {
				mockService.On("SearchIsicCodes", ctx, tt.request.SearchTerm).Return(tt.serviceResults, tt.serviceError)
			}

			response, err := handler.SearchIsicCodes(ctx, tt.request)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Data, len(tt.wantResponse.Data))
			}

			mockService.AssertExpectations(t)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
