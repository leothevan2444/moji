package graphqlapi

import (
	"context"
	"errors"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
	"testing"
	"time"
)

type searchAPISource struct {
	input    torrentsearch.Input
	page     torrentsearch.Page
	selected bool
}

func (s *searchAPISource) snapshot() torrentsearch.Snapshot {
	return torrentsearch.Snapshot{ID: "run", State: "PARTIAL", Revision: 4, Query: indexer.Query{Keywords: "ABCD-123"}, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0), ExpiresAt: time.Unix(3, 0), Partial: true, Total: 1, Sites: []torrentsearch.Site{{InstanceID: "site", Name: "Site", Kind: indexer.Builtin, State: "COMPLETED", Report: indexer.SiteReport{Results: 1, CacheHit: true}}}, Candidates: []torrentsearch.Candidate{{ID: "candidate", InstanceID: "site", InstanceName: "Site", Kind: indexer.Builtin, Title: "ABCD-123"}}}
}
func (s *searchAPISource) StartSearch(_ context.Context, in torrentsearch.Input) (torrentsearch.Snapshot, error) {
	s.input = in
	return s.snapshot(), nil
}
func (s *searchAPISource) Get(_ context.Context, id string, page torrentsearch.Page) (torrentsearch.Snapshot, error) {
	s.page = page
	return s.snapshot(), nil
}
func (s *searchAPISource) CancelSearch(context.Context, string) (torrentsearch.Snapshot, error) {
	out := s.snapshot()
	out.State = "RUNNING"
	out.CancelRequested = true
	return out, nil
}
func (s *searchAPISource) ResourceID(_ context.Context, id, candidate string) (string, error) {
	if id != "run" || candidate != "candidate" {
		return "", torrentsearch.ErrExpired
	}
	s.selected = true
	return "server-private-resource", nil
}
func (s *searchAPISource) Subscribe(context.Context, string) (<-chan torrentsearch.Event, func(), error) {
	return nil, nil, errors.New("unused")
}

type selectedTaskFlow struct {
	req   taskruntime.QueueIndexedResourceRequest
	calls int
}

func (*selectedTaskFlow) CreateFromManualTorrent(context.Context, taskflow.CreateFromManualTorrentInput) (*taskruntime.Task, error) {
	return nil, errors.New("wrong manual entry")
}
func (*selectedTaskFlow) CreateFromSearchCode(context.Context, taskflow.CreateFromSearchCodeInput) (*taskruntime.Task, error) {
	return nil, errors.New("wrong automatic entry")
}
func (s *selectedTaskFlow) QueueTorrentCandidate(_ context.Context, req taskruntime.QueueIndexedResourceRequest) (*taskruntime.Task, error) {
	s.req = req
	s.calls++
	return &taskruntime.Task{ID: "created", Code: req.Code}, nil
}
func TestPersistentSearchGraphQLScopePageUnknownValuesAndCancellation(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, "test")
	source := &searchAPISource{}
	resolver.SearchRuns = source
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Start  *model.TorrentSearch `json:"startTorrentSearch"`
			Read   *model.TorrentSearch `json:"torrentSearch"`
			Cancel *model.TorrentSearch `json:"cancelTorrentSearch"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{startTorrentSearch(input:{keywords:"ABCD-123",instanceIds:["site"],categories:[6000],limit:25,refresh:true,parameters:[{key:"year",value:"2026"}]}){id revision state keywords partial total sites{instanceId kind state results cacheHit fetchedAt} candidates{id title size seeders peers publishedAt categories}}}`, &response)
	if len(response.Errors) != 0 || response.Data.Start == nil || response.Data.Start.Candidates[0].Size != nil || response.Data.Start.Candidates[0].Seeders != nil || response.Data.Start.Sites[0].FetchedAt != nil || source.input.Query.CachePolicy != indexer.CacheRefresh || source.input.Query.InstanceIDs[0] != "site" || source.input.Query.Parameters["year"] != "2026" {
		t.Fatal("search contract lost scope/refresh/unknown observations", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `{torrentSearch(id:"run",offset:100,limit:20,sortBy:SEEDERS_DESC){id revision}}`, &response)
	if len(response.Errors) != 0 || source.page.Offset != 100 || source.page.Limit != 20 || source.page.Sort != "SEEDERS_DESC" {
		t.Fatal("query dropped server paging", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{cancelTorrentSearch(id:"run"){id state cancelRequested}}`, &response)
	if len(response.Errors) != 0 || response.Data.Cancel.State != model.TorrentSearchStateRunning || !response.Data.Cancel.CancelRequested {
		t.Fatal("cancel API invented a terminal state", response.Errors)
	}
	_, err := resolver.startTorrentSearch(context.Background(), model.StartTorrentSearchInput{Keywords: "ABCD-123", Parameters: []*model.TorrentSearchParameterInput{{Key: "x", Value: "1"}, {Key: "x", Value: "2"}}})
	if !errors.Is(err, torrentsearch.ErrInvalid) {
		t.Fatal("duplicate parameters admitted")
	}
}
func TestQueueCandidateGraphQLChecksRunMembershipAndOnlyUsesTaskFlow(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, "test")
	source := &searchAPISource{}
	creator := &selectedTaskFlow{}
	resolver.SearchRuns = source
	resolver.TaskFlow = creator
	var response struct {
		Errors []struct {
			Message    string
			Extensions struct{ Code string }
		}
		Data struct {
			Task *model.Task `json:"queueTorrentCandidate"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{queueTorrentCandidate(input:{code:"ABCD-123",searchId:"other",candidateId:"candidate"}){id}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Extensions.Code != "TORRENT_SEARCH_EXPIRED" || creator.calls != 0 {
		t.Fatal("cross-run selection admitted", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{queueTorrentCandidate(input:{code:"ABCD-123",searchId:"run",candidateId:"candidate",paused:false}){id code}}`, &response)
	if len(response.Errors) != 0 || response.Data.Task == nil || response.Data.Task.ID != "created" || creator.calls != 1 || creator.req.ResourceID != "server-private-resource" || creator.req.Paused == nil || *creator.req.Paused {
		t.Fatal("selection did not use server-owned resource/taskflow", response.Errors)
	}
}
