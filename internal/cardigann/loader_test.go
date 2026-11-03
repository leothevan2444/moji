package cardigann

import (
	"strings"
	"testing"
)

const minimalDefinition = `id: example
name: Example
description: Test fixture
language: en-US
type: public
encoding: UTF-8
links: [https://example.invalid/]
caps:
  categories: {Movies: Movies}
  modes: {search: [q]}
search:
  paths: [{path: /}]
  rows: {selector: article}
  fields:
    title: {selector: h2}
    category: {text: Movies}
    size: {text: 1GB}
    seeders: {text: 0}
    download: {selector: a, attribute: href}
`

func TestLoadRetainsOrderedDuplicateFieldsAndScalarSpelling(t *testing.T) {
	input := minimalDefinition + "    title|append: {text: ' first'}\n    title|append: {text: ' second'}\n"
	plan, err := Load([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	fields := plan.Definition.Search.Fields
	if len(fields) != 7 || fields[0].Name != "title" || string(*fields[6].Selector.Text) != " second" {
		t.Fatalf("fields lost order or duplicate values: %#v", fields)
	}
	if plan.Digest == "" || string(*fields[3].Selector.Text) != "0" {
		t.Fatal("missing digest or changed scalar")
	}
}

func TestLoadRejectsInvalidAndUnboundedRules(t *testing.T) {
	cases := map[string]string{
		"unknown feature":          strings.Replace(minimalDefinition, "  rows:", "  unsupported: true\n  rows:", 1),
		"duplicate identity":       minimalDefinition + "id: other\n",
		"hidden invalid duplicate": minimalDefinition + "    title|append: {unsupported: true}\n    title|append: {text: valid}\n",
		"multiple documents":       minimalDefinition + "---\n{}\n",
		"alias":                    strings.Replace(minimalDefinition, "name: Example", "name: &n Example\ndescription: *n", 1),
		"oversize":                 strings.Repeat(" ", MaxDefinitionBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load([]byte(input)); err == nil {
				t.Fatal("accepted invalid definition")
			}
		})
	}
}

func TestLoadAcceptsJSONSolidusEscapeWithoutRewritingStoredDigest(t *testing.T) {
	input := strings.Replace(minimalDefinition, "{selector: h2}", `{selector: 'h2', filters: [{name: regexp, args: "(.*)\/"}]}`, 1)
	plan, err := Load([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Definition.Search.Fields[0].Selector.Filters[0].Args[0] != "(.*)/" {
		t.Fatal("solidus escape not parsed compatibly")
	}
	plain := strings.ReplaceAll(input, `\/`, `/`)
	other, err := Load([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	if other.Digest == plan.Digest {
		t.Fatal("digest must refer to original bytes")
	}
}
