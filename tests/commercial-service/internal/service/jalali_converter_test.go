package service_test

import (
	"testing"
	"time"

	"metarang/commercial-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJalaliConverter(t *testing.T) {
	c := service.NewJalaliConverter()
	now := c.NowJalali()
	require.NotEmpty(t, now)

	tm := time.Date(2026, 3, 21, 14, 5, 9, 0, time.UTC)
	assert.Equal(t, "1405/01/02", c.FormatJalaliDate(tm))
	assert.Equal(t, "14:5:09", c.FormatJalaliTime(tm))

	// Exercise leap-year branch in gregorian conversion.
	leap := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	assert.NotEmpty(t, c.FormatJalaliDate(leap))
}
