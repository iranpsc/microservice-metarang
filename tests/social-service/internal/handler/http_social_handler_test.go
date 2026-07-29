package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	socialpb "metarang/shared/pb/social"
	authpkg "metarang/shared/pkg/auth"
	"metarang/social-service/internal/handler"
)

type mockFollowAPI struct {
	GetFollowersFunc func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error)
	GetFollowingFunc func(context.Context, *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error)
	FollowFunc       func(context.Context, *socialpb.FollowRequest) (*emptypb.Empty, error)
	UnfollowFunc     func(context.Context, *socialpb.UnfollowRequest) (*emptypb.Empty, error)
	RemoveFunc       func(context.Context, *socialpb.RemoveRequest) (*emptypb.Empty, error)
}

func (m *mockFollowAPI) GetFollowers(ctx context.Context, req *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
	if m.GetFollowersFunc != nil {
		return m.GetFollowersFunc(ctx, req)
	}
	return &socialpb.GetFollowersResponse{}, nil
}

func (m *mockFollowAPI) GetFollowing(ctx context.Context, req *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error) {
	if m.GetFollowingFunc != nil {
		return m.GetFollowingFunc(ctx, req)
	}
	return &socialpb.GetFollowingResponse{}, nil
}

func (m *mockFollowAPI) Follow(ctx context.Context, req *socialpb.FollowRequest) (*emptypb.Empty, error) {
	if m.FollowFunc != nil {
		return m.FollowFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockFollowAPI) Unfollow(ctx context.Context, req *socialpb.UnfollowRequest) (*emptypb.Empty, error) {
	if m.UnfollowFunc != nil {
		return m.UnfollowFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockFollowAPI) Remove(ctx context.Context, req *socialpb.RemoveRequest) (*emptypb.Empty, error) {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

type mockChallengeAPI struct {
	GetTimingsFunc       func(context.Context, *socialpb.GetTimingsRequest) (*socialpb.GetTimingsResponse, error)
	GetQuestionFunc      func(context.Context, *socialpb.GetQuestionRequest) (*socialpb.GetQuestionResponse, error)
	SubmitAnswerFunc     func(context.Context, *socialpb.SubmitAnswerRequest) (*socialpb.SubmitAnswerResponse, error)
	GetAdvertisementFunc func(context.Context, *socialpb.GetAdvertisementRequest) (*socialpb.GetAdvertisementResponse, error)
}

func (m *mockChallengeAPI) GetTimings(ctx context.Context, req *socialpb.GetTimingsRequest) (*socialpb.GetTimingsResponse, error) {
	if m.GetTimingsFunc != nil {
		return m.GetTimingsFunc(ctx, req)
	}
	return &socialpb.GetTimingsResponse{}, nil
}

func (m *mockChallengeAPI) GetQuestion(ctx context.Context, req *socialpb.GetQuestionRequest) (*socialpb.GetQuestionResponse, error) {
	if m.GetQuestionFunc != nil {
		return m.GetQuestionFunc(ctx, req)
	}
	return &socialpb.GetQuestionResponse{}, nil
}

func (m *mockChallengeAPI) SubmitAnswer(ctx context.Context, req *socialpb.SubmitAnswerRequest) (*socialpb.SubmitAnswerResponse, error) {
	if m.SubmitAnswerFunc != nil {
		return m.SubmitAnswerFunc(ctx, req)
	}
	return &socialpb.SubmitAnswerResponse{}, nil
}

func (m *mockChallengeAPI) GetAdvertisement(ctx context.Context, req *socialpb.GetAdvertisementRequest) (*socialpb.GetAdvertisementResponse, error) {
	if m.GetAdvertisementFunc != nil {
		return m.GetAdvertisementFunc(ctx, req)
	}
	return &socialpb.GetAdvertisementResponse{}, nil
}

func requestWithUser(r *http.Request, userID uint64) *http.Request {
	userCtx := &authpkg.UserContext{UserID: userID}
	ctx := context.WithValue(r.Context(), authpkg.UserContextKey{}, userCtx)
	return r.WithContext(ctx)
}

func sampleFollowResources(n int) []*socialpb.FollowResource {
	out := make([]*socialpb.FollowResource, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, &socialpb.FollowResource{
			Id:           uint64(i),
			Name:         "User",
			Code:         "c",
			Level:        "lvl1",
			Online:       i%2 == 0,
			ProfilePhoto: "http://p",
			Followed:     i == 1,
			Can: &socialpb.FollowPermissions{
				Follow:         i != 1,
				Unfollow:       i == 1,
				RemoveFollower: i == 2,
			},
		})
	}
	return out
}

func TestHTTPGetFollowers_PaginatedShapeAndPerPage(t *testing.T) {
	t.Setenv("APP_URL", "http://localhost:8000")

	follow := &mockFollowAPI{}
	follow.GetFollowersFunc = func(_ context.Context, req *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
		require.Equal(t, uint64(42), req.UserId)
		return &socialpb.GetFollowersResponse{Data: sampleFollowResources(12)}, nil
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers", nil), 42)
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data := body["data"].([]interface{})
	require.Len(t, data, 10)

	meta := body["meta"].(map[string]interface{})
	assert.EqualValues(t, 1, meta["current_page"])
	assert.EqualValues(t, 10, meta["per_page"])
	assert.EqualValues(t, 1, meta["from"])
	assert.EqualValues(t, 10, meta["to"])
	assert.Equal(t, "http://localhost:8000/api/followers", meta["path"])

	links := body["links"].(map[string]interface{})
	assert.Equal(t, "http://localhost:8000/api/followers?page=2", links["next"])
	assert.Equal(t, "http://localhost:8000/api/followers?page=1", links["first"])
	assert.Nil(t, links["prev"])
}

func TestHTTPGetFollowers_Page2(t *testing.T) {
	follow := &mockFollowAPI{}
	follow.GetFollowersFunc = func(_ context.Context, _ *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
		return &socialpb.GetFollowersResponse{Data: sampleFollowResources(12)}, nil
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers?page=2", nil), 1)
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data := body["data"].([]interface{})
	require.Len(t, data, 2)

	links := body["links"].(map[string]interface{})
	assert.Nil(t, links["next"])
	assert.NotNil(t, links["prev"])
}

func TestHTTPGetFollowers_UnauthorizedAndErrors(t *testing.T) {
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, &mockChallengeAPI{})

	w := httptest.NewRecorder()
	h.GetFollowers(w, httptest.NewRequest(http.MethodGet, "/api/followers", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/followers", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	follow := &mockFollowAPI{
		GetFollowersFunc: func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
			return nil, errors.New("boom")
		},
	}
	h = handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
	w = httptest.NewRecorder()
	h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers", nil), 1))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHTTPGetFollowing(t *testing.T) {
	follow := &mockFollowAPI{
		GetFollowingFunc: func(_ context.Context, req *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error) {
			require.Equal(t, uint64(5), req.UserId)
			return &socialpb.GetFollowingResponse{Data: sampleFollowResources(3)}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	w := httptest.NewRecorder()
	h.GetFollowing(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/following", nil), 5))
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	h.GetFollowing(w, httptest.NewRequest(http.MethodGet, "/api/following", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	h.GetFollowing(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/following", nil), 5))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	follow.GetFollowingFunc = func(context.Context, *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error) {
		return nil, status.Error(codes.Unavailable, "down")
	}
	w = httptest.NewRecorder()
	h.GetFollowing(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/following", nil), 5))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHTTPFollow_ProfileLimitationDenied(t *testing.T) {
	follow := &mockFollowAPI{}
	follow.FollowFunc = func(_ context.Context, _ *socialpb.FollowRequest) (*emptypb.Empty, error) {
		return nil, status.Error(codes.PermissionDenied, "این کاربر امکان دنبال کردن را  برای شما غیر فعال کرده است.")
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/follow/99", nil), 1)
	w := httptest.NewRecorder()
	h.Follow(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHTTPFollow_SuccessAndValidation(t *testing.T) {
	follow := &mockFollowAPI{}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	w := httptest.NewRecorder()
	h.Follow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/follow/99", nil), 1))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	h.Follow(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/follow/99", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	w = httptest.NewRecorder()
	h.Follow(w, httptest.NewRequest(http.MethodGet, "/api/follow/99", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	h.Follow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/follow/", nil), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	h.Follow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/follow/abc", nil), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user ID")
}

func TestHTTPUnfollowAndRemove(t *testing.T) {
	follow := &mockFollowAPI{}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	w := httptest.NewRecorder()
	h.Unfollow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/unfollow/7", nil), 1))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	h.Remove(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/remove/7", nil), 1))
	assert.Equal(t, http.StatusOK, w.Code)

	follow.UnfollowFunc = func(context.Context, *socialpb.UnfollowRequest) (*emptypb.Empty, error) {
		return nil, status.Error(codes.NotFound, "missing")
	}
	w = httptest.NewRecorder()
	h.Unfollow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/unfollow/7", nil), 1))
	assert.Equal(t, http.StatusNotFound, w.Code)

	follow.RemoveFunc = func(context.Context, *socialpb.RemoveRequest) (*emptypb.Empty, error) {
		return nil, status.Error(codes.AlreadyExists, "exists")
	}
	w = httptest.NewRecorder()
	h.Remove(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/remove/7", nil), 1))
	assert.Equal(t, http.StatusConflict, w.Code)

	w = httptest.NewRecorder()
	h.Unfollow(w, httptest.NewRequest(http.MethodGet, "/api/unfollow/7", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	h.Remove(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/remove/7", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	w = httptest.NewRecorder()
	h.Unfollow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/unfollow/x", nil), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	h.Remove(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/remove/", nil), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHTTPGetTimings_IncludesZeroAnswerCounts(t *testing.T) {
	challenge := &mockChallengeAPI{}
	challenge.GetTimingsFunc = func(_ context.Context, req *socialpb.GetTimingsRequest) (*socialpb.GetTimingsResponse, error) {
		require.Equal(t, uint64(42), req.UserId)
		return &socialpb.GetTimingsResponse{
			Data: &socialpb.TimingsData{
				DisplayAdInterval:       15,
				DisplayQuestionInterval: 15,
				DisplayAnswerInterval:   15,
				Participants:            2,
				CorrectAnswers:          0,
				WrongAnswers:            0,
			},
		}, nil
	}
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, challenge)

	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/timings", nil), 42)
	w := httptest.NewRecorder()
	h.GetTimings(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 15, data["display_ad_interval"])
	assert.EqualValues(t, 0, data["correct_answers"])
	assert.EqualValues(t, 0, data["wrong_answers"])
}

func TestHTTPGetTimings_NilDataAndErrors(t *testing.T) {
	challenge := &mockChallengeAPI{
		GetTimingsFunc: func(context.Context, *socialpb.GetTimingsRequest) (*socialpb.GetTimingsResponse, error) {
			return &socialpb.GetTimingsResponse{Data: nil}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, challenge)

	w := httptest.NewRecorder()
	h.GetTimings(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/timings", nil), 1))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	h.GetTimings(w, httptest.NewRequest(http.MethodGet, "/api/challenge/timings", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	h.GetTimings(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/timings", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	challenge.GetTimingsFunc = func(context.Context, *socialpb.GetTimingsRequest) (*socialpb.GetTimingsResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "nope")
	}
	w = httptest.NewRecorder()
	h.GetTimings(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/timings", nil), 1))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHTTPGetQuestion(t *testing.T) {
	challenge := &mockChallengeAPI{
		GetQuestionFunc: func(_ context.Context, req *socialpb.GetQuestionRequest) (*socialpb.GetQuestionResponse, error) {
			return &socialpb.GetQuestionResponse{
				Data: &socialpb.QuestionResource{Id: 1, Title: "Q"},
			}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, challenge)

	w := httptest.NewRecorder()
	h.GetQuestion(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/question", nil), 1))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"title":"Q"`)

	w = httptest.NewRecorder()
	h.GetQuestion(w, httptest.NewRequest(http.MethodGet, "/api/challenge/question", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	challenge.GetQuestionFunc = func(context.Context, *socialpb.GetQuestionRequest) (*socialpb.GetQuestionResponse, error) {
		return nil, status.Error(codes.FailedPrecondition, "pre")
	}
	w = httptest.NewRecorder()
	h.GetQuestion(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/question", nil), 1))
	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
}

func TestHTTPSubmitAnswer(t *testing.T) {
	challenge := &mockChallengeAPI{
		SubmitAnswerFunc: func(_ context.Context, req *socialpb.SubmitAnswerRequest) (*socialpb.SubmitAnswerResponse, error) {
			require.Equal(t, uint64(1), req.UserId)
			require.Equal(t, uint64(10), req.QuestionId)
			require.Equal(t, uint64(20), req.AnswerId)
			return &socialpb.SubmitAnswerResponse{
				Data: &socialpb.QuestionResource{Id: 10, Title: "done"},
			}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, challenge)

	body := `{"question_id":10,"answer_id":20}`
	w := httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", strings.NewReader(body)), 1))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/answer", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	w = httptest.NewRecorder()
	h.SubmitAnswer(w, httptest.NewRequest(http.MethodPost, "/api/challenge/answer", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req := requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", nil), 1)
	req.Body = nil
	h.SubmitAnswer(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", bytes.NewBufferString(`{`)), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", bytes.NewBufferString(`{"answer_id":20}`)), 1))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	w = httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", bytes.NewBufferString(`{"question_id":10}`)), 1))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	challenge.SubmitAnswerFunc = func(context.Context, *socialpb.SubmitAnswerRequest) (*socialpb.SubmitAnswerResponse, error) {
		return nil, status.Error(codes.InvalidArgument, "bad")
	}
	w = httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", bytes.NewBufferString(body)), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHTTPGetAdvertisement(t *testing.T) {
	challenge := &mockChallengeAPI{
		GetAdvertisementFunc: func(context.Context, *socialpb.GetAdvertisementRequest) (*socialpb.GetAdvertisementResponse, error) {
			return &socialpb.GetAdvertisementResponse{
				Advertisements: []*socialpb.AdvertisementResource{{
					Code: "ad1", Title: "T", Description: "D", InvestmentValue: "1",
					EndsAt: "soon", VideoUrl: "v", ImageUrl: "i", Url: "u", InvestmentAsset: "red",
				}},
			}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, challenge)

	w := httptest.NewRecorder()
	h.GetAdvertisement(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/advertisement", nil), 1))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"ad1"`)

	w = httptest.NewRecorder()
	h.GetAdvertisement(w, httptest.NewRequest(http.MethodGet, "/api/challenge/advertisement", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	h.GetAdvertisement(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/advertisement", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	challenge.GetAdvertisementFunc = func(context.Context, *socialpb.GetAdvertisementRequest) (*socialpb.GetAdvertisementResponse, error) {
		return nil, status.Error(codes.Internal, "fail")
	}
	w = httptest.NewRecorder()
	h.GetAdvertisement(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/advertisement", nil), 1))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRegisterHTTPRoutes_HealthAndAuth(t *testing.T) {
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, &mockChallengeAPI{})
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, requestWithUser(r, 1))
		})
	}
	h.RegisterHTTPRoutes(mux, auth)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/followers", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHTTPPublicBaseURL_ForwardedProto(t *testing.T) {
	t.Setenv("APP_URL", "")
	follow := &mockFollowAPI{
		GetFollowersFunc: func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
			return &socialpb.GetFollowersResponse{Data: sampleFollowResources(1)}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers", nil), 1)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "https://example.com/api/followers")
}
