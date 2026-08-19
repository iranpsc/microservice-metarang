package handler_test

import (
	"testing"

	"metarang/training-service/internal/handler"
)

func TestValidateRequired_StringEmpty(t *testing.T) {
	errs := handler.ValidateRequired("field", "", "en")
	if errs["field"] == "" {
		t.Fatal("expected error message")
	}
}

func TestValidateRequired_StringOK(t *testing.T) {
	errs := handler.ValidateRequired("field", "x", "en")
	if len(errs) != 0 {
		t.Fatal(errs)
	}
}

func TestValidateRequired_FAPLocale(t *testing.T) {
	errs := handler.ValidateRequired("field", "", "fa")
	if errs["field"] == "" {
		t.Fatal("expected fa message")
	}
}
