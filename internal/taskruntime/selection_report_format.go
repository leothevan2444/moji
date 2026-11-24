package taskruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
)

const SelectionReportVersion = 3

type SelectionReportGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SelectionReport stores candidate summaries, without condition-level traces.
type SelectionReport struct {
	ID                                    string
	StartedAt, FinishedAt                 time.Time
	OmittedCandidates, OmittedSites       int
	Version, PolicyVersion, ParserVersion int
	Group                                 *SelectionReportGroup
	Sites                                 []indexer.SiteReport
	Partial                               bool
	End, Selected                         string
	SelectedIndex                         *int
	Candidates                            []resourceselection.Evaluation
	Counts                                map[string]int
}
type selectionReport = SelectionReport

type reportHeader struct {
	ID                string                `json:"id"`
	StartedAt         time.Time             `json:"startedAt"`
	FinishedAt        time.Time             `json:"finishedAt"`
	OmittedCandidates int                   `json:"omittedCandidates"`
	OmittedSites      int                   `json:"omittedSites"`
	Version           int                   `json:"version"`
	PolicyVersion     int                   `json:"policyVersion"`
	ParserVersion     int                   `json:"parserVersion"`
	Group             *SelectionReportGroup `json:"group,omitempty"`
	Sites             []indexer.SiteReport  `json:"sites"`
	Partial           bool                  `json:"partial"`
	End               string                `json:"end"`
	SelectedIndex     *int                  `json:"selectedIndex,omitempty"`
	SelectedTitle     string                `json:"selectedTitle,omitempty"` // Historical selections without an unambiguous retained candidate.
	Counts            map[string]int        `json:"counts"`
}
type reportFacts struct {
	Title   string `json:"title"`
	Size    int64  `json:"size"`
	Seeders int    `json:"seeders"`
}
type reportCandidate struct {
	TierName      string      `json:"tierName,omitempty"`
	Status        string      `json:"status"`
	Reasons       []string    `json:"reasons"`
	ExcludedRules []string    `json:"excludedRules,omitempty"`
	Facts         reportFacts `json:"facts"`
}
type reportWire struct {
	reportHeader
	Candidates []reportCandidate `json:"candidates"`
}

func (r SelectionReport) MarshalJSON() ([]byte, error) {
	selected := r.SelectedIndex
	if selected == nil && r.Selected != "" {
		found := -1
		for i, c := range r.Candidates {
			if c.Facts.Title == r.Selected {
				if found >= 0 {
					found = -1
					break
				}
				found = i
			}
		}
		if found >= 0 {
			selected = &found
		}
	}
	if selected != nil && (*selected < 0 || *selected >= len(r.Candidates)) {
		return nil, fmt.Errorf("invalid selection report candidate index")
	}
	fallback := ""
	if selected == nil {
		fallback = r.Selected
	}
	wire := reportWire{reportHeader: reportHeader{ID: r.ID, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, OmittedCandidates: r.OmittedCandidates, OmittedSites: r.OmittedSites, Version: SelectionReportVersion, PolicyVersion: r.PolicyVersion, ParserVersion: r.ParserVersion, Group: r.Group, Sites: r.Sites, Partial: r.Partial, End: r.End, SelectedIndex: selected, SelectedTitle: fallback, Counts: r.Counts}, Candidates: []reportCandidate{}}
	for _, c := range r.Candidates {
		wire.Candidates = append(wire.Candidates, reportCandidate{TierName: c.TierName, Status: c.Status, Reasons: c.Reasons, ExcludedRules: c.ExcludedRules, Facts: reportFacts{c.Facts.Title, c.Facts.Size, c.Facts.Seeders}})
	}

	return json.Marshal(wire)
}

func (r *SelectionReport) UnmarshalJSON(data []byte) error {
	if len(data) > 256<<10 {
		return fmt.Errorf("selection report exceeds size limit")
	}
	var wire reportWire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if wire.Version != SelectionReportVersion {
		return fmt.Errorf("unsupported selection report version %d", wire.Version)
	}
	result := SelectionReport{ID: wire.ID, StartedAt: wire.StartedAt, FinishedAt: wire.FinishedAt, OmittedCandidates: wire.OmittedCandidates, OmittedSites: wire.OmittedSites, Version: wire.Version, PolicyVersion: wire.PolicyVersion, ParserVersion: wire.ParserVersion, Group: wire.Group, Sites: wire.Sites, Partial: wire.Partial, End: wire.End, Selected: wire.SelectedTitle, SelectedIndex: wire.SelectedIndex, Counts: wire.Counts, Candidates: []resourceselection.Evaluation{}}
	for _, c := range wire.Candidates {
		result.Candidates = append(result.Candidates, resourceselection.Evaluation{TierName: c.TierName, Status: c.Status, Reasons: c.Reasons, ExcludedRules: c.ExcludedRules, Facts: resourceselection.Facts{Title: c.Facts.Title, Size: c.Facts.Size, Seeders: c.Facts.Seeders}})
	}
	if result.SelectedIndex != nil {
		if *result.SelectedIndex < 0 || *result.SelectedIndex >= len(result.Candidates) {
			return fmt.Errorf("invalid selection report candidate index")
		}
		result.Selected = result.Candidates[*result.SelectedIndex].Facts.Title
	}
	*r = result
	return nil
}

func DecodeSelectionReport(raw string) (SelectionReport, error) {
	var report SelectionReport
	err := json.Unmarshal([]byte(raw), &report)
	return report, err
}
