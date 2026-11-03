package cardigann

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSearchNormalizesTypedNumericIdentifiers(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "  rows: {selector: article}", `  inputs:
    year: '{{ .Query.Year }}'
    tmdb: '{{ .Query.TMDBID }}'
    tvdb: '{{ .Query.TVDBID }}'
    rage: '{{ .Query.TVRageID }}'
    maze: '{{ .Query.TVMazeID }}'
    trakt: '{{ .Query.TraktID }}'
    douban: '{{ .Query.DoubanID }}'
  rows: {selector: article}`, 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		u, _ := url.Parse(r.URL)
		want := url.Values{"year": {"2026"}, "tmdb": {"42"}, "tvdb": {"0"}, "rage": {"2147483647"}, "maze": {"-2147483648"}, "douban": {"347"}}
		if u.Query().Encode() != want.Encode() {
			t.Errorf("typed query identifiers differ: %s", r.URL)
		}
		return HTTPResponse{Status: 200}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = execution.Search(context.Background(), Query{Parameters: map[string]string{
		"Year": "002026", "TMDBID": " +00042 ", "TVDBID": "-0", "TVRageID": "2147483647", "TVMazeID": "-2147483648", "DoubanID": "00347",
	}})
	if err != nil || len(session.requests) != 1 {
		t.Fatal("numeric query was not executed once", err)
	}
}

func TestSearchRejectsInvalidTypedIdentifiersBeforeHTTP(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"Year", "TMDBID", "TVDBID", "TVRageID", "TVMazeID", "TraktID", "DoubanID"} {
		for _, value := range []string{"", " ", "bad", "2147483648", "-2147483649", "1.5"} {
			t.Run(key+"/"+value, func(t *testing.T) {
				session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
					t.Error("invalid numeric query reached HTTP")
					return HTTPResponse{Status: 200}, nil
				}}
				execution, err := plan.NewExecution(session, "", nil, time.Now())
				if err != nil {
					t.Fatal(err)
				}
				_, err = execution.Search(context.Background(), Query{Parameters: map[string]string{key: value}})
				if !errors.Is(err, ErrInvalidQuery) || err.Error() != "query "+key+" must be a 32-bit integer" || len(session.requests) != 0 {
					t.Fatal("invalid numeric query was not rejected before HTTP", err)
				}
			})
		}
	}
}
