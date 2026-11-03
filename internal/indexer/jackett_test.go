package indexer

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestJackettRejectsUnrepresentableQueryBeforeHTTP(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(*http.Request) (*http.Response, error) {
		calls++
		return httpResponse(200, nil, `{"Results":[],"Indexers":[]}`), nil
	})
	adapter, err := NewJackettAdapter(network)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()
	instance := InstanceSnapshot{Kind: Jackett, RemoteID: "one", BaseURL: "http://jackett.invalid", JackettAPIKey: "synthetic"}
	for _, query := range []Query{
		{Keywords: "ABCD-123", Parameters: map[string]string{"Year": "2026"}},
		{Keywords: "ABCD-123", Offset: 10},
		{Mode: "movie", Keywords: "ABCD-123"},
	} {
		if _, err := adapter.Search(context.Background(), instance, query); !errors.Is(err, ErrUnsupported) || calls != 0 {
			t.Fatal("external search silently ignored a condition or sent HTTP", err, calls)
		}
	}
}

func TestJackettUsesAPIKeyWithoutManagementLoginAndProjectsUnknownCounts(t *testing.T) {
	var endpoints []string
	network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
		endpoints = append(endpoints, request.URL.Path)
		if request.Header.Get("Cookie") != "" {
			t.Error("external search used management cookies")
		}
		if strings.Contains(request.URL.Path, "torznab") {
			if request.URL.Query().Get("apikey") != "synthetic" || request.URL.Query().Get("t") != "indexers" {
				t.Error("catalog did not use API discovery")
			}
			return httpResponse(200, nil, `<indexers><indexer id="one" configured="true"><title>One</title></indexer><indexer id="off" configured="false"><title>Off</title></indexer></indexers>`), nil
		}
		if request.URL.Path == "/api/v2.0/indexers/all/results" {
			if request.URL.Query().Get("Tracker[]") != "one" || request.URL.Query().Get("Query") != "ABCD-123" {
				t.Error("search scope changed")
			}
			return httpResponse(200, nil, `{"Results":[{"Title":"ABCD-123","Link":"http://127.0.0.1:9117/dl/one","Size":123}],"Indexers":[{"ID":"one","Status":200}]}`), nil
		}
		if request.URL.Path == "/dl/one" {
			return httpResponse(200, nil, "d4:infod4:name8:ABCD-1236:lengthi1eee"), nil
		}
		t.Error("unexpected endpoint")
		return httpResponse(404, nil, ""), nil
	})
	adapter, err := NewJackettAdapter(network)
	if err != nil {
		t.Fatal(err)
	}
	connection := InstanceSnapshot{Kind: Jackett, Enabled: true, BaseURL: "http://127.0.0.1:9117", JackettAPIKey: "synthetic", ConfigRevision: "one"}
	sites, err := adapter.Catalog(context.Background(), connection)
	if err != nil || len(sites) != 1 || sites[0].ID != "jackett/one" {
		t.Fatal("configured catalog failed", err)
	}
	batch, err := adapter.Search(context.Background(), sites[0], Query{Keywords: "ABCD-123"})
	if err != nil || len(batch.Releases) != 1 {
		t.Fatal("external search failed", err)
	}
	release := batch.Releases[0]
	if release.Seeders != nil || release.Peers != nil || release.Size == nil || *release.Size != 123 {
		t.Fatal("unknown counts were invented")
	}
	artifact, err := adapter.Fetch(context.Background(), sites[0], release.Resource)
	if err != nil || len(artifact.TorrentBytes) == 0 {
		t.Fatal("external private endpoint resource failed", err)
	}
	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "Dashboard") {
			t.Fatal("management login required")
		}
	}
}

func TestExternalEndpointGrantDoesNotFollowResourcesToOtherPrivateOrigins(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
		calls++
		return httpResponse(302, http.Header{"Location": []string{"http://10.0.0.9/private"}}, ""), nil
	})
	adapter, _ := NewJackettAdapter(network)
	instance := InstanceSnapshot{ID: "jackett/one", Kind: Jackett, RemoteID: "one", ConfigRevision: "one", BaseURL: "http://127.0.0.1:9117", JackettAPIKey: "synthetic"}
	ref := ResourceRef{Version: 1, ID: "resource", Kind: Jackett, InstanceID: instance.ID, ConfigRevision: "one", Locator: Locator{DownloadURL: instance.BaseURL + "/dl/one", MagnetURI: "magnet:?xt=urn:btih:1111111111111111111111111111111111111111"}}
	if _, err := adapter.Fetch(context.Background(), instance, ref); !errors.Is(err, ErrUnsafeURL) {
		t.Fatal("private-origin grant widened or failure fell back to magnet", err)
	}
	if calls != 1 {
		t.Fatal("redirect contacted another private endpoint")
	}
}

func TestJackettRemoteSiteFailureDoesNotBecomeGatewayRateLimit(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(*http.Request) (*http.Response, error) {
		calls++
		return httpResponse(200, nil, `{"Results":[],"Indexers":[{"ID":"one","Status":429,"Error":"remote site rate limit"},{"ID":"two","Status":200}]}`), nil
	})
	adapter, e := NewJackettAdapter(network)
	if e != nil {
		t.Fatal(e)
	}
	defer adapter.Close()
	instance := InstanceSnapshot{ID: "jackett/one", Kind: Jackett, RemoteID: "one", BaseURL: "http://jackett.invalid", JackettAPIKey: "synthetic"}
	batch, e := adapter.Search(context.Background(), instance, Query{Keywords: "ABCD-123"})
	if !errors.Is(e, ErrTransport) || errors.Is(e, ErrCoolingDown) || !batch.Partial {
		t.Fatal("remote site status confused with gateway admission", e)
	}
	instance.ID = "jackett/two"
	instance.RemoteID = "two"
	batch, e = adapter.Search(context.Background(), instance, Query{Keywords: "ABCD-123"})
	if e != nil || batch.Partial || calls != 2 {
		t.Fatal("one remote site failure blocked other configured sites", e, calls)
	}
}
