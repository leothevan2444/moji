package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestIndexerDiagnosticsGraphQLHistoryAndBounds(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store, _ := indexer.NewDiagnosticStore(manager)
	now := time.Now().UTC()
	if err = store.Record(ctx, indexer.Diagnostic{DiagnosticContext: indexer.DiagnosticContext{Trigger: "AUTOMATIC", TaskID: "task-one", Code: "ABCD-123"}, InstanceID: "one", Name: "Synthetic site", Operation: indexer.HealthSearch, StartedAt: now, FinishedAt: now, Report: indexer.SiteReport{Code: indexer.ErrSolver.Error(), HTTPStatus: 503, CorrelationID: "correlation"}}); err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.IndexerDiagnosticReader = store
	var result struct {
		Data struct {
			Reports struct {
				Items []struct {
					ID, TaskID, Code, ErrorCode string
					HTTPStatus                  int
				}
			} `json:"indexerDiagnostics"`
		}
		Errors []struct{ Message string }
	}
	executeDefinitionGraphQLInto(t, resolver, `{indexerDiagnostics(input:{instanceId:"one",limit:1}){total hasMore items{id taskId code errorCode httpStatus}}}`, &result)
	if len(result.Errors) != 0 || len(result.Data.Reports.Items) != 1 || result.Data.Reports.Items[0].TaskID != "task-one" || result.Data.Reports.Items[0].Code != "ABCD-123" || result.Data.Reports.Items[0].ErrorCode != indexer.ErrSolver.Error() || result.Data.Reports.Items[0].HTTPStatus != 503 {
		t.Fatalf("diagnostic projection: %+v", result)
	}
	for _, query := range []string{`{indexerDiagnostics(input:{limit:101}){total}}`, `{indexerDiagnostics(input:{before:"invalid"}){total}}`, `{indexerDiagnostics(input:{before:"-1"}){total}}`} {
		var rejected struct{ Errors []struct{ Message string } }
		executeDefinitionGraphQLInto(t, resolver, query, &rejected)
		if len(rejected.Errors) != 1 {
			t.Fatal("invalid page accepted", query)
		}
	}
}
