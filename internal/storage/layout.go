package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"os"
	"path/filepath"
	"strings"
)

const layoutFile = ".moji-storage.json"

type ActiveLayout struct {
	Format     int    `json:"format"`
	Generation string `json:"generation"`
}

// ResolveActiveDirectory chooses all database files through one atomic pointer.
// A published generation without a core is an error, never a new empty install.
func ResolveActiveDirectory(root string) (string, error) {
	return resolveDirectory(root, true)
}
func resolveDirectory(root string, requireCore bool) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(filepath.Join(root, layoutFile))
	if errors.Is(err, os.ErrNotExist) {
		return root, nil
	}
	if err != nil {
		return "", err
	}
	var layout ActiveLayout
	if err := json.Unmarshal(raw, &layout); err != nil {
		return "", fmt.Errorf("storage: invalid active layout: %w", err)
	}
	if layout.Format != 1 || layout.Generation == "" || layout.Generation == "." || layout.Generation == ".." || strings.ContainsAny(layout.Generation, "/\\") {
		return "", errors.New("storage: unsupported or invalid active layout")
	}
	directory := filepath.Join(root, ".storage", "generations", layout.Generation)
	if !requireCore {
		return directory, nil
	}
	info, err := os.Stat(filepath.Join(directory, "moji.db"))
	if err != nil {
		return "", fmt.Errorf("storage: active core is missing; explicit restore required: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", errors.New("storage: active core is empty or not a regular file")
	}
	return directory, nil
}

// PublishGeneration requires the caller to hold root's DirectoryLock and to
// stop/join all old-generation users. Publication changes one fsynced pointer;
// interrupted preparation leaves the previous complete generation selected.
func PublishGeneration(ctx context.Context, root, generation string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	root = absolute
	if generation == "" || generation == "." || generation == ".." || strings.ContainsAny(generation, "/\\") {
		return errors.New("storage: invalid generation name")
	}
	directory := filepath.Join(root, ".storage", "generations", generation)
	corePath := filepath.Join(directory, "moji.db")
	info, err := os.Stat(corePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("storage: staged core missing or empty: %v", err)
	}
	if err := validatePublishedCore(ctx, corePath); err != nil {
		return err
	}
	inspected, err := inspectBackup(ctx, corePath)
	if err != nil {
		return err
	}
	if _, ok := inspected.Tables["schema_migrations"]; !ok {
		return ErrUnrecognizedSchema
	}
	for _, role := range []struct {
		Path       string
		Migrations []Migration
	}{{"runtime.db", RuntimeMigrations()}, {filepath.Join("cache", "stashbox", "cache.db"), CacheMigrations()}} {
		path := filepath.Join(directory, role.Path)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			continue
		}
		if err := validateCoreBundle(ctx, path, role.Migrations, false); err != nil {
			return fmt.Errorf("storage: invalid staged %s: %w", role.Path, err)
		}
		if _, err := inspectBackup(ctx, path); err != nil {
			return err
		}
		if wal, err := os.Stat(path + "-wal"); err == nil && wal.Size() > 0 {
			return errors.New("storage: staged derived database has an open WAL")
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := syncFile(path); err != nil {
			return err
		}
		for parent := filepath.Dir(path); parent != root && parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
			if err := syncDirectory(parent); err != nil {
				return err
			}
		}
	}
	// Checkpointed and closed staging files are required: a copied main file alone
	// must contain the full committed core before the pointer can be published.
	if wal, err := os.Stat(corePath + "-wal"); err == nil && wal.Size() > 0 {
		return errors.New("storage: staged core has a WAL; close staging manager before publication")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncFile(corePath); err != nil {
		return err
	}
	for parent := directory; parent != root && parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		if err := syncDirectory(parent); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(ActiveLayout{Format: 1, Generation: generation})
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(root, layoutFile), raw, 0o600)
}

func validatePublishedCore(ctx context.Context, path string) error {
	return validateCoreBundle(ctx, path, CoreMigrations(), false)
}
func validateCoreBundle(ctx context.Context, path string, expected []Migration, allowOlder bool) error {
	db, err := sqlx.Open("sqlite", readOnlyURI(path))
	if err != nil {
		return err
	}
	defer db.Close()
	var applied []struct {
		Version        int
		Name, Checksum string
	}
	if err := db.SelectContext(ctx, &applied, "SELECT version,name,checksum FROM schema_migrations ORDER BY version"); err != nil {
		return ErrUnrecognizedSchema
	}
	if len(applied) == 0 || len(applied) > len(expected) {
		return ErrSchemaTooNew
	}
	if !allowOlder && len(applied) != len(expected) {
		return errors.New("storage: staged core requires a schema upgrade before publication")
	}
	for i, row := range applied {
		if row.Version != expected[i].Version || row.Name != expected[i].Name || row.Checksum != checksum(expected[i].SQL) {
			return ErrMigrationChanged
		}
	}
	return nil
}

// ActivateGeneration protects the selected old core physically, then atomically
// selects a validated prepared generation while holding the instance lock.
func ActivateGeneration(ctx context.Context, root, generation string, extra ...BackupFile) (string, error) {
	lock, err := AcquireDirectory(root)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	root = lock.Directory
	active, err := resolveDirectory(root, false)
	if err != nil {
		return "", err
	}
	preserved := ""
	if _, err := os.Stat(filepath.Join(active, "moji.db")); err == nil {
		preserved, err = preserveRawCore(ctx, root, active, extra)
		if err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := PublishGeneration(ctx, root, generation); err != nil {
		return preserved, err
	}
	return preserved, nil
}
