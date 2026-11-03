package indexerdefs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const MaxStoredBytes int64 = 256 << 20

func (s *Service) cleanupTemporary(ctx context.Context) error {
	return s.repo.db.WithAssetMaintenance(ctx, func() error {
		current := filepath.Dir(filepath.Dir(s.root))
		if err := requireDirectory(current); err != nil {
			return err
		}
		for _, part := range []string{"indexers", "bundles"} {
			current = filepath.Join(current, part)
			if err := requireDirectory(current); errors.Is(err, os.ErrNotExist) {
				return nil
			} else if err != nil {
				return err
			}
		}
		entries, err := os.ReadDir(s.root)
		if err != nil {
			return err
		}
		if len(entries) > 10000 {
			return ErrBudget
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !strings.HasPrefix(entry.Name(), ".staging-") && !strings.HasPrefix(entry.Name(), ".download-") && !strings.HasPrefix(entry.Name(), ".damaged-") {
				continue
			}
			// These are private files without core ownership. An interrupted
			// download is never resumed; committed revision directories survive.
			if err := os.RemoveAll(filepath.Join(s.root, entry.Name())); err != nil {
				return err
			}
		}
		return syncDirectory(s.root)
	})
}

type Lease struct {
	Bundle  *Bundle
	once    sync.Once
	release func()
}

// RequireActive is a short, non-I/O check used while the configuration writer
// owns its lock. Activation takes that configuration gate before changing
// active, so an instance cannot be saved against superseded rules.
func (s *Service) RequireActive(ctx context.Context, revision string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil || s.active.Manifest.Revision != revision {
		return ErrConflict
	}
	return nil
}

// Retain protects a registered revision without compiling its whole library.
// Runtime candidate stores use this during restart; execution still uses
// Acquire to verify/load the actual immutable files.
func (s *Service) Retain(ctx context.Context, revision string) (func(), error) {
	s.leaseMu.Lock()
	defer s.leaseMu.Unlock()
	if _, _, err := s.repo.manifest(ctx, revision); err != nil {
		return nil, err
	}
	s.leases[revision]++
	var once sync.Once
	return func() {
		once.Do(func() {
			s.leaseMu.Lock()
			defer s.leaseMu.Unlock()
			s.leases[revision]--
			if s.leases[revision] == 0 {
				delete(s.leases, revision)
			}
		})
	}, nil
}

func (l *Lease) Release() {
	if l != nil {
		l.once.Do(func() {
			if l.release != nil {
				l.release()
			}
		})
	}
}

// Acquire pins an immutable plan revision for an in-flight search/acquisition.
// An empty revision means the current active bundle. Durable task ownership
// additionally uses PinInTx in the same core transaction as the task intent.
func (s *Service) Acquire(ctx context.Context, revision string) (*Lease, error) {
	s.leaseMu.Lock()
	defer s.leaseMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	active := s.active
	s.mu.Unlock()
	if revision == "" {
		if active == nil {
			return nil, ErrUnavailable
		}
		revision = active.Manifest.Revision
	}
	var bundle *Bundle
	if active != nil && active.Manifest.Revision == revision {
		bundle = active
	} else {
		_, checksum, err := s.repo.manifest(ctx, revision)
		if err != nil {
			return nil, err
		}
		bundle, err = LoadBundle(ctx, filepath.Join(s.root, revision), checksum)
		if err != nil {
			return nil, err
		}
	}
	s.leases[revision]++
	return &Lease{Bundle: bundle, release: func() {
		s.leaseMu.Lock()
		defer s.leaseMu.Unlock()
		s.leases[revision]--
		if s.leases[revision] == 0 {
			delete(s.leases, revision)
		}
	}}, nil
}

func PinInTx(ctx context.Context, tx *sqlx.Tx, kind, id, revision string) error {
	if kind == "" || id == "" || len(kind) > 64 || len(id) > 256 || !ValidRevision(revision) {
		return ErrConflict
	}
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO indexer_definition_refs(owner_kind,owner_id,revision) VALUES(?,?,?)`, kind, id, revision)
	return err
}
func UnpinInTx(ctx context.Context, tx *sqlx.Tx, kind, id string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM indexer_definition_refs WHERE owner_kind=? AND owner_id=?`, kind, id)
	return err
}

func (s *Service) ensureRoot() error {
	current := filepath.Dir(filepath.Dir(s.root))
	if err := requireDirectory(current); err != nil {
		return err
	}
	for _, part := range []string{"indexers", "bundles"} {
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := requireDirectory(current); err != nil {
			return err
		}
	}
	return nil
}

// repair restores the exact assets recorded in core. It cannot change the
// meaning of an existing revision, and serializes replacement with backups/GC.
func (s *Service) repair(ctx context.Context, stage *Bundle) (*Bundle, error) {
	expected, checksum, err := s.repo.manifest(ctx, stage.Manifest.Revision)
	if err != nil {
		return nil, err
	}
	if !sameFiles(expected, stage.Manifest) {
		return nil, ErrInvalidBundle
	}
	data, err := manifestData(expected)
	if err != nil || digest(data) != checksum {
		return nil, ErrInvalidBundle
	}
	manifestFile, err := os.OpenFile(filepath.Join(stage.Directory, "manifest.json"), os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	_, writeErr := manifestFile.Write(data)
	err = errors.Join(writeErr, manifestFile.Sync(), manifestFile.Close())
	if err != nil {
		return nil, err
	}
	stage.Manifest = expected
	stage.ManifestSHA256 = checksum
	err = s.repo.db.WithAssetMaintenance(ctx, func() error {
		s.leaseMu.Lock()
		defer s.leaseMu.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		target := filepath.Join(s.root, stage.Manifest.Revision)
		if info, err := os.Lstat(target); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			// Each replacement is an atomic rename of the exact registered bytes.
			// A healthy active directory is never temporarily removed, even if
			// the process exits halfway through a repair.
			for _, file := range expected.Files {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := os.Rename(filepath.Join(stage.Directory, file.Name), filepath.Join(target, file.Name)); err != nil {
					return err
				}
			}
			if err := os.Rename(filepath.Join(stage.Directory, "manifest.json"), filepath.Join(target, "manifest.json")); err != nil {
				return err
			}
			if err := syncDirectory(target); err != nil {
				return err
			}
			if err := os.Remove(stage.Directory); err != nil {
				return err
			}
			stage.Directory = target
			return syncDirectory(s.root)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		quarantine := filepath.Join(s.root, ".damaged-"+uuid.NewString())
		hadTarget := false
		if _, err := os.Lstat(target); err == nil {
			if err := os.Rename(target, quarantine); err != nil {
				return err
			}
			hadTarget = true
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Rename(stage.Directory, target); err != nil {
			if hadTarget {
				return errors.Join(err, os.Rename(quarantine, target))
			}
			return err
		}
		stage.Directory = target
		if err := syncDirectory(s.root); err != nil {
			return err
		}
		if hadTarget {
			if err := os.RemoveAll(quarantine); err != nil {
				return err
			}
		}
		return syncDirectory(s.root)
	})
	return stage, err
}

// Prune retains active, previous, durable references, live leases and any active
// update target. Metadata is removed first; a crash can only leave orphan files.
func (s *Service) Prune(ctx context.Context) error {
	return s.repo.db.WithAssetMaintenance(ctx, func() error {
		s.leaseMu.Lock()
		defer s.leaseMu.Unlock()
		retained := map[string]bool{}
		for revision := range s.leases {
			retained[revision] = true
		}
		err := s.repo.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
			var protected []string
			if err := tx.SelectContext(ctx, &protected, `SELECT active_revision FROM indexer_definition_state WHERE active_revision IS NOT NULL UNION SELECT previous_revision FROM indexer_definition_state WHERE previous_revision IS NOT NULL UNION SELECT revision FROM indexer_definition_refs UNION SELECT target_revision FROM indexer_definition_jobs WHERE finished_at IS NULL`); err != nil {
				return err
			}
			for _, revision := range protected {
				retained[revision] = true
			}
			var revisions []string
			if err := tx.SelectContext(ctx, &revisions, `SELECT revision FROM indexer_definition_bundles`); err != nil {
				return err
			}
			for _, revision := range revisions {
				if retained[revision] {
					continue
				}
				if _, err := tx.ExecContext(ctx, `DELETE FROM indexer_definition_bundles WHERE revision=?`, revision); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err := requireDirectory(s.root); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		entries, err := os.ReadDir(s.root)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !ValidRevision(entry.Name()) || retained[entry.Name()] {
				continue
			}
			if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return ErrInvalidBundle
			}
			if err := os.RemoveAll(filepath.Join(s.root, entry.Name())); err != nil {
				return err
			}
		}
		return syncDirectory(s.root)
	})
}

func (s *Service) checkStorageBudget(ctx context.Context) error {
	_, err := s.storageUsage(ctx)
	return err
}

func (s *Service) storageUsage(ctx context.Context) (int64, error) {
	var total int64
	var entries int
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > 100000 {
			return ErrBudget
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrInvalidBundle
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return ErrInvalidBundle
		}
		total += info.Size()
		if total > MaxStoredBytes {
			return ErrBudget
		}
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return total, err
}
