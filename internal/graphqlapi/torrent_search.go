package graphqlapi

import (
	"context"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
)

type TorrentSearchService interface {
	StartSearch(context.Context, torrentsearch.Input) (torrentsearch.Snapshot, error)
	Get(context.Context, string, torrentsearch.Page) (torrentsearch.Snapshot, error)
	CancelSearch(context.Context, string) (torrentsearch.Snapshot, error)
	ResourceID(context.Context, string, string) (string, error)
	Subscribe(context.Context, string) (<-chan torrentsearch.Event, func(), error)
}

func nullableText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func torrentSearchToModel(snapshot torrentsearch.Snapshot) *model.TorrentSearch {
	out := &model.TorrentSearch{FilterInstanceID: nullableText(snapshot.FilterInstanceID), CandidateQuery: snapshot.CandidateQuery, ID: snapshot.ID, State: model.TorrentSearchState(snapshot.State), Revision: snapshot.Revision, Keywords: snapshot.Query.Keywords, TaskID: nullableText(snapshot.TaskID), CreatedAt: snapshot.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: snapshot.UpdatedAt.UTC().Format(time.RFC3339Nano), ExpiresAt: snapshot.ExpiresAt.UTC().Format(time.RFC3339Nano), FinishedAt: nullableSearchTime(snapshot.FinishedAt), CancelRequested: snapshot.CancelRequested, Partial: snapshot.Partial, Truncated: snapshot.Truncated, ErrorCode: nullableText(snapshot.ErrorCode), CorrelationID: snapshot.CorrelationID, Total: snapshot.Total, Offset: snapshot.Offset, Limit: snapshot.Limit, Sites: []*model.TorrentSearchSite{}, Candidates: []*model.TorrentSearchCandidate{}}
	for _, site := range snapshot.Sites {
		report := site.Report
		item := &model.TorrentSearchSite{InstanceID: site.InstanceID, Name: site.Name, State: model.TorrentSearchSiteState(site.State), Results: report.Results, CandidateCount: site.CandidateCount, CacheHit: report.CacheHit, FetchedAt: nullableSearchTime(site.FetchedAt), ElapsedMs: report.Elapsed.Milliseconds(), Partial: report.Partial, Truncated: report.Truncated, ErrorCode: nullableText(report.Code), CorrelationID: nullableText(report.CorrelationID), RetryAt: nullableSearchTime(report.RetryAt)}
		if site.Kind != "" {
			kind := model.IndexerSourceKind(site.Kind)
			item.Kind = &kind
		}
		out.Sites = append(out.Sites, item)
	}
	out.Candidates = torrentSearchCandidatesToModel(snapshot.Candidates)
	return out
}
func (r *Resolver) startTorrentSearch(ctx context.Context, input model.StartTorrentSearchInput) (*model.TorrentSearch, error) {
	if r.SearchRuns == nil {
		return nil, indexer.ErrNotConfigured
	}
	query := indexer.Query{Keywords: input.Keywords, Categories: input.Categories, InstanceIDs: input.InstanceIds, Parameters: map[string]string{}}
	if input.Mode != nil {
		query.Mode = *input.Mode
	}
	if input.Limit != nil {
		query.Limit = *input.Limit
	}
	if input.Refresh != nil && *input.Refresh {
		query.CachePolicy = indexer.CacheRefresh
	}
	if len(input.Parameters) > 64 {
		return nil, torrentsearch.ErrInvalid
	}
	for _, pair := range input.Parameters {
		if pair == nil {
			return nil, torrentsearch.ErrInvalid
		}
		if _, exists := query.Parameters[pair.Key]; exists {
			return nil, torrentsearch.ErrInvalid
		}
		query.Parameters[pair.Key] = pair.Value
	}
	taskID := ""
	if input.TaskID != nil {
		taskID = strings.TrimSpace(*input.TaskID)
		if taskID == "" || r.TaskRuntime == nil {
			return nil, torrentsearch.ErrInvalid
		}
		task, err := r.TaskRuntime.FindTask(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if task == nil || task.Code == "" || mediafiles.ExtractCode(query.Keywords) != task.Code {
			return nil, torrentsearch.ErrInvalid
		}
	}
	snapshot, err := r.SearchRuns.StartSearch(ctx, torrentsearch.Input{Query: query, TaskID: taskID})
	if err != nil {
		return nil, err
	}
	return torrentSearchToModel(snapshot), nil
}
func (r *Resolver) torrentSearch(ctx context.Context, id string, offset, limit *int, sort *model.TorrentSearchSort, instanceID, candidateQuery *string) (*model.TorrentSearch, error) {
	if r.SearchRuns == nil {
		return nil, indexer.ErrNotConfigured
	}
	page := torrentsearch.Page{}
	if instanceID != nil {
		page.InstanceID = *instanceID
	}
	if candidateQuery != nil {
		page.CandidateQuery = *candidateQuery
	}
	if offset != nil {
		page.Offset = *offset
	}
	if limit != nil {
		page.Limit = *limit
	}
	if sort != nil {
		page.Sort = string(*sort)
	}
	snapshot, err := r.SearchRuns.Get(ctx, id, page)
	if err != nil {
		return nil, err
	}
	return torrentSearchToModel(snapshot), nil
}
func (r *Resolver) cancelTorrentSearch(ctx context.Context, id string) (*model.TorrentSearch, error) {
	if r.SearchRuns == nil {
		return nil, indexer.ErrNotConfigured
	}
	snapshot, err := r.SearchRuns.CancelSearch(ctx, id)
	if err != nil {
		return nil, err
	}
	return torrentSearchToModel(snapshot), nil
}
func (r *Resolver) queueTorrentCandidate(ctx context.Context, input model.QueueTorrentCandidateInput) (*model.Task, error) {
	if r.SearchRuns == nil {
		return nil, indexer.ErrNotConfigured
	}
	resource, err := r.SearchRuns.ResourceID(ctx, input.SearchID, input.CandidateID)
	if err != nil {
		return nil, err
	}
	creator, ok := r.TaskFlow.(interface {
		QueueTorrentCandidate(context.Context, taskruntime.QueueIndexedResourceRequest) (*taskruntime.Task, error)
	})
	if !ok {
		return nil, indexer.ErrNotConfigured
	}
	req := taskruntime.QueueIndexedResourceRequest{Code: input.Code, ResourceID: resource, Paused: input.Paused}
	if input.SavePath != nil {
		req.SavePath = *input.SavePath
	}
	if input.Category != nil {
		req.Category = *input.Category
	}
	if input.Tags != nil {
		req.Tags = *input.Tags
	}
	task, err := creator.QueueTorrentCandidate(ctx, req)
	if err != nil {
		return nil, err
	}
	return taskToModel(task), nil
}
func (r *Resolver) torrentSearchEvents(ctx context.Context, id string) (<-chan *model.TorrentSearchEvent, error) {
	if r.SearchRuns == nil {
		return nil, indexer.ErrNotConfigured
	}
	events, unsubscribe, err := r.SearchRuns.Subscribe(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make(chan *model.TorrentSearchEvent, 1)
	go func() {
		defer close(out)
		defer unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				item := &model.TorrentSearchEvent{StreamID: event.StreamID, Sequence: event.Sequence, SearchID: event.SearchID}
				select {
				case <-out:
				default:
				}
				out <- item
			}
		}
	}()
	return out, nil
}

func nullableSearchTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func torrentSearchCandidatesToModel(candidates []torrentsearch.Candidate) []*model.TorrentSearchCandidate {
	out := make([]*model.TorrentSearchCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		categories := append([]int{}, candidate.Categories...)
		out = append(out, &model.TorrentSearchCandidate{ID: candidate.ID, InstanceID: candidate.InstanceID, InstanceName: candidate.InstanceName, Kind: model.IndexerSourceKind(candidate.Kind), Title: candidate.Title, DetailsURL: nullableText(candidate.DetailsURL), InfoHash: nullableText(candidate.InfoHash), Size: candidate.Size, Seeders: candidate.Seeders, Peers: candidate.Peers, PublishedAt: nullableSearchTime(candidate.PublishedAt), Categories: categories})
	}
	return out
}
