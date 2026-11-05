package indexer

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestConnectionProbeDoesNotSearchOrFetch(t *testing.T) {
	for _, kind := range []Kind{Builtin, Jackett} {
		t.Run(string(kind), func(t *testing.T) {
			calls := 0
			network := testNetwork(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" {
					t.Fatal("unexpected method")
				}
				if kind == Builtin {
					if req.URL.Path != "/site/" || req.URL.RawQuery != "" {
						t.Fatalf("not a homepage probe: %s", req.URL.Path)
					}
					return httpResponse(200, nil, "<html>Login</html>"), nil
				}
				if req.URL.Path != "/jackett/api/v2.0/indexers/example/results/torznab/api" || req.URL.Query().Get("t") != "caps" || req.URL.Query().Get("apikey") != "synthetic" {
					t.Fatal("not a scoped caps probe")
				}
				return httpResponse(200, nil, "<caps><searching/></caps>"), nil
			})
			base := "https://example.test/site/"
			if kind == Jackett {
				base = "https://example.test/jackett/"
			}
			err := network.CheckConnection(context.Background(), InstanceSnapshot{Kind: kind, BaseURL: base, RemoteID: "example", JackettAPIKey: "synthetic"}, nil)
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}
func TestConnectionProbeRejectsHTTPFailuresAndInvalidJackettDocuments(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   error
	}{
		{401, "unauthorized", ErrAuthentication}, {503, "unavailable", ErrTransport}, {200, "<html>login</html>", ErrTransport}, {200, "<error/>", ErrTransport},
	} {
		network := testNetwork(t, func(*http.Request) (*http.Response, error) { return httpResponse(tc.status, nil, tc.body), nil })
		err := network.CheckConnection(context.Background(), InstanceSnapshot{Kind: Jackett, BaseURL: "https://example.test", RemoteID: "site", JackettAPIKey: "synthetic"}, nil)
		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: %v", tc.status, err)
		}
	}
	network := testNetwork(t, func(*http.Request) (*http.Response, error) { t.Fatal("unsafe request sent"); return nil, nil })
	if err := network.CheckConnection(context.Background(), InstanceSnapshot{Kind: Builtin, BaseURL: "http://127.0.0.1"}, nil); !errors.Is(err, ErrUnsafeURL) {
		t.Fatal(err)
	}
}
