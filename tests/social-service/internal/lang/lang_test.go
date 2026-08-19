package lang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"metarang/social-service/internal/lang"
)

func TestNormalizeLocale(t *testing.T) {
	assert.Equal(t, "en", lang.NormalizeLocale(""))
	assert.Equal(t, "en", lang.NormalizeLocale("en"))
	assert.Equal(t, "en", lang.NormalizeLocale("EN"))
	assert.Equal(t, "fa", lang.NormalizeLocale("fa"))
	assert.Equal(t, "fa", lang.NormalizeLocale(" FA "))
	assert.Equal(t, "en", lang.NormalizeLocale("de"))
}

func TestT_UnknownKeyFallsBack(t *testing.T) {
	got := lang.T("en", "Matrix exit box")
	assert.Equal(t, "Matrix exit box", got)
}
