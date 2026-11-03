package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
	"time"
)

func (r *Resolver) previewTorrentSelection(ctx context.Context, input model.PreviewTorrentSelectionInput) (*model.TorrentSelectionPreview, error) {
	source, ok := r.SearchRuns.(interface {
		Selection(context.Context, string, []string) (torrentsearch.Selection, error)
	})
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	previewer, ok := r.TaskRuntime.(interface {
		PreviewTorrentSelectionContext(context.Context, taskruntime.PreviewTorrentSelectionRequest) (*taskruntime.CandidateSelectionPreview, error)
	})
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	page := torrentsearch.Page{Limit: 100}
	if input.Offset != nil {
		page.Offset = *input.Offset
	}
	if input.Limit != nil {
		page.Limit = *input.Limit
	}
	if input.SortBy != nil {
		page.Sort = string(*input.SortBy)
	}
	// Validate before any file inspection can consume station quota.
	if page.Offset < 0 || page.Offset > 1000 || page.Limit < 1 || page.Limit > 100 {
		return nil, torrentsearch.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	selection, err := source.Selection(ctx, input.SearchID, input.CandidateIds)
	if err != nil {
		return nil, err
	}
	evaluated, err := previewer.PreviewTorrentSelectionContext(ctx, taskruntime.PreviewTorrentSelectionRequest{Query: selection.Query.Keywords, Results: selection.RankingCandidates(), ApplyFastRules: input.ApplyFastRules, ApplyFileRules: input.ApplyFileRules})
	if err != nil {
		return nil, err
	}
	if evaluated == nil {
		return nil, indexer.ErrIncomplete
	}
	candidates, total, err := selection.Project(evaluated.Results, page)
	if err != nil {
		return nil, err
	}
	// A selection must still exist after inspection; expired runs are not revived.
	if _, err = r.SearchRuns.Get(ctx, input.SearchID, torrentsearch.Page{Limit: 1}); err != nil {
		return nil, err
	}
	meta := evaluated.Meta
	return &model.TorrentSelectionPreview{SearchID: selection.SearchID, SearchRevision: selection.Revision, Total: total, Offset: page.Offset, Limit: page.Limit, Candidates: torrentSearchCandidatesToModel(candidates), Meta: &model.TorrentSelectionPreviewMeta{AppliedFastRules: meta.AppliedFastRules, AppliedFileRules: meta.AppliedFileRules, InspectedCount: meta.InspectedCount, InspectableCount: meta.InspectableCount, InspectionCandidateLimit: meta.InspectionCandidateLimit}}, nil
}
