package cardigann

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

var dateTokens = map[string]string{
	"yyyy": "2006", "yyy": "2006", "yy": "06", "y": "6",
	"MMMM": "January", "MMM": "Jan", "MM": "01", "M": "1",
	"dddd": "Monday", "ddd": "Mon", "dd": "02", "d": "2",
	"HH": "15", "H": "15", "hh": "03", "h": "3", "mm": "04", "m": "4", "ss": "05", "s": "5",
	"tt": "PM", "t": "PM", "zzz": "-07:00", "zz": "-07", "z": "-7", "fff": "000", "ff": "00", "f": "0",
}

func ruleDateLayout(format string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(format); {
		if format[i] == '\'' {
			i++
			start := i
			for i < len(format) && format[i] != '\'' {
				i++
			}
			if i == len(format) {
				return "", fmt.Errorf("unclosed date literal")
			}
			out.WriteString(format[start:i])
			i++
			continue
		}
		if format[i] == '\\' && i+1 < len(format) {
			out.WriteByte(format[i+1])
			i += 2
			continue
		}
		end := i + 1
		for end < len(format) && format[end] == format[i] {
			end++
		}
		token := format[i:end]
		if (format[i] == 'f' || format[i] == 'F') && len(token) <= 7 {
			fraction := "0"
			if format[i] == 'F' {
				fraction = "9"
			}
			out.WriteString(strings.Repeat(fraction, len(token)))
		} else if replacement, ok := dateTokens[token]; ok {
			out.WriteString(replacement)
		} else {
			out.WriteString(token)
		}
		i = end
	}
	return out.String(), nil
}

func parseRuleDate(data, format string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	data = strings.TrimSpace(strings.ReplaceAll(data, "\u00a0", " "))
	if strings.EqualFold(data, "now") {
		return now, nil
	}
	if format == "" {
		// Cardigann's intermediate RFC date uses the .NET single z token
		// (for example +0), rather than Go's RFC1123Z offset (+0000).
		if split := strings.LastIndexByte(data, ' '); split >= 0 {
			zone := data[split+1:]
			if len(zone) >= 2 && len(zone) <= 3 && (zone[0] == '+' || zone[0] == '-') {
				if hours, err := strconv.ParseUint(zone[1:], 10, 8); err == nil && hours <= 14 {
					normalized := fmt.Sprintf("%s%c%02d00", data[:split+1], zone[0], hours)
					if parsed, err := time.Parse(time.RFC1123Z, normalized); err == nil {
						return parsed, nil
					}
				}
			}
		}
		return dateparse.ParseIn(data, now.Location())
	}
	layout, err := ruleDateLayout(format)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.ParseInLocation(layout, data, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("date does not match rule format")
	}
	year, month, day := parsed.Date()
	if !strings.Contains(format, "y") {
		year = now.Year()
	}
	if !strings.Contains(format, "M") {
		month = now.Month()
	}
	if !strings.Contains(format, "d") {
		day = now.Day()
	}
	return time.Date(year, month, day, parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), parsed.Location()), nil
}

// Rule filter strings use the upstream UTC-host DateTime representation. Keep
// this distinct from API timestamps and preserve the instant before formatting.
func formatFilterDate(parsed time.Time) string {
	return parsed.UTC().Format("Mon, 02 Jan 2006 15:04:05") + " +0"
}

var relativeDatePattern = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*(years?|yrs?|months?|mons?|weeks?|wks?|days?|hours?|hrs?|minutes?|mins?|seconds?|secs?|[ywdhms])\b`)
var fuzzyAgoPattern = regexp.MustCompile(`(?:^|[^\p{L}\p{Mn}\p{Nd}\p{Pc}])ago`)

func parseFuzzyRelativeDate(data string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	text := strings.ToLower(strings.TrimSpace(data))
	if fuzzyAgoPattern.MatchString(text) {
		return parseRelativeDate(data, now)
	}
	switch text {
	case "now", "just now", "today", "right now":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}
	// Relative duration parts are summed; calendar years/months retain their
	// calendar meaning rather than assuming every month has thirty days.
	matches := relativeDatePattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return time.Time{}, fmt.Errorf("unrecognized relative date")
	}
	result := now
	for _, match := range matches {
		n, err := strconv.ParseFloat(match[1], 64)
		if err != nil || n > 1000000 {
			return time.Time{}, fmt.Errorf("relative date exceeds range")
		}
		unit := match[2]
		switch {
		case strings.HasPrefix(unit, "y"):
			result = result.AddDate(-int(n), 0, 0)
		case strings.HasPrefix(unit, "mon"):
			result = result.AddDate(0, -int(n), 0)
		case strings.HasPrefix(unit, "w"):
			result = result.AddDate(0, 0, -int(n*7))
		case strings.HasPrefix(unit, "d"):
			result = result.AddDate(0, 0, -int(n))
		case strings.HasPrefix(unit, "h"):
			result = result.Add(-time.Duration(n * float64(time.Hour)))
		case strings.HasPrefix(unit, "m"):
			result = result.Add(-time.Duration(n * float64(time.Minute)))
		case strings.HasPrefix(unit, "s"):
			result = result.Add(-time.Duration(n * float64(time.Second)))
		}
	}
	return result, nil
}
