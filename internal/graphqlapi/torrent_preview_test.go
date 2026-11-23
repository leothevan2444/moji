package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/resourceselection"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
	"testing"
)

func (s *searchAPISource) Selection(context.Context, string, []string) (torrentsearch.Selection, error) {
	snapshot := s.snapshot()
	return torrentsearch.Selection{SearchID: snapshot.ID, Revision: snapshot.Revision, Query: snapshot.Query, Candidates: []torrentsearch.SelectionItem{{Candidate: snapshot.Candidates[0], ResourceID: "private-reference"}}}, nil
}

type previewAPIRuntime struct {
	fakeTaskRuntime
	request taskruntime.PreviewTorrentSelectionRequest
	policy  *resourceselection.Policy
	calls   int
}

func (s *previewAPIRuntime) PreviewTorrentSelectionContext(_ context.Context, req taskruntime.PreviewTorrentSelectionRequest) (*taskruntime.CandidateSelectionPreview, error) {
	s.request = req
	s.calls++
	return &taskruntime.CandidateSelectionPreview{Results: req.Results}, nil
}
func TestNeutralPreviewGraphQLUsesOnlyServerSelectionAndReturnsSynchronousMatching(t *testing.T) {
	runtime := &previewAPIRuntime{fakeTaskRuntime: fakeTaskRuntime{findTask: &taskruntime.Task{Film: &film.Snapshot{Code: "ABCD-123"}}}}
	resolver := NewResolver(nil, runtime, nil, "test")
	resolver.SearchRuns = &searchAPISource{}
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Preview *model.TorrentSelectionPreview `json:"previewTorrentSelection"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run",taskId:"task",limit:10}){searchId searchRevision total limit candidates{id title size seeders} partial}}`, &response)
	if len(response.Errors) != 0 || response.Data.Preview == nil || response.Data.Preview.SearchRevision != 4 || response.Data.Preview.Total != 1 || runtime.request.Results[0].ResourceID != "private-reference" || runtime.request.Results[0].Link != "" || runtime.request.Query != "ABCD-123" {
		t.Fatal("neutral preview lost server selection", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run",taskId:"task",limit:1000}){total}}`, &response)
	if len(response.Errors) != 1 || runtime.calls != 1 {
		t.Fatal("invalid page consumed evaluation")
	}
}

func (s *previewAPIRuntime) ResourceSelectionPolicy() resourceselection.Policy {
	if s.policy != nil {
		return *s.policy
	}
	return resourceselection.Default()
}
func TestPreviewRejectsMissingFilmBeforeAcquiringResources(t *testing.T) {
	runtime := &previewAPIRuntime{}
	resolver := NewResolver(nil, runtime, nil, "test")
	resolver.SearchRuns = &searchAPISource{}
	var response struct{ Errors []struct{ Message string } }
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run"}){partial}}`, &response)
	if len(response.Errors) != 1 || runtime.calls != 0 {
		t.Fatal("unconfirmed target consumed matching", response.Errors, runtime.calls)
	}
}

func TestPreviewSelectsOneGroupWithoutChangingDefault(t *testing.T) {
	p := resourceselection.Default()
	p.ActiveGroup = "main"
	for _, id := range []string{"main", "alternate"} {
		p.Groups = append(p.Groups, resourceselection.Group{Fallback: true, ID: id, Name: id, Tiers: []resourceselection.Tier{}})
	}
	runtime := &previewAPIRuntime{policy: &p, fakeTaskRuntime: fakeTaskRuntime{findTask: &taskruntime.Task{Film: &film.Snapshot{Code: "ABCD-123"}}}}
	resolver := NewResolver(nil, runtime, nil, "test")
	resolver.SearchRuns = &searchAPISource{}
	var response struct{ Errors []struct{ Message string } }
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run",taskId:"task",groupID:"alternate"}){partial}}`, &response)
	if len(response.Errors) != 0 || runtime.request.Policy.ActiveGroup != "alternate" || p.ActiveGroup != "main" {
		t.Fatal("group preview mutated default", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run",taskId:"task",groupID:"missing"}){partial}}`, &response)
	if len(response.Errors) != 1 || runtime.calls != 1 {
		t.Fatal("invalid group evaluated resources")
	}
}
