package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteHandlerError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "unauthenticated",
			err:        status.Error(codes.Unauthenticated, "login required"),
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "login required",
		},
		{
			name:       "not found",
			err:        status.Error(codes.NotFound, "missing"),
			wantStatus: http.StatusNotFound,
			wantMsg:    "missing",
		},
		{
			name:       "invalid argument",
			err:        status.Error(codes.InvalidArgument, "bad input"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "bad input",
		},
		{
			name:       "permission denied",
			err:        status.Error(codes.PermissionDenied, "forbidden"),
			wantStatus: http.StatusForbidden,
			wantMsg:    "forbidden",
		},
		{
			name:       "already exists",
			err:        status.Error(codes.AlreadyExists, "duplicate"),
			wantStatus: http.StatusConflict,
			wantMsg:    "duplicate",
		},
		{
			name:       "failed precondition",
			err:        status.Error(codes.FailedPrecondition, "not ready"),
			wantStatus: http.StatusPreconditionFailed,
			wantMsg:    "not ready",
		},
		{
			name:       "unavailable",
			err:        status.Error(codes.Unavailable, "down"),
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "service temporarily unavailable: down",
		},
		{
			name:       "internal",
			err:        status.Error(codes.Internal, "boom"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "boom",
		},
		{
			name:       "non status error",
			err:        errors.New("plain"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeHandlerError(rr, tt.err)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d", rr.Code, tt.wantStatus)
			}

			var body map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["error"] != tt.wantMsg {
				t.Fatalf("error=%q want=%q", body["error"], tt.wantMsg)
			}
		})
	}
}
