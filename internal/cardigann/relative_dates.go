package cardigann

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeAgoParts = regexp.MustCompile(`([\p{Nd}.]+)[\s\v\p{Z}\x{85}]*([^\p{Nd}\s\v\p{Z}\x{85}.]+)`)

const dateTicksPerSecond int64 = 10000000
const dateTicksPerDay = dateTicksPerSecond * 86400

// timeago/reltime are elapsed intervals. Upstream uses float32 quantities and
// 30-day months/365-day years, not calendar month/year subtraction.
func parseRelativeDate(data string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	text := filterLower(data)
	if strings.Contains(text, "now") {
		return now, nil
	}
	text = strings.NewReplacer(",", "", "ago", "", "and", "").Replace(text)
	var ticks int64
	for _, match := range timeAgoParts.FindAllStringSubmatch(text, -1) {
		number := match[1]
		if last := strings.LastIndexByte(number, '.'); last >= 0 {
			number = strings.ReplaceAll(number[:last], ".", "") + number[last:]
		}
		quantity, err := strconv.ParseFloat(number, 32)
		if err != nil || math.IsInf(quantity, 0) || math.IsNaN(quantity) {
			return time.Time{}, fmt.Errorf("invalid relative date quantity")
		}
		unit := match[2]
		var seconds int64
		switch {
		case strings.Contains(unit, "sec") || unit == "s":
			seconds = 1
		case strings.Contains(unit, "min") || unit == "m":
			seconds = 60
		case strings.Contains(unit, "hour") || strings.Contains(unit, "hr") || unit == "h":
			seconds = 3600
		case strings.Contains(unit, "day") || unit == "d":
			seconds = 86400
		case strings.Contains(unit, "week") || strings.Contains(unit, "wk") || unit == "w":
			quantity = float64(float32(quantity) * 7)
			seconds = 86400
		case strings.Contains(unit, "month") || unit == "mo":
			quantity = float64(float32(quantity) * 30)
			seconds = 86400
		case strings.Contains(unit, "year") || unit == "y":
			quantity = float64(float32(quantity) * 365)
			seconds = 86400
		default:
			return time.Time{}, fmt.Errorf("unrecognized relative date unit")
		}
		component := quantity * float64(seconds*dateTicksPerSecond)
		if component >= math.Exp2(63) {
			return time.Time{}, fmt.Errorf("relative date exceeds range")
		}
		part := int64(component)
		if part > math.MaxInt64-ticks {
			return time.Time{}, fmt.Errorf("relative date exceeds range")
		}
		ticks += part
	}
	// Split whole days from the remainder so a supported century-scale interval
	// cannot overflow Go's nanosecond Duration. DateTime's year bounds still apply.
	result := now.AddDate(0, 0, -int(ticks/dateTicksPerDay)).Add(-time.Duration(ticks%dateTicksPerDay) * 100)
	if result.Year() < 1 || result.Year() > 9999 {
		return time.Time{}, fmt.Errorf("relative date exceeds range")
	}
	return result, nil
}
