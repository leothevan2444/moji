package qbittorrent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDownloadServerStateAndCommandCompatibility(t *testing.T) {
	client := NewClient(nil)
	client.baseURL = "http://qbt"
	legacy := false
	calls := []string{}
	client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		status, body := 200, "Ok."
		switch r.URL.Path {
		case "/api/v2/sync/maindata":
			body = `{"server_state":{"free_space_on_disk":12345,"dl_info_speed":999}}`
		case "/api/v2/torrents/stop":
			if legacy {
				status = 404
			}
		case "/api/v2/transfer/setDownloadLimit":
			status = 403
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	ctx := context.Background()
	state, err := client.GetDownloadServerState(ctx)
	if err != nil || state.FreeSpace == nil || *state.FreeSpace != 12345 || state.DownloadSpeed != 999 {
		t.Fatal("wrong server stats", state, err)
	}
	legacy = true
	if err := client.PauseTorrents(ctx, []string{"hash"}); err != nil {
		t.Fatal(err)
	}
	if calls[len(calls)-1] != "/api/v2/torrents/pause" {
		t.Fatal("no v4 fallback")
	}
	if err := client.SetDownloadSpeedLimit(ctx, false, 1024); err == nil {
		t.Fatal("speed limit HTTP failure ignored")
	}
}

func TestTorrentFailuresCannotMasqueradeAsEmptyListsOrSuccessfulCommands(t *testing.T) {
	client := NewClient(nil)
	client.baseURL = "http://qbt"
	body, status := "[]", 500
	client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	ctx := context.Background()
	if _, err := client.GetTorrentList(ctx, nil); err == nil {
		t.Fatal("error response treated as empty list")
	}
	for _, command := range []func() error{func() error { return client.DeleteTorrents(ctx, []string{"hash"}, false) }, func() error { return client.RecheckTorrents(ctx, []string{"hash"}) }, func() error { return client.ReannounceTorrents(ctx, []string{"hash"}) }} {
		if command() == nil {
			t.Fatal("HTTP failure accepted")
		}
	}
	status = 200
	body = "null"
	if _, err := client.GetTorrentList(ctx, nil); err == nil {
		t.Fatal("null treated as confirmed empty list")
	}
}
