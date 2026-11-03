package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/flaresolverr"
)

func TestSolverWireUsesExplicitServiceOriginAndKeepsProtocol500Distinct(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.Host != "solver.internal:8191" || r.URL.Path != "/prefix/v1" || r.Header.Get("Cookie") != "" || r.Header.Get("Proxy-Authorization") != "" {
			t.Fatal("solver endpoint widened or received ambient credentials")
		}
		var c struct {
			Cmd   string
			Proxy *flaresolverr.Proxy
		}
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Cmd != "request.get" || c.Proxy == nil || c.Proxy.Password != "private-proxy" {
			t.Fatal("explicit solver request scope lost", err)
		}
		return httpResponse(500, nil, `{"status":"error","message":"private target url"}`), nil
	})
	protocol, closeClient, err := network.newSolverClient("http://solver.internal:8191/prefix")
	if err != nil {
		t.Fatal(err)
	}
	request := flaresolverr.Request{URL: "https://site.example/", Timeout: 10 * time.Second, Proxy: &flaresolverr.Proxy{URL: "http://proxy.example", Password: "private-proxy"}}
	for i := 0; i < 2; i++ {
		if _, err = protocol.Solve(context.Background(), request); !errors.Is(err, flaresolverr.ErrRejected) {
			t.Fatal("protocol rejection became network cooldown", err)
		}
	}
	if calls != 2 {
		t.Fatal("solver replayed request or created remote session")
	}
	closeClient()
	if _, err = protocol.Solve(context.Background(), request); err == nil || calls != 2 {
		t.Fatal("closed solver transport reopened", err)
	}
}
func TestSolverWireDoesNotFollowServiceRedirectOrForwardPayloadToAnotherOrigin(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(*http.Request) (*http.Response, error) {
		calls++
		return httpResponse(302, http.Header{"Location": []string{"http://other.internal/collect"}}, ""), nil
	})
	client, closeClient, err := network.newSolverClient("http://solver.internal:8191/")
	if err != nil {
		t.Fatal(err)
	}
	defer closeClient()
	_, err = client.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second})
	if !errors.Is(err, flaresolverr.ErrProtocol) || calls != 1 {
		t.Fatal("solver payload followed redirect", err, calls)
	}
}

func TestSolverWireActualJSONRoundTripUsesOnlyExplicitPrivateService(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/v1" || r.Header.Get("Cookie") != "" {
			t.Error("solver wire request widened endpoint or ambient account")
		}
		var request struct {
			Cmd     string
			Cookies []flaresolverr.Cookie
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Cmd != "request.get" || len(request.Cookies) != 1 || request.Cookies[0].Value != "synthetic-account" {
			t.Error("JSON protocol request lost scope", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","solution":{"url":"https://site.example/","status":200,"response":"ok","cookies":[{"name":"cf_clearance","value":"synthetic-clearance"}],"userAgent":"Synthetic browser"}}`)
	}))
	defer server.Close()
	network := NewNetwork(context.Background())
	defer network.Close()
	client, closeClient, err := network.newSolverClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer closeClient()
	solution, err := client.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second, Cookies: []flaresolverr.Cookie{{Name: "account", Value: "synthetic-account"}}})
	if err != nil || solution.UserAgent != "Synthetic browser" || len(solution.Cookies) != 1 || calls.Load() != 1 {
		t.Fatal("actual solver wire round trip failed", err)
	}
	network.mu.Lock()
	for transport := range network.transports {
		if transport.ResponseHeaderTimeout != 0 {
			t.Error("normal 30s header deadline preempted challenge budget")
		}
	}
	network.mu.Unlock()
	closeClient()
	if _, err = client.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second}); err == nil || calls.Load() != 1 {
		t.Fatal("closed solver wire reopened private service", err)
	}
}
func TestSolverWirePreservesEncodedReverseProxyPrefixAndConfiguredQuery(t *testing.T) {
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.EscapedPath() != "/prefix%2Fprivate/v1" || r.URL.RawQuery != "token=synthetic-service-key" {
			t.Fatal("reverse proxy path or service authentication query changed")
		}
		return httpResponse(500, nil, `{"status":"error"}`), nil
	})
	client, closeClient, err := network.newSolverClient("http://solver.internal/prefix%2Fprivate?token=synthetic-service-key")
	if err != nil {
		t.Fatal(err)
	}
	defer closeClient()
	if _, err = client.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second}); !errors.Is(err, flaresolverr.ErrRejected) {
		t.Fatal(err)
	}
}
func TestSolverWireSharesOnlyEndpointTransportAndReleasesAfterLastOwner(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(*http.Request) (*http.Response, error) {
		calls++
		return httpResponse(500, nil, `{"status":"error"}`), nil
	})
	first, closeFirst, err := network.newSolverClient("http://solver.internal:8191")
	if err != nil {
		t.Fatal(err)
	}
	second, closeSecond, err := network.newSolverClient("http://solver.internal:8191/")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("same endpoint did not share transaction admission")
	}
	closeFirst()
	closeFirst()
	if _, err = second.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second}); !errors.Is(err, flaresolverr.ErrRejected) || calls != 1 {
		t.Fatal("one owner closed another's endpoint", err)
	}
	closeSecond()
	if _, err = second.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second}); err == nil || calls != 1 {
		t.Fatal("released endpoint reopened")
	}
}

func TestNetworkCloseInvalidatesBorrowedSolverAndDropsEndpointReferences(t *testing.T) {
	network := testNetwork(t, func(*http.Request) (*http.Response, error) { t.Fatal("closed network reached solver"); return nil, nil })
	client, release, err := network.newSolverClient("http://solver.internal")
	if err != nil {
		t.Fatal(err)
	}
	network.Close()
	if _, err = client.Solve(context.Background(), flaresolverr.Request{URL: "https://site.example/", Timeout: time.Second}); err == nil {
		t.Fatal("closed network solver remained usable")
	}
	network.mu.Lock()
	retained := len(network.solvers)
	network.mu.Unlock()
	if retained != 0 {
		t.Fatal("closed network retained solver endpoint state")
	}
	release()
	release()
	if _, _, err = network.newSolverClient("http://solver.internal"); !errors.Is(err, context.Canceled) {
		t.Fatal("closed network rebuilt solver", err)
	}
}
