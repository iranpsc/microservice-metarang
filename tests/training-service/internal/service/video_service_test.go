package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/training-service/internal/models"
	"metargb/training-service/internal/repository"
)

// Mock repositories
type mockVideoRepository struct {
	getVideosFunc          func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error)
	getVideoBySlugFunc     func(ctx context.Context, slug string) (*models.Video, error)
	getVideoByFileNameFunc func(ctx context.Context, fileName string) (*models.Video, error)
	searchVideosFunc       func(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error)
	getVideoStatsFunc      func(ctx context.Context, videoID uint64) (*models.VideoStats, error)
	incrementViewFunc      func(ctx context.Context, videoID uint64, ipAddress string) error
	addInteractionFunc     func(ctx context.Context, videoID, userID uint64, liked bool, ipAddress string) error
}

func (m *mockVideoRepository) GetVideos(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
	if m.getVideosFunc != nil {
		return m.getVideosFunc(ctx, page, perPage, categoryID, subCategoryID)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockVideoRepository) GetVideoBySlug(ctx context.Context, slug string) (*models.Video, error) {
	if m.getVideoBySlugFunc != nil {
		return m.getVideoBySlugFunc(ctx, slug)
	}
	return nil, errors.New("not implemented")
}

func (m *mockVideoRepository) GetVideoByFileName(ctx context.Context, fileName string) (*models.Video, error) {
	if m.getVideoByFileNameFunc != nil {
		return m.getVideoByFileNameFunc(ctx, fileName)
	}
	return nil, errors.New("not implemented")
}

func (m *mockVideoRepository) SearchVideos(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error) {
	if m.searchVideosFunc != nil {
		return m.searchVideosFunc(ctx, searchTerm, page, perPage)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockVideoRepository) GetVideoStats(ctx context.Context, videoID uint64) (*models.VideoStats, error) {
	if m.getVideoStatsFunc != nil {
		return m.getVideoStatsFunc(ctx, videoID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockVideoRepository) IncrementView(ctx context.Context, videoID uint64, ipAddress string) error {
	if m.incrementViewFunc != nil {
		return m.incrementViewFunc(ctx, videoID, ipAddress)
	}
	return errors.New("not implemented")
}

func (m *mockVideoRepository) AddInteraction(ctx context.Context, videoID, userID uint64, liked bool, ipAddress string) error {
	if m.addInteractionFunc != nil {
		return m.addInteractionFunc(ctx, videoID, userID, liked, ipAddress)
	}
	return errors.New("not implemented")
}

type mockCategoryRepository struct{}

func (m *mockCategoryRepository) GetSubCategoryByID(ctx context.Context, subCategoryID uint64) (*models.VideoSubCategory, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCategoryRepository) GetCategoryByID(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
	return nil, errors.New("not implemented")
}

type mockUserRepository struct {
	getUserBasicByCodeFunc func(ctx context.Context, code string) (*repository.UserBasic, error)
	getUserByIDFunc        func(ctx context.Context, userID uint64) (*repository.UserBasic, error)
}

func (m *mockUserRepository) GetUserBasicByCode(ctx context.Context, code string) (*repository.UserBasic, error) {
	if m.getUserBasicByCodeFunc != nil {
		return m.getUserBasicByCodeFunc(ctx, code)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, userID uint64) (*repository.UserBasic, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

// TestVideoService_SearchVideos tests the SearchVideos method
func TestVideoService_SearchVideos(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		searchTerm  string
		setupMock   func() *mockVideoRepository
		expectError bool
	}{
		{
			name:       "empty search term",
			searchTerm: "",
			setupMock: func() *mockVideoRepository {
				return &mockVideoRepository{}
			},
			expectError: true,
		},
		{
			name:       "valid search",
			searchTerm: "tutorial",
			setupMock: func() *mockVideoRepository {
				return &mockVideoRepository{
					searchVideosFunc: func(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error) {
						return []*models.Video{
							{
								ID:          1,
								Title:       "Test Tutorial",
								Description: "Test Description",
								CreatedAt:   time.Now(),
							},
						}, 1, nil
					},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.setupMock()
			service := NewVideoService(mockRepo, &mockCategoryRepository{}, &mockUserRepository{})

			_, _, err := service.SearchVideos(ctx, tt.searchTerm, 1, 18)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
