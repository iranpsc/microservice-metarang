package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"metarang/auth-service/internal/middleware"
	sharedauth "metarang/shared/pkg/auth"
)

type stubValidator struct {
	user *sharedauth.UserContext
	err  error
}

func (s *stubValidator) ValidateToken(_ context.Context, token string) (*sharedauth.UserContext, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil {
		u := *s.user
		u.Token = token
		return &u, nil
	}
	return &sharedauth.UserContext{UserID: 1, Email: "a@b.com", Token: token}, nil
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("nil validator", func(t *testing.T) {
		h := middleware.AuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not reach next")
		}))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		h := middleware.AuthMiddleware(&stubValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not reach next")
		}))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		h := middleware.AuthMiddleware(&stubValidator{err: errors.New("bad")})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not reach next")
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer badtoken")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("bearer success", func(t *testing.T) {
		var gotUID uint64
		h := middleware.AuthMiddleware(&stubValidator{user: &sharedauth.UserContext{UserID: 42, Email: "u@x.com"}})(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				u, err := middleware.GetUserFromRequest(r)
				if err != nil {
					t.Fatalf("GetUserFromRequest: %v", err)
				}
				gotUID = u.UserID
				w.WriteHeader(http.StatusOK)
			}),
		)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer goodtoken")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK || gotUID != 42 {
			t.Fatalf("status=%d uid=%d", rr.Code, gotUID)
		}
	})

	t.Run("raw authorization without bearer prefix", func(t *testing.T) {
		called := false
		h := middleware.AuthMiddleware(&stubValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "rawtoken")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !called || rr.Code != http.StatusOK {
			t.Fatalf("called=%v status=%d", called, rr.Code)
		}
	})

	t.Run("cookie token", func(t *testing.T) {
		called := false
		h := middleware.AuthMiddleware(&stubValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "cookietoken"})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !called || rr.Code != http.StatusOK {
			t.Fatalf("called=%v status=%d", called, rr.Code)
		}
	})
}

func TestOptionalAuthMiddleware(t *testing.T) {
	t.Run("no token continues", func(t *testing.T) {
		called := false
		h := middleware.OptionalAuthMiddleware(&stubValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			if _, err := middleware.GetUserFromRequest(r); err == nil {
				t.Fatal("expected no user")
			}
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called {
			t.Fatal("expected next")
		}
	})

	t.Run("invalid token ignored", func(t *testing.T) {
		called := false
		h := middleware.OptionalAuthMiddleware(&stubValidator{err: errors.New("bad")})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer x")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !called || rr.Code != http.StatusOK {
			t.Fatalf("called=%v status=%d", called, rr.Code)
		}
	})

	t.Run("valid token sets context", func(t *testing.T) {
		h := middleware.OptionalAuthMiddleware(&stubValidator{user: &sharedauth.UserContext{UserID: 7}})(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				u, err := middleware.GetUserFromRequest(r)
				if err != nil || u.UserID != 7 {
					t.Fatalf("user=%v err=%v", u, err)
				}
				w.WriteHeader(http.StatusOK)
			}),
		)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer ok")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("nil validator", func(t *testing.T) {
		called := false
		h := middleware.OptionalAuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called {
			t.Fatal("expected next")
		}
	})
}

func TestGuestMiddleware(t *testing.T) {
	t.Run("valid token forbidden", func(t *testing.T) {
		h := middleware.GuestMiddleware(&stubValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not reach next")
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer ok")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("invalid or missing continues", func(t *testing.T) {
		called := false
		h := middleware.GuestMiddleware(&stubValidator{err: errors.New("bad")})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !called || rr.Code != http.StatusOK {
			t.Fatalf("called=%v status=%d", called, rr.Code)
		}

		called = false
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called {
			t.Fatal("expected next without token")
		}
	})

	t.Run("nil validator", func(t *testing.T) {
		called := false
		h := middleware.GuestMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called {
			t.Fatal("expected next")
		}
	})
}
