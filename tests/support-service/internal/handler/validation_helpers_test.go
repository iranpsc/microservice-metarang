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

func TestValidateRequired_NumericZeroCurrentlyNotDetected(t *testing.T) {
	// Production uses `case uint64, uint32, int64, int32` so v is interface{} and
	// `v == 0` does not match uint64(0). Empty strings are still validated.
	if errs := handler.ValidateRequired("user_id", uint64(0), "en"); len(errs) != 0 {
		t.Fatalf("current production does not flag uint64(0); got %v", errs)
	}
	if errs := handler.ValidateRequired("title", "", "en"); len(errs) == 0 {
		t.Fatal("empty string should be required")
	}
	if errs := handler.ValidateRequired("ok", "present", "en"); len(errs) != 0 {
		t.Fatalf("non-empty string should pass: %v", errs)
	}
}

func TestValidateMaxLenAndReportSubject(t *testing.T) {
	if errs := handler.ValidateMaxLen("title", "ok", 10, "en"); len(errs) != 0 {
		t.Fatalf("within max: %v", errs)
	}
	if errs := handler.ValidateMaxLen("title", "abcdefghijk", 10, "en"); len(errs) == 0 {
		t.Fatal("expected max error")
	}
	if errs := handler.ValidateReportSubject("", "en"); len(errs) == 0 {
		t.Fatal("empty subject")
	}
	if errs := handler.ValidateReportSubject("not-valid", "en"); len(errs) == 0 {
		t.Fatal("invalid subject")
	}
	if errs := handler.ValidateReportSubject("displayError", "en"); len(errs) != 0 {
		t.Fatalf("valid subject: %v", errs)
	}
}
