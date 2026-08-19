package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "metarang/shared/pb/auth"
	"metarang/social-service/internal/middleware"
	"metarang/social-service/tests/internal/testutil"
)

func TestAuthMiddleware_Success(t *testing.T) {
	auth := &testutil.MockAuthGRPCClient{
		ValidateTokenFunc: func(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
			assert.Equal(t, "good-token", req.Token)
			return &pb.ValidateTokenResponse{Valid: true, UserId: 99, Email: "a@b.com"}, nil
		},
	}
	mw := middleware.AuthMiddleware(auth)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user, err := middleware.GetUserFromRequest(r)
		require.NoError(t, err)
		assert.Equal(t, uint64(99), user.UserID)
		assert.Equal(t, "good-token", user.Token)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestAuthMiddleware_CookieToken(t *testing.T) {
	auth := &testutil.MockAuthGRPCClient{}
	mw := middleware.AuthMiddleware(auth)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := middleware.GetUserFromRequest(r)
		require.NoError(t, err)
		assert.Equal(t, uint64(42), user.UserID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_Unauthenticated(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := middleware.AuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("missing token", func(t *testing.T) {
		handler := middleware.AuthMiddleware(&testutil.MockAuthGRPCClient{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		auth := &testutil.MockAuthGRPCClient{
			ValidateTokenFunc: func(context.Context, *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
				return &pb.ValidateTokenResponse{Valid: false}, nil
			},
		}
		handler := middleware.AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("validate error", func(t *testing.T) {
		auth := &testutil.MockAuthGRPCClient{
			ValidateTokenFunc: func(context.Context, *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
				return nil, errors.New("auth down")
			},
		}
		handler := middleware.AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer x")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestGetUserFromRequest_MissingContext(t *testing.T) {
	_, err := middleware.GetUserFromRequest(httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Error(t, err)
}

func TestAuthMiddleware_RawAuthorizationHeader(t *testing.T) {
	auth := &testutil.MockAuthGRPCClient{
		ValidateTokenFunc: func(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
			assert.Equal(t, "raw-token", req.Token)
			return &pb.ValidateTokenResponse{Valid: true, UserId: 1}, nil
		},
	}
	handler := middleware.AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "raw-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
