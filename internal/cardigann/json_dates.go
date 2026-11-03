package cardigann

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var jsonISODate = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,7}))?([Zz]|[+-][0-9]{2}(?::?[0-9]{2}|:)?)?$`)
var jsonMicrosoftDate = regexp.MustCompile(`^/Date\((-?[0-9]+)(?:[+-][0-9]{2}(?:[0-9]{2})?)?\)/$`)

// JSON.NET reads date-shaped JSON strings as DateTime tokens. Cardigann's
// Value<string>() and array joins expose invariant DateTime text, not the
// original ISO value. This projection follows the pinned UTC reference host;
// it must only be applied to values read from JSON, never HTML/template text.
func jsonDateText(value string) (string, bool) {
	if len(value) > 64 {
		return "", false
	}
	var parsed time.Time
	if match := jsonMicrosoftDate.FindStringSubmatch(value); match != nil {
		milliseconds, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil || milliseconds < -62135596800000 || milliseconds > 253402300799999 {
			return "", false
		}
		// The optional suffix identifies a local token; its milliseconds are
		// already an instant. Applying the suffix again would shift it twice.
		parsed = time.UnixMilli(milliseconds).UTC()
	} else {
		if len(value) < 19 || len(value) > 40 {
			return "", false
		}
		match := jsonISODate.FindStringSubmatch(value)
		if match == nil {
			return "", false
		}
		hour := match[2]
		rollDay := hour == "24"
		if rollDay {
			if match[3] != "00" || match[4] != "00" || strings.Trim(match[5], "0") != "" {
				return "", false
			}
			hour = "00"
		}
		var err error
		parsed, err = time.Parse("2006-01-02T15:04:05", match[1]+"T"+hour+":"+match[3]+":"+match[4])
		if err != nil || parsed.Year() < 1 {
			return "", false
		}
		if rollDay {
			parsed = parsed.AddDate(0, 0, 1)
			if parsed.Year() > 9999 {
				return "", false
			}
		}
		zone := match[6]
		if zone != "" && zone != "Z" && zone != "z" {
			hours, _ := strconv.Atoi(zone[1:3])
			minutes := 0
			if rest := strings.TrimPrefix(zone[3:], ":"); rest != "" {
				minutes, _ = strconv.Atoi(rest)
			}
			offset := time.Duration(hours*60+minutes) * time.Minute
			if zone[0] == '+' {
				offset = -offset
			}
			parsed = parsed.Add(offset)
			// DateTime's explicit-offset conversion saturates at its endpoints.
			if parsed.Year() < 1 {
				parsed = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
			} else if parsed.Year() > 9999 {
				parsed = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
			}
		}
	}
	// Invariant DateTime text omits fractional ticks, including nonzero ticks.
	return parsed.Format("01/02/2006 15:04:05"), true
}
