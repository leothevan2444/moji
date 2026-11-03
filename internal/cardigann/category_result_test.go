package cardigann

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

func searchCategoryFixture(t *testing.T, definition string) Result {
	t.Helper()
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href='/dl'/></article>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{})
	if err != nil || len(result.Results) != 1 {
		t.Fatalf("category search fixture: %#v %v", result, err)
	}
	return result.Results[0]
}

func TestResultCategoryIDsMatchWithoutCaseSensitivity(t *testing.T) {
	for _, categories := range []string{
		"  categories: {FiLm: Movies/HD}",
		"  categorymappings: [{id: FiLm, cat: Movies/HD}]",
	} {
		definition := strings.Replace(minimalDefinition, "  categories: {Movies: Movies}", categories, 1)
		definition = strings.Replace(definition, "category: {text: Movies}", "category: {text: fIlM}", 1)
		result := searchCategoryFixture(t, definition)
		if !slices.Equal(result.Categories, []string{"Movies/HD"}) {
			t.Errorf("case variant lost category: %v", result.Categories)
		}
	}
}

func TestCategoryDescriptionAppendAndReplacementUseMappedCategories(t *testing.T) {
	for _, sample := range []struct {
		name, suffix string
		expected     []string
	}{
		{"append", "", []string{"Movies/HD", "Movies/UHD"}},
		{"unmapped-replacement", "    categorydesc|noappend: {text: missing}\n", []string{"Movies/HD", "Movies/UHD"}},
		{"mapped-replacement", "    categorydesc|noappend: {text: SD}\n", []string{"Movies/SD"}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "  categories: {Movies: Movies}", `  categorymappings:
    - {id: hd, desc: HD, cat: Movies/HD}
    - {id: uhd, desc: UHD, cat: Movies/UHD}
    - {id: sd, desc: SD, cat: Movies/SD}`, 1)
			definition = strings.Replace(definition, "category: {text: Movies}", "categorydesc: {text: HD}", 1)
			definition += "    categorydesc|append: {text: UHD}\n" + sample.suffix
			result := searchCategoryFixture(t, definition)
			if !slices.Equal(result.Categories, sample.expected) {
				t.Fatalf("category modifier changed mapping: %v; want %v", result.Categories, sample.expected)
			}
		})
	}
}
