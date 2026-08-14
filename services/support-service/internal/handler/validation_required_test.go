package handler

import (
	"testing"
)

func TestValidateRequired_NumericZeroCurrentlyNotDetected(t *testing.T) {
	// Production uses `case uint64, uint32, int64, int32` so v is interface{} and
	// `v == 0` does not match uint64(0). Empty strings are still validated.
	if errs := validateRequired("user_id", uint64(0), "en"); len(errs) != 0 {
		t.Fatalf("current production does not flag uint64(0); got %v", errs)
	}
	if errs := validateRequired("title", "", "en"); len(errs) == 0 {
		t.Fatal("empty string should be required")
	}
	if errs := validateRequired("ok", "present", "en"); len(errs) != 0 {
		t.Fatalf("non-empty string should pass: %v", errs)
	}
}

func TestValidateMaxLenAndReportSubject(t *testing.T) {
	if errs := validateMaxLen("title", "ok", 10, "en"); len(errs) != 0 {
		t.Fatalf("within max: %v", errs)
	}
	if errs := validateMaxLen("title", "abcdefghijk", 10, "en"); len(errs) == 0 {
		t.Fatal("expected max error")
	}
	if errs := validateReportSubject("", "en"); len(errs) == 0 {
		t.Fatal("empty subject")
	}
	if errs := validateReportSubject("not-valid", "en"); len(errs) == 0 {
		t.Fatal("invalid subject")
	}
	if errs := validateReportSubject("displayError", "en"); len(errs) != 0 {
		t.Fatalf("valid subject: %v", errs)
	}
}
