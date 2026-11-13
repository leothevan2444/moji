package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloaderCleanupPersistAndValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 4\ndownloader:\n  qbittorrent:\n    url: http://qb.invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.DownloaderCleanup(); got != DefaultDownloaderCleanup() {
		t.Fatal(got)
	}
	want := DownloaderCleanupConfig{Enabled: true, Trigger: "CAPACITY", DeleteFiles: true, MinFreeGB: 123, CapacityOrder: "SEED_TIME"}
	if err := store.UpdateDownloaderCleanup(want); err != nil {
		t.Fatal(err)
	}
	// Other settings writers must preserve cleanup and the qB connection.
	if _, err := store.UpdateStash("http://stash.invalid/graphql", "test"); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.DownloaderCleanup() != want || reopened.Config().Connection.QBittorrent.URL != "http://qb.invalid" {
		t.Fatal("configuration lost after unrelated save")
	}
	before, _ := os.ReadFile(path)
	invalid := want
	invalid.DeleteFiles = false
	if err := store.UpdateDownloaderCleanup(invalid); err == nil {
		t.Fatal("unsafe capacity policy accepted")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) || store.DownloaderCleanup() != want {
		t.Fatal("invalid save changed configuration")
	}
	for _, field := range []string{"trigger: UNKNOWN", "min_free_gb: 0", "capacity_order: UNKNOWN", "ratio: 2"} {
		if _, err := parseStore(path, []byte("config_version: 4\ndownloader:\n  cleanup:\n    "+field+"\n")); err == nil {
			t.Fatal("invalid cleanup field accepted", field)
		}
	}
	if !strings.Contains(string(before), "cleanup:") {
		t.Fatal("cleanup not written to YAML")
	}
}
