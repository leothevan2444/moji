package qbittorrent

import (
	"context"
	"fmt"
	"os"
	"testing"
)

var client *Client

func TestMain(m *testing.M) {
	if os.Getenv("MOJI_RUN_INTEGRATION") != "1" {
		os.Exit(m.Run())
	}

	host := os.Getenv("MOJI_QBT_URL")
	username := os.Getenv("MOJI_QBT_USERNAME")
	password := os.Getenv("MOJI_QBT_PASSWORD")
	if host == "" || username == "" || password == "" {
		os.Exit(m.Run())
	}

	client = NewClient(func() Config { return Config{URL: host, Username: username, Password: password} })
	err := client.Login(context.Background(), username, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to login to qBittorrent: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func requireQBT(t *testing.T) *Client {
	t.Helper()
	if client == nil {
		t.Skip("set MOJI_RUN_INTEGRATION=1, MOJI_QBT_URL, MOJI_QBT_USERNAME, and MOJI_QBT_PASSWORD to run qBittorrent integration tests")
	}
	return client
}

func requireQBTTestHash(t *testing.T) string {
	t.Helper()
	hash := os.Getenv("MOJI_QBT_TEST_HASH")
	if hash == "" {
		t.Skip("set MOJI_QBT_TEST_HASH to run this qBittorrent integration test")
	}
	return hash
}

func requireDestructiveQBT(t *testing.T) *Client {
	t.Helper()
	c := requireQBT(t)
	if os.Getenv("MOJI_RUN_DESTRUCTIVE_QBT_TESTS") != "1" {
		t.Skip("set MOJI_RUN_DESTRUCTIVE_QBT_TESTS=1 to run qBittorrent tests that modify server state")
	}
	return c
}

func TestGetLog(t *testing.T) {
	c := requireQBT(t)
	logs, err := c.GetLog(context.Background(), LogTypeNormal, nil)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("Expected logs, got none")
	}
	t.Logf("Retrieved %d log entries", len(logs))
}

func TestGetPeerLog(t *testing.T) {
	c := requireQBT(t)
	peerLogs, err := c.GetPeerLog(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPeerLog failed: %v", err)
	}
	t.Logf("Retrieved %d peer log entries", len(peerLogs))
}

func TestGetMainData(t *testing.T) {
	c := requireQBT(t)
	mainData, err := c.GetMainData(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetMainData failed: %v", err)
	}
	if mainData == nil {
		t.Fatalf("Expected main data, got nil")
	}
	t.Logf("Retrieved main data with %d torrents", len(mainData.Torrents))
}
