package cardigann

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTVQueryRawAndDerivedEpisodeVariablesMatchReference(t *testing.T) {
	for _, sample := range []struct {
		name, season, ep, expectedSeason, expectedEpisode string
	}{
		{"episode", "1", "2", "1", "S01E02"},
		{"season-pack", "3", "", "3", "S03"},
		{"daily", "2026", "11/03", "2026", "2026.11.03"},
		{"suffix-coercion", "1", "01A", "1", "S01E01"},
		{"zero-season", "0", "5", "", ""},
		{"dot-as-grouping", "1", "1.5", "1", "S01E15"},
		{"grouping-and-zero-fraction", "1", "1,234.0", "1", "S01E1234"},
		{"leading-separator-fallback", "1", ",0", "1", "S01E,0"},
		{"separator-in-fraction-fallback", "1", "1.0,0", "1", "S01E1.0,0"},
		{"text-coercion", "1", "special", "1", "S01E00"},
		{"invalid-daily-falls-back", "2026", "02/30", "2026", "S2026E230"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "modes: {search: [q]}", "modes: {search: [q], tv-search: [q, season, ep]}", 1)
			definition = strings.Replace(definition, "  paths: [{path: /}]", `  paths: [{path: /}]
  inputs:
    q: '{{ .Keywords }}'
    season: '{{ .Query.Season }}'
    ep: '{{ .Query.Ep }}'
    derived: '{{ .Query.Episode }}'`, 1)
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				u, _ := url.Parse(r.URL)
				values := u.Query()
				keywords := "ABCD-123"
				if sample.expectedEpisode != "" {
					keywords += " " + sample.expectedEpisode
				}
				if values.Get("q") != keywords || values.Get("season") != sample.expectedSeason || values.Get("ep") != sample.ep || values.Get("derived") != sample.expectedEpisode {
					t.Errorf("TV query differs from reference: %s", r.URL)
				}
				return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>ABCD-123</h2><a href='/dl'/></article>`)}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			_, err = execution.Search(context.Background(), Query{Mode: "tvsearch", Keywords: "ABCD-123", Parameters: map[string]string{"Season": sample.season, "Ep": sample.ep}})
			if err != nil || len(session.requests) != 1 {
				t.Fatal("TV request did not execute once", err)
			}
		})
	}
}

func TestTVQueryRejectsInvalidNumbersBeforeRequests(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "modes: {search: [q]}", "modes: {search: [q], tv-search: [q, season, ep]}", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	for _, parameters := range []map[string]string{
		{"Season": "bad", "Ep": "1"},
		{"Season": "2147483648", "Ep": "1"},
		{"Season": "1", "Ep": "2147483648"},
		{"Season": "1", "Ep": "1,0.5"},
	} {
		session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
			t.Fatal("invalid query reached HTTP")
			return HTTPResponse{}, nil
		}}
		execution, err := plan.NewExecution(session, "", nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := execution.Search(context.Background(), Query{Mode: "tvsearch", Parameters: parameters}); err == nil || len(session.requests) != 0 {
			t.Fatal("unrepresentable number was not rejected before HTTP", err)
		}
	}
}
