package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if !strings.Contains(string(migrated), "config_version: 2") {
		t.Fatal("missing version")
	}
	if err := run(source, source); err == nil {
		t.Fatal("allowed in-place overwrite")
	}
	if err := run(source, output); err == nil {
		t.Fatal("overwrote an existing output")
	}
}
