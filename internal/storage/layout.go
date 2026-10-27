package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"
)

const (
	DefaultDirectory = ".storage/database"
	layoutFile       = "CURRENT"
)

// LayoutError requires an explicit recovery action. Startup never guesses a
// generation or creates replacement databases inside an existing instance.
type LayoutError struct{ Kind, Path, Recovery string }

func (e *LayoutError) Error() string {
	return fmt.Sprintf("storage: %s: %s; %s", e.Kind, e.Path, e.Recovery)
}
func layoutError(kind, path, recovery string) error { return &LayoutError{kind, path, recovery} }

func validGeneration(name string) bool {
	return name != "" && name != "." && name != ".." && name != "backups" && name != layoutFile && !strings.HasPrefix(name, ".") && strings.TrimSpace(name) == name && !strings.ContainsAny(name, "/\\:{}\"") && !strings.ContainsFunc(name, unicode.IsControl)
}

func ResolveActiveDirectory(root string) (string, error) {
	directory, err := resolveDirectory(root)
	if err != nil {
		return "", err
	}
	if err := validateGenerationFiles(directory); err != nil {
		return "", err
	}
	return directory, nil
}
func resolveDirectory(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pointer := filepath.Join(root, layoutFile)
	info, err := os.Lstat(pointer)
	if errors.Is(err, os.ErrNotExist) {
		return "", layoutError("missing pointer", pointer, "use activate for a verified generation, restore a backup, or explicitly migrate legacy data")
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 255 {
		return "", layoutError("invalid pointer", pointer, "repair CURRENT manually or restore a backup")
	}
	raw, err := os.ReadFile(pointer)
	if err != nil {
		return "", err
	}
	name := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if !validGeneration(name) {
		return "", layoutError("invalid pointer", pointer, "CURRENT must contain only a generation directory name; repair manually or restore a backup")
	}
	directory := filepath.Join(root, name)
	info, err = os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", layoutError("missing or unsafe generation", directory, "activate another verified generation or restore a backup")
	}
	return directory, nil
}

func regularGenerationFile(directory, relative string) error {
	if !filepath.IsLocal(relative) || relative == "." || strings.Contains(relative, "\\") {
		return errors.New("storage: invalid generation file path")
	}
	parts := strings.Split(filepath.Clean(relative), string(filepath.Separator))
	path := directory
	for i, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("storage: symlink in generation")
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return errors.New("storage: invalid generation subdirectory")
			}
		} else if !info.Mode().IsRegular() || info.Size() == 0 {
			return errors.New("storage: empty or irregular generation file")
		}
	}
	return nil
}
func validateGenerationFiles(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return layoutError("missing or unsafe generation", directory, "restore or activate a complete generation")
	}
	for _, role := range []string{"moji.db", "runtime.db", filepath.Join("cache", "stashbox", "cache.db")} {
		if err := regularGenerationFile(directory, role); err != nil {
			recovery := "restore a verified backup or explicitly migrate data"
			if role != "moji.db" {
				name := "runtime"
				if role != "runtime.db" {
					name = "cache"
				}
				recovery = "run moji-storage repair-derived -role " + name + " while Moji is stopped"
			}
			return layoutError("missing or invalid database", filepath.Join(directory, role), recovery)
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			path := filepath.Join(directory, role) + suffix
			info, err := os.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return layoutError("unsafe SQLite sidecar", path, "repair the path manually while Moji is stopped")
			}
		}
	}
	return nil
}

// validateHistory ensures the immutable history referenced by the core remains
// part of the selected generation, including after preparation or recovery.
func validateHistory(ctx context.Context, corePath string) error {
	db, err := sqlx.Open("sqlite", readOnlyURI(corePath))
	if err != nil {
		return err
	}
	defer db.Close()
	var exists int
	if err := db.GetContext(ctx, &exists, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='history_archives'"); err != nil {
		return err
	}
	if exists == 0 {
		return nil
	}
	var rows []struct{ Path, SHA256 string }
	if err := db.SelectContext(ctx, &rows, "SELECT path,sha256 FROM history_archives"); err != nil {
		return err
	}
	for _, row := range rows {
		path := filepath.Join(filepath.Dir(corePath), row.Path)
		if err := regularGenerationFile(filepath.Dir(corePath), row.Path); err != nil {
			return layoutError("missing or unsafe history archive", path, "restore the referenced archive from a verified backup")
		}
		digest, _, err := fileDigest(path)
		if err != nil {
			return err
		}
		if digest != row.SHA256 {
			return layoutError("history checksum mismatch", path, "restore the referenced archive from a verified backup")
		}
	}
	return nil
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
	if !validGeneration(generation) {
		return errors.New("storage: invalid generation name")
	}
	directory := filepath.Join(root, generation)
	if err := validateGenerationFiles(directory); err != nil {
		return err
	}
	corePath := filepath.Join(directory, "moji.db")
	if err := validateHistory(ctx, corePath); err != nil {
		return err
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
	return atomicWrite(filepath.Join(root, layoutFile), []byte(generation+"\n"), 0o600)
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
	if len(applied) == 0 {
		return ErrUnrecognizedSchema
	}
	if len(applied) > len(expected) {
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
	preserved, err := preserveSelected(ctx, root, extra)
	if err != nil {
		return "", err
	}
	if err := PublishGeneration(ctx, root, generation); err != nil {
		return preserved, err
	}
	return preserved, nil
}
