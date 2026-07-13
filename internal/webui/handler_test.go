package webui

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestHandlerReturnsDevelopmentHintWhenDistIsMissing(t *testing.T) {
	handler := NewHandler(filepath.Join(t.TempDir(), "missing"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); body != notBuiltMessage {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestHandlerServesBuiltIndexAndAssets(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<div id=\"root\"></div>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dist, "assets", "app.js"), []byte("console.log('moji')"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	handler := NewHandler(dist)

	for _, route := range []string{"/tasks", "/tasks/123", "/performers/123", "/settings/automation"} {
		indexRec := httptest.NewRecorder()
		handler.ServeHTTP(indexRec, httptest.NewRequest(http.MethodGet, route, nil))
		if body := indexRec.Body.String(); !strings.Contains(body, "root") {
			t.Fatalf("expected SPA fallback index for %s, got %q", route, body)
		}
	}

	assetRec := httptest.NewRecorder()
	handler.ServeHTTP(assetRec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if body := assetRec.Body.String(); !strings.Contains(body, "moji") {
		t.Fatalf("expected asset body, got %q", body)
	}
}

func TestHandlerSetsCachePolicyForDocumentsAndHashedAssets(t *testing.T) {
	dist := buildTestDist(t)
	handler := NewHandler(dist)

	for _, route := range []string{"/", "/tasks/123", "/favicon.svg"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, route, nil))
		if got := recorder.Header().Get("Cache-Control"); got != revalidateCaching {
			t.Fatalf("expected %s for %s, got %s", revalidateCaching, route, got)
		}
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/app-AbCdEf12.js", nil))
	if got := recorder.Header().Get("Cache-Control"); got != immutableCaching {
		t.Fatalf("expected immutable caching for hashed asset, got %s", got)
	}
}

func TestHandlerNegotiatesBrotliAndGzip(t *testing.T) {
	dist := buildTestDist(t)
	handler := NewHandler(dist)
	expected, err := os.ReadFile(filepath.Join(dist, "assets", "app-AbCdEf12.js"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		acceptEncoding string
		wantEncoding   string
		readBody       func(io.Reader) ([]byte, error)
	}{
		{
			name:           "brotli preferred",
			acceptEncoding: "gzip, br",
			wantEncoding:   "br",
			readBody:       func(reader io.Reader) ([]byte, error) { return io.ReadAll(brotli.NewReader(reader)) },
		},
		{
			name:           "gzip fallback",
			acceptEncoding: "br;q=0, gzip;q=1",
			wantEncoding:   "gzip",
			readBody: func(reader io.Reader) ([]byte, error) {
				gzipReader, err := gzip.NewReader(reader)
				if err != nil {
					return nil, err
				}
				defer gzipReader.Close()
				return io.ReadAll(gzipReader)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/assets/app-AbCdEf12.js", nil)
			request.Header.Set("Accept-Encoding", test.acceptEncoding)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if got := recorder.Header().Get("Content-Encoding"); got != test.wantEncoding {
				t.Fatalf("expected encoding %s, got %s", test.wantEncoding, got)
			}
			if got := recorder.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
				t.Fatalf("expected Accept-Encoding in Vary, got %s", got)
			}
			decoded, err := test.readBody(recorder.Body)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if string(decoded) != string(expected) {
				t.Fatal("decoded response did not match the source asset")
			}
		})
	}
}

func TestHandlerLeavesSmallResponsesUncompressed(t *testing.T) {
	dist := buildTestDist(t)
	handler := NewHandler(dist)
	request := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("expected small response to remain uncompressed, got %s", got)
	}
}

func buildTestDist(t *testing.T) string {
	t.Helper()
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<div id=\"root\"></div>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "favicon.svg"), []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("console.log('compressible asset');\n", 128)
	if err := os.WriteFile(filepath.Join(dist, "assets", "app-AbCdEf12.js"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dist
}
