package lang_test

import (
	"strings"
	"testing"

	"metarang/training-service/internal/lang"
)

func TestLang_NormalizeLocale(t *testing.T) {
	if lang.NormalizeLocale("FA") != "fa" {
		t.Fatal("expected fa")
	}
	if lang.NormalizeLocale(" en ") != "en" {
		t.Fatal("expected en")
	}
	if lang.NormalizeLocale("de") != "en" {
		t.Fatal("expected default en")
	}
}

func TestLang_TAndTf(t *testing.T) {
	key := "invalid video data"
	en := lang.T("en", key)
	if en != key {
		t.Fatalf("unexpected en translation: %q", en)
	}
	fa := lang.T("fa", key)
	if fa == "" {
		t.Fatal("expected fa translation")
	}
	if got := lang.Tf("en", "failed to get videos: %v", "x"); !strings.Contains(got, "x") {
		t.Fatalf("Tf=%q", got)
	}
}
