package jackett

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func requireJackettClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("MOJI_RUN_INTEGRATION") != "1" {
		t.Skip("set MOJI_RUN_INTEGRATION=1 to run Jackett integration tests")
	}

	host := os.Getenv("MOJI_JACKETT_URL")
	apiKey := os.Getenv("MOJI_JACKETT_API_KEY")
	if host == "" || apiKey == "" {
		t.Skip("set MOJI_JACKETT_URL and MOJI_JACKETT_API_KEY to run Jackett integration tests")
	}

	return NewClient(host, apiKey, os.Getenv("MOJI_JACKETT_PASSWORD"))
}

func TestClientSearch(t *testing.T) {
	client := requireJackettClient(t)
	results, err := client.Search(SearchRequest{
		Query:    "SONE-786",
		Trackers: []string{"sukebeinyaasi", "onejav", "u3c3"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result, got none")
	}
	for _, result := range results {
		t.Logf("Tracker: %s, Title: %s, PublishDate: %s, Details: %s, Category: %v, Size: %d, InfoHash: %s, MagnetURI: %s",
			result.Tracker, result.Title, result.PublishDate, result.Details,
			result.Category, result.Size, result.InfoHash, result.MagnetURI)
	}
	t.Logf("Search completed successfully with %d results", len(results))
}

func TestClient_GetIndexersReal(t *testing.T) {
	client := requireJackettClient(t)
	indexers, err := client.GetIndexers()
	if err != nil {
		t.Fatal(err)
	}
	if len(indexers) == 0 {
		t.Fatal("expected at least one indexer, got none")
	}
	for _, indexer := range indexers {
		if indexer.Configured {
			t.Logf("Indexer ID: %s, Name: %s, Description: %s, Type: %s, Configured: %t, SiteLink: %s, Language: %s",
				indexer.ID, indexer.Name, indexer.Description, indexer.Type,
				indexer.Configured, indexer.SiteLink, indexer.Language)
		}
	}
	t.Logf("GetIndexers completed successfully with %d indexers", len(indexers))
}

func TestAPIIndexerListingUsesKeyWithoutDashboardSession(t *testing.T) {
	for _, body := range []string{
		`<indexers><indexer id="site" configured="true"><title>Site</title><description>A tracker</description><link>https://tracker.example/</link><language>en-US</language><type>private</type></indexer></indexers>`,
		`<indexers />`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/prefix/api/v2.0/indexers/all/results/torznab/api" || r.URL.Query().Get("apikey") != "key" || r.URL.Query().Get("t") != "indexers" || r.URL.Query().Get("configured") != "true" {
				t.Errorf("unexpected request path: %s", r.URL.Path)
				http.Error(w, "dashboard requires login", http.StatusUnauthorized)
				return
			}
			if len(r.Cookies()) != 0 {
				t.Error("unexpected dashboard cookie")
			}
			_, _ = w.Write([]byte(body))
		}))
		client := NewClient(server.URL+"/prefix", "key", "")
		sites, err := client.GetAPIIndexersContext(context.Background())
		client.CloseIdleConnections()
		server.Close()
		if err != nil || sites == nil {
			t.Fatalf("sites=%v err=%v", sites, err)
		}
		if len(sites) > 0 && (sites[0].Type != "private" || sites[0].SiteLink != "https://tracker.example/" || sites[0].Language != "en-US" || !sites[0].Configured) {
			t.Fatalf("metadata missing: %+v", sites[0])
		}
	}
}

func TestAPIIndexerListingRejectsErrorAndLoginPages(t *testing.T) {
	for _, body := range []string{`<error code="100" description="Invalid API Key"/>`, `<html>Login</html>`, `null`, `<indexers><indexer id="missing-title"/></indexers>`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		client := NewClient(server.URL, "key", "")
		_, err := client.GetAPIIndexersContext(context.Background())
		client.CloseIdleConnections()
		server.Close()
		if err == nil {
			t.Errorf("accepted invalid response: %s", body)
		}
	}
}

func TestSearchResultSeederPresence(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"Title":"sample","Seeders":0}`, 0},
		{`{"Title":"sample","Seeders":17}`, 17},
		{`{"Title":"sample"}`, -1},
		{`{"Title":"sample","Seeders":null}`, -1},
		{`{"Title":"sample","Seeders":-5}`, -1},
	} {
		t.Run(tc.body, func(t *testing.T) {
			var result SearchResult
			if err := json.Unmarshal([]byte(tc.body), &result); err != nil {
				t.Fatal(err)
			}
			if result.Seeders != tc.want || result.Title != "sample" {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}
