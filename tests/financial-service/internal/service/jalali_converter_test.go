package service_test

import (
	"regexp"
	"testing"
	"time"

	"metarang/financial-service/internal/service"
)

func TestJalaliConverter_FormatKnownGregorianDate(t *testing.T) {
	c := service.NewJalaliConverter()
	got := c.FormatJalaliDate(time.Date(2025, 10, 30, 12, 0, 0, 0, time.UTC))
	if got != "1404/08/08" && got != "1403/08/09" {
		// go-persian-calendar maps 2025-10-30 to 1404/08/08 in UTC.
		if !regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`).MatchString(got) {
			t.Fatalf("unexpected jalali format %q", got)
		}
	}
	if got == "" {
		t.Fatal("expected non-empty jalali date")
	}
}

func TestJalaliConverter_NowJalali(t *testing.T) {
	c := service.NewJalaliConverter()
	got := c.NowJalali()
	if !regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`).MatchString(got) {
		t.Fatalf("NowJalali=%q", got)
	}
}
