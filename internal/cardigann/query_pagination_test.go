package cardigann

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSearchDistinguishesUnsetPaginationFromExplicitZero(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "  rows: {selector: article}", `  inputs:
    limit: '{{ .Query.Limit }}'
    offset: '{{ .Query.Offset }}'
    haslimit: '{{ if .Query.Limit }}present{{ else }}absent{{ end }}'
    hasoffset: '{{ if .Query.Offset }}present{{ else }}absent{{ end }}'
  rows: {selector: article}`, 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, Body: []byte(`<article><h2>One</h2><a href="/one">get</a></article><article><h2>Two</h2><a href="/two">get</a></article>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		name   string
		query  Query
		values url.Values
		rows   int
	}{
		{"unset", Query{}, url.Values{"haslimit": {"absent"}, "hasoffset": {"absent"}}, 2},
		{"zero", Query{Limit: queryInteger(0), Offset: queryInteger(0)}, url.Values{"limit": {"0"}, "offset": {"0"}, "haslimit": {"present"}, "hasoffset": {"present"}}, 2},
		{"values", Query{Limit: queryInteger(1), Offset: queryInteger(40)}, url.Values{"limit": {"1"}, "offset": {"40"}, "haslimit": {"present"}, "hasoffset": {"present"}}, 1},
		{"unset-again", Query{}, url.Values{"haslimit": {"absent"}, "hasoffset": {"absent"}}, 2},
	} {
		t.Run(sample.name, func(t *testing.T) {
			result, err := execution.Search(context.Background(), sample.query)
			if err != nil {
				t.Fatal(err)
			}
			target, err := url.Parse(session.requests[len(session.requests)-1].URL)
			if err != nil || target.Query().Encode() != sample.values.Encode() {
				t.Fatalf("pagination values differ: %s (%v)", target, err)
			}
			if len(result.Results) != sample.rows || result.Truncated != (sample.rows == 1) {
				t.Fatalf("result limit differs: %+v", result)
			}
		})
	}
}

func TestSearchInputNamesRetainDeclaredQuerySyntax(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", "paths: [{path: /, method: "+method+"}]", 1)
			definition = strings.Replace(definition, "  rows: {selector: article}", `  inputs:
    'filter[]': '{{ .Keywords }}'
    'filter[language]': 'a b+é'
  rows: {selector: article}`, 1)
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				target, _ := url.Parse(r.URL)
				encoded := target.RawQuery
				if method == "post" {
					encoded = string(r.Body)
				}
				if encoded != "filter[]=A+%26+B&filter[language]=a+b%2B%C3%A9" {
					t.Errorf("declared names or escaped values changed: %q", encoded)
				}
				return HTTPResponse{Status: 200}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := execution.Search(context.Background(), Query{Keywords: "A & B"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
