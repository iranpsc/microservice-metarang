package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

func TestHTTPPersonalInfoKYCSearchWalletErrors(t *testing.T) {
	s := grpc.NewServer()
	defer s.Stop()

	pi := &mockPersonalInfoService{}
	pi.getFunc = func(context.Context, uint64) (*models.PersonalInfo, error) {
		return &models.PersonalInfo{
			Occupation: sql.NullString{String: "eng", Valid: true},
			Education:  sql.NullString{String: "bs", Valid: true},
			Memory:     sql.NullString{String: "m", Valid: true},
			LovedCity:  sql.NullString{String: "c", Valid: true},
			LovedCountry: sql.NullString{String: "ir", Valid: true},
			LovedLanguage: sql.NullString{String: "fa", Valid: true},
			ProblemSolving: sql.NullString{String: "ps", Valid: true},
			Prediction: sql.NullString{String: "pr", Valid: true},
			About: sql.NullString{String: "a", Valid: true},
			Passions: map[string]bool{"music": true},
		}, nil
	}
	piServer := handler.RegisterPersonalInfoHandler(s, pi)

	kycMock := &mockKYCService{}
	kycMock.getKYCFunc = func(context.Context, uint64) (*models.KYC, error) {
		return &models.KYC{
			ID: 1, UserID: 1, Fname: "a", Lname: "b", MelliCode: "0123456789",
			MelliCard: "/c.png", Province: "Tehran", Status: -1,
			Video: sql.NullString{String: "/v.mp4", Valid: true},
			Gender: sql.NullString{String: "male", Valid: true},
			Birthdate: sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			Errors: sql.NullString{String: `["bad"]`, Valid: true},
		}, nil
	}
	kycServer := handler.NewKYCHandler(kycMock, "https://gw")

	walletMock := &mockWalletConnectionService{}
	walletMock.getLinkNonceFunc = func(context.Context, uint64, string) (string, error) {
		return "", status.Error(codes.InvalidArgument, "bad address")
	}
	walletServer := handler.RegisterWalletConnectionHandler(s, walletMock, "en")

	clients := handler.NewLocalClients(
		&pb.UnimplementedAuthServiceServer{},
		&pb.UnimplementedUserServiceServer{},
		kycServer, &pb.UnimplementedCitizenServiceServer{}, piServer,
		&pb.UnimplementedProfileLimitationServiceServer{},
		&pb.UnimplementedProfilePhotoServiceServer{},
		&pb.UnimplementedSettingsServiceServer{},
		&pb.UnimplementedUserEventsServiceServer{},
		handler.NewSearchHandler(stubSearchService{}),
		walletServer,
	)
	httpH := handler.NewHTTPAuthHandler(clients, nil, "en")
	httpW := handler.NewHTTPWalletHandler(clients.WalletConnection, "en")

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/personal-info", nil), 1)
	rr := httptest.NewRecorder()
	httpH.GetPersonalInfo(rr, r)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "occupation") {
		t.Fatalf("pi code=%d body=%s", rr.Code, rr.Body.String())
	}

	r = withUser(httptest.NewRequest(http.MethodGet, "/api/kyc", nil), 1)
	rr = httptest.NewRecorder()
	httpH.GetKYC(rr, r)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "errors") {
		t.Fatalf("kyc code=%d body=%s", rr.Code, rr.Body.String())
	}

	body := bytes.NewBufferString(`{"searchTerm":"ali"}`)
	r = withUser(httptest.NewRequest(http.MethodPost, "/api/search/users", body), 1)
	r.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	httpH.SearchUsers(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("search users code=%d body=%s", rr.Code, rr.Body.String())
	}
	body = bytes.NewBufferString(`{"searchTerm":"feat"}`)
	r = withUser(httptest.NewRequest(http.MethodPost, "/api/search/features", body), 1)
	r.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	httpH.SearchFeatures(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("search features code=%d", rr.Code)
	}
	body = bytes.NewBufferString(`{"searchTerm":"isic"}`)
	r = withUser(httptest.NewRequest(http.MethodPost, "/api/search/isic-codes", body), 1)
	r.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	httpH.SearchIsicCodes(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("search isic code=%d", rr.Code)
	}

	addr := "0x1111111111111111111111111111111111111111"
	r = withUser(httptest.NewRequest(http.MethodGet, "/api/wallet/link/nonce?address="+addr, nil), 1)
	rr = httptest.NewRecorder()
	httpW.GetLinkNonce(rr, r)
	if rr.Code < 400 {
		t.Fatalf("expected wallet grpc error mapping, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPAuthAccountCorePaths(t *testing.T) {
	authSvc := &mockAuthService{}
	authSvc.registerFunc = func(context.Context, string, string) (string, error) { return "https://oauth", nil }
	authSvc.redirectFunc = func(context.Context, string, string) (string, string, error) {
		return "https://oauth", "state", nil
	}
	authSvc.callbackFunc = func(context.Context, string, string, string) (*service.CallbackResult, error) {
		return &service.CallbackResult{Token: "t", ExpiresAt: 3600, RedirectURL: "/"}, nil
	}
	authSvc.getMeFunc = func(context.Context, string) (*service.UserDetails, error) {
		return &service.UserDetails{
			ID: 1, Name: "n", Code: "c", AutomaticLogout: 30,
			HasWallet: true, WalletAddress: "0x1",
			Level: &service.LevelInfo{ID: 1, Title: "L", Description: "d", Score: 10, Slug: "l"},
		}, nil
	}
	authSvc.logoutFunc = func(context.Context, uint64, string, string) error { return nil }
	authSvc.validateTokenFunc = func(context.Context, string) (*models.User, error) {
		return &models.User{ID: 1, Email: "a@x.com"}, nil
	}
	s := grpc.NewServer()
	defer s.Stop()
	tokenRepo := &mockTokenRepository{}
	tokenRepo.validateTokenFunc = func(context.Context, string) (*models.User, error) {
		return &models.User{ID: 1, Email: "a@x.com"}, nil
	}
	authServer := handler.NewAuthHandler(authSvc, tokenRepo, &mockProfilePhotoService{}, "en")
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

	rr := httptest.NewRecorder()
	httpH.Register(rr, httptest.NewRequest(http.MethodGet, "/api/auth/register?back_url=/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("register %d %s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	httpH.Redirect(rr, httptest.NewRequest(http.MethodGet, "/api/auth/redirect?redirect_to=x&back_url=/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("redirect %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	httpH.Callback(rr, httptest.NewRequest(http.MethodGet, "/api/auth/callback?state=s&code=c", nil))
	if rr.Code != http.StatusOK && rr.Code != http.StatusFound {
		t.Fatalf("callback %d %s", rr.Code, rr.Body.String())
	}
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/auth/me", nil), 1)
	r.Header.Set("Authorization", "Bearer tok")
	rr = httptest.NewRecorder()
	httpH.GetMe(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("me %d %s", rr.Code, rr.Body.String())
	}
	r = withUser(httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil), 1)
	rr = httptest.NewRecorder()
	httpH.Logout(rr, r)
	if rr.Code >= 500 {
		t.Fatalf("logout %d %s", rr.Code, rr.Body.String())
	}
	body := strings.NewReader(`{"token":"t"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/validate", body)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	httpH.ValidateToken(rr, req)
}
