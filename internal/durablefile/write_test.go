package durablefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWritePublishesPrivateFileAndPreservesExistingOnFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.yaml")
	if err := Write(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatal("unsafe file permissions", err)
	}
	if err := Write(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	value, err := os.ReadFile(path)
	if err != nil || string(value) != "replacement" {
		t.Fatal("replacement not published", err)
	}
	// A directory cannot be replaced by a file, even after temporary-file fsync.
	if err := Write(root, []byte("invalid"), 0o600); err == nil {
		t.Fatal("invalid replacement accepted")
	}
	value, err = os.ReadFile(path)
	if err != nil || string(value) != "replacement" {
		t.Fatal("failed replacement damaged existing data", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatal("temporary files leaked", err)
	}
}
