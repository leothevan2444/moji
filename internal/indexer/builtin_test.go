package indexer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

const siteFixture = `id: example
name: Example
description: Synthetic account and search
language: en-US
type: private
encoding: UTF-8
links: [https://site.invalid/]
caps:
  categories: {film: Movies/HD}
  modes: {search: [q], movie-search: [q, imdbid]}
settings:
  - {name: username, type: text, label: Username}
  - {name: password, type: password, label: Password}
login:
  path: /login
  method: post
  inputs: {username: '{{ .Config.username }}', password: '{{ .Config.password }}'}
  test: {path: /account, selector: '#member'}
search:
  paths: [{path: /search}]
  inputs: {q: '{{ .Keywords }}', cat: '{{ join .Categories "," }}'}
  rows: {selector: article}
  fields:
    title: {selector: h2}
    category: {text: film}
    size: {text: 1GB}
    seeders: {text: 3}
    leechers: {text: 2}
    download: {selector: a, attribute: href}
`

type fixedBundles struct{ bundle *indexerdefs.Bundle }

func (f fixedBundles) Acquire(ctx context.Context, revision string) (*indexerdefs.Lease, error) {
	if revision != "" && revision != f.bundle.Manifest.Revision {
		return nil, indexerdefs.ErrNotInstalled
	}
	return &indexerdefs.Lease{Bundle: f.bundle}, ctx.Err()
}

func fixtureInstance() InstanceSnapshot {
	username, password := "synthetic-user", "synthetic-password"
	return InstanceSnapshot{ID: "one", Kind: Builtin, Enabled: true, Name: "Synthetic", DefinitionID: "example", ConfigRevision: "one", BaseURL: "https://site.invalid/", Settings: map[string]SettingValue{"username": {String: &username}, "password": {String: &password}}}
}

func TestBuiltinSearchAndFetchKeepAccountSessionAndRestoreCookies(t *testing.T) {
	for _, base := range []string{"https://site.invalid/", ""} {
		name := "explicit-mirror"
		if base == "" {
			name = "rule-default"
		}
		t.Run(name, func(t *testing.T) { testBuiltinSearchAndFetchSession(t, base) })
	}
}
func testBuiltinSearchAndFetchSession(t *testing.T, base string) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	var logins, downloads, searches atomic.Int32
	var expectedOriginal atomic.Int32
	expectedOriginal.Store(2000)
	var expire atomic.Bool
	network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/login":
			logins.Add(1)
			return httpResponse(200, http.Header{"Set-Cookie": []string{"account=synthetic; Path=/; Secure; HttpOnly"}}, "ok"), nil
		case "/account":
			if !strings.Contains(request.Header.Get("Cookie"), "account=synthetic") {
				t.Error("login verification lost cookie")
			}
			return httpResponse(200, http.Header{"Content-Type": []string{"text/html"}}, `<div id="member">user</div>`), nil
		case "/search":
			searches.Add(1)
			if !strings.Contains(request.Header.Get("Cookie"), "account=synthetic") {
				t.Error("search lost account cookie")
			}
			if expire.Swap(false) {
				return httpResponse(401, nil, "login required"), nil
			}
			if request.URL.Query().Get("cat") != "film" {
				t.Error("Torznab category not mapped to rule category")
			}
			if request.URL.Query().Get("limit") != "100" || request.URL.Query().Get("offset") != "0" {
				t.Error("bounded service pagination lost at interpreter boundary", request.URL.Query())
			}
			if request.URL.Query().Get("original") != fmt.Sprint(expectedOriginal.Load()) {
				t.Error("original Torznab category identity changed", request.URL.Query().Get("original"))
			}
			return httpResponse(200, http.Header{"Content-Type": []string{"text/html"}}, `<div id="member"></div><article><h2>ABCD-123</h2><a href="/download">get</a></article>`), nil
		case "/download":
			downloads.Add(1)
			if !strings.Contains(request.Header.Get("Cookie"), "account=synthetic") {
				t.Error("download lost account cookie")
			}
			return httpResponse(200, nil, "d4:infod4:name8:ABCD-1236:lengthi1eee"), nil
		default:
			t.Error("unexpected endpoint")
			return httpResponse(404, nil, ""), nil
		}
	})
	sessions, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	definition := strings.Replace(siteFixture, `cat: '{{ join .Categories "," }}'`, `cat: '{{ join .Categories "," }}', original: '{{ join .Query.Categories "," }}', limit: '{{ .Query.Limit }}', offset: '{{ .Query.Offset }}'`, 1)
	plan, err := cardigann.Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	bundles := fixedBundles{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("a", 40)}, Plans: map[string]*cardigann.Plan{"example": plan}}}
	adapter, err := NewBuiltinAdapter(bundles, sessions)
	if err != nil {
		t.Fatal(err)
	}
	instance := fixtureInstance()
	instance.BaseURL = base
	batch, err := adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Categories: []int{2000}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if batch.release != nil {
		defer batch.release()
	}
	if len(batch.Releases) != 1 {
		t.Fatal("search did not produce candidate")
	}
	release := batch.Releases[0]
	if release.Resource.DefinitionDigest != plan.Digest || release.Seeders == nil || *release.Seeders != 3 || release.Peers == nil || *release.Peers != 5 || len(release.Categories) != 1 || release.Categories[0] != 2040 {
		t.Fatal("candidate projection lost source facts")
	}
	artifact, err := adapter.Fetch(ctx, instance, release.Resource)
	if err != nil || string(artifact.TorrentBytes) != "d4:infod4:name8:ABCD-1236:lengthi1eee" {
		t.Fatal("resource acquisition changed bytes", err)
	}
	if logins.Load() != 1 || downloads.Load() != 1 {
		t.Fatal("search/fetch repeated login or grab")
	}
	restarted, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	adapter.sessions = restarted
	expectedOriginal.Store(2040)
	batch, err = adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Categories: []int{2040}, Limit: 100})
	if err != nil {
		t.Fatal("saved session not restored", err)
	}
	if batch.release != nil {
		batch.release()
	}
	if logins.Load() != 1 {
		t.Fatal("restart unnecessarily logged in")
	}
	expire.Store(true)
	expectedOriginal.Store(2000)
	batch, err = adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Categories: []int{2000}, Limit: 100})
	if err != nil {
		t.Fatal("expired session not reauthenticated", err)
	}
	if batch.release != nil {
		batch.release()
	}
	if logins.Load() != 2 || downloads.Load() != 1 {
		t.Fatal("reauthentication changed grab count")
	}
	before := searches.Load()
	if _, err := adapter.Search(ctx, instance, Query{Parameters: map[string]string{"TMDBID": "bad"}, Limit: 100}); !errors.Is(err, ErrInvalidQuery) || ErrorCode(err) != "INDEXER_QUERY_INVALID" {
		t.Fatal("invalid typed query lost its stable error code", err)
	}
	if searches.Load() != before {
		t.Fatal("invalid typed query issued a search request")
	}
	tampered := release.Resource
	tampered.DefinitionDigest = "wrong"
	if _, err := adapter.Fetch(ctx, instance, tampered); err != ErrReference {
		t.Fatal("changed definition reference accepted", err)
	}
}
