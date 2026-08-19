package lang_test

import (
	"testing"

	"metarang/commercial-service/internal/lang"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLocale(t *testing.T) {
	assert.Equal(t, "en", lang.NormalizeLocale(""))
	assert.Equal(t, "en", lang.NormalizeLocale("EN"))
	assert.Equal(t, "en", lang.NormalizeLocale("en-US"))
	assert.Equal(t, "fa", lang.NormalizeLocale("fa"))
	assert.Equal(t, "fa", lang.NormalizeLocale(" FA "))
}

func TestTAndTf(t *testing.T) {
	key := "failed to get wallet: %v"
	en := lang.T("en", key)
	fa := lang.T("fa", key)
	assert.NotEmpty(t, en)
	assert.NotEmpty(t, fa)

	formatted := lang.Tf("en", key, "db down")
	assert.Contains(t, formatted, "db down")
}
