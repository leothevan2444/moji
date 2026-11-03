package cardigann

import (
	"context"
	"testing"
	"time"
)

func TestTimeAgoIntervalsUseThirtyDayMonthsAndSinglePrecisionScaling(t *testing.T) {
	now := time.Date(2024, 3, 31, 12, 34, 56, 0, time.UTC)
	for _, test := range []struct{ input, want string }{
		{"1 month", "2024-03-01T12:34:56Z"},
		{"1 year", "2023-04-01T12:34:56Z"},
		{"1.5 months", "2024-02-15T12:34:56Z"},
		{"0.1 month", "2024-03-28T12:34:56Z"},
		{"1.5 days", "2024-03-30T00:34:56Z"},
		{"1,000 seconds", "2024-03-31T12:18:16Z"},
		{"1h\u00a02d", "2024-03-29T11:34:56Z"},
		{"yesterday", "2024-03-31T12:34:56Z"},
		{"365 years", "1659-06-28T12:34:56Z"},
	} {
		got, err := parseRelativeDate(test.input, now)
		if err != nil || got.Format(time.RFC3339) != test.want {
			t.Fatalf("%q: got %s, %v; want %s", test.input, got, err, test.want)
		}
	}
	for _, value := range []string{"1 fortnight", "1 yr", ".. hours", "1000000 years"} {
		if _, err := parseRelativeDate(value, now); err == nil {
			t.Fatalf("invalid interval %q accepted", value)
		}
	}
}

func TestFuzzyAgoUsesTheSameIntervalWithoutMatchingArbitraryText(t *testing.T) {
	now := time.Date(2024, 3, 31, 12, 34, 56, 0, time.UTC)
	for _, name := range []string{"timeago", "reltime", "fuzzytime"} {
		got, err := ApplyFilters(context.Background(), "1 month ago", []Filter{{Name: name}}, FilterContext{Now: now})
		if err != nil || got != "Fri, 01 Mar 2024 12:34:56 +0" {
			t.Fatalf("%s: %q %v", name, got, err)
		}
	}
	if _, err := ApplyFilters(context.Background(), "dragon", []Filter{{Name: "fuzzytime"}}, FilterContext{Now: now}); err == nil {
		t.Fatal("a substring of an arbitrary word became a relative date")
	}
}
