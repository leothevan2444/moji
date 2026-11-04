package cardigann

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const fuzzyWordStart = `(?:^|[^\p{L}\p{Mn}\p{Nd}\p{Pc}])`

var fuzzyAgoPattern = regexp.MustCompile(`(?i)` + fuzzyWordStart + `ago`)
var fuzzyDayPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)` + fuzzyWordStart + `(today(?:[ ,]+(?:at)? *|[ ,]*|$))`),
	regexp.MustCompile(`(?i)` + fuzzyWordStart + `(yesterday(?:[ ,]+(?:at)? *|[ ,]*|$))`),
	regexp.MustCompile(`(?i)` + fuzzyWordStart + `(tomorrow(?:[ ,]+(?:at)? *|[ ,]*|$))`),
}
var fuzzyWeekdayPattern = regexp.MustCompile(`(?i)` + fuzzyWordStart + `((monday|tuesday|wednesday|thursday|friday|saturday|sunday) +at +)`)
var fuzzyMissingYear = regexp.MustCompile(`^([0-9]{1,2}-[0-9]{1,2})( |$)`)
var fuzzyMissingYearMonth = regexp.MustCompile(`(?i)^([0-9]{1,2} +[a-z]{3}) +([0-9]{1,2}:[0-9]{1,2}.*)$`)
var fuzzyDigits = regexp.MustCompile(`^[0-9]+$`)
var fuzzyRfcDate = regexp.MustCompile(`^[A-Za-z]{3}, [0-9]{2} [A-Za-z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} [+-][0-9]{1,2}$`)
var fuzzyClockOnly = regexp.MustCompile(`(?i)^([0-9]{1,2}):([0-9]{2})(?::([0-9]{2})(?:\.[0-9]+)?)? *(am|pm)?(?: *(?:UTC|GMT|CST|EST|[+-][0-9]{2}(?::?[0-9]{2})?))?$`)

// FromUnknown has calendar rules distinct from the elapsed FromTimeAgo path.
// These filters follow the pinned UTC reference host, independently of the
// application timezone used for release dates and subscription policy.
func parseFuzzyDate(data string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	text := strings.Join(strings.Fields(data), " ")
	if fuzzyRfcDate.MatchString(text) {
		if parsed, err := parseRuleDate(text, "", now); err == nil {
			return parsed, nil
		}
	}
	if fuzzyDigits.MatchString(text) {
		seconds, err := strconv.ParseInt(text, 10, 64)
		if err != nil || seconds > 253402300799 {
			return time.Time{}, fmt.Errorf("fuzzy Unix timestamp exceeds range")
		}
		return time.Unix(seconds, 0).UTC(), nil
	}
	if strings.Contains(strings.ToLower(text), "now") {
		return now, nil
	}
	if fuzzyAgoPattern.MatchString(text) {
		return parseRelativeDate(text, now)
	}
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for i, pattern := range fuzzyDayPatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			clock, err := fuzzyTimeOfDay(strings.ReplaceAll(text, match[1], ""))
			if err != nil {
				return time.Time{}, err
			}
			offset := []int{0, -1, 1}[i]
			return boundedFuzzyDate(midnight.AddDate(0, 0, offset).Add(clock))
		}
	}
	if match := fuzzyWeekdayPattern.FindStringSubmatch(text); match != nil {
		clock, err := fuzzyTimeOfDay(strings.ReplaceAll(text, match[1], ""))
		if err != nil {
			return time.Time{}, err
		}
		var weekday time.Weekday
		for i, name := range []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"} {
			if strings.EqualFold(match[2], name) {
				weekday = time.Weekday(i)
			}
		}
		days := (int(now.Weekday()) - int(weekday) + 7) % 7
		return boundedFuzzyDate(midnight.AddDate(0, 0, -days).Add(clock))
	}
	if match := fuzzyClockOnly.FindStringSubmatch(text); match != nil {
		hour, _ := strconv.Atoi(match[1])
		minute, _ := strconv.Atoi(match[2])
		second, _ := strconv.Atoi(match[3])
		if hour > 23 || minute > 59 || second > 59 {
			return time.Time{}, fmt.Errorf("invalid fuzzy clock")
		}
		if strings.EqualFold(match[4], "pm") && hour < 12 {
			hour += 12
		} else if strings.EqualFold(match[4], "am") && hour == 12 {
			hour = 0
		}
		// DateTimeRoutines fills absent dates from DefaultDate and exposes
		// integer-second DateTime wall clocks, not UtcDateTime or fractions.
		return boundedFuzzyDate(time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, time.UTC))
	}
	missingYear := false
	if match := fuzzyMissingYear.FindStringSubmatch(text); match != nil {
		text = strings.ReplaceAll(text, match[1], strconv.Itoa(now.Year())+"-"+match[1])
		missingYear = true
	} else if match := fuzzyMissingYearMonth.FindStringSubmatch(text); match != nil {
		text = match[1] + " " + strconv.Itoa(now.Year()) + " " + match[2]
		missingYear = true
	}
	parsed, err := parseRuleDate(text, "", now)
	if err != nil {
		return time.Time{}, err
	}
	// DateTimeRoutines exposes DateTime, not its separate UtcDateTime field.
	// Retain the parsed wall clock here; RFC dates and day phrases above use
	// DateTime.Parse and deliberately apply their offsets instead.
	parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC)
	if missingYear && parsed.After(now) {
		// DateTime.AddYears clamps leap day rather than rolling into March.
		day := parsed.Day()
		if parsed.Month() == time.February && day == 29 {
			day = 28
		}
		parsed = time.Date(parsed.Year()-1, parsed.Month(), day, parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), parsed.Location())
	}
	return boundedFuzzyDate(parsed)
}

func boundedFuzzyDate(parsed time.Time) (time.Time, error) {
	if parsed.Year() < 1 || parsed.Year() > 9999 {
		return time.Time{}, fmt.Errorf("fuzzy date exceeds range")
	}
	return parsed, nil
}

// Calendar phrases use DateTime.Parse(...).TimeOfDay. An explicit offset first
// converts to the UTC reference host, then only its clock is retained.
func fuzzyTimeOfDay(text string) (time.Duration, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	text = strings.ToUpper(text)
	for _, layout := range []string{"15:04:05", "15:04", "3:04:05PM", "3:04PM", "3:04:05 PM", "3:04 PM", "3 PM"} {
		for _, suffix := range []string{"", " Z07:00", " -0700", " MST"} {
			parsed, err := time.Parse(layout+suffix, text)
			if err != nil {
				continue
			}
			if suffix == " MST" {
				zone, _ := parsed.Zone()
				if zone != "UTC" && zone != "GMT" {
					continue
				}
			}
			parsed = parsed.UTC()
			return time.Duration(parsed.Hour())*time.Hour + time.Duration(parsed.Minute())*time.Minute + time.Duration(parsed.Second())*time.Second + time.Duration(parsed.Nanosecond()), nil
		}
	}
	return 0, fmt.Errorf("invalid fuzzy clock")
}
