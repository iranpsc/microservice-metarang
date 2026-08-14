package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/handler"
)

func TestHTTPContract_GetVideoSuccessAndFilters(t *testing.T) {
	liked := true
	video := &mockVideoAPI{
		GetVideoFunc: func(_ context.Context, req *trainingpb.GetVideoRequest) (*trainingpb.VideoResponse, error) {
			if req.Slug != "intro" || req.UserId != 2 || req.IpAddress == "" {
				t.Fatalf("req=%+v", req)
			}
			return &trainingpb.VideoResponse{
				Id: 1, Title: "Intro", Slug: "intro", VideoUrl: "/v.mp4", ImageUrl: "/i.jpg",
				Creator:         &commonpb.UserBasic{Name: "C", Code: "c"},
				Stats:           &trainingpb.VideoStats{ViewsCount: 10, LikesCount: 2, DislikesCount: 1, CommentsCount: 3},
				UserInteraction: &liked,
			}, nil
		},
		GetVideosFunc: func(_ context.Context, req *trainingpb.GetVideosRequest) (*trainingpb.VideosResponse, error) {
			if req.CategoryId != 3 || req.SubCategoryId != 4 {
				t.Fatalf("filters=%+v", req)
			}
			return &trainingpb.VideosResponse{
				Videos:     []*trainingpb.VideoResponse{{Id: 9, Title: "Filtered", Slug: "filtered"}},
				Pagination: &commonpb.PaginationMeta{CurrentPage: 1, PerPage: 18, Total: 1, LastPage: 1},
			}, nil
		},
		GetVideoByFileNameFunc: func(_ context.Context, req *trainingpb.GetVideoByFileNameRequest) (*trainingpb.VideoResponse, error) {
			if req.FileName != "clip.mp4" {
				t.Fatalf("file=%s", req.FileName)
			}
			return &trainingpb.VideoResponse{
				Id: 3, Title: "Clip", VideoUrl: "/uploads/clip.mp4", CreatorCode: "c1",
				Stats: &trainingpb.VideoStats{ViewsCount: 5, LikesCount: 1, DislikesCount: 0},
			}, nil
		},
	}
	h := handler.NewHTTPTrainingHandler(video, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	optional := func(next http.Handler) http.Handler {
		return withUser(2)(forwardedForMW("203.0.113.1")(next))
	}
	mux := newTestMux(h, identityMW, optional)

	rr := doJSON(mux, http.MethodGet, "/api/tutorials/intro", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get video code=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"user_interaction":true`) {
		t.Fatalf("body=%s", rr.Body.String())
	}

	rr = doJSON(mux, http.MethodGet, "/api/tutorials?category_id=3&sub_category_id=4", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list code=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = doJSON(mux, http.MethodPost, "/api/video-tutorials", `{"url":"clip.mp4"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("modal code=%d body=%s", rr.Code, rr.Body.String())
	}
	var modal map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &modal); err != nil {
		t.Fatal(err)
	}
	data, _ := modal["data"].(map[string]interface{})
	if data["views"] == nil || data["likes"] == nil {
		t.Fatalf("expected stats in modal payload: %v", data)
	}
}

func forwardedForMW(ip string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Forwarded-For", ip)
			next.ServeHTTP(w, r)
		})
	}
}
