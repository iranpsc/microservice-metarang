package lang_test

import (
	"strings"
	"testing"

	"metarang/support-service/internal/lang"
)

func TestT_LoadsENAndFAAndMissingKeyFallback(t *testing.T) {
	en := lang.T("en", "user event not found")
	if en != "user event not found" {
		t.Fatalf("en=%q", en)
	}
	fa := lang.T("fa", "user event not found")
	if fa != "رویداد کاربر یافت نشد" {
		t.Fatalf("fa=%q", fa)
	}
	upper := lang.T("FA", "user event not found")
	if upper != fa {
		t.Fatalf("FA normalize got %q", upper)
	}
	missing := lang.T("en", "definitely-missing-key")
	if missing != "definitely-missing-key" {
		t.Fatalf("missing fallback=%q", missing)
	}
	other := lang.T("de", "user event not found")
	if other != en {
		t.Fatalf("unknown locale should use en, got %q", other)
	}
}

func TestTf_FormatsArgs(t *testing.T) {
	got := lang.Tf("en", "failed to create user event: %v", "boom")
	if !strings.Contains(got, "boom") {
		t.Fatalf("got=%q", got)
	}
	plain := lang.Tf("fa", "user event not found")
	if plain != "رویداد کاربر یافت نشد" {
		t.Fatalf("plain=%q", plain)
	}
}

func TestNormalizeLocale(t *testing.T) {
	if lang.NormalizeLocale("FA") != "fa" {
		t.Fatal("FA")
	}
	if lang.NormalizeLocale(" en ") != "en" {
		t.Fatal("en")
	}
	if lang.NormalizeLocale("xx") != "en" {
		t.Fatal("default en")
	}
}
