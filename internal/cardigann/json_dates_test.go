package cardigann

import (
	"context"
	"strings"
	"testing"
)

func TestJSONDateTextMatchesDateTokenProjection(t *testing.T) {
	for _, test := range []struct{ value, want string }{
		{"2024-10-30T10:11:12Z", "10/30/2024 10:11:12"},
		{"2024-10-30T10:11:12z", "10/30/2024 10:11:12"},
		{"2024-10-30T10:11:12+05:30", "10/30/2024 04:41:12"},
		{"2024-10-30T10:11:12-03:30", "10/30/2024 13:41:12"},
		{"2024-10-30T10:11:12+0530", "10/30/2024 04:41:12"},
		{"2024-10-30T10:11:12+05", "10/30/2024 05:11:12"},
		{"2024-10-30T10:11:12.1234567Z", "10/30/2024 10:11:12"},
		{"2024-10-30T10:11:12", "10/30/2024 10:11:12"},
		{"2024-10-30T24:00:00Z", "10/31/2024 00:00:00"},
		{"2024-10-30T24:00:00.0000000Z", "10/31/2024 00:00:00"},
		{"/Date(1730283072000)/", "10/30/2024 10:11:12"},
		{"/Date(1730283072000+0530)/", "10/30/2024 10:11:12"},
		{"/Date(-1)/", "12/31/1969 23:59:59"},
		{"0001-01-01T00:00:00+01:00", "01/01/0001 00:00:00"},
		{"9999-12-31T23:59:59-01:00", "12/31/9999 23:59:59"},
	} {
		t.Run(test.value, func(t *testing.T) {
			if got := jsonScalarText(test.value); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestJSONNonDateStringsRemainUnchanged(t *testing.T) {
	for _, value := range []string{
		"plain", "2024-10-30", "2024-10-30t10:11:12z", "2024-10-30 10:11:12",
		"2024-02-30T10:11:12Z", "2024-10-30T10:11:12.12345678Z",
		"2024-10-30T24:00:01Z", "2024-10-30T24:00:00.1Z", "2024-10-30T10:11:12Z ",
		"2024-10-30T10:11:12+5:30", "0000-01-01T00:00:00Z", "/Date(not-a-number)/",
		strings.Repeat("9", 10000),
	} {
		if got := jsonScalarText(value); got != value {
			t.Fatalf("non-date input changed: %q => %q", value, got)
		}
	}
}

func TestJSONDateProjectionPrecedesCaseAndFiltersButDoesNotConvertLiteralText(t *testing.T) {
	const value = "2024-10-30T10:11:12Z"
	literal := Scalar(value)
	got, found, err := JSONField(context.Background(), nil, Selector{Text: &literal}, FilterContext{})
	if err != nil || !found || got != value {
		t.Fatalf("literal text was date-converted: %q %v", got, err)
	}
	got, found, err = JSONField(context.Background(), map[string]any{"stamp": value}, Selector{Selector: "stamp", Filters: []Filter{{Name: "prepend", Args: Arguments{"seen:"}}}}, FilterContext{})
	if err != nil || !found || got != "seen:10/30/2024 10:11:12" {
		t.Fatalf("filter ran before date projection: %q %v", got, err)
	}
}
