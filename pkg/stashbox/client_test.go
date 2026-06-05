package stashbox

import (
	"context"
	"os"
	"testing"
)

func requireStashboxClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("MOJI_RUN_INTEGRATION") != "1" {
		t.Skip("set MOJI_RUN_INTEGRATION=1 to run StashBox integration tests")
	}

	apiKey := os.Getenv("MOJI_STASHBOX_API_KEY")
	if apiKey == "" {
		t.Skip("set MOJI_STASHBOX_API_KEY to run StashBox integration tests")
	}

	return NewClient(apiKey)
}

func TestClient(t *testing.T) {
	client := requireStashboxClient(t)
	me, err := client.Me(context.Background())
	if err != nil {
		t.Fatalf("failed to get me: %v", err)
	}
	t.Logf("Me: %+v", me)
	version, err := client.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	t.Logf("Version: %+v", version)
}
