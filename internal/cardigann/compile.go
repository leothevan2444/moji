package cardigann

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// validateSyntax rejects unknown or malformed executable constructs at install
// time, including templates in currently unselected conditional branches.
type FeatureUsage struct {
	Filters     map[string]int
	Templates   map[string]int
	DateFormats map[string]int
}

func validateSyntax(n *yaml.Node, path string, jsonSearch bool, templates map[string]*Template, warnings *[]Diagnostic, features *FeatureUsage) error {
	if path == "/encoding" && n.Kind == yaml.ScalarNode {
		if _, err := declaredEncoding(n.Value); err != nil {
			return Diagnostic{Path: path, Line: n.Line, Column: n.Column, Message: err.Error()}
		}
	}
	if n.Kind == yaml.ScalarNode && strings.Contains(n.Value, "{{") {
		features.Templates[n.Value]++
		if _, exists := templates[n.Value]; !exists {
			compiled, err := CompileTemplate(n.Value)
			if err != nil {
				return Diagnostic{Path: path, Line: n.Line, Column: n.Column, Message: err.Error()}
			}
			templates[n.Value] = compiled
		}
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			key, value := n.Content[i].Value, n.Content[i+1]
			childPath := path + "/" + key
			if err := validateSyntax(n.Content[i], path+"/<key>", jsonSearch, templates, warnings, features); err != nil {
				return err
			}
			if strings.HasSuffix(path, "/case") && !strings.Contains(key, "{{") && !(jsonSearch && strings.HasPrefix(path, "/search/")) {
				var err error
				_, err = compileCSS(key)
				if err != nil {
					return Diagnostic{Path: path, Line: n.Content[i].Line, Column: n.Content[i].Column, Message: fmt.Sprintf("invalid case selector: %v", err)}
				}
			}
			if (key == "filters" || key == "keywordsfilters" || key == "preprocessingfilters") && value.Kind == yaml.SequenceNode {
				for _, item := range value.Content {
					var filter Filter
					if err := item.Decode(&filter); err != nil {
						return err
					}
					features.Filters[filter.Name]++
					if (filter.Name == "dateparse" || filter.Name == "timeparse") && len(filter.Args) > 0 {
						features.DateFormats[string(filter.Args[0])]++
					}
					if path == "/search/rows" && filter.Name == "andmatch" {
						if _, err := andMatchLimit(filter); err != nil {
							return Diagnostic{Path: childPath, Line: item.Line, Column: item.Column, Message: err.Error()}
						}
						continue
					}
					if err := ValidateFilter(filter); err != nil {
						return Diagnostic{Path: childPath, Line: item.Line, Column: item.Column, Message: err.Error()}
					}
				}
			}
			if (key == "selector" || key == "remove" || key == "form") && value.Kind == yaml.ScalarNode && value.Value != "" && !strings.Contains(value.Value, "{{") {
				selector := value.Value
				var err error
				if jsonSearch && strings.HasPrefix(path, "/search/") && key != "remove" {
					_, err = parseJSONSelector(selector)
				} else {
					selector = strings.TrimSpace(strings.TrimPrefix(selector, ":root"))
					if selector != "" {
						_, err = compileCSS(selector)
					}
				}
				if err != nil {
					diagnostic := Diagnostic{Path: childPath, Line: value.Line, Column: value.Column, Message: fmt.Sprintf("invalid selector: %v", err)}
					optional := false
					for j := 0; j < len(n.Content); j += 2 {
						if n.Content[j].Value == "optional" && n.Content[j+1].Value == "true" {
							optional = true
						}
					}
					if optional {
						*warnings = append(*warnings, diagnostic)
					} else {
						return diagnostic
					}
				}
			}
			if err := validateSyntax(value, childPath, jsonSearch, templates, warnings, features); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range n.Content {
		if err := validateSyntax(child, path, jsonSearch, templates, warnings, features); err != nil {
			return err
		}
	}
	return nil
}
