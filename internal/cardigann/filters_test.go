package cardigann

import (
	"context"
	"testing"
	"time"
)

func TestFiltersPreserveCommunitySemantics(t *testing.T) {
	fc := FilterContext{Now: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC), Variables: Variables{".Result.prefix": {Text: "X"}}}
	cases := []struct {
		name, input, want string
		args              Arguments
	}{
		{"regexp", "ABCD-123", "123", Arguments{`(?<=ABCD-)(\d+)`}},
		{"regexp", "no number", "", Arguments{`(\d+)`}},
		{"re_replace", "abc123", "X-123", Arguments{`abc(\d+)`, `{{ .Result.prefix }}-$1`}},
		{"split", "a=b=c", "c", Arguments{"=", "-1"}},
		{"querystring", "/dl?a=one&amp;b=two", "", Arguments{"b"}},
		{"dateparse", "2026-10-30 10:11 +08:00", "Fri, 30 Oct 2026 02:11:00 +0", Arguments{"yyyy-MM-dd HH:mm zzz"}},
		{"dateparse", "11/01 10:30", "Sun, 01 Nov 2026 10:30:00 +0", Arguments{"MM/dd HH:mm"}},
		{"timeago", "2 hours 30 minutes ago", "Sun, 01 Nov 2026 09:30:00 +0", nil},
		{"jsonjoinarray", `{"a":["one","two"]}`, "one|two", Arguments{"a", "|"}},
		{"validate", "Action / Drama,Other", "drama,action", Arguments{"Drama, Action"}},
		{"diacritics", "ŠĆŽ-é", "SCZ-e", Arguments{"replace"}},
		{"strdump", "sensitive response", "sensitive response", nil},
		{"re_replace", "Название ABCD-123", " ABCD-123", Arguments{`[\p{IsCyrillic}]+`, ""}},
		{"re_replace", "ABCD-123 中文", "ABCD123", Arguments{`([\p{IsCJKUnifiedIdeographs}\W]+)`, ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name+tc.input, func(t *testing.T) {
			got, err := ApplyFilters(context.Background(), tc.input, []Filter{{Name: tc.name, Args: tc.args}}, fc)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestFilterBudgetsAndInvalidArguments(t *testing.T) {
	for _, filter := range []Filter{{Name: "execute"}, {Name: "split", Args: Arguments{"", "0"}}, {Name: "regexp", Args: Arguments{"("}}, {Name: "replace", Args: Arguments{"one"}}} {
		if _, err := ApplyFilters(context.Background(), "input", []Filter{filter}, FilterContext{}); err == nil {
			t.Fatalf("accepted invalid filter: %#v", filter)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ApplyFilters(ctx, "a", []Filter{{Name: "tolower"}}, FilterContext{}); err != context.Canceled {
		t.Fatal(err)
	}
}
