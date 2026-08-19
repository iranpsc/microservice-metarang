package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

func TestDecodeRequestFormAndQuery(t *testing.T) {
	type payload struct {
		Name  string  `json:"name" form:"name"`
		Count int32   `json:"count" form:"count"`
		Flag  bool    `json:"flag" form:"flag"`
		Amt   float64 `json:"amt" form:"amt"`
		ID    uint64  `json:"id" form:"id"`
	}

	form := url.Values{}
	form.Set("name", "bob")
	form.Set("count", "4")
	form.Set("flag", "true")
	form.Set("amt", "2.5")
	form.Set("id", "12")
	r := httptest.NewRequest(http.MethodPost, "/x?extra=1", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var got payload
	if err := handler.DecodeRequestForTest(r, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "bob" || got.Count != 4 || !got.Flag || got.ID != 12 {
		t.Fatalf("%+v", got)
	}

	r = httptest.NewRequest(http.MethodPost, "/x?name=fromq&count=9", strings.NewReader(`{"name":"frombody"}`))
	r.Header.Set("Content-Type", "application/json")
	got = payload{}
	if err := handler.DecodeRequestBodyForTest(r, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "frombody" || got.Count != 9 {
		t.Fatalf("merge expected body+query, got %+v", got)
	}

	r = httptest.NewRequest(http.MethodGet, "/x?name=onlyq", nil)
	got = payload{}
	if err := handler.DecodeRequestForTest(r, &got); err != nil || got.Name != "onlyq" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestKYCHandler_MapServiceErrors(t *testing.T) {
	ctx := authenticatedContext(1)
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not found", service.ErrKYCNotFound, codes.NotFound},
		{"not owned", service.ErrKYCNotOwned, codes.PermissionDenied},
		{"not rejected", service.ErrKYCNotRejected, codes.PermissionDenied},
		{"storage", service.ErrStorageUnavailable, codes.Internal},
		{"filename", service.ErrMelliCardFilenameRequired, codes.InvalidArgument},
		{"ctype", service.ErrMelliCardContentTypeRequired, codes.InvalidArgument},
		{"too large", service.ErrMelliCardTooLarge, codes.InvalidArgument},
		{"type", service.ErrInvalidMelliCardType, codes.InvalidArgument},
		{"ext", service.ErrInvalidMelliCardExtension, codes.InvalidArgument},
		{"melli", service.ErrInvalidMelliCode, codes.InvalidArgument},
		{"fname", service.ErrInvalidFname, codes.InvalidArgument},
		{"other", errors.New("boom"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &mockKYCService{}
			m.getKYCFunc = func(context.Context, uint64) (*models.KYC, error) {
				return nil, tc.err
			}
			h := handler.NewKYCHandler(m, "")
			_, err := h.GetKYC(ctx, &pb.GetKYCRequest{UserId: 1})
			st, _ := status.FromError(err)
			if st.Code() != tc.code {
				t.Fatalf("code=%v want=%v msg=%v", st.Code(), tc.code, err)
			}
		})
	}
}
