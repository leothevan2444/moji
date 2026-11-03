package cardigann

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestJSONRowCountOnlyValidInt32CanDeclareEmptyBeforeRows(t *testing.T) {
	for _, sample := range []struct {
		name, count, body string
		wantRows          int
		wantCount         *int64
	}{
		{"zero-before-missing-rows", "{selector: total}", `{"total":0}`, 0, countPointer(0)},
		{"negative-before-missing-rows", "{selector: total}", `{"total":-1}`, 0, nil},
		{"positive", "{selector: total}", `{"total":3,"data":[{"name":"Example","url":"/file.torrent"}]}`, 1, countPointer(3)},
		{"signed-and-padded", "{selector: total}", `{"total":" +1 ","data":[{"name":"Example","url":"/file.torrent"}]}`, 1, countPointer(1)},
		{"missing", "{selector: total}", `{"data":[{"name":"Example","url":"/file.torrent"}]}`, 1, nil},
		{"invalid", "{selector: total}", `{"total":"unknown","data":[{"name":"Example","url":"/file.torrent"}]}`, 1, nil},
		{"fraction", "{selector: total}", `{"total":0.5,"data":[{"name":"Example","url":"/file.torrent"}]}`, 1, nil},
		{"overflow", "{selector: total}", `{"total":2147483648,"data":[{"name":"Example","url":"/file.torrent"}]}`, 1, nil},
		{"underflow", "{selector: total}", `{"total":-2147483649,"data":[{"name":"Example","url":"/file.torrent"}]}`, 1, nil},
		{"filtered-before-rows", "{selector: total, filters: [{name: replace, args: [none, '0']}]}", `{"total":"none"}`, 0, countPointer(0)},
	} {
		t.Run(sample.name, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", "paths: [{path: /api, response: {type: json}}]", 1)
			definition = strings.Replace(definition, "rows: {selector: article}", "rows: {selector: data, count: "+sample.count+"}", 1)
			definition = strings.Replace(definition, "title: {selector: h2}", "title: {selector: name}", 1)
			definition = strings.Replace(definition, "download: {selector: a, attribute: href}", "download: {selector: url}", 1)
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
			result, err := execution.Search(context.Background(), Query{Keywords: "Example"})
			if err != nil || len(result.Results) != sample.wantRows || len(result.Issues) != 0 {
				t.Fatalf("count changed result eligibility: %+v %v", result, err)
			}
			if (result.ReportedCount == nil) != (sample.wantCount == nil) || result.ReportedCount != nil && *result.ReportedCount != *sample.wantCount {
				t.Fatalf("invalid or missing count became an observation: %+v", result)
			}
		})
	}
}

func TestHTMLRowCountDoesNotRejectOtherwiseUsableRows(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "rows: {selector: article}", "rows: {selector: article, count: {selector: .missing}}", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, Body: []byte(`<article><h2>Example</h2><a href='/file.torrent'>file</a></article>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{Keywords: "Example"})
	if err != nil || len(result.Results) != 1 || len(result.Issues) != 0 || result.ReportedCount != nil {
		t.Fatalf("JSON-only count executed in HTML: %+v %v", result, err)
	}
}

func countPointer(value int64) *int64 { return &value }
