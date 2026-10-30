package taskruntime

import (
	"context"
	"encoding/json"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSourcingRejectsInvalidLocksAndFallsBackNormally(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2.0/indexers" {
			_ = json.NewEncoder(w).Encode([]jackett.Indexer{{ID: "a", Configured: true}, {ID: "b", Configured: true}, {ID: "c", Configured: true}, {ID: "d", Configured: true}})
			return
		}
		site := r.URL.Query().Get("Tracker[]")
		calls = append(calls, site)
		titles := map[string]string{"a": "WRONG-456 required", "b": "ABCD-123 excluded required", "c": "ABCD-123 ordinary", "d": "ABCD-123 required"}
		_ = json.NewEncoder(w).Encode(map[string]any{"Results": []jackett.SearchResult{{Title: titles[site], MagnetURI: "magnet:?xt=urn:btih:1111111111111111111111111111111111111111"}}})
	}))
	defer server.Close()
	tr := tracker.NewJackettService(func() tracker.JackettConfig {
		return tracker.JackettConfig{URL: server.URL, QueryOrder: []string{"a", "b", "c", "d"}}
	})
	service, err := NewService(tr, &fakeTorrentAdder{}, NewMemoryTaskStore(), WithCandidateSelectionProvider(func() config.CandidateSelectionConfig {
		return candidateSelectionConfig(true, 1, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTitleMatch, Enabled: true, TitleMatch: config.TitleMatchRuleConfig{Clauses: []config.TitleMatchClause{
			{Pattern: "required", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectLock},
			{Pattern: "excluded", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectExclude},
		}}}})
	}))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Code: "ABCD-123"})
	if err != nil || task.Stage != TaskStageDownloading || !reflect.DeepEqual(calls, []string{"a", "b", "c"}) {
		t.Fatalf("task=%+v calls=%v err=%v", task, calls, err)
	}
}
