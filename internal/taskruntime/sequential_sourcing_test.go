package taskruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestSourcingRetainsFallbackAndContinuesUntilPreferred(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2.0/indexers" {
			_ = json.NewEncoder(w).Encode([]jackett.Indexer{{ID: "a", Configured: true}, {ID: "b", Configured: true}, {ID: "c", Configured: true}, {ID: "d", Configured: true}})
			return
		}
		site := r.URL.Query().Get("Tracker[]")
		calls = append(calls, site)
		titles := map[string]string{"a": "WRONG-456 required", "b": "ABCD-123 excluded required", "c": "ABCD-123 ordinary", "d": "ABCD-123 required"}
		_ = json.NewEncoder(w).Encode(map[string]any{"Results": []indexer.SelectionCandidate{{Title: titles[site], MagnetURI: "magnet:?xt=urn:btih:1111111111111111111111111111111111111111"}}})
	}))
	defer server.Close()
	tr := tracker.NewJackettService(func() tracker.JackettConfig {
		return tracker.JackettConfig{URL: server.URL, QueryOrder: []string{"a", "b", "c", "d"}}
	})
	service, err := NewService(tr, &fakeTorrentAdder{}, NewMemoryTaskStore(), WithCandidateSelectionProvider(func() config.CandidateSelectionConfig {
		p := config.DefaultCandidateSelectionConfig()
		p.Rules = []resourceselection.Rule{
			{ID: "preferred", Name: "preferred", Conditions: []resourceselection.Condition{{Scope: "TITLE", Value: "required"}}},
			{ID: "excluded", Name: "excluded", Conditions: []resourceselection.Condition{{Scope: "TITLE", Value: "excluded"}}},
		}
		p.ActiveGroup = "main"
		p.Groups = []resourceselection.Group{{Fallback: true, ID: "main", Name: "Main", Exclude: []string{"excluded"}, Tiers: []resourceselection.Tier{{ID: "first", Name: "First", Rules: []string{"preferred"}}}}}
		return p
	}))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.runSourcingFixture(context.Background(), DownloadRequest{Film: testFilm("ABCD-123"), Code: "ABCD-123"})
	if err != nil || task.Stage != TaskStageDownloading || !reflect.DeepEqual(calls, []string{"a", "b", "c", "d"}) {
		t.Fatalf("task=%+v calls=%v err=%v", task, calls, err)
	}
}
