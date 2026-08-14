package handler_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/support-service/internal/handler"
)

func TestMapServiceError(t *testing.T) {
	tests := []struct {
		err  error
		want codes.Code
	}{
		{nil, codes.OK},
		{errors.New("Unauthorized access"), codes.PermissionDenied},
		{errors.New("resource not found"), codes.NotFound},
		{errors.New("cannot respond to closed ticket"), codes.FailedPrecondition},
		{errors.New("ticket is already closed"), codes.FailedPrecondition},
		{errors.New("db exploded"), codes.Internal},
		{errors.New("note not found"), codes.NotFound},
		{errors.New("Unauthorized: only ticket sender can update"), codes.PermissionDenied},
	}
	for _, tc := range tests {
		got := handler.MapServiceError(tc.err)
		if tc.err == nil {
			if got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
			continue
		}
		st, ok := status.FromError(got)
		if !ok || st.Code() != tc.want {
			t.Fatalf("err=%v want=%v got=%v", tc.err, tc.want, got)
		}
	}
}
