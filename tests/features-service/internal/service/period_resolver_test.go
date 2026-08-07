package service_test

import (
	"strconv"
	"testing"
	"time"

	"metarang/shared/pkg/period"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptime "github.com/yaa110/go-persian-calendar"
)

func TestPeriodResolver_InvalidPeriod(t *testing.T) {
	_, err := period.ResolvePeriod("invalid", time.Now(), time.Time{})
	require.Error(t, err)
}

func TestPeriodResolver_Daily(t *testing.T) {
	ref := time.Date(2026, 5, 15, 14, 30, 45, 0, time.Local)
	window, err := period.ResolvePeriod("daily", ref, time.Time{})
	require.NoError(t, err)

	assert.Equal(t, "daily", window.Period)
	assert.Equal(t, "hourly", window.Granularity)
	assert.Equal(t, 24, len(window.Buckets))

	expectedEnd := ref.Truncate(time.Second).Add(time.Second - time.Nanosecond)
	expectedStart := expectedEnd.Add(-24 * time.Hour).Truncate(time.Second)
	assert.True(t, window.End.Equal(expectedEnd), "end=%v expected=%v", window.End, expectedEnd)
	assert.True(t, window.Start.Equal(expectedStart), "start=%v expected=%v", window.Start, expectedStart)

	first := window.Buckets[0]
	last := window.Buckets[len(window.Buckets)-1]
	assert.Equal(t, first.Start.Format("15:04"), first.Label)
	assert.True(t, first.Start.Before(last.Start), "hourly buckets must be oldest→newest")
	assert.Equal(t, expectedEnd.Truncate(time.Hour), last.Start.Truncate(time.Hour))
	assert.Equal(t, expectedEnd.Add(-23*time.Hour).Truncate(time.Hour), first.Start.Truncate(time.Hour))
}

func TestPeriodResolver_Weekly(t *testing.T) {
	ref := time.Date(2026, 5, 15, 14, 30, 45, 0, time.Local)
	window, err := period.ResolvePeriod("weekly", ref, time.Time{})
	require.NoError(t, err)

	assert.Equal(t, "weekly", window.Period)
	assert.Equal(t, "weekly", window.Granularity)
	assert.Equal(t, 4, len(window.Buckets))

	expectedEnd := ref.Truncate(time.Second).Add(time.Second - time.Nanosecond)
	expectedStart := expectedEnd.Add(-28 * 24 * time.Hour).Truncate(time.Second)
	assert.True(t, window.Start.Equal(expectedStart))
	assert.True(t, window.End.Equal(expectedEnd))

	for i, bucket := range window.Buckets {
		offset := 3 - i
		weekEndDay := expectedEnd.AddDate(0, 0, -offset*7)
		assert.True(t, bucket.End.Equal(endOfDayLocal(weekEndDay)), "bucket %d end", i)
		assert.True(t, bucket.Start.Equal(startOfDayLocal(weekEndDay.AddDate(0, 0, -6))), "bucket %d start", i)
		assert.Equal(t, ptime.New(bucket.Start).Format("yyyy/MM/dd"), bucket.Label)
	}
}

func TestPeriodResolver_Monthly(t *testing.T) {
	ref := time.Date(2026, 5, 15, 14, 30, 45, 0, time.Local)
	window, err := period.ResolvePeriod("monthly", ref, time.Time{})
	require.NoError(t, err)

	assert.Equal(t, "monthly", window.Period)
	assert.Equal(t, "monthly", window.Granularity)
	assert.Equal(t, 12, len(window.Buckets))

	expectedEnd := ref.Truncate(time.Second).Add(time.Second - time.Nanosecond)
	expectedStart := startOfMonthLocal(expectedEnd.AddDate(0, -11, 0))
	assert.True(t, window.Start.Equal(expectedStart))

	first := window.Buckets[0]
	pt := ptime.New(first.Start)
	assert.Equal(t, pt.Month().String()+" "+strconv.Itoa(pt.Year()), first.Label)
}

func TestPeriodResolver_YearlySinceRegistration(t *testing.T) {
	// 2023-08-20 → Jalali 1402; 2026-05-15 → Jalali 1405
	ref := time.Date(2026, 5, 15, 14, 30, 45, 0, time.Local)
	registeredAt := time.Date(2023, 8, 20, 10, 0, 0, 0, time.Local)
	window, err := period.ResolvePeriod("yearly", ref, registeredAt)
	require.NoError(t, err)

	assert.Equal(t, "yearly", window.Period)
	assert.Equal(t, "yearly", window.Granularity)
	assert.Equal(t, 4, len(window.Buckets))

	expectedEnd := ref.Truncate(time.Second).Add(time.Second - time.Nanosecond)
	expectedStart := ptime.New(registeredAt).BeginningOfYear().Time()
	assert.True(t, window.Start.Equal(expectedStart), "start=%v expected=%v", window.Start, expectedStart)
	assert.True(t, window.End.Equal(expectedEnd))

	assert.Equal(t, []string{"1402", "1403", "1404", "1405"}, bucketLabels(window.Buckets))

	first := window.Buckets[0]
	assert.Equal(t, ptime.New(registeredAt).Year(), ptime.New(first.Start).Year())
	assert.True(t, first.Start.Equal(ptime.New(registeredAt).BeginningOfYear().Time()))

	last := window.Buckets[len(window.Buckets)-1]
	assert.Equal(t, "1405", last.Label)
	assert.True(t, last.End.Equal(expectedEnd), "current Jalali year bucket must end at reference")
}

func TestPeriodResolver_YearlyUsesJalaliYearsNotGregorian(t *testing.T) {
	// User registered in Gregorian 2022 after Nowruz → Jalali 1401.
	// Reference in Gregorian Aug 2026 → Jalali 1405.
	// Must start at 1401 (not 1400 from Jan 1 2022) and include 1405 (not stop at 1404).
	ref := time.Date(2026, 8, 7, 12, 0, 0, 0, time.Local)
	registeredAt := time.Date(2022, 6, 15, 10, 0, 0, 0, time.Local)
	require.Equal(t, 1401, ptime.New(registeredAt).Year())
	require.Equal(t, 1405, ptime.New(ref).Year())

	window, err := period.ResolvePeriod("yearly", ref, registeredAt)
	require.NoError(t, err)

	assert.Equal(t, []string{"1401", "1402", "1403", "1404", "1405"}, bucketLabels(window.Buckets))
	assert.True(t, window.Start.Equal(ptime.New(registeredAt).BeginningOfYear().Time()))
	assert.Equal(t, "1401", window.Buckets[0].Label)
	assert.Equal(t, "1405", window.Buckets[len(window.Buckets)-1].Label)
}

func TestPeriodResolver_YearlyWithoutRegistrationFallsBackToCurrentYear(t *testing.T) {
	ref := time.Date(2026, 5, 15, 14, 30, 45, 0, time.Local)
	window, err := period.ResolvePeriod("yearly", ref, time.Time{})
	require.NoError(t, err)

	assert.Equal(t, 1, len(window.Buckets))
	assert.Equal(t, "1405", window.Buckets[0].Label)
	assert.True(t, window.Start.Equal(ptime.New(ref).BeginningOfYear().Time()))
}

func TestPeriodResolver_ResolvePrevious(t *testing.T) {
	ref := time.Date(2026, 5, 15, 14, 30, 45, 0, time.Local)
	current, err := period.ResolvePeriod("daily", ref, time.Time{})
	require.NoError(t, err)

	previous, err := period.ResolvePrevious("daily", ref, time.Time{})
	require.NoError(t, err)

	duration := current.End.Sub(current.Start)
	assert.True(t, previous.Start.Equal(current.Start.Add(-(duration + time.Second))))
	assert.True(t, previous.End.Equal(current.Start.Add(-time.Second)))
}

func startOfDayLocal(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func endOfDayLocal(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

func startOfMonthLocal(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func bucketLabels(buckets []period.PeriodBucket) []string {
	labels := make([]string, len(buckets))
	for i, bucket := range buckets {
		labels[i] = bucket.Label
	}
	return labels
}
