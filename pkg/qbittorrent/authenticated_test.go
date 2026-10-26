package qbittorrent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTorrentCommandsRecoverExpiredSessionsAndRejectFailedLogin(t *testing.T) {
	cfg := Config{URL: "http://qbt", Username: "user", Password: "password"}
	client := NewClient(func() Config { return cfg })
	loggedIn, denyLogin, timeout := false, false, false
	logins, mutations := 0, 0
	client.httpClient.Transport = fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		if timeout {
			return nil, errors.New("timeout")
		}
		status, body := 200, "Ok."
		if r.URL.Path == "/api/v2/auth/login" {
			logins++
			if denyLogin {
				body = "Fails."
			} else {
				loggedIn = true
			}
		} else if !loggedIn {
			status = 403
		} else if r.URL.Path == "/api/v2/torrents/start" {
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("hashes") != "hash" {
				t.Fatal("command body lost after authentication")
			}
			mutations++
		} else {
			body = "[]"
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	ctx := context.Background()
	if err := client.ResumeTorrents(ctx, []string{"hash"}); err != nil {
		t.Fatal(err)
	}
	if logins != 1 || mutations != 1 {
		t.Fatal("wrong auth/mutation count", logins, mutations)
	}
	loggedIn = false
	denyLogin = true
	if _, err := client.GetTorrentList(ctx, nil); err == nil {
		t.Fatal("accepted login failure with HTTP 200")
	}
	timeout = true
	before := logins
	if err := client.ResumeTorrents(ctx, []string{"hash"}); err == nil {
		t.Fatal("ignored timeout")
	}
	if logins != before {
		t.Fatal("retried ambiguous mutation failure")
	}
}
