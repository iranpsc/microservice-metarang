package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "metarang/shared/pb/auth"

	"metarang/calendar-service/internal/middleware"
	caltestutil "metarang/calendar-service/tests/internal/testutil"
)

func TestAuthMiddleware_Success(t *testing.T) {
	auth := &caltestutil.MockAuthGRPCClient{
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

func TestAuthMiddleware_CookieToken(t *testing.T) {
	auth := &caltestutil.MockAuthGRPCClient{}
	mw := middleware.AuthMiddleware(auth)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := middleware.GetUserFromRequest(r)
		if err != nil || user.UserID != 42 {
			t.Fatal(err, user)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestAuthMiddleware_Unauthenticated(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := middleware.AuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("code=%d", rr.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		handler := middleware.AuthMiddleware(&caltestutil.MockAuthGRPCClient{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("code=%d", rr.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		auth := &caltestutil.MockAuthGRPCClient{
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
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("code=%d", rr.Code)
		}
	})

	t.Run("validate error", func(t *testing.T) {
		auth := &caltestutil.MockAuthGRPCClient{
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
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("code=%d", rr.Code)
		}
	})
}

func TestAuthMiddleware_RawAuthorizationHeader(t *testing.T) {
	auth := &caltestutil.MockAuthGRPCClient{
		ValidateTokenFunc: func(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
			if req.Token != "raw-token" {
				t.Fatalf("token=%q", req.Token)
			}
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
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	t.Run("nil client passes through", func(t *testing.T) {
		called := false
		handler := middleware.OptionalAuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusOK || !called {
			t.Fatalf("code=%d called=%v", rr.Code, called)
		}
	})

	t.Run("valid token sets user", func(t *testing.T) {
		auth := &caltestutil.MockAuthGRPCClient{
			ValidateTokenFunc: func(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
				return &pb.ValidateTokenResponse{Valid: true, UserId: 55, Email: "x@y.com"}, nil
			},
		}
		handler := middleware.OptionalAuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := middleware.GetUserFromRequest(r)
			if err != nil || user.UserID != 55 {
				t.Fatal(err, user)
			}
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer tok")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d", rr.Code)
		}
	})

	t.Run("missing token passes through", func(t *testing.T) {
		handler := middleware.OptionalAuthMiddleware(&caltestutil.MockAuthGRPCClient{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := middleware.GetUserFromRequest(r); err == nil {
				t.Fatal("expected no user in context")
			}
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d", rr.Code)
		}
	})

	t.Run("invalid token passes through", func(t *testing.T) {
		auth := &caltestutil.MockAuthGRPCClient{
			ValidateTokenFunc: func(context.Context, *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
				return &pb.ValidateTokenResponse{Valid: false}, nil
			},
		}
		handler := middleware.OptionalAuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := middleware.GetUserFromRequest(r); err == nil {
				t.Fatal("expected no user in context")
			}
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d", rr.Code)
		}
	})
}

func TestGetUserFromRequest_MissingContext(t *testing.T) {
	if _, err := middleware.GetUserFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); err == nil {
		t.Fatal("expected error")
	}
}
