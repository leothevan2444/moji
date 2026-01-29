package qbittorrent

import (
	"context"
	"testing"
)

var host = "http://localhost:8078"
var username = "admin"
var password = "010728sR..."

var client *Client

func TestMain(m *testing.M) {
	client = NewClient(host)
	err := client.Login(context.Background(), username, password)
	if err != nil {
		panic("Failed to login: " + err.Error())
	}
	m.Run()
}

func TestGetLog(t *testing.T) {
	logs, err := client.GetLog(context.Background(), LogTypeNormal, nil)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("Expected logs, got none")
	}
	t.Logf("Retrieved %d log entries", len(logs))
}

func TestGetPeerLog(t *testing.T) {
	peerLogs, err := client.GetPeerLog(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPeerLog failed: %v", err)
	}
	t.Logf("Retrieved %d peer log entries", len(peerLogs))
}

func TestGetMainData(t *testing.T) {
	mainData, err := client.GetMainData(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetMainData failed: %v", err)
	}
	if mainData == nil {
		t.Fatalf("Expected main data, got nil")
	}
	t.Logf("Retrieved main data with %d torrents", len(mainData.Torrents))
}
