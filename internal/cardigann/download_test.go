package cardigann

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDownloadBeforePreservesEmptyAndWhitespaceInputs(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			definition := minimalDefinition + `download:
  before:
    path: /prepare
    method: ` + method + `
    inputs:
      token: ""
      blank: "  "
  selectors:
    - {selector: a, attribute: href, usebeforeresponse: true}
`
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			file := []byte("d4:infod4:name8:ABCD-123ee")
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				u, err := url.Parse(r.URL)
				if err != nil {
					t.Fatal(err)
				}
				if u.Path == "/prepare" {
					encoded := u.RawQuery
					values := u.Query()
					if method == "post" {
						encoded = string(r.Body)
						values, err = url.ParseQuery(string(r.Body))
						if err != nil {
							t.Fatal(err)
						}
					}
					if encoded != "token=&blank=++" {
						t.Errorf("before input order changed: %s", encoded)
					}
					if r.Method != strings.ToUpper(method) || !values.Has("token") || values.Get("token") != "" || values.Get("blank") != "  " {
						t.Fatalf("generic before inputs changed: %s %#v", r.Method, values)
					}
					return HTTPResponse{Status: 200, Body: []byte(`<a href="/file.torrent">file</a>`)}, nil
				}
				if u.Path != "/file.torrent" {
					t.Fatalf("unexpected request: %s", r.URL)
				}
				return HTTPResponse{Status: 200, Body: file}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := execution.Fetch(context.Background(), Result{DownloadURL: "https://example.invalid/details"})
			if err != nil || string(artifact.TorrentBytes) != string(file) || len(session.requests) != 2 {
				t.Fatalf("before-to-torrent flow: %v, %d requests", err, len(session.requests))
			}
		})
	}
}

func TestDirectDownloadPreservesDeclaredReferer(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition + `download:
  headers:
    Referer: [https://example.invalid/declared]
`))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		if r.Headers.Get("Referer") != "https://example.invalid/declared" {
			t.Error("declared referer overwritten", r.Headers.Get("Referer"))
		}
		return HTTPResponse{Status: 200, Body: []byte("d4:infod4:name8:ABCD-123ee")}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execution.Fetch(context.Background(), Result{DownloadURL: "https://example.invalid/file.torrent"}); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadUsesBeforeResponseAndReturnsTestedBytesOnce(t *testing.T) {
	definition := minimalDefinition + `download:
  before:
    path: /prepare
    method: post
    inputs:
      id: '{{ .DownloadUri.Query.id }}'
  selectors:
    - selector: a.download
      attribute: href
      usebeforeresponse: true
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	file := []byte("d4:infod4:name8:ABCD-123ee")
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		switch r.URL {
		case "https://example.invalid/prepare":
			if r.Method != http.MethodPost || string(r.Body) != "id=123" {
				t.Fatalf("wrong before request: %#v", r)
			}
			return HTTPResponse{Status: 200, Body: []byte(`<a class="download" href="/file.torrent">file</a>`)}, nil
		case "https://example.invalid/file.torrent":
			return HTTPResponse{Status: 200, Body: file}, nil
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return HTTPResponse{}, nil
		}
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := execution.Fetch(context.Background(), Result{Title: "ABCD-123", DownloadURL: "https://example.invalid/details?id=123"})
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.TorrentBytes) != string(file) || len(session.requests) != 2 {
		t.Fatalf("downloaded bytes changed or grabbed twice: %#v %#v", artifact, session.requests)
	}
}

func TestDownloadRedirectsToMagnetAndDoesNotReplayAmbiguousFailures(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition))
	if err != nil {
		t.Fatal(err)
	}
	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 302, Headers: http.Header{"Location": []string{magnet}}}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := execution.Fetch(context.Background(), Result{DownloadURL: "https://example.invalid/dl"})
	if err != nil || artifact.Magnet != magnet || len(session.requests) != 1 {
		t.Fatalf("magnet redirect: %#v %v", artifact, err)
	}
	definition := minimalDefinition + `download:
  selectors:
    - {selector: 'a.first', attribute: href}
    - {selector: 'a.second', attribute: href}
`
	plan, err = Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("ambiguous transport failure")
	session = &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		if r.URL == "https://example.invalid/details" {
			return HTTPResponse{Status: 200, Body: []byte(`<a class="first" href="/first"/><a class="second" href="/second"/>`)}, nil
		}
		return HTTPResponse{}, failure
	}}
	execution, err = plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = execution.Fetch(context.Background(), Result{DownloadURL: "https://example.invalid/details"}); !errors.Is(err, failure) || len(session.requests) != 2 {
		t.Fatalf("uncertain grab replayed: %v %#v", err, session.requests)
	}
}

func TestDownloadInfoHashAndInvalidMagnet(t *testing.T) {
	definition := minimalDefinition + `download:
  infohash:
    hash: {selector: code}
    title: {selector: h1}
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, Body: []byte(`<code>` + strings.Repeat("a", 40) + `</code><h1>ABCD-123</h1>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := execution.Fetch(context.Background(), Result{DownloadURL: "https://example.invalid/details"})
	if err != nil || !strings.Contains(artifact.Magnet, "dn=ABCD-123") {
		t.Fatalf("infohash: %#v %v", artifact, err)
	}
	if _, err = magnetArtifact("magnet:?xt=urn:btih:invalid"); err == nil {
		t.Fatal("invalid hash accepted")
	}
}
