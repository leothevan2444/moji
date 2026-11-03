package cardigann

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const MaxDefinitionBytes = 1 << 20
const MaxDefinitionDepth = 64
const MaxDefinitionNodes = 100000

// The validation contract is pinned independently of runtime rule downloads.
// Source: Prowlarr/Indexers, commit 5f5db4e58d06be18619c3a858b9a91ade6f341ca.
//
//go:embed schema_v11.json
var schemaV11 []byte

var validationSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := validationDocument()
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("https://moji.invalid/cardigann/v11", doc); err != nil {
		return nil, err
	}
	return c.Compile("https://moji.invalid/cardigann/v11")
})

// Plan is immutable after Load. Callers must not modify its Definition.
// Compiled execution data is private so site sessions cannot mutate it.
type Plan struct {
	Definition Definition
	Digest     string
	templates  map[string]*Template
	Warnings   []Diagnostic
	Features   FeatureUsage
}

// Diagnostic identifies rule locations without including response bodies,
// credentials, query URLs, or private per-instance settings.
type Diagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", d.Path, d.Line, d.Column, d.Message)
}

func Load(data []byte) (*Plan, error) {
	if len(data) == 0 || len(data) > MaxDefinitionBytes {
		return nil, fmt.Errorf("definition must contain 1..%d bytes", MaxDefinitionBytes)
	}
	var root yaml.Node
	// YamlDotNet accepts the JSON solidus escape used by community rules.
	// yaml.v3 rejects it; normalize this one equivalent spelling for parsing,
	// while retaining the digest of the untouched downloaded bytes.
	decoder := yaml.NewDecoder(bytes.NewReader(normalizeSolidusEscape(data)))
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("definition YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("definition must contain exactly one YAML document")
	}
	remaining := MaxDefinitionNodes
	value, err := yamlValue(&root, "", 0, &remaining)
	if err != nil {
		return nil, err
	}
	schema, err := validationSchema()
	if err != nil {
		return nil, fmt.Errorf("compile v11 schema: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return nil, fmt.Errorf("v11 schema: %w", err)
	}
	var def Definition
	if err := root.Decode(&def); err != nil {
		return nil, fmt.Errorf("definition fields: %w", err)
	}
	if strings.TrimSpace(def.ID) == "" || strings.TrimSpace(def.Name) == "" || len(def.Links) == 0 {
		return nil, fmt.Errorf("definition requires id, name and links")
	}
	sum := sha256.Sum256(data)
	plan := &Plan{Definition: def, Digest: hex.EncodeToString(sum[:]), templates: map[string]*Template{}, Features: FeatureUsage{Filters: map[string]int{}, Templates: map[string]int{}, DateFormats: map[string]int{}}}
	jsonSearch := false
	for _, p := range def.Search.Paths {
		if p.Response.Type == "json" {
			jsonSearch = true
		}
	}
	if err := validateSyntax(&root, "", jsonSearch, plan.templates, &plan.Warnings, &plan.Features); err != nil {
		return nil, err
	}
	return plan, nil
}

func normalizeSolidusEscape(data []byte) []byte {
	out := make([]byte, 0, len(data))
	quoted, single, comment := false, false, false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if c == '\n' {
			comment = false
		}
		if comment {
			out = append(out, c)
			continue
		}
		if !quoted && !single && c == '#' {
			comment = true
		}
		if single {
			out = append(out, c)
			if c == '\'' {
				if i+1 < len(data) && data[i+1] == '\'' {
					i++
					out = append(out, data[i])
				} else {
					single = false
				}
			}
			continue
		}
		if quoted && c == '\\' && i+1 < len(data) {
			i++
			if data[i] != '/' {
				out = append(out, c)
			}
			out = append(out, data[i])
			continue
		}
		if c == '"' {
			quoted = !quoted
		}
		if !quoted && c == '\'' {
			single = true
		}
		out = append(out, c)
	}
	return out
}

// yamlValue converts only bounded JSON-compatible YAML. Aliases are rejected
// rather than expanded; repeated search fields are independently schema-checked
// while their order and occurrences remain intact in the typed model.
func yamlValue(n *yaml.Node, path string, depth int, remaining *int) (any, error) {
	*remaining--
	if depth > MaxDefinitionDepth || *remaining < 0 {
		return nil, Diagnostic{Path: path, Line: n.Line, Column: n.Column, Message: "definition nesting/node budget exceeded"}
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) != 1 {
			return nil, fmt.Errorf("definition must have one root")
		}
		return yamlValue(n.Content[0], path, depth+1, remaining)
	case yaml.MappingNode:
		result := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("line %d: mapping key must be scalar", key.Line)
			}
			name := key.Value
			childPath := path + "/" + name
			value, err := yamlValue(n.Content[i+1], childPath, depth+1, remaining)
			if err != nil {
				return nil, err
			}
			if _, found := result[name]; found && path != "/search/fields" {
				return nil, Diagnostic{Path: childPath, Line: key.Line, Column: key.Column, Message: "duplicate mapping key"}
			}
			if path == "/search/fields" {
				// Each occurrence receives the same SelectorBlock validation,
				// including occurrences hidden by JSON object's unique keys.
				schema, err := selectorSchema()
				if err != nil {
					return nil, err
				}
				if err := schema.Validate(value); err != nil {
					return nil, fmt.Errorf("%s: %w", childPath, err)
				}
			}
			result[name] = value
		}
		return result, nil
	case yaml.SequenceNode:
		items := make([]any, 0, len(n.Content))
		for i, child := range n.Content {
			value, err := yamlValue(child, path+"/"+strconv.Itoa(i), depth+1, remaining)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!bool":
			return n.Value == "true", nil
		case "!!null":
			return nil, nil
		case "!!int", "!!float":
			var value any
			if err := n.Decode(&value); err != nil {
				return nil, err
			}
			return value, nil
		}
	}
	return nil, Diagnostic{Path: path, Line: n.Line, Column: n.Column, Message: "unsupported YAML node or tag"}
}

var selectorSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := validationDocument()
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("https://moji.invalid/cardigann/v11", doc); err != nil {
		return nil, err
	}
	return c.Compile("https://moji.invalid/cardigann/v11#/definitions/SelectorBlock")
})

func validationDocument() (map[string]any, error) {
	var doc map[string]any
	if err := json.Unmarshal(schemaV11, &doc); err != nil {
		return nil, err
	}
	definitions := doc["definitions"].(map[string]any)
	// Upstream's v11 schema references SelectorBlock for download.selectors,
	// while its C# SelectorField explicitly supports usebeforeresponse.
	// Extend only this location, retaining all other SelectorBlock rules.
	encoded, err := json.Marshal(definitions["SelectorBlock"])
	if err != nil {
		return nil, err
	}
	var selector map[string]any
	if err := json.Unmarshal(encoded, &selector); err != nil {
		return nil, err
	}
	selector["properties"].(map[string]any)["usebeforeresponse"] = map[string]any{"type": "boolean"}
	definitions["DownloadSelectorBlock"] = selector
	download := definitions["DownloadBlock"].(map[string]any)["properties"].(map[string]any)
	download["selectors"].(map[string]any)["items"] = map[string]any{"$ref": "#/definitions/DownloadSelectorBlock"}
	return doc, nil
}
