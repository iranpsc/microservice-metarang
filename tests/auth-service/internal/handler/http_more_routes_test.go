package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"metarang/auth-service/internal/handler"
	pb "metarang/shared/pb/auth"
	sharedauth "metarang/shared/pkg/auth"
)

func TestProfileLimitationValidationForTest(t *testing.T) {
	body := `{"limited_user_id":2,"options":{"follow":true,"send_message":true,"share":true,"send_ticket":true,"view_profile_images":true,"view_features_locations":true},"note":"hi"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	in, errs := handler.ParseCreateProfileLimitationBodyForTest(r)
	if len(errs) != 0 || in == nil || in.LimitedUserID != 2 {
		t.Fatalf("in=%v errs=%v", in, errs)
	}

	r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	_, errs = handler.ParseCreateProfileLimitationBodyForTest(r)
	if len(errs) == 0 {
		t.Fatal("expected errors")
	}

	updBody := `{"options":{"follow":false,"send_message":true,"share":true,"send_ticket":true,"view_profile_images":true,"view_features_locations":true},"note":null}`
	r = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(updBody))
	r.Header.Set("Content-Type", "application/json")
	uin, uerrs := handler.ParseUpdateProfileLimitationBodyForTest(r)
	if len(uerrs) != 0 || uin == nil {
		t.Fatalf("uin=%v errs=%v", uin, uerrs)
	}

	ttrue := true
	jsonMap := handler.ProfileLimitationResourceJSONForTest(&pb.ProfileLimitation{
		Id: 1, LimiterUserId: 1, LimitedUserId: 2,
		Options: &pb.ProfileLimitationOptions{
			Follow: &ttrue, SendMessage: &ttrue, Share: &ttrue, SendTicket: &ttrue,
			ViewProfileImages: &ttrue, ViewFeaturesLocations: &ttrue,
		},
	}, 1)
	if jsonMap["id"] == nil {
		t.Fatalf("%v", jsonMap)
	}
}

func TestHTTPWalletHandler(t *testing.T) {
	s := grpc.NewServer()
	defer s.Stop()
	walletServer := handler.RegisterWalletConnectionHandler(s, &mockWalletConnectionService{}, "en")
	clients := handler.NewLocalClients(
		&pb.UnimplementedAuthServiceServer{},
		&pb.UnimplementedUserServiceServer{},
		&pb.UnimplementedKYCServiceServer{},
		&pb.UnimplementedCitizenServiceServer{},
		&pb.UnimplementedPersonalInfoServiceServer{},
		&pb.UnimplementedProfileLimitationServiceServer{},
		&pb.UnimplementedProfilePhotoServiceServer{},
		&pb.UnimplementedSettingsServiceServer{},
		&pb.UnimplementedUserEventsServiceServer{},
		&pb.UnimplementedSearchServiceServer{},
		walletServer,
	)
	h := handler.NewHTTPWalletHandler(clients.WalletConnection, "en")
	addr := "0x1111111111111111111111111111111111111111"

	withAuth := func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), sharedauth.UserContextKey{}, &sharedauth.UserContext{UserID: 1})
		return r.WithContext(ctx)
	}

	r := withAuth(httptest.NewRequest(http.MethodGet, "/api/wallet/link/nonce?address="+addr, nil))
	rr := httptest.NewRecorder()
	h.GetLinkNonce(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}

	body := bytes.NewBufferString(`{"address":"` + addr + `","signature":"sig"}`)
	r = withAuth(httptest.NewRequest(http.MethodPost, "/api/wallet/link", body))
	r.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.LinkWallet(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}

	r = withAuth(httptest.NewRequest(http.MethodGet, "/api/wallet/security/nonce?address="+addr, nil))
	rr = httptest.NewRecorder()
	h.GetSecurityNonce(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}

	body = bytes.NewBufferString(`{"address":"` + addr + `","signature":"sig","duration":15}`)
	r = withAuth(httptest.NewRequest(http.MethodPost, "/api/wallet/security/verify", body))
	r.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.VerifySecuritySignature(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}

	t.Run("error branches", func(t *testing.T) {
		cases := []struct {
			name string
			call func(http.ResponseWriter, *http.Request)
			req  *http.Request
			code int
		}{
			{"link nonce unauth", h.GetLinkNonce, httptest.NewRequest(http.MethodGet, "/api/wallet/link/nonce?address="+addr, nil), http.StatusUnauthorized},
			{"link wallet unauth", h.LinkWallet, httptest.NewRequest(http.MethodPost, "/api/wallet/link", strings.NewReader(`{}`)), http.StatusUnauthorized},
			{"security nonce unauth", h.GetSecurityNonce, httptest.NewRequest(http.MethodGet, "/api/wallet/security/nonce?address="+addr, nil), http.StatusUnauthorized},
			{"verify unauth", h.VerifySecuritySignature, httptest.NewRequest(http.MethodPost, "/api/wallet/security/verify", strings.NewReader(`{}`)), http.StatusUnauthorized},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rr := httptest.NewRecorder()
				tc.call(rr, tc.req)
				if rr.Code != tc.code {
					t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
				}
			})
		}

		empty := withAuth(httptest.NewRequest(http.MethodPost, "/api/wallet/link", strings.NewReader("")))
		empty.Header.Set("Content-Type", "application/json")
		empty.ContentLength = -1
		rr := httptest.NewRecorder()
		h.LinkWallet(rr, empty)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("empty body code=%d body=%s", rr.Code, rr.Body.String())
		}

		bad := withAuth(httptest.NewRequest(http.MethodPost, "/api/wallet/link", strings.NewReader(`{`)))
		bad.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		h.LinkWallet(rr, bad)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json code=%d", rr.Code)
		}

		emptySec := withAuth(httptest.NewRequest(http.MethodPost, "/api/wallet/security/verify", strings.NewReader("")))
		emptySec.Header.Set("Content-Type", "application/json")
		emptySec.ContentLength = -1
		rr = httptest.NewRecorder()
		h.VerifySecuritySignature(rr, emptySec)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("verify empty code=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}
