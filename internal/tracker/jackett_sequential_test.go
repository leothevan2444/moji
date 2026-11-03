package tracker

import (
	"context"
	"encoding/json"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/pkg/jackett"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSequentialSiteSearchStopsOnAcceptanceAndFallsBack(t *testing.T) {
	for _, scenario := range []string{"hit", "empty", "failure", "reject", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			calls := []string{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v2.0/indexers" {
					_ = json.NewEncoder(w).Encode([]jackett.Indexer{{ID: "c", Configured: true}, {ID: "a", Configured: true}, {ID: "b", Configured: true}, {ID: "off", Configured: false}})
					return
				}
				ids := r.URL.Query()["Tracker[]"]
				if len(ids) != 1 {
					t.Errorf("expected one site per request, got %v", ids)
					w.WriteHeader(400)
					return
				}
				id := ids[0]
				calls = append(calls, id)
				if id == "a" && scenario == "failure" {
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				if id == "a" && scenario == "cancel" {
					cancel()
					return
				}
				results := []indexer.SelectionCandidate{{Title: id}}
				if id == "a" && scenario == "empty" {
					results = nil
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"Results": results})
			}))
			defer server.Close()
			service := NewJackettService(func() JackettConfig {
				return JackettConfig{URL: server.URL, QueryOrder: []string{"missing", "a", "b", "a"}}
			})
			err := service.SearchSequential(ctx, "ABCD-123", func(batch []indexer.SelectionCandidate) (bool, error) {
				if len(batch) == 0 {
					return false, nil
				}
				return scenario == "hit" || batch[0].Title == "b", nil
			})
			if scenario == "cancel" {
				if err == nil || !reflect.DeepEqual(calls, []string{"a"}) {
					t.Fatalf("cancellation failed: %v, %v", calls, err)
				}
				return
			}
			want := []string{"a", "b"}
			if scenario == "hit" {
				want = []string{"a"}
			}
			if err != nil || !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls=%v want=%v err=%v", calls, want, err)
			}
			// Manual searches collect all configured sites in the same sequential order.
			calls = nil
			_, err = service.Search("ABCD-123")
			if err != nil || !reflect.DeepEqual(calls, []string{"a", "b", "c"}) {
				t.Fatalf("manual calls=%v err=%v", calls, err)
			}
		})
	}
}
