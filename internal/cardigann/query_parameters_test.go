package cardigann

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNamedParametersCannotReplaceQueryContext(t *testing.T) {
	for _, parameters := range []map[string]string{
		{"Q": "replacement"}, {"Type": "movie"}, {"Limit": "999999"}, {"Offset": "40"},
		{"Categories": "6000"}, {"IsSearch": "fake"}, {"IMDBIDShort": "forged"},
		{"Episode": "forged"}, {"APIKey": "synthetic"}, {"unknown": "value"},
		{"Year": "2026", "year": "2025"},
	} {
		plan, err := Load([]byte(minimalDefinition))
		if err != nil {
			t.Fatal(err)
		}
		session := &fakeSession{respond: func(Request) (HTTPResponse, error) { return HTTPResponse{Status: 200}, nil }}
		execution, err := plan.NewExecution(session, "", nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		_, err = execution.Search(context.Background(), Query{Keywords: "ABCD-123", Parameters: parameters})
		if !errors.Is(err, ErrInvalidQuery) || len(session.requests) != 0 {
			t.Fatalf("invalid parameters reached search HTTP: keys=%v requests=%d err=%v", parameters, len(session.requests), err)
		}
	}
}

func TestNamedParameterCaseResolvesToDeclaredTemplateField(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "  rows:", "  inputs:\n    year: '{{ .Query.Year }}'\n    q: '{{ .Keywords }}'\n  rows:", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(request Request) (HTTPResponse, error) {
		parsed, err := url.Parse(request.URL)
		if err != nil || parsed.Query().Get("year") != "2026" || parsed.Query().Get("q") != "ABCD-123 2026" {
			t.Error("lowercase parameter was lost at template boundary", request.URL, err)
		}
		return HTTPResponse{Status: 200}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execution.Search(context.Background(), Query{Keywords: "ABCD-123", Parameters: map[string]string{"year": "002026"}}); err != nil {
		t.Fatal(err)
	}
}
