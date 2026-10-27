package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RestoreResult struct {
	Generation        string          `json:"generation"`
	Directory         string          `json:"directory"`
	PreviousRawBackup string          `json:"previous_raw_backup,omitempty"`
	PreviousBackup    *BackupManifest `json:"previous_backup,omitempty"`
	// Configuration and receipts are preserved in the generation alongside the
	// core. Operators restore these to their configured locations independently.
	Artifacts []string `json:"artifacts"`
}

// RestoreCore verifies a backup, protects the old core, prepares a new complete
// generation and publishes one pointer. Runtime/cache data is rebuilt; no
// external client is initialized and no pending command is replayed.
func RestoreCore(ctx context.Context, root, manifestPath string, extra ...BackupFile) (result RestoreResult, err error) {
	saved, err := VerifyBackup(ctx, manifestPath)
	if err != nil {
		return result, err
	}
	source := filepath.Join(filepath.Dir(manifestPath), saved.Database)
	if err := validateCoreBundle(ctx, source, CoreMigrations(), true); err != nil {
		return result, err
	}
	lock, err := AcquireDirectory(root)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	root = lock.Directory
	active, err := resolveDirectory(root, false)
	if err != nil {
		return result, err
	}
	currentPath := filepath.Join(active, "moji.db")
	if info, statErr := os.Stat(currentPath); statErr == nil {
		if !info.Mode().IsRegular() {
			return result, errors.New("storage: existing core is not a regular file")
		}
		raw, err := preserveRawCore(ctx, root, active, extra)
		if err != nil {
			return result, err
		}
		result.PreviousRawBackup = raw
		// Even an invalid/damaged old core is preserved before recovery. A logical
		// backup is attempted in addition; failure cannot prevent disaster recovery.
		old, openErr := Open(ctx, Options{Path: currentPath, Durability: Durable})
		if openErr == nil {
			preserved, backupErr := old.Backup(ctx, filepath.Join(root, "backups"), "before-restore", true, extra...)
			closeErr := old.Close()
			if backupErr == nil && closeErr == nil {
				result.PreviousBackup = &preserved
			}
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return result, statErr
	}

	result.Generation = "restore-" + time.Now().UTC().Format("20060102T150405Z") + "-" + uuid.NewString()
	result.Directory = filepath.Join(root, ".storage", "generations", result.Generation)
	if err := copyPrivateFile(ctx, source, filepath.Join(result.Directory, "moji.db")); err != nil {
		return result, err
	}
	prefix := strings.TrimSuffix(saved.Database, ".sqlite") + ".assets/"
	for _, artifact := range saved.Artifacts {
		name := strings.TrimPrefix(artifact.Path, prefix)
		if !filepath.IsLocal(name) {
			return result, errors.New("storage: invalid restore artifact")
		}
		target := filepath.Join(result.Directory, name)
		if target == filepath.Join(result.Directory, "moji.db") {
			return result, errors.New("storage: artifact collides with core")
		}
		if err := copyPrivateFile(ctx, filepath.Join(filepath.Dir(manifestPath), artifact.Path), target); err != nil {
			return result, err
		}
		digest, size, err := fileDigest(target)
		if err != nil {
			return result, err
		}
		if digest != artifact.SHA256 || size != artifact.Bytes {
			return result, errors.New("storage: artifact changed while restoring")
		}
		result.Artifacts = append(result.Artifacts, name)
	}
	// Catch checksums changed after verification before publication.
	actual, err := inspectBackup(ctx, filepath.Join(result.Directory, "moji.db"))
	if err != nil {
		return result, err
	}
	if actual.SHA256 != saved.SHA256 {
		return result, fmt.Errorf("storage: backup changed while restoring")
	}
	staged, err := OpenManager(ctx, DefaultManagerOptions(result.Directory))
	if err != nil {
		return result, err
	}
	if err := staged.Close(); err != nil {
		return result, err
	}
	if err := PublishGeneration(ctx, root, result.Generation); err != nil {
		return result, err
	}
	return result, nil
}
