package cardigann

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCategoryVariablesMappingAndPathEligibility(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "  categories: {Movies: Movies}", `  categorymappings:
    - {id: 10, cat: Movies/HD}
    - {id: 11, cat: Movies/SD}
    - {id: 12, cat: Audio}
    - {id: 10, cat: Movies/UHD}
    - {id: 99, cat: Other, default: true}`, 1)
	definition = strings.Replace(definition, "  paths: [{path: /}]", `  inputs:
    original: '{{ join .Query.Categories "," }}'
    mapped: '{{ if .Categories }}{{ join .Categories "," }}{{ else }}NONE{{ end }}'
  paths:
    - {path: /positive, categories: ['10']}
    - {path: /negative, categories: ['!', '12']}`, 1)
	for _, sample := range []struct {
		name            string
		categories      []int
		original        string
		paths           map[string]string
		withoutDefaults bool
	}{
		{"parent-and-duplicates", []int{2000}, "2000", map[string]string{"/positive": "10", "/negative": "NONE"}, false},
		{"child", []int{2040}, "2040", map[string]string{"/positive": "10", "/negative": "NONE"}, false},
		{"excluded", []int{3000}, "3000", map[string]string{}, false},
		{"defaults", nil, "", map[string]string{"/negative": "NONE"}, false},
		{"unrestricted", nil, "", map[string]string{"/positive": "NONE", "/negative": "NONE"}, true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			source := definition
			if sample.withoutDefaults {
				source = strings.Replace(source, "    - {id: 99, cat: Other, default: true}\n", "", 1)
			}
			plan, err := Load([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				target, err := url.Parse(r.URL)
				if err != nil {
					t.Fatal(err)
				}
				want, ok := sample.paths[target.Path]
				if !ok || seen[target.Path] || target.Query().Get("original") != sample.original || target.Query().Get("mapped") != want {
					t.Errorf("wrong category request: %s", r.URL)
				}
				seen[target.Path] = true
				return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href='/dl'/></article>`)}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := execution.Search(context.Background(), Query{Categories: sample.categories}); err != nil {
				t.Fatal(err)
			}
			if len(seen) != len(sample.paths) {
				t.Fatalf("missing eligible paths: %v", seen)
			}
		})
	}
}
