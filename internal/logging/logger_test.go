package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerEntriesRespectRetentionAndLevel(t *testing.T) {
	logger, err := New(Options{
		ConsoleWriter: &bytes.Buffer{},
		MaxEntries:    2,
		Level:         "debug",
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.logf(parseLevel("info"), "first")
	logger.logf(parseLevel("warn"), "second")
	logger.logf(parseLevel("error"), "third")

	entries := logger.Entries(0, "info")
	if len(entries) != 2 {
		t.Fatalf("expected 2 retained entries, got %d", len(entries))
	}
	if entries[0].Message != "third" || entries[1].Message != "second" {
		t.Fatalf("unexpected entry order: %+v", entries)
	}

	warnOnly := logger.Entries(0, "warn")
	if len(warnOnly) != 2 {
		t.Fatalf("expected 2 warn-or-higher entries, got %d", len(warnOnly))
	}

	errorOnly := logger.Entries(1, "error")
	if len(errorOnly) != 1 || errorOnly[0].Message != "third" {
		t.Fatalf("unexpected error entries: %+v", errorOnly)
	}
}

func TestRollingFileWriterRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "moji.log")

	writer, err := newRollingFileWriter(path, 24, 2)
	if err != nil {
		t.Fatalf("new rolling file writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	for range 4 {
		if _, err := writer.Write([]byte("1234567890\n")); err != nil {
			t.Fatalf("write log: %v", err)
		}
	}

	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current log: %v", err)
	}
	if len(current) == 0 {
		t.Fatal("expected current log file to contain data")
	}

	backup, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatalf("read backup log: %v", err)
	}
	if !strings.Contains(string(backup), "1234567890") {
		t.Fatalf("expected backup to contain rotated content, got %q", string(backup))
	}
}

func TestDefaultLogFilePath(t *testing.T) {
	for _, tc := range []struct{ config, want string }{
		{"", "logs/moji.log"},
		{"config.yaml", "logs/moji.log"},
		{" /data/config.yaml ", "/data/logs/moji.log"},
	} {
		if got := DefaultLogFilePath(tc.config); got != tc.want {
			t.Errorf("DefaultLogFilePath(%q) = %q, want %q", tc.config, got, tc.want)
		}
	}
}

func TestRollingFileLayoutRetentionAndRestart(t *testing.T) {
	path := DefaultLogFilePath(filepath.Join(t.TempDir(), "config.yaml"))
	writer, err := newRollingFileWriter(path, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"aaa\n", "bbb\n", "ccc\n", "ddd\n"} {
		if _, err := writer.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	writer, err = newRollingFileWriter(path, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.Write([]byte("eee\n")); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{path: "eee\n", path + ".1": "ddd\n", path + ".2": "ccc\n"} {
		got, err := os.ReadFile(name)
		if err != nil || string(got) != want {
			t.Fatalf("read %s = %q, %v; want %q", name, got, err, want)
		}
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(files) != 3 {
		t.Fatalf("expected current + 2 backups, got %d, %v", len(files), err)
	}
}
