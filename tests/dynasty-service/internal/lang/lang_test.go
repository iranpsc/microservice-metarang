package lang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"metarang/dynasty-service/internal/lang"
)

func TestLang_NormalizeLocale(t *testing.T) {
	assert.Equal(t, "fa", lang.NormalizeLocale("FA"))
	assert.Equal(t, "fa", lang.NormalizeLocale(" fa "))
	assert.Equal(t, "en", lang.NormalizeLocale("en"))
	assert.Equal(t, "en", lang.NormalizeLocale(""))
	assert.Equal(t, "en", lang.NormalizeLocale("de"))
}

func TestLang_T_And_Tf(t *testing.T) {
	// Keys may or may not exist; ensure no panic and locale normalization works.
	en := lang.T("en", "nonexistent.key")
	fa := lang.T("fa", "nonexistent.key")
	assert.NotEmpty(t, en)
	assert.NotEmpty(t, fa)

	formatted := lang.Tf("en", "nonexistent.key", "arg")
	assert.NotEmpty(t, formatted)
}
