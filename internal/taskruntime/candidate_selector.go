package taskruntime

import (
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
	"slices"
)

type defaultCandidateSelector struct{}

func candidateFacts(r indexer.SelectionCandidate, position int) resourceselection.Facts {
	return resourceselection.Facts{ID: preferredTorrentURL(r), Title: r.Title, Size: r.Size, Seeders: r.Seeders, Position: position}
}
func (s defaultCandidateSelector) Preview(ctx context.Context, code string, rows []indexer.SelectionCandidate, cfg config.CandidateSelectionConfig) (CandidateSelectionPreview, error) {
	cfg = cfg.Effective()
	if err := cfg.Validate(); err != nil {
		return CandidateSelectionPreview{}, err
	}
	facts := make([]resourceselection.Facts, len(rows))
	for i, r := range rows {
		facts[i] = candidateFacts(r, i)
	}
	evaluations := resourceselection.Rank(cfg, facts)
	for i := range evaluations {
		if _, err := validateCodeEvidence(code, evaluations[i].Facts.Title); err != nil {
			evaluations[i].Status = "REJECTED"
			evaluations[i].Preferred = false
			evaluations[i].Reasons = append(evaluations[i].Reasons, TaskStageErrorCandidateUnverified)
		}
	}
	slices.SortStableFunc(evaluations, resourceselection.Compare)
	out := CandidateSelectionPreview{Evaluations: evaluations, Results: []indexer.SelectionCandidate{}}
	for _, e := range evaluations {
		out.Results = append(out.Results, rows[e.Facts.Position])
	}
	return out, ctx.Err()
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
