package cardigann

import (
	"context"
	"testing"
	"time"
)

func TestDateFilterPreservesInstantThroughIntermediateText(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		input, format, want string
	}{
		{"2024-10-30 00:11:12 +05:30", "yyyy-MM-dd HH:mm:ss zzz", "2024-10-29T18:41:12Z"},
		{"2024-10-30 00:11:12 -03:30", "yyyy-MM-dd HH:mm:ss zzz", "2024-10-30T03:41:12Z"},
		{"2024-10-30T10:11:12.1234567+00:00", "yyyy-MM-dd'T'HH:mm:ss.fffffffzzz", "2024-10-30T10:11:12Z"},
		{"2024-10-30T10:11:12.12+00:00", "yyyy-MM-dd'T'HH:mm:ss.FFFFFFFzzz", "2024-10-30T10:11:12Z"},
	} {
		t.Run(test.input, func(t *testing.T) {
			filtered, err := ApplyFilters(context.Background(), test.input, []Filter{{Name: "dateparse", Args: Arguments{Scalar(test.format)}}}, FilterContext{Now: now})
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := parseRuleDate(filtered, "", now)
			if err != nil || parsed.UTC().Format(time.RFC3339) != test.want {
				t.Fatalf("intermediate %q lost instant: %s %v", filtered, parsed, err)
			}
		})
	}
}

func TestDateFilterLeavesFormatFailureUnchanged(t *testing.T) {
	input := "2024-10-30\t 10:11:12 +05:30"
	got, err := ApplyFilters(context.Background(), input, []Filter{{Name: "dateparse", Args: Arguments{"yyyy-MM-dd HH:mm:ss zzz"}}}, FilterContext{})
	if err != nil || got != input {
		t.Fatalf("format failure changed original text: %q %v", got, err)
	}
}
