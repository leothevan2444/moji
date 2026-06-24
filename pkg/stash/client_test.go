package stash

import (
	"context"
	"os"
	"testing"
)

func requireStashClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("MOJI_RUN_INTEGRATION") != "1" {
		t.Skip("set MOJI_RUN_INTEGRATION=1 to run Stash integration tests")
	}

	host := os.Getenv("MOJI_STASH_GRAPHQL_URL")
	if host == "" {
		t.Skip("set MOJI_STASH_GRAPHQL_URL to run Stash integration tests")
	}

	return NewClient(func() Config { return Config{URL: host, APIKey: os.Getenv("MOJI_STASH_API_KEY")} })
}

func TestClient(t *testing.T) {
	client := requireStashClient(t)
	version, err := client.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	t.Logf("Version: %+v", *version)
}
