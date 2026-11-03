package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexercheck"
)

type fakeIndexerCheck struct {
	input indexercheck.Input
	calls int
}

func (f *fakeIndexerCheck) Check(_ context.Context, input indexercheck.Input) (indexercheck.Report, error) {
	f.input = input
	f.calls++
	return indexercheck.Report{ID: "correlation", InstanceID: "diagnostic-only", Kind: indexer.Builtin, ConfigRevision: "configuration", RuleRevision: "rules", StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Stages: []indexercheck.Stage{{Name: "CONFIGURATION", State: "PASSED", Elapsed: 3 * time.Millisecond}, {Name: "AUTHENTICATION", State: "FAILED", ErrorCode: indexer.ErrChallenge.Error()}, {Name: "SEARCH", State: "SKIPPED"}, {Name: "DOWNLOAD", State: "NOT_REQUESTED"}}}, nil
}
func TestIndexerCheckGraphQLPreservesActualDraftAndRequestedScopeWithoutReturningSecrets(t *testing.T) {
	checker := &fakeIndexerCheck{}
	r := NewResolver(nil, nil, nil, "test")
	r.IndexerChecks = checker
	var response struct {
		Errors []any
		Data   struct {
			Report *model.IndexerCheckReport `json:"testIndexer"`
		}
	}
	executeDefinitionGraphQLInto(t, r, `mutation{testIndexer(input:{expectedRevision:"configuration",definitionId:"example",draft:{name:"Unsaved name",enabled:false,settings:[{name:"password",clear:true},{name:"filter",value:{boolean:false}}]}}){id instanceId kind configRevision ruleRevision results partial truncated startedAt finishedAt stages{name state errorCode elapsedMs}}}`, &response)
	if len(response.Errors) != 0 || response.Data.Report == nil || checker.calls != 1 || checker.input.Search || checker.input.Download || checker.input.InstanceID != "" || checker.input.ExpectedRevision != "configuration" || checker.input.Draft == nil || *checker.input.Draft.Definition != "example" || *checker.input.Draft.Name != "Unsaved name" || *checker.input.Draft.Enabled || !checker.input.Draft.Settings["password"].Clear || *checker.input.Draft.Settings["filter"].Value.Boolean {
		t.Fatal("test API changed draft/default operation scope", response.Errors, checker.input)
	}
	got := response.Data.Report
	if len(got.Stages) != 4 || got.Stages[0].ElapsedMs != 3 || got.Stages[1].State != model.IndexerCheckStageStateFailed || got.Stages[1].ErrorCode != indexer.ErrChallenge.Error() || got.Stages[3].State != model.IndexerCheckStageStateNotRequested {
		t.Fatal("stage mapping falsely reported success", got)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, r, `mutation{testIndexer(input:{instanceId:"jackett/one",expectedRevision:"configuration",search:true,keywords:"ABCD-123",download:true}){id}}`, &response)
	if len(response.Errors) != 0 || !checker.input.Search || !checker.input.Download || checker.input.Keywords != "ABCD-123" || checker.input.Draft != nil || checker.input.InstanceID != "jackett/one" {
		t.Fatal("explicit external search/download scope lost", response.Errors, checker.input)
	}
}
func TestIndexerCheckRejectsDefinitionOverridesAndMalformedSettingChangesBeforeExecution(t *testing.T) {
	checker := &fakeIndexerCheck{}
	r := NewResolver(nil, nil, nil, "test")
	r.IndexerChecks = checker
	for _, input := range []string{`instanceId:"existing",expectedRevision:"configuration",definitionId:"other"`, `expectedRevision:"configuration",draft:{settings:[{name:"same",clear:true},{name:"same",clear:true}]}`} {
		var reply struct{ Errors []any }
		executeDefinitionGraphQLInto(t, r, `mutation{testIndexer(input:{`+input+`}){id}}`, &reply)
		if len(reply.Errors) == 0 || checker.calls != 0 {
			t.Fatal("invalid draft reached site test", reply)
		}
	}
}
