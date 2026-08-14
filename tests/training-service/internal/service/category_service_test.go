package service_test

import (
	"context"
	"errors"
	"testing"

	"metarang/training-service/internal/models"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

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
