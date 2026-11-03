package cardigann

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFetchAlreadyCancelledDoesNotReturnAnArtifactOrSendRequests(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition))
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []Result{
		{MagnetURI: "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)},
		{InfoHash: strings.Repeat("a", 40), Title: "Example"},
		{DownloadURL: "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)},
		{DownloadURL: "https://example.invalid/file.torrent"},
	} {
		session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
			t.Fatal("cancelled fetch sent HTTP")
			return HTTPResponse{}, nil
		}}
		execution, err := plan.NewExecution(session, "", nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		artifact, err := execution.Fetch(ctx, result)
		if !errors.Is(err, context.Canceled) || artifact.Magnet != "" || len(artifact.TorrentBytes) != 0 || len(session.requests) != 0 {
			t.Fatalf("already cancelled fetch returned success: %+v %v", artifact, err)
		}
	}
}

func TestInfoHashSelectorPreservesCancellationCause(t *testing.T) {
	definition := minimalDefinition + `download:
  infohash:
    hash: {selector: code, filters: [{name: trim}]}
    title: {selector: h1}
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
		cancel()
		return HTTPResponse{Status: 200, Body: []byte("<code>" + strings.Repeat("a", 40) + "</code><h1>Example</h1>")}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := execution.Fetch(ctx, Result{DownloadURL: "https://example.invalid/details"})
	if !errors.Is(err, context.Canceled) || artifact.Magnet != "" || len(artifact.TorrentBytes) != 0 || len(session.requests) != 1 {
		t.Fatalf("selector hid cancellation or continued: %+v %v", artifact, err)
	}
}

func TestDownloadFallbackSelectorsDoNotHideCancellation(t *testing.T) {
	definition := minimalDefinition + `download:
  selectors:
    - {selector: a.first, attribute: href}
    - {selector: a.second, attribute: href}
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
		cancel()
		return HTTPResponse{Status: 200, Body: []byte(`<a class='first' href='/file.torrent'></a><a class='second' href='/other.torrent'></a>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := execution.Fetch(ctx, Result{DownloadURL: "https://example.invalid/details"})
	if !errors.Is(err, context.Canceled) || artifact.Magnet != "" || len(artifact.TorrentBytes) != 0 || len(session.requests) != 1 {
		t.Fatalf("fallback hid cancellation or continued: %+v %v", artifact, err)
	}
}

func TestFetchRetainsConfirmedTorrentReceivedAtCancellation(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	file := "d4:infod4:name8:ABCD-1236:lengthi1eee"
	session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
		cancel()
		return HTTPResponse{Status: 200, Body: []byte(file)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := execution.Fetch(ctx, Result{DownloadURL: "https://example.invalid/file.torrent"})
	// Keep a confirmed receipt so the outer cache can retain it while the
	// cancelled task's generation fence prevents accepting a late selection.
	if err != nil || string(artifact.TorrentBytes) != file || len(session.requests) != 1 {
		t.Fatalf("confirmed late grab was discarded: %+v %v", artifact, err)
	}
}
