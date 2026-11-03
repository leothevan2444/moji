// moji-indexer-check examines a locally downloaded, pinned definition corpus.
// It never reads application config or contacts configured production services.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leothevan2444/moji/internal/cardigann"
)

type report struct {
	RequestFeatures    map[string]int                    `json:"requestFeatures"`
	RequestDefinitions map[string][]string               `json:"requestDefinitions"`
	Encodings          map[string]int                    `json:"encodings"`
	QueryModes         map[string]int                    `json:"queryModes"`
	QueryVariables     map[string]int                    `json:"queryVariables"`
	Definitions        int                               `json:"definitions"`
	Loaded             int                               `json:"loaded"`
	Failures           map[string]string                 `json:"failures"`
	Warnings           map[string][]cardigann.Diagnostic `json:"warnings"`
	Filters            map[string]int                    `json:"filters"`
	LoginMethods       map[string]int                    `json:"loginMethods"`
	ResponseTypes      map[string]int                    `json:"responseTypes"`
	DateFormats        map[string]int                    `json:"dateFormats"`
	Templates          map[string]int                    `json:"templates"`
}

func main() {
	directory := flag.String("definitions", "", "local v11 definition directory (required)")
	output := flag.String("report", "", "write JSON report to this path instead of stdout")
	flag.Parse()
	if *directory == "" {
		fmt.Fprintln(os.Stderr, "-definitions is required")
		os.Exit(2)
	}
	r, err := inspect(*directory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *output == "" {
		_, err = os.Stdout.Write(append(data, '\n'))
	} else {
		err = os.WriteFile(*output, append(data, '\n'), 0600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(r.Failures) > 0 {
		os.Exit(1)
	}
}

func inspect(directory string) (report, error) {
	r := report{RequestFeatures: map[string]int{}, RequestDefinitions: map[string][]string{}, Encodings: map[string]int{}, QueryModes: map[string]int{}, QueryVariables: map[string]int{}, Failures: map[string]string{}, Warnings: map[string][]cardigann.Diagnostic{}, Filters: map[string]int{}, LoginMethods: map[string]int{}, ResponseTypes: map[string]int{}, DateFormats: map[string]int{}, Templates: map[string]int{}}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return r, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		r.Definitions++
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return r, err
		}
		plan, err := cardigann.Load(data)
		if err != nil {
			r.Failures[entry.Name()] = err.Error()
		} else {
			r.Loaded++
			collectRequestFeatures(&r, entry.Name(), plan)
			if len(plan.Warnings) > 0 {
				r.Warnings[entry.Name()] = plan.Warnings
			}
			mergeCounts(r.Filters, plan.Features.Filters)
			mergeCounts(r.Templates, plan.Features.Templates)
			mergeCounts(r.DateFormats, plan.Features.DateFormats)
			if plan.Definition.Login != nil {
				r.LoginMethods[plan.Definition.Login.Method]++
			}
			for _, p := range plan.Definition.Search.Paths {
				if p.Response.Type != "" {
					r.ResponseTypes[p.Response.Type]++
				}
			}
		}
	}
	return r, nil
}

func mergeCounts(target, source map[string]int) {
	for key, count := range source {
		target[key] += count
	}
}
