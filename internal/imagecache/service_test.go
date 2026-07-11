package imagecache

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

var tinyPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func imageTransport(contentType string, body []byte, calls *atomic.Int32, check func(*http.Request)) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if check != nil {
			check(r)
		}
		h := make(http.Header)
		h.Set("Content-Type", contentType)
		h.Set("Cache-Control", "max-age=3600")
		return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
	})
}

func newTestService(t *testing.T, cfg Config) *Service {
	t.Helper()
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "moji.db"), filepath.Join(dir, "images"), func() Config { return cfg })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestProxyCachesAndForwardsStashAPIKey(t *testing.T) {
	var calls atomic.Int32
	s := newTestService(t, DefaultConfig())
	s.client.Transport = imageTransport("image/png", tinyPNG, &calls, func(r *http.Request) {
		if r.Header.Get("ApiKey") != "secret" {
			t.Errorf("missing api key")
		}
	})
	proxy, err := s.Register(context.Background(), Descriptor{Kind: SourceStash, InstanceURL: "http://stash.test", ImageURL: "/poster.png", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, proxy, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), tinyPNG) {
			t.Fatalf("unexpected response %d", rec.Code)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one upstream request, got %d", calls.Load())
	}
	status, _ := s.Status(context.Background())
	if status.EntryCount != 1 || status.UsedBytes == 0 {
		t.Fatalf("unexpected status %+v", status)
	}
	if _, err := s.Clear(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, _ = s.Status(context.Background())
	if status.EntryCount != 0 {
		t.Fatalf("cache not cleared")
	}
}

func TestProxyRejectsSVGAndUnknownKey(t *testing.T) {
	s := newTestService(t, DefaultConfig())
	var calls atomic.Int32
	s.client.Transport = imageTransport("image/svg+xml", []byte("<svg/>"), &calls, nil)
	proxy, _ := s.Register(context.Background(), Descriptor{Kind: SourceStash, InstanceURL: "http://stash.test", ImageURL: "/x.svg"})
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, proxy, nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/images/"+strings.Repeat("0", 64), nil))
	if rec.Code != http.StatusBadGateway && rec.Code != http.StatusNotFound {
		t.Fatalf("unexpected unknown response %d", rec.Code)
	}
}

func TestCacheDisabledDoesNotPersistFile(t *testing.T) {
	s := newTestService(t, Config{Enabled: false, MaxSizeMB: 64, RetentionDays: 1})
	var calls atomic.Int32
	s.client.Transport = imageTransport("image/png", tinyPNG, &calls, nil)
	proxy, _ := s.Register(context.Background(), Descriptor{Kind: SourceStash, InstanceURL: "http://stash.test", ImageURL: "/poster.png"})
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, proxy, nil))
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	status, _ := s.Status(context.Background())
	if status.EntryCount != 0 || status.UsedBytes != 0 {
		t.Fatalf("unexpected persisted cache %+v", status)
	}
}
