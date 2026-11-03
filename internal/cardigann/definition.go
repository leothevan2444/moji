// Package cardigann interprets the Prowlarr Cardigann v11 declarative format.
// It does not own application configuration, task selection, or persistence.
package cardigann

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Scalar preserves rule spelling (including numeric category identifiers).
// Only the scalar values supported by the rule format are accepted.
type Scalar string

func (s *Scalar) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode || (n.Tag != "!!str" && n.Tag != "!!int" && n.Tag != "!!float" && n.Tag != "!!bool") {
		return fmt.Errorf("line %d: expected string, number or boolean", n.Line)
	}
	*s = Scalar(n.Value)
	return nil
}

type Definition struct {
	ID              string       `yaml:"id"`
	Replaces        []string     `yaml:"replaces"`
	Name            string       `yaml:"name"`
	Description     string       `yaml:"description"`
	Language        string       `yaml:"language"`
	Type            string       `yaml:"type"`
	Encoding        string       `yaml:"encoding"`
	FollowRedirect  bool         `yaml:"followredirect"`
	TestLinkTorrent *bool        `yaml:"testlinktorrent"`
	RequestDelay    float64      `yaml:"requestDelay"`
	Links           []string     `yaml:"links"`
	LegacyLinks     []string     `yaml:"legacylinks"`
	Certificates    []string     `yaml:"certificates"`
	Caps            Capabilities `yaml:"caps"`
	Settings        []Setting    `yaml:"settings"`
	Login           *Login       `yaml:"login"`
	Search          Search       `yaml:"search"`
	Download        *Download    `yaml:"download"`
}

type Capabilities struct {
	Categories        ScalarFields        `yaml:"categories"`
	CategoryMappings  []CategoryMapping   `yaml:"categorymappings"`
	Modes             map[string][]string `yaml:"modes"`
	AllowRawSearch    bool                `yaml:"allowrawsearch"`
	AllowTVSearchIMDB bool                `yaml:"allowtvsearchimdb"`
}

type CategoryMapping struct {
	ID      Scalar `yaml:"id"`
	Cat     string `yaml:"cat"`
	Desc    string `yaml:"desc"`
	Default bool   `yaml:"default"`
}

type Setting struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	Label    string            `yaml:"label"`
	Default  Scalar            `yaml:"default"`
	Defaults []Scalar          `yaml:"defaults"`
	Options  map[string]Scalar `yaml:"options"`
}

type Selector struct {
	Selector  string   `yaml:"selector"`
	Attribute string   `yaml:"attribute"`
	Optional  bool     `yaml:"optional"`
	Default   *Scalar  `yaml:"default"`
	Text      *Scalar  `yaml:"text"`
	Remove    string   `yaml:"remove"`
	Case      Cases    `yaml:"case"`
	Filters   []Filter `yaml:"filters"`
}

// Fields and Cases must retain YAML order. Fields additionally permits repeated
// names: a later field may append to a previously computed result.
type Field struct {
	Name     string
	Selector Selector
}
type Fields []Field
type Case struct {
	Selector string
	Value    Scalar
}
type Cases []Case

func (f *Fields) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: fields must be a mapping", n.Line)
	}
	for i := 0; i < len(n.Content); i += 2 {
		var s Selector
		if err := n.Content[i+1].Decode(&s); err != nil {
			return err
		}
		*f = append(*f, Field{Name: n.Content[i].Value, Selector: s})
	}
	return nil
}

func (c *Cases) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: case must be a mapping", n.Line)
	}
	for i := 0; i < len(n.Content); i += 2 {
		var v Scalar
		if err := n.Content[i+1].Decode(&v); err != nil {
			return err
		}
		*c = append(*c, Case{Selector: n.Content[i].Value, Value: v})
	}
	return nil
}

type Filter struct {
	Name string    `yaml:"name"`
	Args Arguments `yaml:"args"`
}
type Arguments []Scalar

func (a *Arguments) UnmarshalYAML(n *yaml.Node) error {
	nodes := []*yaml.Node{n}
	if n.Kind == yaml.SequenceNode {
		nodes = n.Content
	}
	for _, node := range nodes {
		var s Scalar
		if err := node.Decode(&s); err != nil {
			return err
		}
		*a = append(*a, s)
	}
	return nil
}

type RequestBlock struct {
	Path           string       `yaml:"path"`
	Method         string       `yaml:"method"`
	Inputs         ScalarFields `yaml:"inputs"`
	QuerySeparator string       `yaml:"queryseparator"`
}
type SearchPath struct {
	RequestBlock   `yaml:",inline"`
	Categories     []Scalar `yaml:"categories"`
	InheritInputs  *bool    `yaml:"inheritinputs"`
	FollowRedirect bool     `yaml:"followredirect"`
	Response       Response `yaml:"response"`
}
type Response struct {
	Type             string `yaml:"type"`
	NoResultsMessage string `yaml:"noResultsMessage"`
}
type Rows struct {
	Selector                        `yaml:",inline"`
	After                           int       `yaml:"after"`
	DateHeaders                     *Selector `yaml:"dateheaders"`
	Count                           *Selector `yaml:"count"`
	Multiple                        bool      `yaml:"multiple"`
	MissingAttributeEqualsNoResults bool      `yaml:"missingAttributeEqualsNoResults"`
}
type Search struct {
	Path                 string              `yaml:"path"`
	Paths                []SearchPath        `yaml:"paths"`
	AllowEmptyInputs     bool                `yaml:"allowEmptyInputs"`
	Inputs               ScalarFields        `yaml:"inputs"`
	Headers              map[string][]string `yaml:"headers"`
	KeywordsFilters      []Filter            `yaml:"keywordsfilters"`
	Error                []ErrorBlock        `yaml:"error"`
	PreprocessingFilters []Filter            `yaml:"preprocessingfilters"`
	Rows                 Rows                `yaml:"rows"`
	Fields               Fields              `yaml:"fields"`
}
type ErrorBlock struct {
	Path     string   `yaml:"path"`
	Selector string   `yaml:"selector"`
	Message  Selector `yaml:"message"`
}
type PageTest struct {
	Path     string `yaml:"path"`
	Selector string `yaml:"selector"`
}
type Captcha struct {
	Type     string `yaml:"type"`
	Selector string `yaml:"selector"`
	Input    string `yaml:"input"`
}
type Login struct {
	Path              string              `yaml:"path"`
	SubmitPath        string              `yaml:"submitpath"`
	Cookies           []string            `yaml:"cookies"`
	Method            string              `yaml:"method"`
	Form              string              `yaml:"form"`
	Selectors         bool                `yaml:"selectors"`
	Inputs            ScalarFields        `yaml:"inputs"`
	SelectorInputs    Fields              `yaml:"selectorinputs"`
	GetSelectorInputs Fields              `yaml:"getselectorinputs"`
	Error             []ErrorBlock        `yaml:"error"`
	Test              *PageTest           `yaml:"test"`
	Captcha           *Captcha            `yaml:"captcha"`
	Headers           map[string][]string `yaml:"headers"`
}
type DownloadSelector struct {
	Selector          `yaml:",inline"`
	UseBeforeResponse bool `yaml:"usebeforeresponse"`
}
type Before struct {
	RequestBlock `yaml:",inline"`
	PathSelector *DownloadSelector `yaml:"pathselector"`
}
type InfoHash struct {
	Hash              DownloadSelector `yaml:"hash"`
	Title             DownloadSelector `yaml:"title"`
	UseBeforeResponse bool             `yaml:"usebeforeresponse"`
}
type Download struct {
	Method    string              `yaml:"method"`
	Before    *Before             `yaml:"before"`
	Selectors []DownloadSelector  `yaml:"selectors"`
	InfoHash  *InfoHash           `yaml:"infohash"`
	Headers   map[string][]string `yaml:"headers"`
}
