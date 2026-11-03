package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"gopkg.in/yaml.v3"
)

func TestPackagedExampleUsesV3AndLeavesIndexerSourcesDisabled(t *testing.T) {
	source := filepath.Join("..", "..", "docker", "config.example.yaml")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var header struct {
		Version int `yaml:"config_version"`
	}
	if err := yaml.Unmarshal(raw, &header); err != nil || header.Version != 3 {
		t.Fatal("packaged example does not use the current configuration layout", err)
	}
	cfg, err := config.LoadFromPath(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexers.Jackett.Enabled || cfg.Connection.Jackett.URL != "" || cfg.Connection.Jackett.APIKey != "" || len(cfg.Indexers.Builtin.Instances) != 0 {
		t.Fatal("new installation implicitly enabled or configured an indexer")
	}
	if cfg.Indexers.Builtin.Definitions.Source != "prowlarr" || cfg.Indexers.Builtin.Definitions.SchemaVersion != 11 || cfg.Indexers.Builtin.Definitions.AutoUpdate {
		t.Fatal("definition download policy differs from the documented installation")
	}
	if err := run(source, filepath.Join(t.TempDir(), "validated-copy.yaml")); err != nil {
		t.Fatal("packaged example did not survive offline save/validation", err)
	}
	unchanged, err := os.ReadFile(source)
	if err != nil || string(unchanged) != string(raw) {
		t.Fatal("offline validation modified the packaged example", err)
	}
}

func TestMigrationCopyPreservesSourceAndRejectsOverwrite(t *testing.T) {
	source := filepath.Join(t.TempDir(), "old.yaml")
	output := filepath.Join(t.TempDir(), "new.yaml")
	raw := []byte("connection:\n  jackett: {url: 'http://localhost', api_key: test}\nautomation: {}\n")
	if err := os.WriteFile(source, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(source, output); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(source)
	if string(original) != string(raw) {
		t.Fatal("modified source")
	}
	migrated, _ := os.ReadFile(output)
	if !strings.Contains(string(migrated), "config_version: 3") {
		t.Fatal("missing version")
	}
	if err := run(source, source); err == nil {
		t.Fatal("allowed in-place overwrite")
	}
	if err := run(source, output); err == nil {
		t.Fatal("overwrote an existing output")
	}
}
