package resourceselection

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func testPolicy() Policy {
	p := Default()
	p.Rules = []Rule{{ID: "label", Name: "User label", Conditions: []Condition{{Scope: "TITLE", Value: "preferred"}}}, {ID: "bad", Name: "Blocked label", Conditions: []Condition{{Scope: "TITLE", Value: "blocked"}}}}
	p.ActiveGroup = "main"
	p.Groups = []Group{{ID: "main", Name: "Main", Fallback: true, Exclude: []string{"bad"}, Tiers: []Tier{{ID: "first", Name: "First", Rules: []string{"label"}}}}}
	return p
}
func TestLiteralAndDelimitedRegexAreDeterministic(t *testing.T) {
	p := testPolicy()
	p.Rules[0].Conditions[0].Value = "a.b"
	if Evaluate(p, Facts{Title: "A.B"}).Preferred != true || Evaluate(p, Facts{Title: "axb"}).Preferred {
		t.Fatal("literal punctuation interpreted as regex")
	}
	p.Rules[0].Conditions[0].Value = "/a.b/"
	if !Evaluate(p, Facts{Title: "axb"}).Preferred {
		t.Fatal("delimited regex failed")
	}
	for _, v := range []string{"/[/", "//"} {
		p.Rules[0].Conditions[0].Value = v
		if p.Validate() == nil {
			t.Fatal("invalid regex accepted", v)
		}
	}
	p.Rules[0].Conditions[0].Value = "/folder"
	if !Evaluate(p, Facts{Title: "/folder/video.mp4"}).Preferred {
		t.Fatal("undelimited path not literal")
	}
}
func TestInclusiveNumericRangesAndUnknown(t *testing.T) {
	p := testPolicy()
	p.Rules[0] = Rule{ID: "label", Name: "Range", Conditions: []Condition{{Scope: "SIZE", Minimum: "5", Maximum: "10"}}}
	for _, v := range []int64{4, 5, 7, 10, 11, -1} {
		e := Evaluate(p, Facts{Title: "film", Size: v})
		if e.Preferred != (v >= 5 && v <= 10) {
			t.Fatal("range boundary", v, e)
		}
		if v < 0 && e.Status != "INSUFFICIENT_DATA" {
			t.Fatal("unknown range passed")
		}
	}

}
func TestFixedGroupCombinationsAndFallback(t *testing.T) {
	p := testPolicy()
	p.Rules = append(p.Rules, Rule{ID: "files", Name: "Files", Conditions: []Condition{{Scope: "SEEDERS", Minimum: "1"}}})
	g := &p.Groups[0]
	g.Require = []string{"files", "label"}
	if !Evaluate(p, Facts{Title: "preferred", Seeders: -1}).Preferred {
		t.Fatal("any admission did not resolve unknown")
	}
	g.Require = []string{"files"}
	if Evaluate(p, Facts{Title: "preferred", Seeders: -1}).Status != "INSUFFICIENT_DATA" {
		t.Fatal("unknown admission passed")
	}
	g.Require = nil
	g.Exclude = []string{"files", "bad"}
	if Evaluate(p, Facts{Title: "preferred", Seeders: -1}).Status != "INSUFFICIENT_DATA" {
		t.Fatal("unknown exclusion passed")
	}
	if Evaluate(p, Facts{Title: "preferred blocked"}).Status != "REJECTED" {
		t.Fatal("any exclusion did not reject")
	}
	g.Exclude = nil
	g.Tiers[0].Rules = append(g.Tiers[0].Rules, "files")
	if Evaluate(p, Facts{Title: "preferred", Seeders: -1}).Status != "INSUFFICIENT_DATA" {
		t.Fatal("tier ALL passed unknown")
	}
	f := Facts{Title: "preferred", Seeders: 0}
	e := Evaluate(p, f)
	if e.Status != "ACCEPTABLE" || e.Preferred || e.Ranks[0] != 1 || !slices.Contains(e.Reasons, "FALLBACK_ALLOWED") || e.TierName != "" {
		t.Fatal("fallback confused with priority", e)
	}
	f.Seeders = 1
	if !Evaluate(p, f).Preferred {
		t.Fatal("all tier conditions should match")
	}
	g.Fallback = false
	f.Seeders = 0
	if Evaluate(p, f).Status != "REJECTED" {
		t.Fatal("fallback switch off ignored")
	}
	g.Require = []string{"files"}
	f.Seeders = -1
	f.Title = "ordinary"
	if Evaluate(p, f).Status != "REJECTED" {
		t.Fatal("irrelevant unknown caused inspection although no tier could match")
	}
}
func TestFallbackWithoutTiersAndPreFilterSafeguards(t *testing.T) {
	p := testPolicy()
	g := &p.Groups[0]
	g.Tiers = nil
	g.Fallback = true
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	e := Evaluate(p, Facts{Title: "ordinary"})
	if e.Status != "ACCEPTABLE" || e.Preferred || e.Ranks[0] != 0 {
		t.Fatal("fallback only group", e)
	}
	if Evaluate(p, Facts{Title: "blocked"}).Status != "REJECTED" {
		t.Fatal("fallback bypassed exclusion")
	}
	g.Require = []string{"label"}
	if Evaluate(p, Facts{Title: "ordinary"}).Status != "REJECTED" {
		t.Fatal("fallback bypassed admission")
	}
	if Evaluate(p, Facts{}).Status != "INSUFFICIENT_DATA" {
		t.Fatal("fallback bypassed unknown")
	}
	g.Fallback = false
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if Evaluate(p, Facts{Title: "preferred", Seeders: -1}).Status != "REJECTED" {
		t.Fatal("no tiers and no fallback selected a resource")
	}
}
func TestReusableRulesGroupSelectionAndDeepCopy(t *testing.T) {
	p := testPolicy()
	p.Groups = append(p.Groups, Group{ID: "alternate", Name: "Alternate", Fallback: true, Tiers: []Tier{{ID: "first", Name: "Blocked label", Rules: []string{"bad"}}}})
	q, err := p.WithGroup("alternate")
	if err != nil {
		t.Fatal(err)
	}
	if !Evaluate(q, Facts{Title: "blocked"}).Preferred || Evaluate(p, Facts{Title: "blocked"}).Preferred || p.ActiveGroup != "main" {
		t.Fatal("shared rule changed")
	}
	q.Rules[0].Conditions[0].Value = "edited"
	if p.Rules[0].Conditions[0].Value != "preferred" {
		t.Fatal("policy alias")
	}

}
func TestValidationAndPolicyBudget(t *testing.T) {
	for _, edit := range []func(*Policy){func(p *Policy) { p.Version = 2 }, func(p *Policy) { p.Rules[0].Conditions[0].Scope = "MODIFIED" }, func(p *Policy) { p.Rules[0].Conditions[0].Value = "/[/" }, func(p *Policy) { p.Rules = p.Rules[1:] }, func(p *Policy) {
		p.Rules[0] = Rule{ID: "label", Name: "Range", Conditions: []Condition{{Scope: "SEEDERS", Minimum: "11", Maximum: "10"}}}
	}, func(p *Policy) { p.Rules[0].Conditions[0].Minimum = "1" }} {
		p := testPolicy()
		edit(&p)
		if p.Validate() == nil {
			t.Fatal("invalid policy admitted")
		}
	}
	p := Default()
	for i := 0; i < 64; i++ {
		p.Rules = append(p.Rules, Rule{ID: fmt.Sprint(i), Name: strings.Repeat("x", 160), Conditions: []Condition{{Scope: "TITLE", Value: strings.Repeat("<", 512)}}})
	}
	if p.Validate() == nil {
		t.Fatal("JSON escaped policy budget ignored")
	}
}
func TestStableOrderingAndNoBuiltInMediaJudgments(t *testing.T) {
	p := Default()
	rng := rand.New(rand.NewSource(3))
	facts := make([]Facts, 90)
	for i := range facts {
		facts[i] = Facts{ID: fmt.Sprint(i), Title: []string{"ordinary", "无码破解 中字 4k"}[rng.Intn(2)], Seeders: rng.Intn(20) - 1, Site: rng.Intn(5), Position: i}
	}
	a := Rank(p, facts)
	if !reflect.DeepEqual(a, Rank(p, facts)) {
		t.Fatal("non deterministic")
	}
	for i := range a {
		if !a[i].Preferred {
			t.Fatal("implicit media preference")
		}
		for j := i; j < len(a); j++ {
			if Compare(a[i], a[j]) > 0 {
				t.Fatal("nontransitive")
			}
		}
	}
}
func TestLegacyPolicyVersionIsNotReinterpreted(t *testing.T) {
	var p Policy
	if err := json.Unmarshal([]byte(`{"version":2,"defaultGroup":"main","rules":[{"id":"old","mode":"ANY","conditions":[{"scope":"TITLE","operator":"CONTAINS","value":"x"}]}]}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Version != 2 || p.ActiveGroup != "main" || p.Validate() == nil {
		t.Fatal("legacy snapshot became executable")
	}
	data, _ := json.Marshal(testPolicy())
	if !strings.Contains(string(data), "conditions") || strings.Contains(string(data), "operator") {
		t.Fatal("implicit-any snapshots contain retired fields")
	}
}
func BenchmarkRank1000Rules(b *testing.B) {
	p := testPolicy()
	facts := make([]Facts, 1000)
	for i := range facts {
		facts[i] = Facts{ID: fmt.Sprint(i), Title: "preferred", Position: i}
	}
	b.ReportAllocs()
	for b.Loop() {
		Rank(p, facts)
	}
}

func TestRuleConditionsAlwaysUseAnyIncludingUnknown(t *testing.T) {
	for _, title := range []Truth{Yes, No, Unknown} {
		for _, size := range []Truth{Yes, No, Unknown} {
			for _, seeders := range []Truth{Yes, No, Unknown} {
				p := testPolicy()
				p.Rules = []Rule{{ID: "label", Name: "Mixed feature", Conditions: []Condition{{Scope: "TITLE", Value: "hit"}, {Scope: "SIZE", Minimum: "1", Maximum: "1"}, {Scope: "SEEDERS", Minimum: "1", Maximum: "1"}}}}
				p.Groups[0].Exclude = nil
				p.Groups[0].Fallback = false
				f := Facts{Title: "ordinary", Size: 0, Seeders: 0}
				if title == Yes {
					f.Title = "hit"
				}
				if title == Unknown {
					f.Title = ""
				}
				if size == Yes {
					f.Size = 1
				}
				if size == Unknown {
					f.Size = -1
				}
				if seeders == Yes {
					f.Seeders = 1
				}
				if seeders == Unknown {
					f.Seeders = -1
				}
				expected := No
				for _, v := range []Truth{title, size, seeders} {
					if v == Yes {
						expected = Yes
						break
					}
					if v == Unknown {
						expected = Unknown
					}
				}
				status := "REJECTED"
				if expected == Yes {
					status = "ACCEPTABLE"
				}
				if expected == Unknown {
					status = "INSUFFICIENT_DATA"
				}
				got := Evaluate(p, f)
				if got.Status != status || got.Preferred != (expected == Yes) {
					t.Fatalf("truths %s/%s/%s: %s", title, size, seeders, got.Status)
				}
			}
		}
	}
}
func TestConditionsHaveNoCountLimitAndEvidenceRemainsBounded(t *testing.T) {
	p := testPolicy()
	p.Rules[0].Conditions = nil
	p.Groups[0].Exclude = nil
	for i := 0; i < 300; i++ {
		p.Rules[0].Conditions = append(p.Rules[0].Conditions, Condition{Scope: "TITLE", Value: fmt.Sprintf("absent-%d", i)})
	}
	p.Rules[0].Conditions = append(p.Rules[0].Conditions, Condition{Scope: "TITLE", Value: "preferred"})
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	got := Evaluate(p, Facts{Title: "preferred", Seeders: -1})
	if !got.Preferred || len(got.Facts.Evidence) != 32 || got.EvidenceOmitted != 269 {
		t.Fatalf("large rule lost meaning or exceeded evidence budget: %#v", got)
	}
	p.Rules[0].Conditions = nil
	if err := p.Validate(); err != nil {
		t.Fatal("empty rule rejected", err)
	}
	if got := Evaluate(p, Facts{Title: "preferred", Seeders: 1}); got.Preferred {
		t.Fatal("empty rule matched", got)
	}
}

func TestAdmissionAndExclusionCannotReferenceTheSameRule(t *testing.T) {
	p := testPolicy()
	p.Groups[0].Require = []string{"label"}
	p.Groups[0].Exclude = append(p.Groups[0].Exclude, "label")
	if err := p.Validate(); err == nil {
		t.Fatal("contradictory pre-filter accepted")
	}
	p.Groups[0].Require = nil
	p.Groups[0].Tiers = nil
	if err := p.Validate(); err != nil {
		t.Fatal("valid exclusion incorrectly prohibited", err)
	}
}

func TestRuleLibraryOrderingDoesNotChangeSelection(t *testing.T) {
	p := testPolicy()
	facts := []Facts{{ID: "one", Title: "preferred", Seeders: 1}, {ID: "two", Title: "blocked", Seeders: 10}, {ID: "three", Title: "ordinary", Seeders: 2}}
	before := Rank(p, facts)
	slices.Reverse(p.Rules)
	if !reflect.DeepEqual(before, Rank(p, facts)) {
		t.Fatal("library organization changed resource ranking")
	}
}

func TestPriorityAndExclusionCannotReferenceTheSameRule(t *testing.T) {
	p := testPolicy()
	p.Groups[0].Exclude = append(p.Groups[0].Exclude, "label")
	if err := p.Validate(); err == nil {
		t.Fatal("priority exclusion overlap accepted")
	}
}

func TestEmptyPriorityTiersAreValidAndNeverMatch(t *testing.T) {
	p := testPolicy()
	first := p.Groups[0].Tiers[0]
	p.Groups[0].Tiers = append([]Tier{{ID: "empty", Name: "Empty", Rules: []string{}}}, first)
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	got := Evaluate(p, Facts{Title: "preferred", Seeders: 1})
	if got.TierName != first.Name || got.Ranks[0] != 1 {
		t.Fatal("empty tier took priority", got)
	}
	p.Groups[0].Tiers = p.Groups[0].Tiers[:1]
	p.Groups[0].Fallback = false
	if got := Evaluate(p, Facts{Title: "preferred", Seeders: 1}); got.Status != "REJECTED" {
		t.Fatal("empty tier matched", got)
	}
	p.Groups[0].Fallback = true
	if got := Evaluate(p, Facts{Title: "preferred", Seeders: 1}); got.Status != "ACCEPTABLE" || got.Preferred {
		t.Fatal("empty tier changed fallback", got)
	}
}

func TestCurrentPolicyRejectsObjectReferencesAndHistoricalPolicyStaysVersioned(t *testing.T) {
	raw := `{"version":7,"groups":[{"id":"main","tiers":[{"id":"first","rules":[{"ruleID":"label","negate":true}]}]}]}`
	var p Policy
	if err := json.Unmarshal([]byte(raw), &p); err == nil {
		t.Fatal("v7 accepted removed relation")
	}
	if err := json.Unmarshal([]byte(strings.Replace(raw, `"version":7`, `"version":6`, 1)), &p); err != nil {
		t.Fatal("historical report unreadable", err)
	}
	if p.Version != 6 || p.Groups[0].Tiers[0].Rules[0] != "label" {
		t.Fatal("historical context lost")
	}
	if p.Validate() == nil {
		t.Fatal("historical policy admitted for new selection")
	}
	current := testPolicy()
	current.Groups[0].Tiers[0].Rules = []string{"label", "label"}
	if current.Validate() == nil {
		t.Fatal("duplicate positive tier reference accepted")
	}
}
