package jalali

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// JalaliToCarbon converts Jalali date string to time.Time
// Format: Y/m/d (e.g., "1403/08/09")
func JalaliToCarbon(jalaliDate string) (time.Time, error) {
	parts := strings.Split(jalaliDate, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid jalali date format: expected Y/m/d")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %w", err)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %w", err)
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %w", err)
	}

	return jalaliToGregorian(year, month, day)
}

// CarbonToJalali converts time.Time to Jalali date string
// Format: Y/m/d (e.g., "1403/08/09")
func CarbonToJalali(t time.Time) string {
	year, month, day := gregorianToJalali(t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%d/%02d/%02d", year, month, day)
}

// CarbonToJalaliDateTime converts time.Time to Jalali date-time string
// Format: Y/m/d H:i (e.g., "1403/08/09 14:30")
func CarbonToJalaliDateTime(t time.Time) string {
	year, month, day := gregorianToJalali(t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%d/%02d/%02d %02d:%02d", year, month, day, t.Hour(), t.Minute())
}

// gregorianToJalali converts Gregorian date to Jalali date
func gregorianToJalali(gy, gm, gd int) (jy, jm, jd int) {
	var g_d_m = []int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	
	if gy > 1600 {
		jy = 979
		gy -= 1600
	} else {
		jy = 0
		gy -= 621
	}

	if gm > 2 {
		gy2 := gy + 1
		if (gy2%4 == 0 && gy2%100 != 0) || (gy2%400 == 0) {
			g_d_m[2] = 60
		}
	}

	gy2 := gy
	if (gy2%4 == 0 && gy2%100 != 0) || (gy2%400 == 0) {
		// leap year
		if gm > 2 {
			gd++
		}
	}

	days := 365*gy + ((gy + 3) / 4) - ((gy + 99) / 100) + ((gy + 399) / 400) + gd + g_d_m[gm-1] - 1

	jy += 33 * (days / 12053)
	days %= 12053

	jy += 4 * (days / 1461)
	days %= 1461

	if days > 365 {
		jy += (days - 1) / 365
		days = (days - 1) % 365
	}

	if days < 186 {
		jm = 1 + days/31
		jd = 1 + (days % 31)
	} else {
		jm = 7 + (days-186)/30
		jd = 1 + ((days - 186) % 30)
	}

	return
}

// jalaliToGregorian converts Jalali date to Gregorian date
func jalaliToGregorian(jy, jm, jd int) (time.Time, error) {
	var sal_a = []int{
		// -61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210,
		// 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
		-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210,
		1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
	}

	gy := jy + 621
	var leap int = -14

	jp := sal_a[0]
	for i := 1; i < len(sal_a); i++ {
		j := sal_a[i]
		leap = leap + (j-jp)/33*8 + ((j-jp)%33)/4
		if jy < j {
			break
		}
		jp = j
	}

	n := jy - jp
	if n < 0 {
		n = -n
	}

	leap = leap + n/33*8 + (n%33+3)/4
	if (jy%33+1)%4 == 0 {
		leap++
	}

	if jm < 7 {
		n = 31*jm - 31
	} else {
		n = 30*jm - 30
	}

	n = n + jd + leap + 79

	gy2 := gy
	if (gy2%4 == 0 && gy2%100 != 0) || (gy2%400 == 0) {
		leap = 1
	} else {
		leap = 0
	}

	if n > 366+leap {
		gy++
		n = n - (366 + leap)
	}

	var gm int
	var g_d_m = []int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}

	if leap == 1 {
		g_d_m[2] = 60
	}

	for i := 0; i < 12; i++ {
		v := g_d_m[i]
		if n <= v {
			gm = i
			break
		}
		gm = i + 1
	}

	gd := n
	if gm > 0 {
		gd = n - g_d_m[gm-1]
	}

	return time.Date(gy, time.Month(gm), gd, 0, 0, 0, 0, time.UTC), nil
}

