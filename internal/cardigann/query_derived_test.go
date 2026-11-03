package cardigann

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDerivedQueryFlagsAreRenderedFromPresenceAndRawEpisode(t *testing.T) {
	for _, sample := range []struct {
		name       string
		parameters map[string]string
		want       string
	}{
		{"rss", nil, "id=|rss=True|imdb=|genre=|episode="},
		{"empty-genre", map[string]string{"Genre": ""}, "id=|rss=True|imdb=|genre=True|episode="},
		{"empty-imdb", map[string]string{"IMDBID": ""}, "id=True|rss=|imdb=True|genre=|episode="},
		{"zero-year", map[string]string{"Year": "0"}, "id=True|rss=|imdb=|genre=|episode="},
		{"episode-without-season", map[string]string{"Ep": "9"}, "id=True|rss=|imdb=|genre=|episode="},
		{"blank-episode", map[string]string{"Ep": " "}, "id=|rss=True|imdb=|genre=|episode="},
		{"negative-season", map[string]string{"Season": "-1"}, "id=|rss=True|imdb=|genre=|episode=S-01"},
		{"publisher", map[string]string{"Publisher": "Publisher"}, "id=True|rss=|imdb=|genre=|episode="},
	} {
		t.Run(sample.name, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "  rows:", "  inputs:\n    flags: 'id={{ .Query.IsIdSearch }}|rss={{ .Query.IsRssSearch }}|imdb={{ .Query.IsImdbQuery }}|genre={{ .Query.IsGenreQuery }}|episode={{ .Query.Episode }}'\n  rows:", 1)
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(request Request) (HTTPResponse, error) {
				parsed, err := url.Parse(request.URL)
				if err != nil || parsed.Query().Get("flags") != sample.want {
					t.Errorf("derived flags did not reach request: %s %v", request.URL, err)
				}
				return HTTPResponse{Status: 200, Body: []byte("<div>no results</div>")}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := execution.Search(context.Background(), Query{Parameters: sample.parameters}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
