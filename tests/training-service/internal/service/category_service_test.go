package service_test

import (
	"context"
	"errors"
	"testing"

	"metarang/training-service/internal/models"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func TestCategoryService_GetCategoriesAndVideosSuccess(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoriesFunc: func(ctx context.Context, page, perPage int32) ([]*models.VideoCategory, int32, error) {
			return []*models.VideoCategory{{ID: 1, Name: "C"}}, 1, nil
		},
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 3, Slug: slug}, nil
		},
	}
	mv := &testutil.MockVideoRepo{
		GetVideosFunc: func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
			if categoryID == nil || *categoryID != 3 || subCategoryID != nil {
				t.Fatalf("cat=%v sub=%v", categoryID, subCategoryID)
			}
			return []*models.Video{{ID: 9}}, 1, nil
		},
	}
	svc := service.NewCategoryService(mc, mv)
	cats, total, err := svc.GetCategories(context.Background(), 1, 30)
	if err != nil || total != 1 || len(cats) != 1 {
		t.Fatal(err, total, len(cats))
	}
	videos, vtotal, err := svc.GetCategoryVideos(context.Background(), "c", 1, 18)
	if err != nil || vtotal != 1 || len(videos) != 1 {
		t.Fatal(err, vtotal, len(videos))
	}
}

func TestCategoryService_GetCategoryBySlug_NotFoundAndFullDetails(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return nil, nil
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	if _, err := svc.GetCategoryBySlug(context.Background(), "missing"); err == nil {
		t.Fatal("expected not found")
	}

	mc = &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 1, Slug: slug}, nil
		},
		GetSubCategoriesByCategoryIDFunc: func(ctx context.Context, categoryID uint64) ([]*models.VideoSubCategory, error) {
			return []*models.VideoSubCategory{{ID: 2}}, nil
		},
		GetSubCategoryStatsByCategoryIDFunc: func(ctx context.Context, categoryID uint64) (map[uint64]*models.SubCategoryStats, error) {
			return map[uint64]*models.SubCategoryStats{2: {VideosCount: 4}}, nil
		},
		GetCategoryStatsFunc: func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
			return &models.CategoryStats{VideosCount: 4}, nil
		},
	}
	svc = service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	d, err := svc.GetCategoryBySlug(context.Background(), "cat")
	if err != nil || d.Stats == nil || d.SubCategoryStats[2].VideosCount != 4 {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestCategoryService_GetCategoryBySlug_RepoErrorAndPartialStats(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return nil, context.Canceled
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	_, err := svc.GetCategoryBySlug(context.Background(), "x")
	if err == nil {
		t.Fatal("expected wrapped repo error")
	}

	mc = &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 1, Slug: slug}, nil
		},
		GetSubCategoriesByCategoryIDFunc: func(ctx context.Context, categoryID uint64) ([]*models.VideoSubCategory, error) {
			return nil, errors.New("subs failed")
		},
		GetCategoryStatsFunc: func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
			return nil, errors.New("stats failed")
		},
	}
	svc = service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	d, err := svc.GetCategoryBySlug(context.Background(), "cat")
	if err != nil || d.Category.ID != 1 || d.SubCategories != nil || d.Stats != nil {
		t.Fatalf("partial details=%+v err=%v", d, err)
	}
}

func TestCategoryService_GetSubCategoryBySlugs_NotFoundAndRepoError(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryBySlugsFunc: func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
			return nil, nil
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	_, err := svc.GetSubCategoryBySlugs(context.Background(), "c", "s")
	if err == nil {
		t.Fatal("expected not found")
	}

	mc.GetSubCategoryBySlugsFunc = func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
		return nil, context.Canceled
	}
	_, err = svc.GetSubCategoryBySlugs(context.Background(), "c", "s")
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestCategoryService_GetSubCategoryBySlugs_WithCategoryAndStats(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryBySlugsFunc: func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
			return &models.VideoSubCategory{ID: 4, VideoCategoryID: 2, Slug: subCategorySlug}, nil
		},
		GetCategoryByIDFunc: func(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: categoryID, Name: "Cat"}, nil
		},
		GetSubCategoryStatsFunc: func(ctx context.Context, subCategoryID uint64) (*models.SubCategoryStats, error) {
			return &models.SubCategoryStats{VideosCount: 3, ViewsCount: 8}, nil
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	d, err := svc.GetSubCategoryBySlugs(context.Background(), "c", "s")
	if err != nil || d.Category == nil || d.Stats == nil || d.Stats.VideosCount != 3 {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestCategoryService_GetSubCategoryBySlugs_IgnoresCategoryAndStatsErrors(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryBySlugsFunc: func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
			return &models.VideoSubCategory{ID: 4, VideoCategoryID: 2, Slug: subCategorySlug}, nil
		},
		GetCategoryByIDFunc: func(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
			return nil, errors.New("missing cat")
		},
		GetSubCategoryStatsFunc: func(ctx context.Context, subCategoryID uint64) (*models.SubCategoryStats, error) {
			return nil, errors.New("stats")
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	d, err := svc.GetSubCategoryBySlugs(context.Background(), "c", "s")
	if err != nil || d.SubCategory.ID != 4 || d.Category != nil || d.Stats != nil {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestCategoryService_GetCategoryVideos_NotFoundAndRepoError(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return nil, nil
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	_, _, err := svc.GetCategoryVideos(context.Background(), "missing", 1, 18)
	if err == nil {
		t.Fatal("expected not found")
	}

	mc.GetCategoryBySlugFunc = func(ctx context.Context, slug string) (*models.VideoCategory, error) {
		return nil, context.Canceled
	}
	_, _, err = svc.GetCategoryVideos(context.Background(), "c", 1, 18)
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestCategoryService_GetCategoryStats(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryStatsFunc: func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
			if categoryID != 9 {
				t.Fatalf("id=%d", categoryID)
			}
			return &models.CategoryStats{VideosCount: 6, ViewsCount: 10}, nil
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	st, err := svc.GetCategoryStats(context.Background(), 9)
	if err != nil || st.VideosCount != 6 {
		t.Fatal(err, st)
	}
}

func TestCategoryService_GetCategoryBySlug_SubStatsErrorStillReturnsSubs(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 1, Slug: slug}, nil
		},
		GetSubCategoriesByCategoryIDFunc: func(ctx context.Context, categoryID uint64) ([]*models.VideoSubCategory, error) {
			return []*models.VideoSubCategory{{ID: 2}}, nil
		},
		GetSubCategoryStatsByCategoryIDFunc: func(ctx context.Context, categoryID uint64) (map[uint64]*models.SubCategoryStats, error) {
			return nil, errors.New("batch stats")
		},
		GetCategoryStatsFunc: func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
			return &models.CategoryStats{VideosCount: 1}, nil
		},
	}
	svc := service.NewCategoryService(mc, &testutil.MockVideoRepo{})
	d, err := svc.GetCategoryBySlug(context.Background(), "cat")
	if err != nil || len(d.SubCategories) != 1 || d.SubCategoryStats != nil || d.Stats == nil {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}
