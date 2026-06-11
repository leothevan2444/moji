package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreUpdateQBittorrentPreservesUnmodeledSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `database:
  dsn: "user:password@tcp(localhost:3306)/media_library?parseTime=true"
qbittorrent:
  url: "http://localhost:8080"
  username: "admin"
  password: "secret"
tasks:
  progress_sync_interval_seconds: 60
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	cfg, err := store.UpdateQBittorrent("http://localhost:8081", "operator", nil, "/downloads", "moji", "auto")
	if err != nil {
		t.Fatalf("update qbittorrent: %v", err)
	}
	if cfg.QBittorrent.Password != "secret" {
		t.Fatalf("expected password preserved, got %q", cfg.QBittorrent.Password)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(updated)
	if !strings.Contains(text, `database:`) || !strings.Contains(text, `dsn: "user:password@tcp(localhost:3306)/media_library?parseTime=true"`) {
		t.Fatalf("expected database section preserved, got:\n%s", text)
	}

	reloaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.Tasks.ProgressSyncIntervalSeconds != 60 {
		t.Fatalf("expected numeric progress sync interval preserved, got %d", reloaded.Tasks.ProgressSyncIntervalSeconds)
	}
	if reloaded.QBittorrent.Username != "operator" {
		t.Fatalf("expected updated username, got %q", reloaded.QBittorrent.Username)
	}
}

func TestLoadFromPathNormalizesLegacyStashGraphQLURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `stash:
  graphql_url: "http://stash.example/graphql"
  api_key: "secret"
  library_path: "/library"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Stash.URL != "http://stash.example" {
		t.Fatalf("expected normalized stash url, got %q", cfg.Stash.URL)
	}
	if cfg.Stash.GraphQLEndpoint() != "http://stash.example/graphql" {
		t.Fatalf("expected derived graphql endpoint, got %q", cfg.Stash.GraphQLEndpoint())
	}
}
