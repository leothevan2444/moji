package resourceselection

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Truth string

const (
	Unknown Truth = "UNKNOWN"
	Yes     Truth = "YES"
	No      Truth = "NO"
)

type Evidence struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
	Source    string `json:"source"`
	Text      string `json:"text"`
}
type Facts struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Size     int64      `json:"size"`
	Seeders  int        `json:"seeders"`
	Site     int        `json:"site"`
	Position int        `json:"position"`
	Evidence []Evidence `json:"evidence"`
}
type Evaluation struct {
	EvidenceOmitted int      `json:"evidenceOmitted"`
	GroupName       string   `json:"groupName"`
	TierName        string   `json:"tierName"`
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	Preferred       bool     `json:"preferred"`
	Ranks           []int    `json:"ranks"`
	Reasons         []string `json:"reasons"`
	UnknownRules    []string `json:"unknownRules"`
	Facts           Facts    `json:"facts"`
}

func truncate(v string, n int) string {
	r := []rune(v)
	if len(r) > n {
		return string(r[:n])
	}
	return v
}
func number(v string) (int64, bool) {
	n, e := strconv.ParseInt(v, 10, 64)
	return n, e == nil && n >= 0
}
func matches(r Condition, f Facts, matchers map[string]*regexp.Regexp) (Truth, string) {
	if r.Numeric() {
		value := f.Size
		if r.Scope == "SEEDERS" {
			value = int64(f.Seeders)
		}
		if value < 0 {
			return Unknown, ""
		}
		text := strconv.FormatInt(value, 10)
		if r.Minimum != "" {
			minimum, ok := number(r.Minimum)
			if !ok {
				return Unknown, ""
			}
			if value < minimum {
				return No, text
			}
		}
		if r.Maximum != "" {
			maximum, ok := number(r.Maximum)
			if !ok {
				return Unknown, ""
			}
			if value > maximum {
				return No, text
			}
		}
		return Yes, text
	}
	if strings.TrimSpace(f.Title) == "" {
		return Unknown, ""
	}
	_, regex := textExpression(r.Value)
	if regex {
		matcher := matchers[r.Value]
		if matcher == nil {
			return Unknown, ""
		}
		return truth(matcher.MatchString(f.Title)), f.Title
	}
	return truth(strings.Contains(strings.ToLower(f.Title), strings.ToLower(r.Value))), f.Title
}
func truth(v bool) Truth {
	if v {
		return Yes
	}
	return No
}
func Evaluate(p Policy, f Facts) Evaluation {
	p = Prepare(p)
	e := Evaluation{ID: f.ID, Status: "ACCEPTABLE", Preferred: true, Ranks: []int{0}, Reasons: []string{}, UnknownRules: []string{}, Facts: f}
	e.Facts.Evidence = []Evidence{}
	if p.ActiveGroup == "" {
		e.Reasons = append(e.Reasons, "NO_GROUP_FILTER")
		return e
	}
	var group *Group
	for i := range p.Groups {
		if p.Groups[i].ID == p.ActiveGroup {
			group = &p.Groups[i]
			break
		}
	}
	if group == nil {
		e.Status = "REJECTED"
		e.Preferred = false
		e.Reasons = append(e.Reasons, "GROUP_NOT_FOUND")
		return e
	}
	e.GroupName = group.Name
	rules := map[string]Rule{}
	for _, r := range p.Rules {
		rules[r.ID] = r
	}
	cache := map[string]Truth{}
	ref := func(id string) Truth {
		v, ok := cache[id]
		if !ok {
			rule, exists := rules[id]
			if !exists {
				return Unknown
			}
			v = No
			unknown := false
			for _, condition := range rule.Conditions {
				result, text := matches(condition, f, p.matchers)
				expression := condition.Value
				if condition.Numeric() {
					expression = condition.Minimum + "…" + condition.Maximum
				}
				if len(e.Facts.Evidence) < 32 {
					e.Facts.Evidence = append(e.Facts.Evidence, Evidence{Attribute: rule.Name + ": " + expression, Value: string(result), Source: condition.Scope, Text: truncate(text, 240)})
				} else {
					e.EvidenceOmitted++
				}
				if result == Yes {
					v = Yes
					break
				}
				unknown = unknown || result == Unknown
			}
			if v != Yes && unknown {
				v = Unknown
			}
			cache[id] = v

		}

		return v
	}
	combine := func(refs []string, mode string, empty Truth) Truth {
		if len(refs) == 0 {
			return empty
		}
		unknown := []string{}
		for _, r := range refs {
			v := ref(r)
			if mode == "ALL" && v == No {
				return No
			}
			if mode == "ANY" && v == Yes {
				return Yes
			}
			if v == Unknown {
				if rule, ok := rules[r]; ok {
					unknown = append(unknown, rule.Name)
				}
			}
		}
		if len(unknown) > 0 {
			for _, name := range unknown {
				if !slices.Contains(e.UnknownRules, name) {
					e.UnknownRules = append(e.UnknownRules, name)
				}
			}
			return Unknown
		}
		return truth(mode == "ALL")
	}
	blockedUnknown := false
	switch combine(group.Require, "ANY", Yes) {
	case No:
		e.Status = "REJECTED"
		e.Reasons = append(e.Reasons, "REQUIRED_RULE_FAILED")
	case Unknown:
		blockedUnknown = true
	}
	switch combine(group.Exclude, "ANY", No) {
	case Yes:
		e.Status = "REJECTED"
		e.Reasons = append(e.Reasons, "EXCLUDED_RULE_MATCHED")
	case Unknown:
		blockedUnknown = true
	}
	if e.Status == "REJECTED" {
		e.Preferred = false
		return e
	}
	earlierUnknown := false
	for i, t := range group.Tiers {
		v := combine(t.Rules, "ALL", No)
		if v == Unknown {
			earlierUnknown = true
			continue
		}
		if v == Yes {
			e.Ranks = []int{i}
			e.TierName = t.Name
			e.Preferred = i == 0 && !blockedUnknown && !earlierUnknown
			e.Reasons = append(e.Reasons, "TIER_MATCHED")
			if blockedUnknown || earlierUnknown {
				e.Status = "INSUFFICIENT_DATA"
				e.Preferred = false
			}
			return e
		}
	}
	e.Preferred = false
	if group.Fallback {
		e.Ranks = []int{len(group.Tiers)}
		e.Reasons = append(e.Reasons, "FALLBACK_ALLOWED")
		if blockedUnknown || earlierUnknown {
			e.Status = "INSUFFICIENT_DATA"
		}
	} else if earlierUnknown {
		e.Status = "INSUFFICIENT_DATA"
	} else {
		e.Status = "REJECTED"
		e.Reasons = append(e.Reasons, "NO_TIER_MATCHED")
	}

	return e
}

// Compare uses ascending ranks and a stable source/position tie-breaker.
func Compare(a, b Evaluation) int {
	status := func(v string) int {
		switch v {
		case "ACCEPTABLE":
			return 0
		case "INSUFFICIENT_DATA":
			return 1
		default:
			return 2
		}
	}
	if c := cmp.Compare(status(a.Status), status(b.Status)); c != 0 {
		return c
	}
	if c := slices.Compare(a.Ranks, b.Ranks); c != 0 {
		return c
	}
	seedRank := func(n int) int {
		if n > 0 {
			return 0
		}
		if n < 0 {
			return 1
		}
		return 2
	}
	if c := cmp.Compare(seedRank(a.Facts.Seeders), seedRank(b.Facts.Seeders)); c != 0 {
		return c
	}
	if a.Facts.Seeders > 0 && b.Facts.Seeders > 0 {
		if c := cmp.Compare(b.Facts.Seeders, a.Facts.Seeders); c != 0 {
			return c
		}
	}
	if c := cmp.Compare(a.Facts.Site, b.Facts.Site); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Facts.Position, b.Facts.Position); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}
func Rank(p Policy, facts []Facts) []Evaluation {
	p = Prepare(p)
	out := make([]Evaluation, 0, len(facts))
	for _, f := range facts {
		out = append(out, Evaluate(p, f))
	}
	slices.SortStableFunc(out, Compare)
	return out
}
