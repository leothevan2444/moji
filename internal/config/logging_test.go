package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/logging"
)

func TestLogFileLayout(t *testing.T) {
	for _, prefix := range []string{"", "config_version: 2\n"} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		raw := prefix + "logging: {file_path: /retired/server.log, level: debug}\n"
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		store, err := OpenStore(path)
		if err != nil {
			t.Fatal(err)
		}
		want := logging.DefaultLogFilePath(path)
		if got := store.Config().EffectiveLogFilePath(); got != want {
			t.Fatalf("log path = %s, want %s", got, want)
		}
		if err := store.Migrate(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "file_path") {
			t.Fatal("retired path override retained on save")
		}
		cfg, err := LoadFromPath(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.EffectiveLogFilePath() != want || cfg.EffectiveLogLevel() != "debug" {
			t.Fatal("save changed log directory or level")
		}
	}
}
