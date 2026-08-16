package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	authpkg "metarang/shared/pkg/auth"
	"metarang/training-service/internal/handler"
)

type mockVideoAPI struct {
	GetVideosFunc          func(context.Context, *trainingpb.GetVideosRequest) (*trainingpb.VideosResponse, error)
	GetVideoFunc           func(context.Context, *trainingpb.GetVideoRequest) (*trainingpb.VideoResponse, error)
	GetVideoByFileNameFunc func(context.Context, *trainingpb.GetVideoByFileNameRequest) (*trainingpb.VideoResponse, error)
	SearchVideosFunc       func(context.Context, *trainingpb.SearchVideosRequest) (*trainingpb.VideosResponse, error)
	AddInteractionFunc     func(context.Context, *trainingpb.AddInteractionRequest) (*commonpb.Empty, error)
}

func (m *mockVideoAPI) GetVideos(ctx context.Context, req *trainingpb.GetVideosRequest) (*trainingpb.VideosResponse, error) {
	if m.GetVideosFunc != nil {
		return m.GetVideosFunc(ctx, req)
	}
	return &trainingpb.VideosResponse{}, nil
}
func (m *mockVideoAPI) GetVideo(ctx context.Context, req *trainingpb.GetVideoRequest) (*trainingpb.VideoResponse, error) {
	if m.GetVideoFunc != nil {
		return m.GetVideoFunc(ctx, req)
	}
	return &trainingpb.VideoResponse{}, nil
}
func (m *mockVideoAPI) GetVideoByFileName(ctx context.Context, req *trainingpb.GetVideoByFileNameRequest) (*trainingpb.VideoResponse, error) {
	if m.GetVideoByFileNameFunc != nil {
		return m.GetVideoByFileNameFunc(ctx, req)
	}
	return &trainingpb.VideoResponse{}, nil
}
func (m *mockVideoAPI) SearchVideos(ctx context.Context, req *trainingpb.SearchVideosRequest) (*trainingpb.VideosResponse, error) {
	if m.SearchVideosFunc != nil {
		return m.SearchVideosFunc(ctx, req)
	}
	return &trainingpb.VideosResponse{}, nil
}
func (m *mockVideoAPI) AddInteraction(ctx context.Context, req *trainingpb.AddInteractionRequest) (*commonpb.Empty, error) {
	if m.AddInteractionFunc != nil {
		return m.AddInteractionFunc(ctx, req)
	}
	return &commonpb.Empty{}, nil
}

type mockCategoryAPI struct {
	GetCategoriesFunc     func(context.Context, *trainingpb.GetCategoriesRequest) (*trainingpb.CategoriesResponse, error)
	GetCategoryFunc       func(context.Context, *trainingpb.GetCategoryRequest) (*trainingpb.CategoryResponse, error)
	GetSubCategoryFunc    func(context.Context, *trainingpb.GetSubCategoryRequest) (*trainingpb.SubCategoryResponse, error)
	GetCategoryVideosFunc func(context.Context, *trainingpb.GetCategoryVideosRequest) (*trainingpb.VideosResponse, error)
}

func (m *mockCategoryAPI) GetCategories(ctx context.Context, req *trainingpb.GetCategoriesRequest) (*trainingpb.CategoriesResponse, error) {
	if m.GetCategoriesFunc != nil {
		return m.GetCategoriesFunc(ctx, req)
	}
	return &trainingpb.CategoriesResponse{}, nil
}
func (m *mockCategoryAPI) GetCategory(ctx context.Context, req *trainingpb.GetCategoryRequest) (*trainingpb.CategoryResponse, error) {
	if m.GetCategoryFunc != nil {
		return m.GetCategoryFunc(ctx, req)
	}
	return &trainingpb.CategoryResponse{}, nil
}
func (m *mockCategoryAPI) GetSubCategory(ctx context.Context, req *trainingpb.GetSubCategoryRequest) (*trainingpb.SubCategoryResponse, error) {
	if m.GetSubCategoryFunc != nil {
		return m.GetSubCategoryFunc(ctx, req)
	}
	return &trainingpb.SubCategoryResponse{}, nil
}
func (m *mockCategoryAPI) GetCategoryVideos(ctx context.Context, req *trainingpb.GetCategoryVideosRequest) (*trainingpb.VideosResponse, error) {
	if m.GetCategoryVideosFunc != nil {
		return m.GetCategoryVideosFunc(ctx, req)
	}
	return &trainingpb.VideosResponse{}, nil
}

type mockCommentAPI struct {
	GetCommentsFunc           func(context.Context, *trainingpb.GetCommentsRequest) (*trainingpb.CommentsResponse, error)
	AddCommentFunc            func(context.Context, *trainingpb.AddCommentRequest) (*trainingpb.CommentResponse, error)
	UpdateCommentFunc         func(context.Context, *trainingpb.UpdateCommentRequest) (*trainingpb.CommentResponse, error)
	DeleteCommentFunc         func(context.Context, *trainingpb.DeleteCommentRequest) (*commonpb.Empty, error)
	AddCommentInteractionFunc func(context.Context, *trainingpb.AddCommentInteractionRequest) (*commonpb.Empty, error)
	ReportCommentFunc         func(context.Context, *trainingpb.ReportCommentRequest) (*commonpb.Empty, error)
}

func (m *mockCommentAPI) GetComments(ctx context.Context, req *trainingpb.GetCommentsRequest) (*trainingpb.CommentsResponse, error) {
	if m.GetCommentsFunc != nil {
		return m.GetCommentsFunc(ctx, req)
	}
	return &trainingpb.CommentsResponse{}, nil
}
func (m *mockCommentAPI) AddComment(ctx context.Context, req *trainingpb.AddCommentRequest) (*trainingpb.CommentResponse, error) {
	if m.AddCommentFunc != nil {
		return m.AddCommentFunc(ctx, req)
	}
	return &trainingpb.CommentResponse{}, nil
}
func (m *mockCommentAPI) UpdateComment(ctx context.Context, req *trainingpb.UpdateCommentRequest) (*trainingpb.CommentResponse, error) {
	if m.UpdateCommentFunc != nil {
		return m.UpdateCommentFunc(ctx, req)
	}
	return &trainingpb.CommentResponse{}, nil
}
func (m *mockCommentAPI) DeleteComment(ctx context.Context, req *trainingpb.DeleteCommentRequest) (*commonpb.Empty, error) {
	if m.DeleteCommentFunc != nil {
		return m.DeleteCommentFunc(ctx, req)
	}
	return &commonpb.Empty{}, nil
}
func (m *mockCommentAPI) AddCommentInteraction(ctx context.Context, req *trainingpb.AddCommentInteractionRequest) (*commonpb.Empty, error) {
	if m.AddCommentInteractionFunc != nil {
		return m.AddCommentInteractionFunc(ctx, req)
	}
	return &commonpb.Empty{}, nil
}
func (m *mockCommentAPI) ReportComment(ctx context.Context, req *trainingpb.ReportCommentRequest) (*commonpb.Empty, error) {
	if m.ReportCommentFunc != nil {
		return m.ReportCommentFunc(ctx, req)
	}
	return &commonpb.Empty{}, nil
}

type mockReplyAPI struct {
	GetRepliesFunc          func(context.Context, *trainingpb.GetRepliesRequest) (*trainingpb.RepliesResponse, error)
	AddReplyFunc            func(context.Context, *trainingpb.AddReplyRequest) (*trainingpb.CommentResponse, error)
	UpdateReplyFunc         func(context.Context, *trainingpb.UpdateReplyRequest) (*trainingpb.CommentResponse, error)
	DeleteReplyFunc         func(context.Context, *trainingpb.DeleteReplyRequest) (*commonpb.Empty, error)
	AddReplyInteractionFunc func(context.Context, *trainingpb.AddReplyInteractionRequest) (*commonpb.Empty, error)
}

func (m *mockReplyAPI) GetReplies(ctx context.Context, req *trainingpb.GetRepliesRequest) (*trainingpb.RepliesResponse, error) {
	if m.GetRepliesFunc != nil {
		return m.GetRepliesFunc(ctx, req)
	}
	return &trainingpb.RepliesResponse{}, nil
}
func (m *mockReplyAPI) AddReply(ctx context.Context, req *trainingpb.AddReplyRequest) (*trainingpb.CommentResponse, error) {
	if m.AddReplyFunc != nil {
		return m.AddReplyFunc(ctx, req)
	}
	return &trainingpb.CommentResponse{}, nil
}
func (m *mockReplyAPI) UpdateReply(ctx context.Context, req *trainingpb.UpdateReplyRequest) (*trainingpb.CommentResponse, error) {
	if m.UpdateReplyFunc != nil {
		return m.UpdateReplyFunc(ctx, req)
	}
	return &trainingpb.CommentResponse{}, nil
}
func (m *mockReplyAPI) DeleteReply(ctx context.Context, req *trainingpb.DeleteReplyRequest) (*commonpb.Empty, error) {
	if m.DeleteReplyFunc != nil {
		return m.DeleteReplyFunc(ctx, req)
	}
	return &commonpb.Empty{}, nil
}
func (m *mockReplyAPI) AddReplyInteraction(ctx context.Context, req *trainingpb.AddReplyInteractionRequest) (*commonpb.Empty, error) {
	if m.AddReplyInteractionFunc != nil {
		return m.AddReplyInteractionFunc(ctx, req)
	}
	return &commonpb.Empty{}, nil
}

func identityMW(next http.Handler) http.Handler { return next }

func withUser(userID uint64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newTestMux(h *handler.HTTPTrainingHandler, authMW, optionalMW func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, authMW, optionalMW)
	return mux
}

func TestHTTPHealth(t *testing.T) {
	h := handler.NewHTTPTrainingHandler(&mockVideoAPI{}, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
	if body := rr.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("body=%q", body)
	}
}

func TestHTTPGetVideos(t *testing.T) {
	video := &mockVideoAPI{}
	video.GetVideosFunc = func(_ context.Context, req *trainingpb.GetVideosRequest) (*trainingpb.VideosResponse, error) {
		if req.Pagination == nil || req.Pagination.PerPage != 18 {
			t.Fatalf("pagination=%+v", req.Pagination)
		}
		return &trainingpb.VideosResponse{
			Videos: []*trainingpb.VideoResponse{
				{Id: 1, Title: "Intro", Slug: "intro"},
			},
			Pagination: &commonpb.PaginationMeta{CurrentPage: 1, PerPage: 18, Total: 1, LastPage: 1},
		}, nil
	}
	h := handler.NewHTTPTrainingHandler(video, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/tutorials", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("data=%v", body["data"])
	}
}

func TestHTTPGetVideo_UserInteraction(t *testing.T) {
	liked := true
	video := &mockVideoAPI{}
	video.GetVideoFunc = func(_ context.Context, req *trainingpb.GetVideoRequest) (*trainingpb.VideoResponse, error) {
		if req.Slug != "intro" {
			t.Fatalf("slug=%s", req.Slug)
		}
		if req.UserId != 7 {
			t.Fatalf("userID=%d", req.UserId)
		}
		return &trainingpb.VideoResponse{
			Id:              1,
			Title:           "Intro",
			Slug:            "intro",
			UserInteraction: &liked,
			Stats:           &trainingpb.VideoStats{ViewsCount: 10, LikesCount: 2},
		}, nil
	}
	h := handler.NewHTTPTrainingHandler(video, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, withUser(7))

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/tutorials/intro", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok || data["user_interaction"] != true {
		t.Fatalf("user_interaction=%v body=%v", data["user_interaction"], body)
	}
}

func TestHTTPSearchVideos_Validation(t *testing.T) {
	h := handler.NewHTTPTrainingHandler(&mockVideoAPI{}, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/tutorials/search", bytes.NewBufferString(`{"searchTerm":""}`)))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPAddInteraction_RequiresAuth(t *testing.T) {
	h := handler.NewHTTPTrainingHandler(&mockVideoAPI{}, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/tutorials/9/interactions?liked=1", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPAddInteraction_Success(t *testing.T) {
	video := &mockVideoAPI{}
	video.AddInteractionFunc = func(_ context.Context, req *trainingpb.AddInteractionRequest) (*commonpb.Empty, error) {
		if req.VideoId != 9 || req.UserId != 3 || !req.Liked {
			t.Fatalf("req=%+v", req)
		}
		return &commonpb.Empty{}, nil
	}
	h := handler.NewHTTPTrainingHandler(video, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, withUser(3), identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/tutorials/9/interactions?liked=1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPGetComments_WithInteraction(t *testing.T) {
	liked := false
	comment := &mockCommentAPI{}
	comment.GetCommentsFunc = func(_ context.Context, req *trainingpb.GetCommentsRequest) (*trainingpb.CommentsResponse, error) {
		if req.VideoId != 5 || req.UserId != 2 {
			t.Fatalf("req=%+v", req)
		}
		return &trainingpb.CommentsResponse{
			Comments: []*trainingpb.CommentResponse{
				{Id: 11, VideoId: 5, Content: "hi", UserInteraction: &liked},
			},
		}, nil
	}
	h := handler.NewHTTPTrainingHandler(&mockVideoAPI{}, &mockCategoryAPI{}, comment, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, withUser(2))

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/tutorials/5/comments", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]interface{})
	first := data[0].(map[string]interface{})
	if first["user_interaction"] != false {
		t.Fatalf("user_interaction=%v", first["user_interaction"])
	}
}

func TestHTTPGetVideo_NotFound(t *testing.T) {
	video := &mockVideoAPI{}
	video.GetVideoFunc = func(context.Context, *trainingpb.GetVideoRequest) (*trainingpb.VideoResponse, error) {
		return nil, status.Error(codes.NotFound, "video not found")
	}
	h := handler.NewHTTPTrainingHandler(video, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/tutorials/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPGetCategories(t *testing.T) {
	cat := &mockCategoryAPI{}
	cat.GetCategoriesFunc = func(context.Context, *trainingpb.GetCategoriesRequest) (*trainingpb.CategoriesResponse, error) {
		return &trainingpb.CategoriesResponse{
			Categories: []*trainingpb.CategoryResponse{{Id: 1, Name: "Basics", Slug: "basics"}},
		}, nil
	}
	h := handler.NewHTTPTrainingHandler(&mockVideoAPI{}, cat, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/tutorials/categories", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestHTTPVideoTutorialsModal(t *testing.T) {
	video := &mockVideoAPI{}
	video.GetVideoByFileNameFunc = func(_ context.Context, req *trainingpb.GetVideoByFileNameRequest) (*trainingpb.VideoResponse, error) {
		if req.FileName != "clip.mp4" {
			t.Fatalf("file=%s", req.FileName)
		}
		return &trainingpb.VideoResponse{Id: 3, Title: "Clip", VideoUrl: "/uploads/clip.mp4"}, nil
	}
	h := handler.NewHTTPTrainingHandler(video, &mockCategoryAPI{}, &mockCommentAPI{}, &mockReplyAPI{})
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/video-tutorials", bytes.NewBufferString(`{"url":"clip.mp4"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPGetReplies(t *testing.T) {
	reply := &mockReplyAPI{}
	reply.GetRepliesFunc = func(_ context.Context, req *trainingpb.GetRepliesRequest) (*trainingpb.RepliesResponse, error) {
		if req.CommentId != 8 {
			t.Fatalf("comment=%d", req.CommentId)
		}
		return &trainingpb.RepliesResponse{
			Replies: []*trainingpb.CommentResponse{{Id: 20, Content: "reply", ParentId: 8}},
		}, nil
	}
	h := handler.NewHTTPTrainingHandler(&mockVideoAPI{}, &mockCategoryAPI{}, &mockCommentAPI{}, reply)
	mux := newTestMux(h, identityMW, identityMW)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/comments/8/replies", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}
