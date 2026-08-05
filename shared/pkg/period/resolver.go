// Package period resolves reporting periods for analytics and summaries.
package period

import (
	"fmt"
	"strconv"
	"time"

	ptime "github.com/yaa110/go-persian-calendar"
)

var ValidPeriods = []string{"daily", "weekly", "monthly", "yearly"}

// PeriodBucket is a single chart bucket with a Jalali label.
type PeriodBucket struct {
	Start time.Time
	End   time.Time
	Label string
}

// PeriodWindow is the resolved period window and chart buckets.
type PeriodWindow struct {
	Period      string
	Start       time.Time
	End         time.Time
	Granularity string
	Buckets     []PeriodBucket
}

// PreviousWindow is the immediately preceding period of equal length.
type PreviousWindow struct {
	Start time.Time
	End   time.Time
}

// ResolvePeriod resolves a reporting window and chart buckets.
// registeredAt is used for yearly periods (one bucket per Jalali year since registration).
// When registeredAt is zero, yearly falls back to the current Gregorian year only.
func ResolvePeriod(period string, reference time.Time, registeredAt time.Time) (*PeriodWindow, error) {
	if !isValidPeriod(period) {
		return nil, fmt.Errorf("invalid period [%s] provided", period)
	}

	end := endOfSecond(reference)
	var start time.Time
	switch period {
	case "daily":
		start = startOfSecond(end.Add(-24 * time.Hour))
	case "weekly":
		start = startOfSecond(end.Add(-28 * 24 * time.Hour))
	case "monthly":
		start = startOfMonth(end.AddDate(0, -11, 0))
	case "yearly":
		start = startOfYear(registrationYear(registeredAt, end))
	}

	return &PeriodWindow{
		Period:      period,
		Start:       start,
		End:         end,
		Granularity: granularityFor(period),
		Buckets:     buildBuckets(period, start, end, registeredAt),
	}, nil
}

// ResolvePrevious returns the immediately preceding period of equal length.
func ResolvePrevious(period string, reference time.Time, registeredAt time.Time) (*PreviousWindow, error) {
	current, err := ResolvePeriod(period, reference, registeredAt)
	if err != nil {
		return nil, err
	}
	duration := current.End.Sub(current.Start)
	return &PreviousWindow{
		Start: current.Start.Add(-(duration + time.Second)),
		End:   current.Start.Add(-time.Second),
	}, nil
}

// NormalizePeriod returns period if valid, otherwise "daily".
func NormalizePeriod(period string) string {
	if isValidPeriod(period) {
		return period
	}
	return "daily"
}

func isValidPeriod(period string) bool {
	for _, p := range ValidPeriods {
		if p == period {
			return true
		}
	}
	return false
}

func granularityFor(period string) string {
	switch period {
	case "daily":
		return "hourly"
	case "weekly":
		return "weekly"
	case "monthly":
		return "monthly"
	case "yearly":
		return "yearly"
	default:
		return ""
	}
}

func buildBuckets(period string, start, end time.Time, registeredAt time.Time) []PeriodBucket {
	switch period {
	case "daily":
		return hourlyBuckets(end)
	case "weekly":
		return weeklyBuckets(end, 4)
	case "monthly":
		return monthlyBuckets(end, 12)
	case "yearly":
		return yearlyBuckets(registrationYear(registeredAt, end), end)
	default:
		return nil
	}
}

func registrationYear(registeredAt, reference time.Time) time.Time {
	if registeredAt.IsZero() {
		return reference
	}
	return registeredAt
}

func hourlyBuckets(end time.Time) []PeriodBucket {
	// Chronological order (oldest → newest), consistent with weekly/monthly/yearly buckets.
	buckets := make([]PeriodBucket, 0, 24)
	for offset := 23; offset >= 0; offset-- {
		bucketEnd := endOfHour(end.Add(-time.Duration(offset) * time.Hour))
		bucketStart := startOfHour(bucketEnd)
		buckets = append(buckets, PeriodBucket{
			Start: bucketStart,
			End:   bucketEnd,
			Label: bucketStart.Format("15:04"),
		})
	}
	return buckets
}

func weeklyBuckets(end time.Time, weeks int) []PeriodBucket {
	buckets := make([]PeriodBucket, 0, weeks)
	for offset := weeks - 1; offset >= 0; offset-- {
		weekEndDay := end.AddDate(0, 0, -offset*7)
		bucketEnd := endOfDay(weekEndDay)
		bucketStart := startOfDay(weekEndDay.AddDate(0, 0, -6))
		buckets = append(buckets, PeriodBucket{
			Start: bucketStart,
			End:   bucketEnd,
			Label: ptime.New(bucketStart).Format("yyyy/MM/dd"),
		})
	}
	return buckets
}

func monthlyBuckets(end time.Time, months int) []PeriodBucket {
	buckets := make([]PeriodBucket, 0, months)
	for offset := months - 1; offset >= 0; offset-- {
		bucketDate := end.AddDate(0, -offset, 0)
		bucketStart := startOfMonth(bucketDate)
		bucketEnd := endOfMonth(bucketDate)
		pt := ptime.New(bucketStart)
		buckets = append(buckets, PeriodBucket{
			Start: bucketStart,
			End:   bucketEnd,
			Label: pt.Month().String() + " " + strconv.Itoa(pt.Year()),
		})
	}
	return buckets
}

func yearlyBuckets(registeredAt, end time.Time) []PeriodBucket {
	startYear := registeredAt.Year()
	endYear := end.Year()
	if startYear > endYear {
		startYear = endYear
	}

	buckets := make([]PeriodBucket, 0, endYear-startYear+1)
	for year := startYear; year <= endYear; year++ {
		bucketStart := startOfYear(time.Date(year, 1, 1, 0, 0, 0, 0, end.Location()))
		bucketEnd := endOfYear(time.Date(year, 1, 1, 0, 0, 0, 0, end.Location()))
		if bucketEnd.After(end) {
			bucketEnd = end
		}
		pt := ptime.New(bucketStart)
		buckets = append(buckets, PeriodBucket{
			Start: bucketStart,
			End:   bucketEnd,
			Label: strconv.Itoa(pt.Year()),
		})
	}
	return buckets
}

func endOfSecond(t time.Time) time.Time {
	return t.Truncate(time.Second).Add(time.Second - time.Nanosecond)
}

func startOfSecond(t time.Time) time.Time {
	return t.Truncate(time.Second)
}

func startOfHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

func endOfHour(t time.Time) time.Time {
	return startOfHour(t).Add(time.Hour - time.Nanosecond)
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

func startOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func endOfMonth(t time.Time) time.Time {
	return startOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

func startOfYear(t time.Time) time.Time {
	y, _, _ := t.Date()
	return time.Date(y, 1, 1, 0, 0, 0, 0, t.Location())
}

func endOfYear(t time.Time) time.Time {
	return startOfYear(t).AddDate(1, 0, 0).Add(-time.Nanosecond)
}
