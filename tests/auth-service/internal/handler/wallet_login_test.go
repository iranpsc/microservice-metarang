package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
	sharedauth "metarang/shared/pkg/auth"
)

func TestHTTPCallback_ForwardsWalletLoginQuery(t *testing.T) {
	cases := []struct {
		query    string
		expected bool
	}{
		{query: "state=s&code=c&wallet_login=true", expected: true},
		{query: "state=s&code=c&wallet_login=1", expected: true},
		{query: "state=s&code=c&wallet_login=false", expected: false},
		{query: "state=s&code=c&wallet_login=0", expected: false},
		{query: "state=s&code=c", expected: false},
		{query: "state=s&code=c&wallet_login=maybe", expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			authSvc := &mockAuthService{}
			var got bool
			authSvc.callbackFunc = func(_ context.Context, _, _ string, _ string, walletLogin bool) (*service.CallbackResult, error) {
				got = walletLogin
				return &service.CallbackResult{Token: "tok", ExpiresAt: 55, RedirectURL: "https://app/cb"}, nil
			}
			httpH := newHTTPAuthHandlerForWalletLogin(t, authSvc)

			r := httptest.NewRequest(http.MethodGet, "/api/auth/callback?"+tc.query, nil)
			rr := httptest.NewRecorder()
			httpH.Callback(rr, r)
			if rr.Code != http.StatusFound {
				t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
			}
			if got != tc.expected {
				t.Fatalf("wallet_login=%v want %v", got, tc.expected)
			}
		})
	}
}

func TestGRPCCallback_PassesWalletLoginToService(t *testing.T) {
	m := &mockAuthService{}
	var got bool
	m.callbackFunc = func(_ context.Context, state, code, _ string, walletLogin bool) (*service.CallbackResult, error) {
		if state != "s" || code != "c" {
			t.Fatalf("state=%s code=%s", state, code)
		}
		got = walletLogin
		return &service.CallbackResult{Token: "t", ExpiresAt: 55, RedirectURL: "u"}, nil
	}
	h := handler.NewAuthHandler(m, &mockTokenRepository{}, &mockProfilePhotoService{}, "en")
	_, err := h.Callback(context.Background(), &pb.CallbackRequest{State: "s", Code: "c", WalletLogin: true})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected wallet_login true to reach the service")
	}
}

func TestHTTPGetMe_AlwaysIncludesWalletLogin(t *testing.T) {
	cases := []bool{true, false}
	for _, walletLogin := range cases {
		t.Run(boolName(walletLogin), func(t *testing.T) {
			authSvc := &mockAuthService{}
			authSvc.getMeFunc = func(context.Context, string) (*service.UserDetails, error) {
				return &service.UserDetails{
					ID: 1, Name: "n", Code: "c", AutomaticLogout: 30, Token: "tok",
					WalletLogin: walletLogin,
				}, nil
			}
			httpH := newHTTPAuthHandlerForWalletLogin(t, authSvc)

			r := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
			r = r.WithContext(context.WithValue(r.Context(), sharedauth.UserContextKey{}, &sharedauth.UserContext{
				UserID: 1, Token: "tok",
			}))
			rr := httptest.NewRecorder()
			httpH.GetMe(rr, r)
			if rr.Code != http.StatusOK {
				t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			data, ok := payload["data"].(map[string]interface{})
			if !ok {
				t.Fatalf("missing data: %s", rr.Body.String())
			}
			got, exists := data["wallet_login"]
			if !exists {
				t.Fatalf("wallet_login missing from GetMe response: %s", rr.Body.String())
			}
			gotBool, ok := got.(bool)
			if !ok {
				t.Fatalf("wallet_login type %T value %v", got, got)
			}
			if gotBool != walletLogin {
				t.Fatalf("wallet_login=%v want %v", gotBool, walletLogin)
			}
		})
	}
}

func TestGRPCGetMe_IncludesWalletLogin(t *testing.T) {
	m := &mockAuthService{}
	m.getMeFunc = func(context.Context, string) (*service.UserDetails, error) {
		return &service.UserDetails{ID: 1, Name: "n", Code: "c", AutomaticLogout: 30, WalletLogin: true}, nil
	}
	h := handler.NewAuthHandler(m, &mockTokenRepository{}, &mockProfilePhotoService{}, "en")
	resp, err := h.GetMe(context.Background(), &pb.GetMeRequest{Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.WalletLogin {
		t.Fatal("expected wallet_login on UserResponse")
	}
}

func newHTTPAuthHandlerForWalletLogin(t *testing.T, authSvc *mockAuthService) *handler.HTTPAuthHandler {
	t.Helper()
	authServer := handler.NewAuthHandler(authSvc, &mockTokenRepository{}, &mockProfilePhotoService{gatewayURL: "https://cdn"}, "en")
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
	return handler.NewHTTPAuthHandler(clients, nil, "en")
}

func boolName(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
