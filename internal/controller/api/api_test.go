package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestHealthzIsStableRESTHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(&fakeTracker{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); body != "ok\n" {
		t.Fatalf("unexpected body: %q", body)
	}
	if got := rec.Header().Get("Deprecation"); got != "" {
		t.Fatalf("healthz should not be deprecated, got %q", got)
	}
}

func TestTrackerSearchRESTEndpointIsDeprecatedDebugCompatibility(t *testing.T) {
	tr := fakeTracker{
		results: []jackett.SearchResult{
			{Title: "ABCD-123", Tracker: "demo"},
		},
	}
	mux := http.NewServeMux()
	NewHandler(&tr).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/tracker/search?q=ABCD-123&trackers=a,b&categories=1,2&limit=3", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("expected Deprecation header %q, got %q", "true", got)
	}
	if got := rec.Header().Get("Link"); got != `</graphql>; rel="successor-version"` {
		t.Fatalf("unexpected Link header: %q", got)
	}
	if got := rec.Header().Get("Warning"); got == "" {
		t.Fatal("expected Warning header")
	}

	var resp trackerSearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if tr.query != "ABCD-123" {
		t.Fatalf("expected query %q, got %q", "ABCD-123", tr.query)
	}
	if tr.options.Limit != 3 {
		t.Fatalf("expected limit %d, got %d", 3, tr.options.Limit)
	}
	if len(tr.options.Trackers) != 2 || tr.options.Trackers[0] != "a" || tr.options.Trackers[1] != "b" {
		t.Fatalf("unexpected trackers: %#v", tr.options.Trackers)
	}
	if len(tr.options.Categories) != 2 || tr.options.Categories[0] != 1 || tr.options.Categories[1] != 2 {
		t.Fatalf("unexpected categories: %#v", tr.options.Categories)
	}
}

type fakeTracker struct {
	results []jackett.SearchResult
	query   string
	options tracker.SearchOptions
}

func (f *fakeTracker) Search(query string, options ...tracker.SearchOption) ([]jackett.SearchResult, error) {
	f.query = query
	for _, option := range options {
		option(&f.options)
	}
	return f.results, nil
}
