package utils_test

import (
	"strings"
	"testing"
	"time"

	"metarang/support-service/internal/utils"
)

func TestFormatJalaliDate(t *testing.T) {
	ts := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	s := utils.FormatJalaliDate(ts)
	if !strings.Contains(s, "/") || len(s) < 8 {
		t.Fatalf("unexpected %q", s)
	}
}

func TestFormatJalaliTime(t *testing.T) {
	ts := time.Date(2024, 1, 1, 9, 5, 3, 0, time.UTC)
	if got := utils.FormatJalaliTime(ts); got != "09:05:03" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatJalaliDateTime(t *testing.T) {
	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	s := utils.FormatJalaliDateTime(ts)
	if !strings.Contains(s, "14:30:00") {
		t.Fatalf("unexpected %q", s)
	}
}

func TestGregorianToJalali_Branches(t *testing.T) {
	j := utils.GregorianToJalali(time.Date(900, 3, 1, 0, 0, 0, 0, time.UTC))
	if j.Year <= 0 || j.Month < 1 || j.Day < 1 {
		t.Fatalf("unexpected %+v", j)
	}
	j2 := utils.GregorianToJalali(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC))
	if j2.Month < 1 {
		t.Fatalf("unexpected %+v", j2)
	}
	janLeap := utils.GregorianToJalali(time.Date(2000, 1, 15, 8, 9, 10, 0, time.UTC))
	if janLeap.Month < 1 || janLeap.Hour != 8 {
		t.Fatalf("jan leap %+v", janLeap)
	}
	_ = utils.GregorianToJalali(time.Date(2003, 6, 1, 0, 0, 0, 0, time.UTC))
	lateYear := utils.GregorianToJalali(time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC))
	if lateYear.Month < 7 {
		t.Fatalf("expected second-half jalali month %+v", lateYear)
	}
}
