package cardigann

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFuzzyCalendarRulesPreserveTheirCalendarMeaning(t *testing.T) {
	now := time.Date(2024, 3, 31, 12, 34, 56, 500000000, time.UTC)
	for _, test := range []struct{ input, want string }{
		{"today", "Sun, 31 Mar 2024 00:00:00 +0"},
		{"yesterday", "Sat, 30 Mar 2024 00:00:00 +0"},
		{"tomorrow", "Mon, 01 Apr 2024 00:00:00 +0"},
		{"Today, at 10:20", "Sun, 31 Mar 2024 10:20:00 +0"},
		{"yesterday 9:05 pm", "Sat, 30 Mar 2024 21:05:00 +0"},
		{"Sunday at 23:59", "Sun, 31 Mar 2024 23:59:00 +0"},
		{"Monday at 10:20", "Mon, 25 Mar 2024 10:20:00 +0"},
		{"Wednesday at 10:20", "Wed, 27 Mar 2024 10:20:00 +0"},
		{"unknown", "Sun, 31 Mar 2024 12:34:56 +0"},
		{"0", "Thu, 01 Jan 1970 00:00:00 +0"},
		{"1", "Thu, 01 Jan 1970 00:00:01 +0"},
		{"12-31 10:20", "Sun, 31 Dec 2023 10:20:00 +0"},
		{"31 Dec 10:30", "Sun, 31 Dec 2023 10:30:00 +0"},
		{"29 Feb 10:30", "Thu, 29 Feb 2024 10:30:00 +0"},
		{"2020-05-14 18:20:30 +05:30", "Thu, 14 May 2020 18:20:30 +0"},
		{"today 00:30 +08:00", "Sun, 31 Mar 2024 16:30:00 +0"},
	} {
		got, err := ApplyFilters(context.Background(), test.input, []Filter{{Name: "fuzzytime"}}, FilterContext{Now: now})
		if err != nil || got != test.want {
			t.Fatalf("%q: got %q, %v; want %q", test.input, got, err, test.want)
		}
	}
	for _, text := range []string{"not a date", "today 25:00", "1 month", "253402300800"} {
		if _, err := ApplyFilters(context.Background(), text, []Filter{{Name: "fuzzytime"}}, FilterContext{Now: now}); err == nil {
			t.Fatalf("invalid fuzzy date %q accepted", text)
		}
	}
}

func TestSearchFieldsAndDateHeadersUseTheSameCalendarParser(t *testing.T) {
	for _, header := range []bool{false, true} {
		definition := minimalDefinition
		if header {
			definition = strings.Replace(definition, "rows: {selector: article}", "rows: {selector: article, dateheaders: {selector: h3}}", 1)
		} else {
			definition += "    date: {selector: time}\n"
		}
		plan, err := Load([]byte(definition))
		if err != nil {
			t.Fatal(err)
		}
		session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
			return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<h3>Yesterday 9:05 pm</h3><article><h2>ABCD-123</h2><a href="/file.torrent">download</a><time>Yesterday 9:05 pm</time></article>`)}, nil
		}}
		execution, err := plan.NewExecution(session, "https://example.invalid/", nil, time.Date(2024, 3, 31, 12, 34, 56, 0, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		result, err := execution.Search(context.Background(), Query{Keywords: "ABCD-123"})
		if err != nil || len(result.Results) != 1 || len(result.Issues) != 0 || result.Results[0].PublishedAt == nil || result.Results[0].PublishedAt.Format(time.RFC3339) != "2024-03-30T21:05:00Z" {
			t.Fatalf("header=%v: calendar parser was bypassed: %+v, %v", header, result, err)
		}
	}
}
