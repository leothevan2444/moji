package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReportFindsRequestFeaturesAndAffectedDefinitions(t *testing.T) {
	directory := t.TempDir()
	rule := `id: example
name: Example
description: Synthetic inventory fixture
language: en-US
type: public
encoding: windows-1251
links: [https://example.invalid/]
caps:
  categories: {Movies: Movies}
  modes: {search: [q], tv-search: [q, season, ep]}
search:
  inputs: {q: '{{ .Query.Q }}', 'category[]': film}
  paths:
    - path: /search
      method: post
      inputs: {q: local, $raw: 'season={{ .Query.Season }}'}
    - path: /isolated
      inheritinputs: false
      inputs: {q: '{{ .Query.Ep }}'}
  rows: {selector: article}
  fields:
    title: {selector: h2}
    download: {selector: a, attribute: href}
    category: {text: Movies}
    size: {text: 1GB}
    seeders: {text: 0}
`
	if e := os.WriteFile(filepath.Join(directory, "example.yml"), []byte(rule), 0600); e != nil {
		t.Fatal(e)
	}
	r, e := inspect(directory)
	if e != nil || r.Loaded != 1 || len(r.Failures) != 0 {
		t.Fatal("inventory did not compile fixture", e, r.Failures)
	}
	for _, feature := range []string{"search.post", "search.raw", "inheritinputs.false", "inheritinputs.repeated-key", "search.literal-input-name"} {
		if r.RequestFeatures[feature] != 1 || len(r.RequestDefinitions[feature]) != 1 || r.RequestDefinitions[feature][0] != "example.yml" {
			t.Fatal("request feature missing or duplicated", feature)
		}
	}
	if r.Encodings["windows-1251"] != 1 || r.QueryModes["tv-search"] != 1 || r.QueryVariables[".Query.Season"] == 0 || r.QueryVariables[".Query.Ep"] == 0 {
		t.Fatal("encoding/query inventory missing")
	}
}
