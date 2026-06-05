package jackett

import (
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
