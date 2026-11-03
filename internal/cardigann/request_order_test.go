package cardigann

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSearchKeepsDeclaredInputOrderAcrossInheritedAndRawParameters(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "  paths: [{path: /}]", `  inputs:
    z: first
    $raw: 'r=1&r=2'
    a: last
  paths:
    - path: /search
      method: `+method+`
      inputs:
        z: local
        b: end`, 1)
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				u, _ := url.Parse(r.URL)
				encoded := u.RawQuery
				if r.Method == http.MethodPost {
					encoded = string(r.Body)
				}
				if encoded != "z=first&r=1&r=2&a=last&z=local&b=end" {
					t.Errorf("declared input order changed: %s", encoded)
				}
				return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href='/dl'/></article>`)}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := execution.Search(context.Background(), Query{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMappedCategoriesKeepDefinitionOrder(t *testing.T) {
	for _, sample := range []struct {
		categories string
		requested  []int
	}{
		{"  categories: {'9': Movies/HD, '2': Movies/SD}", []int{2000}},
		{"  categorymappings: [{id: 9, cat: Movies/HD}, {id: 2, cat: Movies/SD}, {id: 9, cat: Movies/UHD}]", []int{2000}},
		{"  categorymappings: [{id: 9, cat: Movies/HD, default: true}, {id: 2, cat: Movies/SD, default: true}]", nil},
	} {
		definition := strings.Replace(minimalDefinition, "  categories: {Movies: Movies}", sample.categories, 1)
		definition = strings.Replace(definition, "  paths: [{path: /}]", "  paths: [{path: /}]\n  inputs: {cat: '{{ join .Categories \",\" }}'}", 1)
		plan, err := Load([]byte(definition))
		if err != nil {
			t.Fatal(err)
		}
		session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
			u, _ := url.Parse(r.URL)
			if u.Query().Get("cat") != "9,2" {
				t.Errorf("declared category order changed: %s", u.RawQuery)
			}
			return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href='/dl'/></article>`)}, nil
		}}
		execution, err := plan.NewExecution(session, "", nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := execution.Search(context.Background(), Query{Categories: sample.requested}); err != nil {
			t.Fatal(err)
		}
	}
}
