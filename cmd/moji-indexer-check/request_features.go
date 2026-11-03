package main

import (
	"regexp"
	"slices"
	"strings"

	"github.com/leothevan2444/moji/internal/cardigann"
)

var queryVariable = regexp.MustCompile(`\.Query\.[A-Za-z][A-Za-z0-9]*`)

func collectRequestFeatures(r *report, name string, plan *cardigann.Plan) {
	add := func(feature string) {
		r.RequestFeatures[feature]++
		if !slices.Contains(r.RequestDefinitions[feature], name) {
			r.RequestDefinitions[feature] = append(r.RequestDefinitions[feature], name)
		}
	}
	encoding := plan.Definition.Encoding
	if encoding == "" {
		encoding = "UTF-8"
	}
	r.Encodings[strings.ToLower(encoding)]++
	for mode := range plan.Definition.Caps.Modes {
		r.QueryModes[mode]++
	}
	for template, count := range plan.Features.Templates {
		for _, variable := range queryVariable.FindAllString(template, -1) {
			r.QueryVariables[variable] += count
		}
	}
	search := plan.Definition.Search
	for _, path := range search.Paths {
		inherited := path.InheritInputs == nil || *path.InheritInputs
		if !inherited {
			add("inheritinputs.false")
		}
		for _, input := range path.Inputs {
			if _, ok := search.Inputs.Lookup(input.Name); inherited && ok {
				add("inheritinputs.repeated-key")
			}
		}
		_, globalRaw := search.Inputs.Lookup("$raw")
		_, localRaw := path.Inputs.Lookup("$raw")
		if inherited && globalRaw || localRaw {
			add("search.raw")
		}
		if inherited && hasLiteralInputName(search.Inputs) || hasLiteralInputName(path.Inputs) {
			add("search.literal-input-name")
		}
		if strings.EqualFold(path.Method, "post") {
			add("search.post")
		}
		if path.QuerySeparator != "" && path.QuerySeparator != "&" {
			add("search.custom-separator")
		}
		if len(path.Categories) > 0 {
			add("search.categories")
			if path.Categories[0] == "!" {
				add("search.categories.exclude")
			}
		}
	}
}

func hasLiteralInputName(inputs cardigann.ScalarFields) bool {
	for _, input := range inputs {
		if strings.ContainsAny(input.Name, "[]") {
			return true
		}
	}
	return false
}
