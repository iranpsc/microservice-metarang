package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteJSON_WrapsAndSkipsExistingData(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]interface{}{"id": 1})
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["data"] == nil {
		t.Fatalf("expected wrap: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]interface{}{"data": []int{1}})
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("should skip wrap: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusBadRequest, map[string]string{"error": "nope"})
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"error"`)) {
		t.Fatalf("error payload %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("nil data code=%d", rr.Code)
	}
}

func TestWriteHandlerError_Mappings(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{errors.New("plain"), http.StatusInternalServerError},
		{status.Error(codes.Unauthenticated, "x"), http.StatusUnauthorized},
		{status.Error(codes.NotFound, "x"), http.StatusNotFound},
		{status.Error(codes.PermissionDenied, "x"), http.StatusForbidden},
		{status.Error(codes.AlreadyExists, "x"), http.StatusConflict},
		{status.Error(codes.FailedPrecondition, "x"), http.StatusPreconditionFailed},
		{status.Error(codes.Unavailable, "x"), http.StatusServiceUnavailable},
		{status.Error(codes.Internal, "x"), http.StatusInternalServerError},
		{status.Error(codes.InvalidArgument, "bad field"), http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeHandlerError(rr, tc.err)
		if rr.Code != tc.code {
			t.Fatalf("err=%v code=%d want=%d body=%s", tc.err, rr.Code, tc.code, rr.Body.String())
		}
	}
}

func TestExtractIDFromPathAndSplitJalaliDateTime(t *testing.T) {
	if got := extractIDFromPath("/api/tickets/12", "/api/tickets/", "/api/support/tickets/"); got != "12" {
		t.Fatalf("got=%q", got)
	}
	if got := extractIDFromPath("/api/support/tickets/12/extra", "/api/tickets/", "/api/support/tickets/"); got != "12" {
		t.Fatalf("got=%q", got)
	}
	if got := extractIDFromPath("/nope/1", "/api/tickets/"); got != "" {
		t.Fatalf("got=%q", got)
	}

	d, tm := splitJalaliDateTime("1403/01/01 12:00:00")
	if d != "1403/01/01" || tm != "12:00:00" {
		t.Fatalf("d=%q tm=%q", d, tm)
	}
	d, tm = splitJalaliDateTime("1403/01/01")
	if d != "1403/01/01" || tm != "" {
		t.Fatalf("date only d=%q tm=%q", d, tm)
	}
	d, tm = splitJalaliDateTime("  ")
	if d != "" || tm != "" {
		t.Fatalf("empty d=%q tm=%q", d, tm)
	}
}

func TestDecodeJSONBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"a":1}`))
	var v map[string]int
	if err := decodeJSONBody(req, &v); err != nil || v["a"] != 1 {
		t.Fatalf("v=%v err=%v", v, err)
	}
}

func TestSpoofedMethodFromValues(t *testing.T) {
	if spoofedMethodFromValues(nil) != "" {
		t.Fatal("empty")
	}
	if spoofedMethodFromValues([]string{" patch "}) != "PATCH" {
		t.Fatal("upper")
	}
}
