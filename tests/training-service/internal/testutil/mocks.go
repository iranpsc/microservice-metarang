package testutil

import (
	"context"

	"metarang/training-service/internal/models"
	"metarang/training-service/internal/repository"
)

// MockVideoRepo implements repository.VideoRepositoryInterface for tests.
type MockVideoRepo struct {
	GetVideosFunc          func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error)
	GetVideoBySlugFunc     func(ctx context.Context, slug string) (*models.Video, error)
	GetVideoByFileNameFunc func(ctx context.Context, fileName string) (*models.Video, error)
	SearchVideosFunc       func(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error)
	GetVideoStatsFunc      func(ctx context.Context, videoID uint64) (*models.VideoStats, error)
	GetUserInteractionFunc func(ctx context.Context, videoID, userID uint64) (*bool, error)
	IncrementViewFunc      func(ctx context.Context, videoID uint64, ipAddress string) error
	AddInteractionFunc     func(ctx context.Context, videoID, userID uint64, liked bool, ipAddress string) error
}

func (m *MockVideoRepo) GetVideos(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
	if m.GetVideosFunc != nil {
		return m.GetVideosFunc(ctx, page, perPage, categoryID, subCategoryID)
	}
	return nil, 0, nil
}

func (m *MockVideoRepo) GetVideoBySlug(ctx context.Context, slug string) (*models.Video, error) {
	if m.GetVideoBySlugFunc != nil {
		return m.GetVideoBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *MockVideoRepo) GetVideoByFileName(ctx context.Context, fileName string) (*models.Video, error) {
	if m.GetVideoByFileNameFunc != nil {
		return m.GetVideoByFileNameFunc(ctx, fileName)
	}
	return nil, nil
}

func (m *MockVideoRepo) SearchVideos(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error) {
	if m.SearchVideosFunc != nil {
		return m.SearchVideosFunc(ctx, searchTerm, page, perPage)
	}
	return nil, 0, nil
}

func (m *MockVideoRepo) GetVideoStats(ctx context.Context, videoID uint64) (*models.VideoStats, error) {
	if m.GetVideoStatsFunc != nil {
		return m.GetVideoStatsFunc(ctx, videoID)
	}
	return &models.VideoStats{}, nil
}

func (m *MockVideoRepo) GetUserInteraction(ctx context.Context, videoID, userID uint64) (*bool, error) {
	if m.GetUserInteractionFunc != nil {
		return m.GetUserInteractionFunc(ctx, videoID, userID)
	}
	return nil, nil
}

func (m *MockVideoRepo) IncrementView(ctx context.Context, videoID uint64, ipAddress string) error {
	if m.IncrementViewFunc != nil {
		return m.IncrementViewFunc(ctx, videoID, ipAddress)
	}
	return nil
}

func (m *MockVideoRepo) AddInteraction(ctx context.Context, videoID, userID uint64, liked bool, ipAddress string) error {
	if m.AddInteractionFunc != nil {
		return m.AddInteractionFunc(ctx, videoID, userID, liked, ipAddress)
	}
	return nil
}

var _ repository.VideoRepositoryInterface = (*MockVideoRepo)(nil)

// MockCategoryRepo implements repository.CategoryRepositoryInterface for tests.
type MockCategoryRepo struct {
	GetCategoriesFunc                   func(ctx context.Context, page, perPage int32) ([]*models.VideoCategory, int32, error)
	GetCategoryByIDFunc                 func(ctx context.Context, categoryID uint64) (*models.VideoCategory, error)
	GetCategoryBySlugFunc               func(ctx context.Context, slug string) (*models.VideoCategory, error)
	GetSubCategoryByIDFunc              func(ctx context.Context, subCategoryID uint64) (*models.VideoSubCategory, error)
	GetSubCategoryBySlugsFunc           func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error)
	GetSubCategoriesByCategoryIDFunc    func(ctx context.Context, categoryID uint64) ([]*models.VideoSubCategory, error)
	GetCategoryStatsFunc                func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error)
	GetSubCategoryStatsFunc             func(ctx context.Context, subCategoryID uint64) (*models.SubCategoryStats, error)
	GetSubCategoryStatsByCategoryIDFunc func(ctx context.Context, categoryID uint64) (map[uint64]*models.SubCategoryStats, error)
}

func (m *MockCategoryRepo) GetCategories(ctx context.Context, page, perPage int32) ([]*models.VideoCategory, int32, error) {
	if m.GetCategoriesFunc != nil {
		return m.GetCategoriesFunc(ctx, page, perPage)
	}
	return nil, 0, nil
}

func (m *MockCategoryRepo) GetCategoryByID(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
	if m.GetCategoryByIDFunc != nil {
		return m.GetCategoryByIDFunc(ctx, categoryID)
	}
	return nil, nil
}

func (m *MockCategoryRepo) GetCategoryBySlug(ctx context.Context, slug string) (*models.VideoCategory, error) {
	if m.GetCategoryBySlugFunc != nil {
		return m.GetCategoryBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *MockCategoryRepo) GetSubCategoryByID(ctx context.Context, subCategoryID uint64) (*models.VideoSubCategory, error) {
	if m.GetSubCategoryByIDFunc != nil {
		return m.GetSubCategoryByIDFunc(ctx, subCategoryID)
	}
	return nil, nil
}

func (m *MockCategoryRepo) GetSubCategoryBySlugs(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
	if m.GetSubCategoryBySlugsFunc != nil {
		return m.GetSubCategoryBySlugsFunc(ctx, categorySlug, subCategorySlug)
	}
	return nil, nil
}

func (m *MockCategoryRepo) GetSubCategoriesByCategoryID(ctx context.Context, categoryID uint64) ([]*models.VideoSubCategory, error) {
	if m.GetSubCategoriesByCategoryIDFunc != nil {
		return m.GetSubCategoriesByCategoryIDFunc(ctx, categoryID)
	}
	return nil, nil
}

func (m *MockCategoryRepo) GetCategoryStats(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
	if m.GetCategoryStatsFunc != nil {
		return m.GetCategoryStatsFunc(ctx, categoryID)
	}
	return &models.CategoryStats{}, nil
}

func (m *MockCategoryRepo) GetSubCategoryStats(ctx context.Context, subCategoryID uint64) (*models.SubCategoryStats, error) {
	if m.GetSubCategoryStatsFunc != nil {
		return m.GetSubCategoryStatsFunc(ctx, subCategoryID)
	}
	return &models.SubCategoryStats{}, nil
}

func (m *MockCategoryRepo) GetSubCategoryStatsByCategoryID(ctx context.Context, categoryID uint64) (map[uint64]*models.SubCategoryStats, error) {
	if m.GetSubCategoryStatsByCategoryIDFunc != nil {
		return m.GetSubCategoryStatsByCategoryIDFunc(ctx, categoryID)
	}
	return map[uint64]*models.SubCategoryStats{}, nil
}

var _ repository.CategoryRepositoryInterface = (*MockCategoryRepo)(nil)

// MockCommentRepo implements repository.CommentRepositoryInterface for tests.
type MockCommentRepo struct {
	GetCommentsFunc                    func(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error)
	GetRepliesFunc                     func(ctx context.Context, commentID uint64, page, perPage int32) ([]*models.Comment, int32, error)
	GetCommentByIDFunc                 func(ctx context.Context, commentID uint64) (*models.Comment, error)
	AddCommentFunc                     func(ctx context.Context, videoID, userID uint64, content string) (*models.Comment, error)
	UpdateCommentFunc                  func(ctx context.Context, commentID, userID uint64, content string) error
	DeleteCommentFunc                  func(ctx context.Context, commentID, userID uint64) error
	AddReplyFunc                       func(ctx context.Context, parentCommentID, userID uint64, content string) (*models.Comment, error)
	UpdateReplyFunc                    func(ctx context.Context, replyID, userID uint64, content string) error
	DeleteReplyFunc                    func(ctx context.Context, replyID, userID uint64) error
	GetCommentStatsFunc                func(ctx context.Context, commentID uint64) (*models.CommentStats, error)
	AddCommentInteractionFunc          func(ctx context.Context, commentID, userID uint64, liked bool, ipAddress string) error
	AddReplyInteractionFunc            func(ctx context.Context, replyID, userID uint64, liked bool, ipAddress string) error
	ReportCommentFunc                  func(ctx context.Context, videoID, commentID, userID uint64, content string) error
	GetUserInteractionsForCommentsFunc func(ctx context.Context, commentIDs []uint64, userID uint64) (map[uint64]bool, error)
}

func (m *MockCommentRepo) GetComments(ctx context.Context, videoID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
	if m.GetCommentsFunc != nil {
		return m.GetCommentsFunc(ctx, videoID, page, perPage)
	}
	return nil, 0, nil
}

func (m *MockCommentRepo) GetReplies(ctx context.Context, commentID uint64, page, perPage int32) ([]*models.Comment, int32, error) {
	if m.GetRepliesFunc != nil {
		return m.GetRepliesFunc(ctx, commentID, page, perPage)
	}
	return nil, 0, nil
}

func (m *MockCommentRepo) GetCommentByID(ctx context.Context, commentID uint64) (*models.Comment, error) {
	if m.GetCommentByIDFunc != nil {
		return m.GetCommentByIDFunc(ctx, commentID)
	}
	return nil, nil
}

func (m *MockCommentRepo) AddComment(ctx context.Context, videoID, userID uint64, content string) (*models.Comment, error) {
	if m.AddCommentFunc != nil {
		return m.AddCommentFunc(ctx, videoID, userID, content)
	}
	return nil, nil
}

func (m *MockCommentRepo) UpdateComment(ctx context.Context, commentID, userID uint64, content string) error {
	if m.UpdateCommentFunc != nil {
		return m.UpdateCommentFunc(ctx, commentID, userID, content)
	}
	return nil
}

func (m *MockCommentRepo) DeleteComment(ctx context.Context, commentID, userID uint64) error {
	if m.DeleteCommentFunc != nil {
		return m.DeleteCommentFunc(ctx, commentID, userID)
	}
	return nil
}

func (m *MockCommentRepo) AddReply(ctx context.Context, parentCommentID, userID uint64, content string) (*models.Comment, error) {
	if m.AddReplyFunc != nil {
		return m.AddReplyFunc(ctx, parentCommentID, userID, content)
	}
	return nil, nil
}

func (m *MockCommentRepo) UpdateReply(ctx context.Context, replyID, userID uint64, content string) error {
	if m.UpdateReplyFunc != nil {
		return m.UpdateReplyFunc(ctx, replyID, userID, content)
	}
	return nil
}

func (m *MockCommentRepo) DeleteReply(ctx context.Context, replyID, userID uint64) error {
	if m.DeleteReplyFunc != nil {
		return m.DeleteReplyFunc(ctx, replyID, userID)
	}
	return nil
}

func (m *MockCommentRepo) GetCommentStats(ctx context.Context, commentID uint64) (*models.CommentStats, error) {
	if m.GetCommentStatsFunc != nil {
		return m.GetCommentStatsFunc(ctx, commentID)
	}
	return &models.CommentStats{}, nil
}

func (m *MockCommentRepo) AddCommentInteraction(ctx context.Context, commentID, userID uint64, liked bool, ipAddress string) error {
	if m.AddCommentInteractionFunc != nil {
		return m.AddCommentInteractionFunc(ctx, commentID, userID, liked, ipAddress)
	}
	return nil
}

func (m *MockCommentRepo) AddReplyInteraction(ctx context.Context, replyID, userID uint64, liked bool, ipAddress string) error {
	if m.AddReplyInteractionFunc != nil {
		return m.AddReplyInteractionFunc(ctx, replyID, userID, liked, ipAddress)
	}
	return nil
}

func (m *MockCommentRepo) ReportComment(ctx context.Context, videoID, commentID, userID uint64, content string) error {
	if m.ReportCommentFunc != nil {
		return m.ReportCommentFunc(ctx, videoID, commentID, userID, content)
	}
	return nil
}

func (m *MockCommentRepo) GetUserInteractionsForComments(ctx context.Context, commentIDs []uint64, userID uint64) (map[uint64]bool, error) {
	if m.GetUserInteractionsForCommentsFunc != nil {
		return m.GetUserInteractionsForCommentsFunc(ctx, commentIDs, userID)
	}
	return map[uint64]bool{}, nil
}

var _ repository.CommentRepositoryInterface = (*MockCommentRepo)(nil)

// MockUserRepo implements repository.UserRepositoryInterface for tests.
type MockUserRepo struct {
	GetUserBasicByCodeFunc func(ctx context.Context, code string) (*repository.UserBasic, error)
	GetUserByIDFunc        func(ctx context.Context, userID uint64) (*repository.UserBasic, error)
}

func (m *MockUserRepo) GetUserBasicByCode(ctx context.Context, code string) (*repository.UserBasic, error) {
	if m.GetUserBasicByCodeFunc != nil {
		return m.GetUserBasicByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *MockUserRepo) GetUserByID(ctx context.Context, userID uint64) (*repository.UserBasic, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, userID)
	}
	return &repository.UserBasic{ID: userID, Name: "Test", Code: "u1"}, nil
}

var _ repository.UserRepositoryInterface = (*MockUserRepo)(nil)
