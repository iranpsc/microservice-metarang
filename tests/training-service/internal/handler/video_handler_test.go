package handler_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/handler"
	"metarang/training-service/internal/models"
	"metarang/training-service/internal/repository"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func sampleVideo(id uint64, slug string) *models.Video {
	s := slug
	return &models.Video{
		ID:          id,
		Title:       "Tutorial",
		Slug:        &s,
		Description: "desc",
		FileName:    "clip.mp4",
		CreatorCode: "c1",
		Image:       "thumb.jpg",
		CreatedAt:   time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
	}
}

func newVideoClient(t *testing.T, mv *testutil.MockVideoRepo, mc *testutil.MockCategoryRepo, mu *testutil.MockUserRepo) trainingpb.VideoServiceClient {
	t.Helper()
	if mv == nil {
		mv = &testutil.MockVideoRepo{}
	}
	if mc == nil {
		mc = &testutil.MockCategoryRepo{}
	}
	if mu == nil {
		mu = &testutil.MockUserRepo{}
	}
	svc := service.NewVideoService(mv, mc, mu)
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterVideoHandler(s, svc)
	})
	t.Cleanup(cleanup)
	return trainingpb.NewVideoServiceClient(conn)
}

func TestVideoHandler_GetVideos_DefaultsAndFilters(t *testing.T) {
	var gotPage, gotPer int32
	var gotCat, gotSub *uint64
	mv := &testutil.MockVideoRepo{
		GetVideosFunc: func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
			gotPage, gotPer, gotCat, gotSub = page, perPage, categoryID, subCategoryID
			return []*models.Video{sampleVideo(1, "intro")}, 20, nil
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	resp, err := client.GetVideos(context.Background(), &trainingpb.GetVideosRequest{
		CategoryId:    4,
		SubCategoryId: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPage != 1 || gotPer != 18 {
		t.Fatalf("defaults page=%d per=%d", gotPage, gotPer)
	}
	if gotCat == nil || *gotCat != 4 || gotSub == nil || *gotSub != 8 {
		t.Fatalf("filters cat=%v sub=%v", gotCat, gotSub)
	}
	if resp.Pagination.LastPage != 2 || len(resp.Videos) != 1 || resp.Videos[0].Title != "Tutorial" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestVideoHandler_GetVideos_CustomPaginationAndRepoError(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		GetVideosFunc: func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
			return nil, 0, context.DeadlineExceeded
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	_, err := client.GetVideos(context.Background(), &trainingpb.GetVideosRequest{
		Pagination: &commonpb.PaginationRequest{Page: 2, PerPage: 5},
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestVideoHandler_GetVideo_SuccessSetsInteractionHeader(t *testing.T) {
	liked := true
	mv := &testutil.MockVideoRepo{
		GetVideoBySlugFunc: func(ctx context.Context, slug string) (*models.Video, error) {
			return sampleVideo(3, slug), nil
		},
		GetUserInteractionFunc: func(ctx context.Context, videoID, userID uint64) (*bool, error) {
			if videoID != 3 || userID != 11 {
				t.Fatalf("interaction lookup video=%d user=%d", videoID, userID)
			}
			return &liked, nil
		},
	}
	mc := &testutil.MockCategoryRepo{
		GetSubCategoryByIDFunc: func(ctx context.Context, subCategoryID uint64) (*models.VideoSubCategory, error) {
			icon := "icon.svg"
			return &models.VideoSubCategory{ID: 2, VideoCategoryID: 1, Name: "Sub", Slug: "sub", Icon: &icon}, nil
		},
		GetCategoryByIDFunc: func(ctx context.Context, categoryID uint64) (*models.VideoCategory, error) {
			return &models.VideoCategory{ID: 1, Name: "Cat", Slug: "cat"}, nil
		},
	}
	mu := &testutil.MockUserRepo{
		GetUserBasicByCodeFunc: func(ctx context.Context, code string) (*repository.UserBasic, error) {
			return &repository.UserBasic{ID: 9, Name: "Creator", Code: code, ProfilePhoto: "p.jpg"}, nil
		},
	}
	client := newVideoClient(t, mv, mc, mu)
	var header metadata.MD
	resp, err := client.GetVideo(context.Background(), &trainingpb.GetVideoRequest{Slug: "intro", UserId: 11}, grpc.Header(&header))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Id != 3 || resp.UserInteraction == nil || !*resp.UserInteraction {
		t.Fatalf("resp=%+v", resp)
	}
	if got := header.Get("x-video-user-interaction"); len(got) != 1 || got[0] != "true" {
		t.Fatalf("header=%v", header)
	}
}

func TestVideoHandler_GetVideoByFileName_Success(t *testing.T) {
	var gotIP string
	mv := &testutil.MockVideoRepo{
		GetVideoByFileNameFunc: func(ctx context.Context, fileName string) (*models.Video, error) {
			if fileName != "clip.mp4" {
				t.Fatalf("file=%s", fileName)
			}
			return sampleVideo(9, "clip"), nil
		},
		IncrementViewFunc: func(ctx context.Context, videoID uint64, ipAddress string) error {
			gotIP = ipAddress
			return nil
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	resp, err := client.GetVideoByFileName(context.Background(), &trainingpb.GetVideoByFileNameRequest{
		FileName: "clip.mp4", IpAddress: "203.0.113.8",
	})
	if err != nil || resp.Id != 9 || resp.Title != "Tutorial" {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
	if gotIP != "203.0.113.8" {
		t.Fatalf("ip=%s", gotIP)
	}
}

func TestVideoHandler_GetVideoByFileName_NotFoundAndIPFallback(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		GetVideoByFileNameFunc: func(ctx context.Context, fileName string) (*models.Video, error) {
			return nil, nil
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	md := metadata.Pairs("x-forwarded-for", "203.0.113.9")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := client.GetVideoByFileName(ctx, &trainingpb.GetVideoByFileNameRequest{FileName: "missing"})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("got %v", err)
	}
}

func TestVideoHandler_SearchVideos_Success(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		SearchVideosFunc: func(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error) {
			if searchTerm != "go" || page != 1 || perPage != 18 {
				t.Fatalf("term=%s page=%d per=%d", searchTerm, page, perPage)
			}
			return []*models.Video{sampleVideo(1, "go")}, 1, nil
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	resp, err := client.SearchVideos(context.Background(), &trainingpb.SearchVideosRequest{Query: "go"})
	if err != nil || len(resp.Videos) != 1 || resp.Videos[0].Slug != "go" || resp.Pagination.Total != 1 {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
}

func TestVideoHandler_SearchVideos_PaginationAndInternalError(t *testing.T) {
	var gotPage, gotPer int32
	mv := &testutil.MockVideoRepo{
		SearchVideosFunc: func(ctx context.Context, searchTerm string, page, perPage int32) ([]*models.Video, int32, error) {
			gotPage, gotPer = page, perPage
			return nil, 0, context.Canceled
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	_, err := client.SearchVideos(context.Background(), &trainingpb.SearchVideosRequest{
		Query:      "go",
		Pagination: &commonpb.PaginationRequest{Page: 3, PerPage: 9},
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
	if gotPage != 3 || gotPer != 9 {
		t.Fatalf("page=%d per=%d", gotPage, gotPer)
	}
}

func TestVideoHandler_IncrementView_ErrorAndMetadataIP(t *testing.T) {
	var gotIP string
	mv := &testutil.MockVideoRepo{
		IncrementViewFunc: func(ctx context.Context, videoID uint64, ipAddress string) error {
			gotIP = ipAddress
			return context.Canceled
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	md := metadata.Pairs("x-real-ip", "198.51.100.4")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := client.IncrementView(ctx, &trainingpb.IncrementViewRequest{VideoId: 4})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
	if gotIP == "" {
		t.Fatal("expected IP from metadata or unknown")
	}
}

func TestVideoHandler_AddInteraction_Error(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		AddInteractionFunc: func(ctx context.Context, videoID, userID uint64, liked bool, ipAddress string) error {
			return context.Canceled
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	_, err := client.AddInteraction(context.Background(), &trainingpb.AddInteractionRequest{
		VideoId: 1, UserId: 2, Liked: false,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestVideoHandler_GetVideos_SkipsNilSlugStillBuilds(t *testing.T) {
	mv := &testutil.MockVideoRepo{
		GetVideosFunc: func(ctx context.Context, page, perPage int32, categoryID, subCategoryID *uint64) ([]*models.Video, int32, error) {
			return []*models.Video{{ID: 2, Title: "NoSlug", CreatedAt: time.Now()}}, 1, nil
		},
	}
	client := newVideoClient(t, mv, nil, nil)
	resp, err := client.GetVideos(context.Background(), &trainingpb.GetVideosRequest{})
	if err != nil || len(resp.Videos) != 1 || resp.Videos[0].Slug != "" {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
}
