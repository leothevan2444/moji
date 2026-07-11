package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
