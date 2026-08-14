package handler_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/handler"
	"metarang/training-service/internal/models"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func newCategoryClient(t *testing.T, mc *testutil.MockCategoryRepo, mv *testutil.MockVideoRepo) trainingpb.CategoryServiceClient {
	t.Helper()
	if mc == nil {
		mc = &testutil.MockCategoryRepo{}
	}
	if mv == nil {
		mv = &testutil.MockVideoRepo{}
	}
	catSvc := service.NewCategoryService(mc, mv)
	vidSvc := service.NewVideoService(mv, mc, &testutil.MockUserRepo{})
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterCategoryHandler(s, catSvc, vidSvc)
	})
	t.Cleanup(cleanup)
	return trainingpb.NewCategoryServiceClient(conn)
}

func TestCategoryHandler_GetCategories_DefaultsAndInternalError(t *testing.T) {
	var gotPage, gotPer int32
	mc := &testutil.MockCategoryRepo{
		GetCategoriesFunc: func(ctx context.Context, page, perPage int32) ([]*models.VideoCategory, int32, error) {
			gotPage, gotPer = page, perPage
			return nil, 0, context.Canceled
		},
	}
	client := newCategoryClient(t, mc, nil)
	_, err := client.GetCategories(context.Background(), &trainingpb.GetCategoriesRequest{})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
	if gotPage != 1 || gotPer != 30 {
		t.Fatalf("defaults page=%d per=%d", gotPage, gotPer)
	}
}

func TestCategoryHandler_GetCategories_WithIconsAndLastPage(t *testing.T) {
	icon := "cat.svg"
	mc := &testutil.MockCategoryRepo{
		GetCategoriesFunc: func(ctx context.Context, page, perPage int32) ([]*models.VideoCategory, int32, error) {
			return []*models.VideoCategory{{
				ID: 1, Name: "Basics", Slug: "basics", Description: "d", Image: "img.png", Icon: &icon,
			}}, 61, nil
		},
		GetCategoryStatsFunc: func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
			return &models.CategoryStats{VideosCount: 4, ViewsCount: 10, LikesCount: 2, DislikesCount: 1}, nil
		},
	}
	client := newCategoryClient(t, mc, nil)
	resp, err := client.GetCategories(context.Background(), &trainingpb.GetCategoriesRequest{
		Pagination: &commonpb.PaginationRequest{Page: 2, PerPage: 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Pagination.LastPage != 3 || len(resp.Categories) != 1 {
		t.Fatalf("pagination=%+v cats=%d", resp.Pagination, len(resp.Categories))
	}
	cat := resp.Categories[0]
	if cat.IconUrl == "" || cat.VideosCount != 4 || cat.Stats == nil || cat.Stats.ViewsCount != 10 {
		t.Fatalf("cat=%+v", cat)
	}
}

func TestCategoryHandler_GetCategory_WithSubcategories(t *testing.T) {
	icon := "sub.svg"
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 7, Name: "Cat", Slug: slug, Image: "c.png"}, nil
		},
		GetSubCategoriesByCategoryIDFunc: func(ctx context.Context, categoryID uint64) ([]*models.VideoSubCategory, error) {
			return []*models.VideoSubCategory{
				{ID: 2, Name: "Sub", Slug: "sub", Description: "sd", Image: "s.png", Icon: &icon},
			}, nil
		},
		GetSubCategoryStatsByCategoryIDFunc: func(ctx context.Context, categoryID uint64) (map[uint64]*models.SubCategoryStats, error) {
			return map[uint64]*models.SubCategoryStats{2: {VideosCount: 3, ViewsCount: 9}}, nil
		},
		GetCategoryStatsFunc: func(ctx context.Context, categoryID uint64) (*models.CategoryStats, error) {
			return &models.CategoryStats{VideosCount: 3}, nil
		},
	}
	client := newCategoryClient(t, mc, nil)
	resp, err := client.GetCategory(context.Background(), &trainingpb.GetCategoryRequest{Slug: "cat"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.SubCategories) != 1 || resp.SubCategories[0].VideosCount != 3 || resp.SubCategories[0].IconUrl == "" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestCategoryHandler_GetSubCategory_NotFound(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryBySlugsFunc: func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
			return nil, nil
		},
	}
	client := newCategoryClient(t, mc, nil)
	_, err := client.GetSubCategory(context.Background(), &trainingpb.GetSubCategoryRequest{
		CategorySlug: "c", SubCategorySlug: "s",
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("got %v", err)
	}
}

func TestCategoryHandler_GetCategoryVideos_DefaultsAndError(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return nil, context.Canceled
		},
	}
	client := newCategoryClient(t, mc, nil)
	_, err := client.GetCategoryVideos(context.Background(), &trainingpb.GetCategoryVideosRequest{CategorySlug: "c"})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestCategoryHandler_GetCategoryVideos_EmptyList(t *testing.T) {
	mc := &testutil.MockCategoryRepo{
		GetCategoryBySlugFunc: func(ctx context.Context, slug string) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 1, Slug: slug}, nil
		},
	}
	mv := &testutil.MockVideoRepo{
		GetVideosFunc: func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
			if page != 1 || perPage != 18 {
				t.Fatalf("page=%d per=%d", page, perPage)
			}
			return []*models.Video{}, 0, nil
		},
	}
	client := newCategoryClient(t, mc, mv)
	resp, err := client.GetCategoryVideos(context.Background(), &trainingpb.GetCategoryVideosRequest{CategorySlug: "c"})
	if err != nil || len(resp.Videos) != 0 || resp.Pagination.Total != 0 {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
}

func TestCategoryHandler_GetSubCategory_WithStatsAndIcon(t *testing.T) {
	icon := "i.svg"
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryBySlugsFunc: func(ctx context.Context, categorySlug, subCategorySlug string) (*models.VideoSubCategory, error) {
			return &models.VideoSubCategory{
				ID: 4, Name: "Sub", Slug: subCategorySlug, VideoCategoryID: 2,
				Description: "d", Image: "img.png", Icon: &icon, CreatedAt: time.Now(),
			}, nil
		},
		GetCategoryByIDFunc: func(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: categoryID, Name: "Cat", Slug: "cslug"}, nil
		},
		GetSubCategoryStatsFunc: func(ctx context.Context, subCategoryID uint64) (*models.SubCategoryStats, error) {
			return &models.SubCategoryStats{VideosCount: 5, ViewsCount: 11, LikesCount: 2, DislikesCount: 1}, nil
		},
	}
	client := newCategoryClient(t, mc, nil)
	resp, err := client.GetSubCategory(context.Background(), &trainingpb.GetSubCategoryRequest{
		CategorySlug: "c", SubCategorySlug: "s",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideosCount != 5 || resp.Category == nil || resp.Category.Slug != "cslug" || resp.IconUrl == "" {
		t.Fatalf("resp=%+v", resp)
	}
}
