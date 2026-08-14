package handler_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	"metarang/social-service/internal/handler"
)

func firstFollowItem(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &parsed))
	data, ok := parsed["data"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, data)
	item, ok := data[0].(map[string]interface{})
	require.True(t, ok)
	return item
}

func TestHTTPFollowLists_ProfilePhotoURLAndNull(t *testing.T) {
	photoURL := "https://cdn.example.com/users/7.jpg"

	t.Run("followers profile_photo is URL", func(t *testing.T) {
		follow := &mockFollowAPI{
			GetFollowersFunc: func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
				return &socialpb.GetFollowersResponse{
					Data: []*socialpb.FollowResource{{
						Id: 7, Name: "Ada", Code: "ada", ProfilePhoto: photoURL,
						Level: "gold", Online: true, Followed: false,
						Can: &socialpb.FollowPermissions{Follow: true, Unfollow: false, RemoveFollower: true},
					}},
				}, nil
			},
		}
		h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
		w := httptest.NewRecorder()
		h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)

		item := firstFollowItem(t, w.Body.Bytes())
		assert.Equal(t, photoURL, item["profile_photo"])
		assert.EqualValues(t, 7, item["id"])
		assert.Equal(t, "Ada", item["name"])
		assert.Equal(t, "ada", item["code"])
		assert.Equal(t, "gold", item["level"])
		assert.Equal(t, true, item["online"])
		assert.Equal(t, false, item["followed"])
		can := item["can"].(map[string]interface{})
		assert.Equal(t, true, can["follow"])
		assert.Equal(t, false, can["unfollow"])
		assert.Equal(t, true, can["remove_follower"])
	})

	t.Run("following profile_photo is URL", func(t *testing.T) {
		follow := &mockFollowAPI{
			GetFollowingFunc: func(context.Context, *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error) {
				return &socialpb.GetFollowingResponse{
					Data: []*socialpb.FollowResource{{
						Id: 8, Name: "Bob", Code: "bob", ProfilePhoto: photoURL,
						Can: &socialpb.FollowPermissions{Follow: false, Unfollow: true},
					}},
				}, nil
			},
		}
		h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
		w := httptest.NewRecorder()
		h.GetFollowing(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/following", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)
		item := firstFollowItem(t, w.Body.Bytes())
		assert.Equal(t, photoURL, item["profile_photo"])
	})

	t.Run("empty profile_photo encodes as JSON null", func(t *testing.T) {
		follow := &mockFollowAPI{
			GetFollowersFunc: func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
				return &socialpb.GetFollowersResponse{
					Data: []*socialpb.FollowResource{{Id: 9, Name: "NoPhoto", Code: "np"}},
				}, nil
			},
		}
		h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
		w := httptest.NewRecorder()
		h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"profile_photo":null`)
		assert.NotContains(t, w.Body.String(), `"profile_photo":""`)
		item := firstFollowItem(t, w.Body.Bytes())
		assert.Nil(t, item["profile_photo"])
	})

	t.Run("following empty profile_photo is JSON null", func(t *testing.T) {
		follow := &mockFollowAPI{
			GetFollowingFunc: func(context.Context, *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error) {
				return &socialpb.GetFollowingResponse{
					Data: []*socialpb.FollowResource{{Id: 10, Name: "Empty"}},
				}, nil
			},
		}
		h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
		w := httptest.NewRecorder()
		h.GetFollowing(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/following", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)
		item := firstFollowItem(t, w.Body.Bytes())
		assert.Nil(t, item["profile_photo"])
	})
}

func TestHTTPGetFollowers_PaginationOverflowAndInvalidPage(t *testing.T) {
	follow := &mockFollowAPI{
		GetFollowersFunc: func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
			return &socialpb.GetFollowersResponse{Data: sampleFollowResources(3)}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	t.Run("page overflow yields empty data", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers?page=99", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		data := body["data"].([]interface{})
		assert.Empty(t, data)
		meta := body["meta"].(map[string]interface{})
		assert.EqualValues(t, 99, meta["current_page"])
		assert.Nil(t, meta["from"])
		assert.Nil(t, meta["to"])
	})

	t.Run("invalid page query ignored", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers?page=abc", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		meta := body["meta"].(map[string]interface{})
		assert.EqualValues(t, 1, meta["current_page"])
		assert.Len(t, body["data"].([]interface{}), 3)
	})

	t.Run("zero page query ignored", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.GetFollowers(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers?page=0", nil), 1))
		require.Equal(t, http.StatusOK, w.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		meta := body["meta"].(map[string]interface{})
		assert.EqualValues(t, 1, meta["current_page"])
	})
}

func TestHTTPGetQuestion_MethodNotAllowedAndNilData(t *testing.T) {
	challenge := &mockChallengeAPI{
		GetQuestionFunc: func(context.Context, *socialpb.GetQuestionRequest) (*socialpb.GetQuestionResponse, error) {
			return &socialpb.GetQuestionResponse{Data: nil}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, challenge)

	w := httptest.NewRecorder()
	h.GetQuestion(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/question", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	w = httptest.NewRecorder()
	h.GetQuestion(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/challenge/question", nil), 1))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{}\n", w.Body.String())
}

func TestHTTPUnfollowRemove_RemainingErrorMapping(t *testing.T) {
	follow := &mockFollowAPI{}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})

	w := httptest.NewRecorder()
	h.Unfollow(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/unfollow/7", nil), 1))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	w = httptest.NewRecorder()
	h.Remove(w, httptest.NewRequest(http.MethodGet, "/api/remove/7", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	follow.UnfollowFunc = func(context.Context, *socialpb.UnfollowRequest) (*emptypb.Empty, error) {
		return nil, status.Error(codes.Internal, "unfollow failed")
	}
	w = httptest.NewRecorder()
	h.Unfollow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/unfollow/7", nil), 1))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	follow.RemoveFunc = func(context.Context, *socialpb.RemoveRequest) (*emptypb.Empty, error) {
		return nil, status.Error(codes.PermissionDenied, "cannot remove")
	}
	w = httptest.NewRecorder()
	h.Remove(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/remove/7", nil), 1))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHTTPFollow_InternalError(t *testing.T) {
	follow := &mockFollowAPI{
		FollowFunc: func(context.Context, *socialpb.FollowRequest) (*emptypb.Empty, error) {
			return nil, status.Error(codes.ResourceExhausted, "rate limited")
		},
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
	w := httptest.NewRecorder()
	h.Follow(w, requestWithUser(httptest.NewRequest(http.MethodGet, "/api/follow/9", nil), 1))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHTTPSubmitAnswer_EmptyBodyEOF(t *testing.T) {
	h := handler.NewHTTPSocialHandler(&mockFollowAPI{}, &mockChallengeAPI{})
	w := httptest.NewRecorder()
	h.SubmitAnswer(w, requestWithUser(httptest.NewRequest(http.MethodPost, "/api/challenge/answer", strings.NewReader("")), 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "request body is required")
}

func TestHTTPPublicBaseURL_TLSWithoutForwardedProto(t *testing.T) {
	t.Setenv("APP_URL", "")
	follow := &mockFollowAPI{
		GetFollowersFunc: func(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error) {
			return &socialpb.GetFollowersResponse{Data: sampleFollowResources(1)}, nil
		},
	}
	h := handler.NewHTTPSocialHandler(follow, &mockChallengeAPI{})
	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/followers", nil), 1)
	req.Host = "secure.example.com"
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "https://secure.example.com/api/followers")
}
