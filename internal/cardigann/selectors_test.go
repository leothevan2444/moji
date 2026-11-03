package cardigann

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestHTMLSelectorCaseRemovalAndMissingValues(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html><body><article class="trusted"><h2>A &amp; B<span>noise</span></h2><a href="/dl">download</a></article></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	row := doc.Find("article")
	got, found, err := HTMLField(context.Background(), row, Selector{Selector: "h2", Remove: "span"}, FilterContext{})
	if err != nil || !found || got != "A & B" {
		t.Fatalf("%q %v %v", got, found, err)
	}
	got, _, err = HTMLField(context.Background(), row, Selector{Case: Cases{{Selector: "article.trusted", Value: "yes"}, {Selector: "article", Value: "no"}}}, FilterContext{})
	if err != nil || got != "yes" {
		t.Fatalf("%q %v", got, err)
	}
	defaultText := Scalar("fallback")
	got, found, err = HTMLField(context.Background(), row, Selector{Selector: "missing", Optional: true, Default: &defaultText}, FilterContext{})
	if err != nil || !found || got != "fallback" {
		t.Fatalf("%q %v %v", got, found, err)
	}
	if _, _, err = HTMLField(context.Background(), row, Selector{Selector: "missing"}, FilterContext{}); err == nil {
		t.Fatal("missing required selector accepted")
	}
}

func TestJSONSelectorNestedConditionsAndNullPresence(t *testing.T) {
	root := map[string]any{"meta": map[string]any{"label": "good", "present": nil}, "title": "ABCD-123"}
	for _, tc := range []struct {
		selector string
		found    bool
	}{{"meta:has(label:contains(good)):has(present)", true}, {"meta:not(missing)", true}, {"meta:not(present)", false}, {"meta:has(label:contains(bad))", false}} {
		selector, err := parseJSONSelector(tc.selector)
		if err != nil {
			t.Fatal(err)
		}
		values, err := selector.selectValues(root)
		if err != nil || (len(values) > 0) != tc.found {
			t.Fatalf("%s: %#v %v", tc.selector, values, err)
		}
	}
	got, found, err := JSONField(context.Background(), root, Selector{Selector: "title"}, FilterContext{})
	if err != nil || !found || got != "ABCD-123" {
		t.Fatalf("%q %v %v", got, found, err)
	}
}

func TestCSSScopeRelativeHasAndUnquotedContains(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<main><article id="one"><span><a href="/one">Only Upload</a></span></article><article id="two"><span><a href="/two">other</a></span></article></main>`))
	if err != nil {
		t.Fatal(err)
	}
	row := doc.Find("#one")
	selection, err := selectHTML(row, `:scope > span > a:contains(Only Upload)`, false)
	if err != nil || selection.Length() != 1 || selection.Text() != "Only Upload" {
		t.Fatalf("scope/contains failed: %v %#v", err, selection)
	}
	selection, err = selectHTML(doc.Selection, `article:has(~ article a[href="/two"])`, true)
	if err != nil || selection.Length() != 1 || selection.AttrOr("id", "") != "one" {
		t.Fatalf("relative has failed: %v %#v", err, selection)
	}
	if doc.Find("["+scopeAttribute+"]").Length() != 0 || doc.Find("[data-moji-cardigann-relative-0]").Length() != 0 {
		t.Fatal("scope markers leaked")
	}
}

func TestJSONCasesCompareLiteralValuesAndArraysUseCommaText(t *testing.T) {
	root := map[string]any{"leechType": "-75%", "validated": true, "tags": []any{"one", "two"}}
	cases := []struct {
		selector Selector
		want     string
	}{
		{Selector{Selector: "leechType", Case: Cases{{Selector: "-75%", Value: "0.25"}, {Selector: "*", Value: "1"}}}, "0.25"},
		{Selector{Selector: "validated", Case: Cases{{Selector: "True", Value: "yes"}, {Selector: "False", Value: "no"}}}, "yes"},
		{Selector{Selector: "tags"}, "one,two"},
	}
	for _, tc := range cases {
		got, found, err := JSONField(context.Background(), root, tc.selector, FilterContext{})
		if err != nil || !found || got != tc.want {
			t.Fatalf("got %q/%v/%v, want %q", got, found, err, tc.want)
		}
	}
}
