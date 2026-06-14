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
