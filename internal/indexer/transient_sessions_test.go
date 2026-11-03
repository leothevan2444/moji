package indexer

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestTransientDiagnosticsAuthenticateIndependentlyWithoutReadingOrReplacingSavedCookies(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	logins, searches := 0, 0
	var observed []string
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/login":
			logins++
			return httpResponse(200, http.Header{"Set-Cookie": []string{"account=session-" + strconv.Itoa(logins) + "; Path=/; Secure; HttpOnly"}}, "ok"), nil
		case "/account":
			if !strings.Contains(r.Header.Get("Cookie"), "account=session-") {
				t.Error("authentication verification lost its own cookie")
			}
			return httpResponse(200, nil, `<div id="member"></div>`), nil
		case "/search":
			searches++
			observed = append(observed, r.Header.Get("Cookie"))
			return httpResponse(200, nil, `<div id="member"></div><article><h2>ABCD-123</h2><a href="/download">get</a></article>`), nil
		}
		t.Error("unexpected request", r.URL.Path)
		return httpResponse(404, nil, ""), nil
	})
	plan, err := cardigann.Load([]byte(siteFixture))
	if err != nil {
		t.Fatal(err)
	}
	bundles := fixedBundles{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("a", 40)}, Plans: map[string]*cardigann.Plan{"example": plan}}}
	instance := fixtureInstance()
	instance.BaseURL = ""
	persistent, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewBuiltinAdapter(bundles, persistent)
	if err != nil {
		t.Fatal(err)
	}
	if err = adapter.Authenticate(ctx, instance); err != nil || logins != 1 || searches != 0 {
		t.Fatal("standalone authentication unexpectedly searched", err, logins, searches)
	}
	persistent.Close()
	var before string
	if err = manager.Runtime.Reader.Get(&before, `SELECT payload_json FROM indexer_sessions`); err != nil {
		t.Fatal(err)
	}
	transient, err := NewTransientSessions(network)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err = NewBuiltinAdapter(bundles, transient)
	if err != nil {
		t.Fatal(err)
	}
	if err = adapter.Authenticate(ctx, instance); err != nil || logins != 2 || searches != 0 {
		t.Fatal("diagnostic reused the saved login", err, logins, searches)
	}
	batch, err := adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Limit: 20})
	batch.Close()
	if err != nil || logins != 2 || searches != 1 || !strings.Contains(observed[0], "session-2") {
		t.Fatal("diagnostic did not reuse only its own isolated login", err)
	}
	transient.Close()
	transient.Close()
	if err = adapter.Authenticate(ctx, instance); !errors.Is(err, context.Canceled) {
		t.Fatal("closed session collection reopened", err)
	}
	var after string
	if err = manager.Runtime.Reader.Get(&after, `SELECT payload_json FROM indexer_sessions`); err != nil || before != after {
		t.Fatal("transient check wrote account cookies to runtime storage", err)
	}
	restored, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	adapter, err = NewBuiltinAdapter(bundles, restored)
	if err != nil {
		t.Fatal(err)
	}
	batch, err = adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Limit: 20})
	batch.Close()
	if err != nil || logins != 2 || len(observed) != 2 || !strings.Contains(observed[1], "session-1") {
		t.Fatal("diagnostic replaced production account state", err, observed)
	}
}
