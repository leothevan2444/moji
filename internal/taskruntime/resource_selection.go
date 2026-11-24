package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
)

type selectionItem struct {
	result     indexer.SelectionCandidate
	evaluation resourceselection.Evaluation
}
type selectionRound struct {
	selected      *selectionItem
	validationErr error
	now           func() time.Time
	code          string
	policy        resourceselection.Policy
	items         []*selectionItem
	seen          map[string]bool
	sites         int
	report        selectionReport
}

func newSelectionRound(now func() time.Time, code string, p resourceselection.Policy) *selectionRound {
	p = p.Effective()
	err := p.Validate()
	if err == nil {
		p = resourceselection.Prepare(p)
	}
	return &selectionRound{validationErr: err, now: now, code: code, policy: p, seen: map[string]bool{}, report: selectionReport{ID: uuid.NewString(), StartedAt: now().UTC(), Version: SelectionReportVersion, ParserVersion: resourceselection.ParserVersion, PolicyVersion: p.Version, Group: selectionGroup(p), Candidates: []resourceselection.Evaluation{}, Sites: []indexer.SiteReport{}, Counts: map[string]int{}}}
}
func (r *selectionRound) add(ctx context.Context, rows []indexer.SelectionCandidate) (bool, error) {
	if r.validationErr != nil {
		return false, r.validationErr
	}
	site := r.sites
	r.sites++
	for i, row := range rows {
		if len(r.items) >= 1000 {
			r.report.Partial = true
			break
		}
		key := preferredTorrentURL(row)
		dedupKey := strconv.Itoa(site) + "\x00" + key + "\x00" + row.Title
		if key == "" || r.seen[dedupKey] {
			continue
		}
		r.seen[dedupKey] = true
		f := candidateFacts(row, len(r.items))
		f.Site = site
		e := resourceselection.EvaluateSummary(r.policy, f)
		if _, err := validateCodeEvidence(r.code, row.Title); err != nil {
			e.Status = "REJECTED"
			e.Preferred = false
			e.Reasons = append(e.Reasons, TaskStageErrorCandidateUnverified)
		}
		r.items = append(r.items, &selectionItem{result: row, evaluation: e})
		if i >= 99 {
			r.report.Partial = r.report.Partial || len(rows) > 100
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if r.preferred() != nil {
		return true, nil
	}
	if len(r.items) >= 1000 {
		r.report.Partial = true
		return true, nil
	}

	return false, nil
}
func (r *selectionRound) sorted() []*selectionItem {
	items := append([]*selectionItem(nil), r.items...)
	sort.SliceStable(items, func(i, j int) bool { return resourceselection.Compare(items[i].evaluation, items[j].evaluation) < 0 })
	return items
}
func (r *selectionRound) preferred() *selectionItem {
	for _, i := range r.sorted() {
		if i.evaluation.Status == "ACCEPTABLE" && i.evaluation.Preferred {
			return i
		}
	}
	return nil
}
func (r *selectionRound) finish(ctx context.Context) (indexer.SelectionCandidate, error) {
	if r.validationErr != nil {
		return indexer.SelectionCandidate{}, r.validationErr
	}
	r.report.Partial = r.report.Partial || ctx.Err() != nil
	for _, item := range r.sorted() {
		if item.evaluation.Status == "ACCEPTABLE" {
			r.selected = item
			r.report.Selected = item.result.Title
			r.report.End = "BEST_AVAILABLE"
			if item.evaluation.Preferred {
				r.report.End = "PREFERRED"
			}
			return item.result, nil
		}
	}
	r.report.End = "NO_ACCEPTABLE_RESOURCE"
	if r.report.Partial {
		r.report.End = "INCOMPLETE"
	}
	return indexer.SelectionCandidate{}, errors.New("no acceptable resource available from indexer results")
}
func (r *selectionRound) json() string {
	r.report.FinishedAt = r.now().UTC()
	r.report.Selected = reportText(r.report.Selected)
	r.report.Counts = map[string]int{}
	r.report.Candidates = []resourceselection.Evaluation{}
	keepIndex := -1
	for _, item := range r.sorted() {
		r.report.Counts[item.evaluation.Status]++
		if len(r.report.Candidates) < 20 || item == r.selected {
			if item == r.selected {
				keepIndex = len(r.report.Candidates)
			}
			e := item.evaluation
			e.ID = ""
			e.Facts.ID = "" // opaque upstream locators and credentials must never enter reports
			e.Facts.Title = reportText(e.Facts.Title)
			r.report.Candidates = append(r.report.Candidates, e)
		}
	}

	if keepIndex >= 0 {
		r.report.SelectedIndex = &keepIndex
	}
	r.report.OmittedCandidates = len(r.items) - len(r.report.Candidates)
	for {
		data, err := json.Marshal(r.report)
		if err != nil {
			return ""
		}
		if len(data) <= 256<<10 {
			return string(data)
		}
		removed := false
		for i := len(r.report.Candidates) - 1; i >= 0; i-- {
			if i != keepIndex {
				if keepIndex > i {
					keepIndex--
				}
				r.report.Candidates = append(r.report.Candidates[:i], r.report.Candidates[i+1:]...)
				r.report.OmittedCandidates++
				removed = true
				break
			}
		}
		if removed {
			continue
		}
		if len(r.report.Sites) > 0 {
			r.report.Sites = r.report.Sites[:len(r.report.Sites)-1]
			r.report.OmittedSites++
			continue
		}
		return ""
	}

}

// Manual adoption records the override instead of retaining a stale automatic
// recommendation. Acquisition does not regenerate reports; only an explicit
// choice creates a new report, so expired history is never reinterpreted.
func (s *Service) manualSelectionReport(task *Task) string {
	p := s.ResourceSelectionPolicy()
	if old, err := DecodeSelectionReport(task.SelectionReportJSON); err == nil && old.End == "USER_SELECTED" {
		return task.SelectionReportJSON
	}
	round := newSelectionRound(s.now, task.Code, p)
	title := task.Candidate.Title
	if title == task.TorrentURL && (strings.HasPrefix(title, "http") || strings.HasPrefix(title, "magnet:")) {
		title = ""
	}
	f := resourceselection.Facts{Title: title, Seeders: task.Candidate.Seeders, Size: task.Candidate.Size}
	e := resourceselection.EvaluateSummary(p, f)
	round.items = []*selectionItem{{evaluation: e}}
	round.selected = round.items[0]
	round.report.End = "USER_SELECTED"
	round.report.Selected = title
	return round.json()
}

func reportText(v string) string {
	r := []rune(v)
	if len(r) > 240 {
		return string(r[:240]) + "…"
	}
	return v
}

func selectionGroup(p resourceselection.Policy) *SelectionReportGroup {
	for _, g := range p.Groups {
		if p.ActiveGroup != "" && g.ID == p.ActiveGroup {
			return &SelectionReportGroup{ID: g.ID, Name: g.Name}
		}
	}
	return nil
}
