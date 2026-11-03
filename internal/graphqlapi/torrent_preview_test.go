package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
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
	calls   int
}

func (s *previewAPIRuntime) PreviewTorrentSelectionContext(_ context.Context, req taskruntime.PreviewTorrentSelectionRequest) (*taskruntime.CandidateSelectionPreview, error) {
	s.request = req
	s.calls++
	return &taskruntime.CandidateSelectionPreview{Results: req.Results, Meta: taskruntime.CandidateSelectionPreviewMeta{AppliedFastRules: req.ApplyFastRules, AppliedFileRules: req.ApplyFileRules, InspectedCount: 1, InspectableCount: 2, InspectionCandidateLimit: 5}}, nil
}
func TestNeutralPreviewGraphQLUsesOnlyServerSelectionAndReportsBudget(t *testing.T) {
	runtime := &previewAPIRuntime{}
	resolver := NewResolver(nil, runtime, nil, "test")
	resolver.SearchRuns = &searchAPISource{}
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Preview *model.TorrentSelectionPreview `json:"previewTorrentSelection"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run",applyFastRules:true,applyFileRules:true,limit:10}){searchId searchRevision total limit candidates{id title size seeders} meta{appliedFastRules appliedFileRules inspectedCount inspectableCount inspectionCandidateLimit}}}`, &response)
	if len(response.Errors) != 0 || response.Data.Preview == nil || response.Data.Preview.SearchRevision != 4 || response.Data.Preview.Total != 1 || response.Data.Preview.Meta.InspectionCandidateLimit != 5 || runtime.request.Results[0].ResourceID != "private-reference" || runtime.request.Results[0].Link != "" || runtime.request.Query != "ABCD-123" {
		t.Fatal("neutral preview lost server selection/budget", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{previewTorrentSelection(input:{searchId:"run",applyFileRules:true,limit:1000}){total}}`, &response)
	if len(response.Errors) != 1 || runtime.calls != 1 {
		t.Fatal("invalid page consumed file inspection quota")
	}
}
