package taskruntime

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/resourceselection"
)

func TestReportStoresOnlyCandidateSummary(t *testing.T) {
	index := 0
	r := SelectionReport{SelectedIndex: &index, Group: &SelectionReportGroup{ID: "g", Name: "Frozen group"}, Candidates: []resourceselection.Evaluation{{Status: "REJECTED", TierName: "", Reasons: []string{"EXCLUDED_RULE_MATCHED"}, ExcludedRules: []string{"Unwanted label"}, Facts: resourceselection.Facts{Title: "unique title", Size: 1024, Seeders: 0, ID: "private", Evidence: []resourceselection.Evidence{{Attribute: "private trace", Value: "YES", Source: "TITLE", Text: "duplicated title"}}}}}}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"rules":`, `"conditions":`, `"matches":`, `"evidence":`, `"preferred":`, `"ranks":`, `"unknownRules":`, `"position":`, `"site":`, "private trace", "duplicated title"} {
		if strings.Contains(string(raw), key) {
			t.Fatal("report retained removed detail", key)
		}
	}
	got, err := DecodeSelectionReport(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Selected != "unique title" || got.Candidates[0].Status != "REJECTED" || got.Candidates[0].ExcludedRules[0] != "Unwanted label" || got.Candidates[0].Facts.Seeders != 0 {
		t.Fatal("candidate summary changed")
	}
	if strings.Count(string(raw), "unique title") != 1 {
		t.Fatal("selected title duplicated")
	}
}

func TestReportDoesNotGuessSelectedCandidateFromDuplicateTitles(t *testing.T) {
	r := SelectionReport{Selected: "same", Candidates: []resourceselection.Evaluation{{Facts: resourceselection.Facts{Title: "same"}}, {Facts: resourceselection.Facts{Title: "same"}}}}
	data, err := json.Marshal(r)
	if err != nil || r.SelectedIndex != nil {
		t.Fatal("ambiguous historical title was treated as an identity", err)
	}
	saved, err := DecodeSelectionReport(string(data))
	if err != nil || saved.Selected != "same" || saved.SelectedIndex != nil {
		t.Fatal("historical title lost", err)
	}
}

func TestReportRejectsUnknownVersionsAndInvalidReferences(t *testing.T) {
	for _, raw := range []string{`{"version":1}`, `{"version":2}`, `{"version":4}`, `{"version":3,"selectedIndex":0,"candidates":[]}`, `{"version":3,"conditions":[],"candidates":[{"matches":[{"condition":0}]}]}`, `{"version":3,"conditions":[{}],"candidates":[{"matches":[{"condition":-1}]}]}`} {
		if _, err := DecodeSelectionReport(raw); err == nil {
			t.Fatal("invalid report accepted", raw)
		}
	}
}

func TestReportTrimmingKeepsSelectedIndexValid(t *testing.T) {
	round := newSelectionRound(selectionService(t).now, "TEST-123", resourceselection.Default())
	for i := 0; i < 25; i++ {
		item := &selectionItem{evaluation: resourceselection.Evaluation{Status: "REJECTED", Ranks: []int{i}, Facts: resourceselection.Facts{Title: strings.Repeat("x", i+1)}}}
		for j := 0; j < 32; j++ {
			item.evaluation.Reasons = append(item.evaluation.Reasons, strings.Repeat("reason", 350))
		}
		round.items = append(round.items, item)
	}
	round.selected = round.items[24]
	round.report.Selected = round.selected.evaluation.Facts.Title
	round.report.End = "USER_SELECTED"
	raw := round.json()
	decoded, err := DecodeSelectionReport(raw)
	if err != nil || raw == "" || len(raw) > 256<<10 || decoded.SelectedIndex == nil || decoded.Selected != round.report.Selected || decoded.OmittedCandidates == 0 {
		t.Fatal("trimming lost selection or its reference", err)
	}
}
