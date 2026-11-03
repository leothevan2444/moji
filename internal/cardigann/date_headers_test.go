package cardigann

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDateHeadersDoNotBorrowDatesAcrossInvalidOrUnrelatedGroups(t *testing.T) {
	const header = "<h3>Mon, 02 Nov 2026 00:00:00 +0000</h3>"
	const article = "<article><h2>ABCD-123</h2><a href='/file.torrent'>file</a></article>"
	for _, sample := range []struct {
		name, body string
		optional   bool
		wantRows   int
		issue      string
	}{
		{"sibling", header + article, false, 1, ""},
		{"parent-sibling", header + "<section>" + article + "</section>", false, 1, ""},
		{"nested-section", header + "<section><div>" + article + "</div></section>", false, 0, "REQUIRED_FIELD_MISSING"},
		{"invalid-nearest", header + "<h3>completely invalid date</h3>" + article, false, 0, "FIELD_VALUE_INVALID"},
		{"invalid-nearest-optional", header + "<h3>completely invalid date</h3>" + article, true, 0, "FIELD_VALUE_INVALID"},
		{"optional-missing", article, true, 1, ""},
	} {
		t.Run(sample.name, func(t *testing.T) {
			optional := ""
			if sample.optional {
				optional = ", optional: true"
			}
			definition := strings.Replace(minimalDefinition, "rows: {selector: article}", "rows: {selector: article, dateheaders: {selector: h3"+optional+"}}", 1)
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
				return HTTPResponse{Status: 200, Body: []byte(sample.body)}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			result, err := execution.Search(context.Background(), Query{Keywords: "ABCD-123"})
			if err != nil || len(result.Results) != sample.wantRows {
				t.Fatal("wrong date-header eligibility", result, err)
			}
			if sample.issue == "" {
				if len(result.Issues) != 0 {
					t.Fatal("valid/optional header produced an issue", result.Issues)
				}
				if sample.name != "optional-missing" && (result.Results[0].PublishedAt == nil || result.Results[0].PublishedAt.Format(time.RFC3339) != "2026-11-02T00:00:00Z") {
					t.Fatal("matched header changed its date", result)
				}
			} else if len(result.Issues) != 1 || result.Issues[0].Field != "dateheaders" || result.Issues[0].Code != sample.issue {
				t.Fatal("invalid header lost partial-result diagnosis", result.Issues)
			}
		})
	}
}
