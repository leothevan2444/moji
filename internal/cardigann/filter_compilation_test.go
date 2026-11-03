package cardigann

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadRejectsInvalidSearchFilterPipelinesBeforeActivation(t *testing.T) {
	for _, pipeline := range []string{"keywordsfilters", "preprocessingfilters"} {
		for _, filter := range []string{
			"{name: regexp, args: '('}",
			"{name: replace, args: [one]}",
		} {
			t.Run(pipeline+"/"+filter, func(t *testing.T) {
				definition := strings.Replace(minimalDefinition, "  rows: {selector: article}", "  "+pipeline+": ["+filter+"]\n  rows: {selector: article}", 1)
				_, err := Load([]byte(definition))
				var diagnostic Diagnostic
				if !errors.As(err, &diagnostic) || diagnostic.Path != "/search/"+pipeline || diagnostic.Line == 0 || diagnostic.Column == 0 {
					t.Fatalf("invalid executable filter was not rejected at its source: %v", err)
				}
			})
		}
	}
}

func TestSearchFilterPipelinesAppearInStaticFeatureReport(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "  rows: {selector: article}", `  keywordsfilters: [{name: re_replace, args: ['a', 'b']}]
  preprocessingfilters: [{name: dateparse, args: 'yyyy-MM-dd'}]
  rows: {selector: article}`, 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Features.Filters["re_replace"] != 1 || plan.Features.Filters["dateparse"] != 1 || plan.Features.DateFormats["yyyy-MM-dd"] != 1 {
		t.Fatalf("search filter pipelines were omitted from the capability report: %+v", plan.Features)
	}
}
