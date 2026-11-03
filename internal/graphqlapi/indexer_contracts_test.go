package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func TestIndexerPublicContractHasOnlyServerOwnedSearchAndSelection(t *testing.T) {
	executable := generated.NewExecutableSchema(generated.Config{Resolvers: NewResolver(nil, nil, nil, "test")})
	schema := executable.Schema()
	for _, name := range []string{"jackettSearch", "previewJackettSelection", "jackettIndexers", "blockedTaskTorrentCandidates"} {
		if schema.Query.Fields.ForName(name) != nil {
			t.Fatal("removed raw candidate API remains public", name)
		}
		payload, _ := json.Marshal(map[string]string{"query": "query{" + name + "}"})
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		reply := httptest.NewRecorder()
		NewGraphQLServer(executable).ServeHTTP(reply, req)
		if reply.Code != http.StatusUnprocessableEntity {
			t.Fatal("legacy field bypassed validation", name, reply.Code)
		}
	}
	for _, name := range []string{"JackettSearchInput", "JackettSearchResult", "JackettIndexer", "JackettSortBy", "PreviewJackettSelectionInput", "PreviewJackettSelectionResult"} {
		if schema.Types[name] != nil {
			t.Fatal("removed client-locator type remains public", name)
		}
	}
	if schema.Query.Fields.ForName("torrentSearch") == nil {
		t.Fatal("durable search missing")
	}
	for _, name := range []string{"startTorrentSearch", "previewTorrentSelection", "queueTorrentCandidate", "resolveTaskSourcingCandidate", "prepareTorrentDownload"} {
		if schema.Mutation.Fields.ForName(name) == nil {
			t.Fatal("neutral operation missing", name)
		}
	}
}
func TestWaitingResourceTaskStartsDurableSearchWithItsServerCode(t *testing.T) {
	next := time.Now().Add(time.Hour)
	runtime := &fakeTaskRuntime{findTask: &taskruntime.Task{ID: "waiting", Code: "OFJE-657", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusPending, SourcingNextCheckAt: &next}}
	source := &searchAPISource{}
	r := NewResolver(nil, runtime, nil, "test")
	r.SearchRuns = source
	taskID := "waiting"
	_, err := r.startTorrentSearch(context.Background(), model.StartTorrentSearchInput{Keywords: "OFJE-657", TaskID: &taskID})
	if err != nil || source.input.TaskID != "waiting" || source.input.Query.Keywords != "OFJE-657" {
		t.Fatal("waiting source task lost durable search admission", err, source.input)
	}
}
