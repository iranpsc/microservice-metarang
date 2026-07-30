package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "metarang/shared/pb/auth"

	"metarang/financial-service/internal/middleware"
	fintestutil "metarang/financial-service/tests/internal/testutil"
)

func TestAuthMiddleware_Success(t *testing.T) {
	auth := &fintestutil.MockAuthGRPCClient{
		ValidateTokenFunc: func(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
			if req.Token != "good-token" {
				t.Fatalf("token=%q", req.Token)
			}
			return &pb.ValidateTokenResponse{Valid: true, UserId: 99, Email: "a@b.com"}, nil
		},
	}
	mw := middleware.AuthMiddleware(auth)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user, err := middleware.GetUserFromRequest(r)
		if err != nil {
			t.Fatal(err)
		}
		if user.UserID != 99 || user.Token != "good-token" {
			t.Fatalf("user=%+v", user)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called {
		t.Fatalf("code=%d called=%v", rr.Code, called)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	auth := &fintestutil.MockAuthGRPCClient{}
	mw := middleware.AuthMiddleware(auth)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestAuthMiddleware_NilClient(t *testing.T) {
	mw := middleware.AuthMiddleware(nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer x")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestOptionalAuthMiddleware_WithValidToken(t *testing.T) {
	auth := &fintestutil.MockAuthGRPCClient{}
	mw := middleware.OptionalAuthMiddleware(auth)
	var userID uint64
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, err := middleware.GetUserFromRequest(r); err == nil {
			userID = user.UserID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || userID != 42 {
		t.Fatalf("code=%d userID=%d", rr.Code, userID)
	}
}

func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	auth := &fintestutil.MockAuthGRPCClient{}
	mw := middleware.OptionalAuthMiddleware(auth)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestOptionalAuthMiddleware_CookieToken(t *testing.T) {
	auth := &fintestutil.MockAuthGRPCClient{}
	mw := middleware.OptionalAuthMiddleware(auth)
	var userID uint64
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, err := middleware.GetUserFromRequest(r); err == nil {
			userID = user.UserID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if userID != 42 {
		t.Fatalf("userID=%d", userID)
	}
}
