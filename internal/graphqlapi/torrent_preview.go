package graphqlapi

import (
	"context"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
)

type selectionPreviewService interface {
	Selection(context.Context, string, []string) (torrentsearch.Selection, error)
}

func previewPage(offset, limit *int, sortBy *model.TorrentSearchSort) (torrentsearch.Page, error) {
	p := torrentsearch.Page{Limit: 100}
	if offset != nil {
		p.Offset = *offset
	}
	if limit != nil {
		p.Limit = *limit
	}
	if sortBy != nil {
		p.Sort = string(*sortBy)
	}
	if p.Offset < 0 || p.Offset > 1000 || p.Limit < 1 || p.Limit > 100 {
		return p, torrentsearch.ErrInvalid
	}
	return p, nil
}
func (r *Resolver) previewTorrentSelection(ctx context.Context, input model.PreviewTorrentSelectionInput) (*model.TorrentSelectionPreview, error) {
	page, err := previewPage(input.Offset, input.Limit, input.SortBy)
	if err != nil {
		return nil, err
	}
	source, ok := r.SearchRuns.(selectionPreviewService)
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	runtime, ok := r.TaskRuntime.(interface {
		PreviewTorrentSelectionContext(context.Context, taskruntime.PreviewTorrentSelectionRequest) (*taskruntime.CandidateSelectionPreview, error)
		ResourceSelectionPolicy() resourceselection.Policy
	})
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	var code string
	if input.TaskID != nil {
		task, err := r.TaskRuntime.FindTask(ctx, *input.TaskID)
		if err != nil {
			return nil, err
		}
		if task == nil || task.Film == nil {
			return nil, film.ErrRequired
		}
		code = task.Film.Code
	} else {
		creator, ok := r.TaskFlow.(filmAdmission)
		if !ok || input.Endpoint == nil || input.SceneID == nil || input.ConfirmationDigest == nil {
			return nil, film.ErrRequired
		}
		v, err := creator.ResolveFilm(ctx, *input.Endpoint, *input.SceneID)
		if err != nil {
			return nil, err
		}
		if v.Digest() != *input.ConfirmationDigest {
			return nil, film.ErrChanged
		}
		code = v.Code
	}
	// Always evaluate the complete run. Candidate ids restrict only the projection.
	if input.CandidateIds != nil {
		if _, err := source.Selection(ctx, input.SearchID, input.CandidateIds); err != nil {
			return nil, err
		}
	}
	selection, err := source.Selection(ctx, input.SearchID, nil)
	if err != nil {
		return nil, err
	}
	policy := runtime.ResourceSelectionPolicy()
	if input.GroupID != nil {
		policy, err = policy.WithGroup(*input.GroupID)
		if err != nil {
			return nil, config.ErrInvalidSettings
		}
	}
	req := taskruntime.PreviewTorrentSelectionRequest{Query: code, Results: selection.RankingCandidates(), Policy: &policy}
	out, err := runtime.PreviewTorrentSelectionContext(ctx, req)
	if err != nil {
		return nil, err
	}
	out.VisibleIDs = input.CandidateIds
	return projectSelectionPreview(selection, out, page)
}
func projectSelectionPreview(selection torrentsearch.Selection, out *taskruntime.CandidateSelectionPreview, page torrentsearch.Page) (*model.TorrentSelectionPreview, error) {
	ranked := out.Results
	if out.VisibleIDs != nil {
		wanted := map[string]bool{}
		for _, id := range out.VisibleIDs {
			wanted[id] = true
		}
		ranked = nil
		for _, row := range out.Results {
			if wanted[row.GUID] {
				ranked = append(ranked, row)
			}
		}
	}
	rows, total, err := selection.Project(ranked, page)
	if err != nil {
		return nil, err
	}
	byReference := map[string]string{}
	for _, item := range selection.Candidates {
		byReference["indexer-resource:"+item.ResourceID] = item.Candidate.ID
	}
	visible := map[string]bool{}
	for _, row := range rows {
		visible[row.ID] = true
	}
	evaluations := []*model.ResourceEvaluation{}
	for _, e := range out.Evaluations {
		if !visible[byReference[e.ID]] {
			continue
		}
		v := evaluationModel(e)
		v.CandidateID = nilIfEmpty(byReference[e.ID])
		evaluations = append(evaluations, v)
	}
	return &model.TorrentSelectionPreview{SearchID: selection.SearchID, SearchRevision: selection.Revision, Partial: selection.Partial, Total: total, Offset: page.Offset, Limit: page.Limit, Candidates: torrentSearchCandidatesToModel(rows), Evaluations: evaluations}, nil
}
func evaluationModel(e resourceselection.Evaluation) *model.ResourceEvaluation {
	out := &model.ResourceEvaluation{EvidenceOmitted: e.EvidenceOmitted, GroupName: e.GroupName, TierName: e.TierName, Title: e.Facts.Title, Status: e.Status, Preferred: e.Preferred, Reasons: e.Reasons, UnknownRules: e.UnknownRules, Evidence: []*model.ResourceEvidence{}}
	for _, v := range e.Facts.Evidence {
		out.Evidence = append(out.Evidence, &model.ResourceEvidence{Attribute: v.Attribute, Value: v.Value, Source: v.Source, Text: v.Text})
	}
	return out
}
func selectionReportModel(raw string) *model.ResourceSelectionReport {
	if raw == "" {
		return nil
	}
	report, err := taskruntime.DecodeSelectionReport(raw)
	if err != nil {
		return nil
	}
	out := &model.ResourceSelectionReport{End: report.End, Partial: report.Partial, Evaluations: []*model.ResourceSelectionCandidate{}}
	if report.Group != nil {
		if report.Group.Name != "" {
			name := report.Group.Name
			out.GroupName = &name
		}
	} else if report.PolicyVersion > 0 {
		name := ""
		out.GroupName = &name
	}
	for index, e := range report.Candidates {
		candidate := &model.ResourceSelectionCandidate{Selected: report.SelectedIndex != nil && *report.SelectedIndex == index, Title: e.Facts.Title, Status: e.Status, TierName: e.TierName, Reasons: e.Reasons, ExcludedRules: e.ExcludedRules}
		if candidate.ExcludedRules == nil {
			candidate.ExcludedRules = []string{}
		}
		if e.Facts.Size >= 0 {
			size := e.Facts.Size
			candidate.Size = &size
		}
		if e.Facts.Seeders >= 0 {
			seeders := e.Facts.Seeders
			candidate.Seeders = &seeders
		}
		out.Evaluations = append(out.Evaluations, candidate)
	}
	return out
}
