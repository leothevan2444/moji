package qbittorrent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTransferPreferencesSparseFormPatch(t *testing.T) {
	c := NewClient(nil)
	c.baseURL = "http://qb"
	calls := 0
	c.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		body := `{"queueing_enabled":false,"max_ratio":1.5,"share_limits_mode":"MatchAll","save_path":"/downloads","web_ui_password":"must not escape"}`
		if r.Method == http.MethodPost {
			if r.URL.Path != "/api/v2/app/setPreferences" {
				t.Fatal(r.URL.Path)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal([]byte(r.Form.Get("json")), &fields); err != nil {
				t.Fatal(err)
			}
			if len(fields) != 2 || string(fields["queueing_enabled"]) != "false" || string(fields["max_ratio"]) != "1.5" {
				t.Fatalf("patch was not sparse: %v", fields)
			}
			body = "Ok."
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	p, err := c.GetTorrentPreferences(context.Background())
	if err != nil || p.QueueingEnabled == nil || *p.QueueingEnabled || p.ShareLimitMode == nil || *p.ShareLimitMode != "MatchAll" || p.MaxInactiveSeedingTime != nil {
		t.Fatal(p, err)
	}
	disabled := false
	ratio := 1.5
	if err := c.PatchTransferPreferences(context.Background(), TransferPreferences{QueueingEnabled: &disabled, MaxRatio: &ratio}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
