package indexer

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestBuiltinSearchClampsFutureDatesWithoutChangingDownloadFields(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	const definition = `id: example
name: Example
description: Synthetic publication dates
language: en-US
type: public
encoding: UTF-8
links: [https://site.invalid/]
caps:
  categories: {Movies: Movies}
  modes: {search: [q]}
search:
  paths: [{path: /search}]
  rows: {selector: article}
  fields:
    title: {selector: h2}
    category: {text: Movies}
    size: {text: 1GB}
    seeders: {text: 0}
    date: {selector: time, optional: true}
    download: {selector: a, attribute: href}
download:
  before:
    path: /prepare
    method: get
    inputs: {date: '{{ .Result.date }}'}
`
	plan, err := cardigann.Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	prepared := false
	network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/search":
			return httpResponse(200, nil, `<article><h2>Future A</h2><time>9999-12-31T00:00:00Z</time><a href="/download">get</a></article>
<article><h2>Future B</h2><time>9999-12-31T00:00:00Z</time><a href="/download">get</a></article>
<article><h2>Past</h2><time>2001-01-02T00:00:00Z</time><a href="/download">get</a></article>
<article><h2>Unknown</h2><a href="/download">get</a></article>`), nil
		case "/prepare":
			prepared = true
			if !strings.Contains(request.URL.Query().Get("date"), "9999") {
				t.Error("download template lost original future date")
			}
			return httpResponse(200, nil, "ok"), nil
		case "/download":
			return httpResponse(200, nil, "d4:infod4:name8:ABCD-1236:lengthi1eee"), nil
		default:
			t.Fatalf("unexpected request: %s", request.URL.Path)
			return nil, nil
		}
	})
	sessions, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	bundles := fixedBundles{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("a", 40)}, Plans: map[string]*cardigann.Plan{"example": plan}}}
	adapter, err := NewBuiltinAdapter(bundles, sessions)
	if err != nil {
		t.Fatal(err)
	}
	instance := fixtureInstance()
	instance.Settings = nil
	before := time.Now().UTC()
	batch, err := adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Limit: 100})
	after := time.Now().UTC()
	if err != nil {
		t.Fatal(err)
	}
	if batch.release != nil {
		defer batch.release()
	}
	if len(batch.Releases) != 4 {
		t.Fatalf("got %d releases, want 4", len(batch.Releases))
	}
	for _, release := range batch.Releases[:2] {
		if release.PublishedAt == nil || release.PublishedAt.Before(before) || release.PublishedAt.After(after) {
			t.Fatalf("future publication not clamped to retrieval: %v, interval [%v, %v]", release.PublishedAt, before, after)
		}
		if !strings.Contains(release.Resource.Locator.Fields["date"], "9999") {
			t.Fatal("resource locator lost original future date")
		}
	}
	if !batch.Releases[0].PublishedAt.Equal(*batch.Releases[1].PublishedAt) {
		t.Fatal("one search batch used different retrieval timestamps")
	}
	if batch.Releases[0].PublishedAt == batch.Releases[1].PublishedAt {
		t.Fatal("release dates share a mutable pointer")
	}
	past := time.Date(2001, 1, 2, 0, 0, 0, 0, time.UTC)
	if got := batch.Releases[2].PublishedAt; got == nil || !got.Equal(past) {
		t.Fatalf("historical publication changed: %v", got)
	}
	if batch.Releases[3].PublishedAt != nil {
		t.Fatal("unknown publication was invented")
	}
	if _, err := adapter.Fetch(ctx, instance, batch.Releases[0].Resource); err != nil {
		t.Fatal(err)
	}
	if !prepared {
		t.Fatal("download preparation was skipped")
	}
}
