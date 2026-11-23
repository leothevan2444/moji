// Package resourceselection evaluates user-defined resource rules without I/O.
package resourceselection

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const Version = 7
const ParserVersion = 7

type Policy struct {
	matchers    map[string]*regexp.Regexp
	Version     int     `yaml:"version" json:"version"`
	ActiveGroup string  `yaml:"active_group" json:"activeGroup"`
	Rules       []Rule  `yaml:"rules" json:"rules"`
	Groups      []Group `yaml:"groups" json:"groups"`
}
type Rule struct {
	ID         string      `yaml:"id" json:"id"`
	Name       string      `yaml:"name" json:"name"`
	Conditions []Condition `yaml:"conditions" json:"conditions"`
}
type Condition struct {
	Scope   string `yaml:"scope" json:"scope"`
	Value   string `yaml:"value,omitempty" json:"value,omitempty"`
	Minimum string `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Maximum string `yaml:"maximum,omitempty" json:"maximum,omitempty"`
}
type Group struct {
	Fallback bool     `yaml:"fallback" json:"fallback"`
	ID       string   `yaml:"id" json:"id"`
	Name     string   `yaml:"name" json:"name"`
	Require  []string `yaml:"require" json:"require"`
	Exclude  []string `yaml:"exclude" json:"exclude"`
	Tiers    []Tier   `yaml:"tiers" json:"tiers"`
}
type Tier struct {
	ID    string   `yaml:"id" json:"id"`
	Name  string   `yaml:"name" json:"name"`
	Rules []string `yaml:"rules" json:"rules"`
}

func Default() Policy {
	return Policy{Version: Version, Rules: []Rule{}, Groups: []Group{}}
}
func (p Policy) Effective() Policy {
	if p.Version == 0 {
		return Default()
	}
	p.matchers = nil
	p.Rules = slices.Clone(p.Rules)
	for i := range p.Rules {
		p.Rules[i].Conditions = slices.Clone(p.Rules[i].Conditions)
	}
	p.Groups = slices.Clone(p.Groups)
	for i := range p.Groups {
		g := &p.Groups[i]
		g.Require = slices.Clone(g.Require)
		g.Exclude = slices.Clone(g.Exclude)
		g.Tiers = slices.Clone(g.Tiers)
		for j := range g.Tiers {
			g.Tiers[j].Rules = slices.Clone(g.Tiers[j].Rules)
		}
	}
	return p
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,79}$`)

func validName(s string) bool { return strings.TrimSpace(s) != "" && len(s) <= 160 }
func (p Policy) Validate() error {
	if p.Version != Version {
		return fmt.Errorf("unsupported resource selection version %d; back up and manually replace with version 7 positive-only rules and groups", p.Version)
	}
	if len(p.Rules) > 64 || len(p.Groups) > 16 {
		return fmt.Errorf("at most 64 rules and 16 groups are allowed")
	}
	ids := map[string]bool{}
	for _, r := range p.Rules {
		if !identifier.MatchString(r.ID) || ids[r.ID] || !validName(r.Name) {
			return fmt.Errorf("invalid or duplicate resource rule %q", r.Name)
		}
		ids[r.ID] = true
		for _, c := range r.Conditions {
			if !slices.Contains([]string{"TITLE", "SIZE", "SEEDERS"}, c.Scope) {
				return fmt.Errorf("invalid match field in %q", r.Name)
			}
			if c.Numeric() {
				if c.Value != "" || c.Minimum == "" && c.Maximum == "" {
					return fmt.Errorf("numeric rule %q needs at least one range bound and no text", r.Name)
				}
				min, max := int64(0), int64(0)
				var ok bool
				if c.Minimum != "" {
					min, ok = number(c.Minimum)
					if !ok {
						return fmt.Errorf("invalid minimum in %q", r.Name)
					}
				}
				if c.Maximum != "" {
					max, ok = number(c.Maximum)
					if !ok {
						return fmt.Errorf("invalid maximum in %q", r.Name)
					}
				}
				if c.Minimum != "" && c.Maximum != "" && min > max {
					return fmt.Errorf("minimum exceeds maximum in %q", r.Name)
				}
			} else {
				if strings.TrimSpace(c.Value) == "" || c.Minimum != "" || c.Maximum != "" {
					return fmt.Errorf("text rule %q needs an expression and no range", r.Name)
				}
				expression, regex := textExpression(c.Value)
				if len(expression) > 512 || regex && expression == "" {
					return fmt.Errorf("invalid expression length in %q", r.Name)
				}
				if regex {
					if _, err := regexp.Compile("(?i)" + expression); err != nil {
						return fmt.Errorf("invalid regular expression in %q: %w", r.Name, err)
					}
				}
			}
		}
	}

	directRefs := func(values []string) error {
		seen := map[string]bool{}
		for _, id := range values {
			if !ids[id] || seen[id] {
				return fmt.Errorf("unknown or duplicate rule reference %q", id)
			}
			seen[id] = true
		}
		return nil
	}
	groups := map[string]bool{}
	for _, g := range p.Groups {
		if !identifier.MatchString(g.ID) || groups[g.ID] || !validName(g.Name) || len(g.Tiers) > 16 {
			return fmt.Errorf("invalid or duplicate resource group %q", g.Name)
		}
		groups[g.ID] = true
		if err := directRefs(g.Require); err != nil {
			return err
		}
		if err := directRefs(g.Exclude); err != nil {
			return err
		}
		for _, id := range g.Require {
			if slices.Contains(g.Exclude, id) {
				return fmt.Errorf("rule %q cannot be used for both admission and exclusion in group %q", id, g.Name)
			}
		}
		tiers := map[string]bool{}
		for _, t := range g.Tiers {
			if !identifier.MatchString(t.ID) || tiers[t.ID] || !validName(t.Name) {
				return fmt.Errorf("invalid tier in %q", g.Name)
			}
			tiers[t.ID] = true
			for _, ref := range t.Rules {
				if slices.Contains(g.Exclude, ref) {
					return fmt.Errorf("rule %q cannot be used for both priority and exclusion in group %q", ref, g.Name)
				}
			}
			if len(t.Rules) > 32 {
				return fmt.Errorf("at most 32 rule references are allowed")
			}
			if err := directRefs(t.Rules); err != nil {
				return err
			}
		}
	}
	if p.ActiveGroup != "" && !groups[p.ActiveGroup] {
		return fmt.Errorf("unknown active resource group %q", p.ActiveGroup)
	}
	if len(p.Groups) > 0 && p.ActiveGroup == "" {
		return fmt.Errorf("select an active resource group")
	}
	data, err := json.Marshal(p)
	if err != nil || len(data) > 192<<10 {
		return fmt.Errorf("resource policy exceeds 192 KiB serialized budget")
	}
	return nil
}

// WithGroup selects one group for a preview without mutating the saved active group.
func (p Policy) WithGroup(id string) (Policy, error) {
	p = p.Effective()
	p.ActiveGroup = id
	if err := p.Validate(); err != nil {
		return Policy{}, err
	}
	return p, nil
}
func Prepare(p Policy) Policy {
	if p.matchers != nil {
		return p
	}
	p.matchers = map[string]*regexp.Regexp{}
	for _, r := range p.Rules {
		for _, c := range r.Conditions {
			expression, regex := textExpression(c.Value)
			if regex {
				if _, ok := p.matchers[c.Value]; !ok {
					if re, err := regexp.Compile("(?i)" + expression); err == nil {
						p.matchers[c.Value] = re
					}
				}
			}
		}
	}

	return p
}

// UnmarshalJSON preserves the selected group in historical report and preview
// policy snapshots. YAML configuration deliberately accepts only active_group.
func (p *Policy) UnmarshalJSON(data []byte) error {
	var err error
	data, err = historicalTierIDs(data)
	if err != nil {
		return err
	}
	type payload Policy
	var current payload
	if err := json.Unmarshal(data, &current); err != nil {
		return err
	}
	var keys struct {
		Active *string `json:"activeGroup"`
		Legacy *string `json:"defaultGroup"`
	}
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	if keys.Active != nil && keys.Legacy != nil && *keys.Active != *keys.Legacy {
		return fmt.Errorf("conflicting policy group snapshot fields")
	}
	if keys.Active == nil && keys.Legacy != nil {
		current.ActiveGroup = *keys.Legacy
	}
	*p = Policy(current)
	return nil
}

func (r Condition) Numeric() bool { return r.Scope == "SIZE" || r.Scope == "SEEDERS" }
func textExpression(value string) (string, bool) {
	if len(value) >= 2 && strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") {
		return value[1 : len(value)-1], true
	}
	return value, false
}
