package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// preserveRawCore is an offline recovery precaution, including a damaged core
// which cannot pass SQLite backup/integrity validation. The caller owns the
// directory lock; these files are never used as a verified logical snapshot.
func preserveRawCore(ctx context.Context, root, active string, extra []BackupFile) (string, error) {
	directory := filepath.Join(root, "backups", "before-restore-raw-"+time.Now().UTC().Format("20060102T150405Z")+"-"+uuid.NewString())
	files := []BackupFile{}
	for _, name := range []string{"moji.db", "moji.db-wal", "moji.db-shm"} {
		if active == "" {
			break
		}
		path := filepath.Join(active, name)
		if info, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return "", err
		} else if !info.Mode().IsRegular() {
			return "", layoutError("unsafe recovery file", path, "repair the path manually before recovery")
		}
		files = append(files, BackupFile{Name: name, Path: path})
	}
	if _, err := os.Stat(filepath.Join(root, layoutFile)); err == nil {
		files = append(files, BackupFile{Name: layoutFile, Path: filepath.Join(root, layoutFile)})
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	files = append(files, extra...)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	var entries []BackupArtifact
	for _, file := range files {
		if !filepath.IsLocal(file.Name) {
			return "", errors.New("storage: invalid raw preservation name")
		}
		target := filepath.Join(directory, file.Name)
		if err := copyPrivateFile(ctx, file.Path, target); err != nil {
			return "", err
		}
		digest, size, err := fileDigest(target)
		if err != nil {
			return "", err
		}
		sourceDigest, sourceSize, err := fileDigest(file.Path)
		if err != nil {
			return "", err
		}
		if sourceDigest != digest || sourceSize != size {
			return "", errors.New("storage: source changed while preserving raw core")
		}
		entries = append(entries, BackupArtifact{Path: file.Name, SHA256: digest, Bytes: size})
	}
	manifest := struct {
		Format    int              `json:"format"`
		Protected bool             `json:"protected"`
		Physical  bool             `json:"physical_recovery_only"`
		Files     []BackupArtifact `json:"files"`
	}{1, true, true, entries}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err := atomicWrite(filepath.Join(directory, "manifest.json"), raw, 0o600); err != nil {
		return "", err
	}
	return directory, syncDirectory(filepath.Dir(directory))
}

// A broken pointer never authorizes guessing an old generation. Preserve the
// pointer itself and leave every unselected generation untouched for recovery.
func preserveSelected(ctx context.Context, root string, extra []BackupFile) (string, error) {
	active, err := resolveDirectory(root)
	if err != nil {
		var layout *LayoutError
		if !errors.As(err, &layout) {
			return "", err
		}
		active = ""
	}
	return preserveRawCore(ctx, root, active, extra)
}
