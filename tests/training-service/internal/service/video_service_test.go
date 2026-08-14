package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metarang/training-service/internal/models"
	"metarang/training-service/internal/repository"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func TestVideoService_GetVideoByFileName_NotFoundAndViewErrorIgnored(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		GetVideoByFileNameFunc: func(ctx context.Context, fileName string) (*models.Video, error) {
			return nil, nil
		},
	}
	svc := service.NewVideoService(mv, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	_, err := svc.GetVideoByFileName(context.Background(), "x", "ip")
	if err == nil {
		t.Fatal("expected not found")
	}

	mv.GetVideoByFileNameFunc = func(ctx context.Context, fileName string) (*models.Video, error) {
		return nil, context.Canceled
	}
	_, err = svc.GetVideoByFileName(context.Background(), "x", "ip")
	if err == nil {
		t.Fatal("expected repo error")
	}

	s := "sl"
	mv.GetVideoByFileNameFunc = func(ctx context.Context, fileName string) (*models.Video, error) {
		return &models.Video{ID: 4, Slug: &s}, nil
	}
	mv.IncrementViewFunc = func(ctx context.Context, videoID uint64, ipAddress string) error {
		return errors.New("view failed")
	}
	v, err := svc.GetVideoByFileName(context.Background(), "frag", "ip")
	if err != nil || v == nil || v.ID != 4 {
		t.Fatalf("view error should be ignored: err=%v v=%+v", err, v)
	}
}

func TestVideoService_GetVideoBySlug_ViewErrorIgnored(t *testing.T) {
	s := "intro"
	mv := &testutil.MockVideoRepo{
		GetVideoBySlugFunc: func(ctx context.Context, slug string) (*models.Video, error) {
			return &models.Video{ID: 2, Slug: &s}, nil
		},
		IncrementViewFunc: func(ctx context.Context, videoID uint64, ipAddress string) error {
			return errors.New("view failed")
		},
	}
	svc := service.NewVideoService(mv, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	v, err := svc.GetVideoBySlug(context.Background(), "intro", nil, "1.1.1.1")
	if err != nil || v.ID != 2 {
		t.Fatalf("err=%v v=%+v", err, v)
	}
}

func TestVideoService_GetVideoWithDetails_UserInteraction(t *testing.T) {
	liked := false
	mv := &testutil.MockVideoRepo{
		GetVideoStatsFunc: func(ctx context.Context, videoID uint64) (*models.VideoStats, error) {
			return &models.VideoStats{ViewsCount: 4}, nil
		},
		GetUserInteractionFunc: func(ctx context.Context, videoID, userID uint64) (*bool, error) {
			if userID != 12 {
				t.Fatalf("user=%d", userID)
			}
			return &liked, nil
		},
	}
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryByIDFunc: func(ctx context.Context, subCategoryID uint64) (*models.VideoSubCategory, error) {
			return &models.VideoSubCategory{ID: 5, VideoCategoryID: 7}, nil
		},
		GetCategoryByIDFunc: func(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 7, Name: "Cat"}, nil
		},
	}
	mu := &testutil.MockUserRepo{
		GetUserBasicByCodeFunc: func(ctx context.Context, code string) (*repository.UserBasic, error) {
			return &repository.UserBasic{ID: 1, Code: code, Name: "C"}, nil
		},
	}
	svc := service.NewVideoService(mv, mc, mu)
	uid := uint64(12)
	d, err := svc.GetVideoWithDetails(context.Background(), &models.Video{
		ID: 1, VideoSubCategoryID: 5, CreatorCode: "c1", CreatedAt: time.Now(),
	}, &uid)
	if err != nil || d.UserInteraction == nil || *d.UserInteraction || d.CreatedAtJalali == "" {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestVideoService_GetVideoWithDetails_IgnoresLookupErrors(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		GetVideoStatsFunc: func(ctx context.Context, videoID uint64) (*models.VideoStats, error) {
			return nil, errors.New("stats")
		},
		GetUserInteractionFunc: func(ctx context.Context, videoID, userID uint64) (*bool, error) {
			return nil, errors.New("interaction")
		},
	}
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryByIDFunc: func(ctx context.Context, subCategoryID uint64) (*models.VideoSubCategory, error) {
			return nil, errors.New("sub")
		},
	}
	mu := &testutil.MockUserRepo{
		GetUserBasicByCodeFunc: func(ctx context.Context, code string) (*repository.UserBasic, error) {
			return nil, errors.New("user")
		},
	}
	svc := service.NewVideoService(mv, mc, mu)
	uid := uint64(1)
	d, err := svc.GetVideoWithDetails(context.Background(), &models.Video{
		ID: 1, CreatorCode: "c", CreatedAt: time.Time{},
	}, &uid)
	if err != nil || d.Creator != nil || d.Category != nil || d.Stats != nil || d.UserInteraction != nil {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestVideoService_GetVideos_PassesFilters(t *testing.T) {
	cat, sub := uint64(3), uint64(4)
	mv := &testutil.MockVideoRepo{
		GetVideosFunc: func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
			if page != 2 || perPage != 9 || categoryID == nil || *categoryID != 3 || subCategoryID == nil || *subCategoryID != 4 {
				t.Fatalf("page=%d per=%d cat=%v sub=%v", page, perPage, categoryID, subCategoryID)
			}
			return []*models.Video{{ID: 1}}, 1, nil
		},
	}
	svc := service.NewVideoService(mv, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	list, total, err := svc.GetVideos(context.Background(), 2, 9, &cat, &sub)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatal(err, total, len(list))
	}
}
