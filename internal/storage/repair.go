package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RepairDerived explicitly rebuilds disposable state while preserving the core
// and the old derived files. It never runs as an automatic startup fallback.
func RepairDerived(ctx context.Context, root, role string, extra ...BackupFile) (string, error) {
	relative := "runtime.db"
	migrations := RuntimeMigrations()
	if role == "cache" {
		relative = filepath.Join("cache", "stashbox", "cache.db")
		migrations = CacheMigrations()
	} else if role != "runtime" {
		return "", errors.New("storage: repair role must be runtime or cache")
	}
	lock, err := AcquireDirectory(root)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	root = lock.Directory
	active, err := resolveDirectory(root)
	if err != nil {
		return "", err
	}
	if err = regularGenerationFile(active, "moji.db"); err != nil {
		return "", err
	}
	core := filepath.Join(active, "moji.db")
	if err = validatePublishedCore(ctx, core); err != nil {
		return "", err
	}
	if _, err = inspectBackup(ctx, core); err != nil {
		return "", err
	}
	if err = validateHistory(ctx, core); err != nil {
		return "", err
	}
	// Parent directories may be absent, but must never redirect a repair outside
	// the selected generation through symlinks.
	target := filepath.Join(active, relative)
	for parent := filepath.Dir(target); parent != active; parent = filepath.Dir(parent) {
		info, e := os.Lstat(parent)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return "", e
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("storage: unsafe repair directory %s", parent)
		}
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		path := target + suffix
		info, e := os.Lstat(path)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return "", e
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("storage: unsafe repair file %s", path)
		}
		extra = append(extra, BackupFile{Name: filepath.Join("derived", role+suffix), Path: path})
	}
	preserved, err := preserveRawCore(ctx, root, active, extra)
	if err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(root, "repair-")
	if err != nil {
		return preserved, err
	}
	staged := filepath.Join(staging, "database.db")
	d, err := Open(ctx, Options{Path: staged, Durability: Rebuildable})
	if err != nil {
		return preserved, err
	}
	migrateErr := d.Migrate(ctx, migrations)
	if err = errors.Join(migrateErr, d.Close()); err != nil {
		return preserved, err
	}
	if err = syncFile(staged); err != nil {
		return preserved, err
	}
	if err = ctx.Err(); err != nil {
		return preserved, err
	}
	// Original files already have checksum-verified physical copies. Removing the
	// live sidecars is safe only here, offline under the instance lock.
	for _, suffix := range []string{"-wal", "-shm"} {
		if err = os.Remove(target + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return preserved, err
		}
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return preserved, err
	}
	if err = os.Rename(staged, target); err != nil {
		return preserved, err
	}
	for parent := filepath.Dir(target); parent != root; parent = filepath.Dir(parent) {
		if err = syncDirectory(parent); err != nil {
			return preserved, err
		}
	}
	if err = os.Remove(staging); err != nil {
		return preserved, err
	}
	return preserved, syncDirectory(root)
}
