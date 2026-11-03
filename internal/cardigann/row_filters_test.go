package cardigann

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRowAndMatchUsesFilteredQueryAndDeclaredIdentifierCapabilities(t *testing.T) {
	for _, sample := range []struct {
		name, query, title, argument, keywordFilter, modes string
		parameters                                         map[string]string
		want                                               bool
	}{
		{"hyphen", "ABCD-123", "ABCD 123", "", "", "", nil, true},
		{"positive-limit", "ABCD123extra", "ABCD123", ", args: 7", "", "", nil, true},
		{"zero-is-unlimited", "ABCD123extra", "ABCD123", ", args: 0", "", "", nil, false},
		{"negative-is-unlimited", "ABCD123extra", "ABCD123", ", args: -1", "", "", nil, false},
		{"stopwords", "The and an ABCD123 A", "ABCD123", "", "", "", nil, true},
		{"single-cjk", "中", "Other", "", "", "", nil, true},
		{"utf16-prefix", "AB😀CD", "AB", ", args: 5", "", "", nil, true},
		{"filtered", "ABCD123 extra", "ABCD123", "", "[{name: re_replace, args: ['extra', '']}]", "", nil, true},
		{"empty-filter-fallback", "AB&CD", "ABCD", "", "[{name: re_replace, args: ['.*', '']}]", "", nil, true},
		{"supported-imdb", "Different", "Other", "", "", "{search: [q], movie-search: [q, imdbid]}", map[string]string{"IMDBID": "tt1234567"}, true},
		{"unsupported-imdb", "Different", "Other", "", "", "", map[string]string{"IMDBID": "tt1234567"}, false},
		{"supported-tmdb", "Different", "Other", "", "", "{search: [q], movie-search: [q, tmdbid]}", map[string]string{"TMDBID": "123"}, true},
		{"missing-term", "ABCD123 missing", "ABCD123", "", "", "", nil, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "rows: {selector: article}", "rows: {selector: article, filters: [{name: andmatch"+sample.argument+"}]}", 1)
			if sample.modes != "" {
				definition = strings.Replace(definition, "modes: {search: [q]}", "modes: "+sample.modes, 1)
			}
			if sample.keywordFilter != "" {
				definition = strings.Replace(definition, "  rows:", "  keywordsfilters: "+sample.keywordFilter+"\n  rows:", 1)
			}
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
				return HTTPResponse{Status: 200, Body: []byte("<article><h2>" + sample.title + "</h2><a href='/file.torrent'>file</a></article>")}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			result, err := execution.Search(context.Background(), Query{Keywords: sample.query, Parameters: sample.parameters})
			if err != nil || (len(result.Results) == 1) != sample.want || len(result.Issues) != 0 {
				t.Fatalf("row filter changed search eligibility: %+v %v", result, err)
			}
		})
	}
}

func TestLoadRejectsInvalidAndMatchLimitAtSource(t *testing.T) {
	for _, argument := range []string{"bad", "''", "'1.5'", "2147483648"} {
		definition := strings.Replace(minimalDefinition, "rows: {selector: article}", "rows: {selector: article, filters: [{name: andmatch, args: "+argument+"}]}", 1)
		_, err := Load([]byte(definition))
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Path != "/search/rows/filters" || diagnostic.Line == 0 {
			t.Fatal("invalid row filter limit passed installation", err)
		}
	}
}
