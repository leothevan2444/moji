package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

type Manager struct {
	Core, Runtime, Cache *Database
	Observations         *ObservationStore
	lock                 *DirectoryLock
	BackupDirectory      string
	Supplemental         func(context.Context) ([]BackupFile, error)
}
type ManagerOptions struct {
	BackupDirectory                                    string
	Directory                                          string
	Supplemental                                       func(context.Context) ([]BackupFile, error)
	CoreMigrations, RuntimeMigrations, CacheMigrations []Migration
}

// OpenManager is the only production owner of all three database files. The
// migration CLI uses the same schema bundles against isolated staging files.
func OpenManager(ctx context.Context, o ManagerOptions) (m *Manager, err error) {
	if o.Directory == "" {
		return nil, errors.New("storage: data directory is required")
	}
	lock, err := AcquireDirectory(o.Directory)
	if err != nil {
		return nil, err
	}
	m = &Manager{lock: lock, BackupDirectory: filepath.Join(lock.Directory, "backups"), Supplemental: o.Supplemental}
	activeDirectory, e := ResolveActiveDirectory(lock.Directory)
	if e != nil {
		lock.Close()
		return nil, e
	}
	if o.BackupDirectory != "" {
		directory, e := filepath.Abs(o.BackupDirectory)
		if e != nil {
			lock.Close()
			return nil, e
		}
		m.BackupDirectory = directory
	}
	owner := m
	defer func() {
		if err != nil {
			owner.Close()
		}
	}()
	for _, item := range []struct {
		destination **Database
		path        string
		durability  Durability
		migrations  []Migration
	}{
		{&m.Core, "moji.db", Durable, o.CoreMigrations},
		{&m.Runtime, "runtime.db", Rebuildable, o.RuntimeMigrations},
		{&m.Cache, filepath.Join("cache", "stashbox", "cache.db"), Rebuildable, o.CacheMigrations},
	} {
		d, e := Open(ctx, Options{Path: filepath.Join(activeDirectory, item.path), Durability: item.durability})
		if e != nil {
			return nil, e
		}
		*item.destination = d
		// Every nonempty upgrade receives a protected backup, before changing DDL.
		var tables int
		if e = d.Reader.GetContext(ctx, &tables, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"); e != nil {
			return nil, e
		}
		if tables > 0 {
			version, e := ReadVersion(ctx, d.Reader)
			if e != nil {
				return nil, fmt.Errorf("%w: %s", ErrUnrecognizedSchema, item.path)
			}
			if version < len(item.migrations) {
				var backupErr error
				if item.path == "moji.db" {
					_, backupErr = m.Backup(ctx, "before-migration:"+item.path, true)
				} else {
					_, backupErr = d.Backup(ctx, m.BackupDirectory, "before-migration:"+item.path, true)
				}
				if backupErr != nil {
					e = backupErr
					return nil, e
				}
			}
		}
		if e = d.Migrate(ctx, item.migrations); e != nil {
			return nil, fmt.Errorf("%s: %w", item.path, e)
		}
	}
	m.Observations, err = NewObservationStore(ctx, m.Runtime)
	if err != nil {
		return nil, err
	}
	if err = m.recoverDownloadPaths(ctx); err != nil {
		return nil, err
	}
	return m, nil
}
func (m *Manager) Close() error {
	var errs []error
	if m.Observations != nil {
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		errs = append(errs, m.Observations.FlushAll(flushCtx))
		cancel()
	}
	for _, d := range []*Database{m.Cache, m.Runtime, m.Core} {
		if d != nil {
			errs = append(errs, d.Close())
		}
	}
	if m.lock != nil {
		errs = append(errs, m.lock.Close())
	}
	return errors.Join(errs...)
}

type MaintenanceResult struct {
	At          time.Time
	Checkpoints map[string]Checkpoint
	Backup      *BackupManifest
	Err         error
}

// Maintain performs passive checkpoints and periodic online core backups. The
// caller owns the goroutine and joins it before closing the manager.
func (m *Manager) Maintain(ctx context.Context, report func(MaintenanceResult)) {
	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	observations := time.NewTicker(3 * time.Second)
	defer observations.Stop()
	var lastBackup time.Time
	run := func() {
		result := MaintenanceResult{At: time.Now().UTC(), Checkpoints: map[string]Checkpoint{}}
		_, cleanupErr := m.PruneRuntime(ctx, result.At)
		result.Err = errors.Join(result.Err, cleanupErr)
		for name, d := range map[string]*Database{"core": m.Core, "runtime": m.Runtime, "cache": m.Cache} {
			checkpoint, err := d.Checkpoint(ctx)
			result.Checkpoints[name] = checkpoint
			result.Err = errors.Join(result.Err, err)
		}
		if lastBackup.IsZero() || time.Since(lastBackup) >= time.Hour {
			work, cancel := context.WithTimeout(ctx, 5*time.Minute)
			snapshot, err := m.Backup(work, "scheduled", false)
			cancel()
			result.Err = errors.Join(result.Err, err)
			if err == nil {
				lastBackup = time.Now()
				result.Backup = &snapshot
				result.Err = errors.Join(result.Err, PruneBackups(m.BackupDirectory, time.Now().UTC()))
			}
		}
		if report != nil {
			report(result)
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-observations.C:
			if err := m.Observations.Flush(ctx); err != nil && report != nil {
				report(MaintenanceResult{At: time.Now().UTC(), Err: err})
			}
		case <-timer.C:
			run()
		}
	}
}

// Backup captures the durable core and required non-SQL state together in the
// manifest. These files are individually atomic; there is no cross-file SQL tx.
func (m *Manager) Backup(ctx context.Context, reason string, protect bool) (BackupManifest, error) {
	var files []BackupFile
	if m.Supplemental != nil {
		var err error
		files, err = m.Supplemental(ctx)
		if err != nil {
			return BackupManifest{}, err
		}
	}
	return m.Core.Backup(ctx, m.BackupDirectory, reason, protect, files...)
}
