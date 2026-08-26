package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
	sharedauth "metarang/shared/pkg/auth"
)

func TestHTTPAuthHandler_CoreRoutes(t *testing.T) {
	photo := &mockProfilePhotoService{gatewayURL: "https://cdn"}
	tokenRepo := &mockTokenRepository{}
	authSvc := &mockAuthService{}
	authSvc.registerFunc = func(context.Context, string, string) (string, error) {
		return "https://oauth.example/register", nil
	}
	authSvc.redirectFunc = func(context.Context, string, string) (string, string, error) {
		return "https://oauth.example/redirect", "st", nil
	}
	authSvc.callbackFunc = func(context.Context, string, string, string, bool) (*service.CallbackResult, error) {
		return &service.CallbackResult{Token: "tok", ExpiresAt: 55, RedirectURL: "https://app/cb"}, nil
	}
	authSvc.getMeFunc = func(context.Context, string) (*service.UserDetails, error) {
		return &service.UserDetails{ID: 1, Name: "n", Code: "c", AutomaticLogout: 30}, nil
	}
	authSvc.validateTokenFunc = func(context.Context, string) (*models.User, error) {
		return &models.User{ID: 1, Email: "e@x.com"}, nil
	}
	authSvc.logoutFunc = func(context.Context, uint64, string, string) error { return nil }
	tokenRepo.validateTokenFunc = func(context.Context, string) (*models.User, error) {
		return &models.User{ID: 1}, nil
	}

	authServer := handler.NewAuthHandler(authSvc, tokenRepo, photo, "en")
	clients := handler.NewLocalClients(
		authServer,
		&pb.UnimplementedUserServiceServer{},
		&pb.UnimplementedKYCServiceServer{},
		&pb.UnimplementedCitizenServiceServer{},
		&pb.UnimplementedPersonalInfoServiceServer{},
		&pb.UnimplementedProfileLimitationServiceServer{},
		&pb.UnimplementedProfilePhotoServiceServer{},
		&pb.UnimplementedSettingsServiceServer{},
		&pb.UnimplementedUserEventsServiceServer{},
		&pb.UnimplementedSearchServiceServer{},
		&pb.UnimplementedWalletConnectionServiceServer{},
	)
	httpH := handler.NewHTTPAuthHandler(clients, nil, "en")

	t.Run("register", func(t *testing.T) {
		body := bytes.NewBufferString(`{"back_url":"https://app","referral":""}`)
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
		r.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		httpH.Register(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("redirect", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/auth/redirect?redirect_to=a&back_url=b", nil)
		rr := httptest.NewRecorder()
		httpH.Redirect(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("callback redirect", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/auth/callback?state=s&code=c", nil)
		rr := httptest.NewRecorder()
		httpH.Callback(rr, r)
		if rr.Code != http.StatusFound {
			t.Fatalf("code=%d", rr.Code)
		}
	})

	t.Run("validate", func(t *testing.T) {
		body := bytes.NewBufferString(`{"token":"t"}`)
		r := httptest.NewRequest(http.MethodPost, "/api/auth/validate", body)
		r.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		httpH.ValidateToken(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("get me", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
		ctx := context.WithValue(r.Context(), sharedauth.UserContextKey{}, &sharedauth.UserContext{
			UserID: 1, Token: "tok",
		})
		r = r.WithContext(ctx)
		rr := httptest.NewRecorder()
		httpH.GetMe(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("logout", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		ctx := context.WithValue(r.Context(), sharedauth.UserContextKey{}, &sharedauth.UserContext{
			UserID: 1, Token: "tok",
		})
		r = r.WithContext(ctx)
		rr := httptest.NewRecorder()
		httpH.Logout(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}
