package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PruneBackups retains protected snapshots and the newest snapshot in each of
// the last 24 hourly, 14 daily and 8 weekly buckets. Unknown files are untouched.
// Only a verified new backup should trigger pruning; errors stop deletion.
func PruneBackups(directory string, now time.Time) error {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Resume only explicit prior prune intentions, never infer deletion from an
	// orphan database/asset file.
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json.pruning") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var saved BackupManifest
		if err := json.Unmarshal(raw, &saved); err != nil {
			return err
		}
		if err := finishPrune(directory, path, saved); err != nil {
			return err
		}
	}
	type record struct {
		manifest BackupManifest
		path     string
	}
	var records []record
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var m BackupManifest
		if err = json.Unmarshal(payload, &m); err != nil {
			return fmt.Errorf("storage: invalid retention manifest %s: %w", entry.Name(), err)
		}
		if m.Format != 1 || filepath.Base(m.Database) != m.Database || entry.Name() != m.Database+".json" {
			continue
		}
		if m.CreatedAt.IsZero() {
			return fmt.Errorf("storage: missing backup timestamp: %s", entry.Name())
		}
		records = append(records, record{m, path})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].manifest.CreatedAt.After(records[j].manifest.CreatedAt) })
	hourly, daily, weekly := map[int64]bool{}, map[int64]bool{}, map[int64]bool{}
	newestRegular := true
	for _, r := range records {
		m := r.manifest
		if m.Protected {
			continue
		}
		keep := newestRegular || m.CreatedAt.After(now)
		newestRegular = false
		hour := now.Unix()/3600 - m.CreatedAt.Unix()/3600
		day := now.Unix()/86400 - m.CreatedAt.Unix()/86400
		// Absolute week buckets avoid locale- or daylight-saving-dependent retention.
		week := now.Unix()/(7*86400) - m.CreatedAt.Unix()/(7*86400)
		if hour >= 0 && hour < 24 && !hourly[hour] {
			hourly[hour] = true
			keep = true
		}
		if day >= 0 && day < 14 && !daily[day] {
			daily[day] = true
			keep = true
		}
		if week >= 0 && week < 8 && !weekly[week] {
			weekly[week] = true
			keep = true
		}
		if keep {
			continue
		}
		database := filepath.Join(directory, m.Database)
		// A changed or missing snapshot must be surfaced, never silently discarded.
		digest, size, err := fileDigest(database)
		if err != nil {
			return err
		}
		if digest != m.SHA256 || size != m.Bytes {
			return fmt.Errorf("storage: refusing to prune modified backup: %s", m.Database)
		}
		if err := verifyArtifacts(r.path, m); err != nil {
			return err
		}
		tombstone := r.path + ".pruning"
		if err := os.Rename(r.path, tombstone); err != nil {
			return err
		}
		if err := syncDirectory(directory); err != nil {
			return err
		}
		if err := finishPrune(directory, tombstone, m); err != nil {
			return err
		}
	}
	return syncDirectory(directory)
}

// A synced tombstone authorizes idempotent completion after interruption. Missing
// files are expected only in this phase; modified surviving files stop cleanup.
func finishPrune(directory, tombstone string, m BackupManifest) error {
	if m.Protected || m.Format != 1 || filepath.Base(m.Database) != m.Database || !strings.HasSuffix(m.Database, ".sqlite") || filepath.Base(tombstone) != m.Database+".json.pruning" {
		return errors.New("storage: invalid backup prune intent")
	}
	check := func(path, expected string, size int64) error {
		digest, actual, err := fileDigest(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if digest != expected || actual != size {
			return fmt.Errorf("storage: modified file during prune: %s", filepath.Base(path))
		}
		return nil
	}
	database := filepath.Join(directory, m.Database)
	if err := check(database, m.SHA256, m.Bytes); err != nil {
		return err
	}
	prefix := strings.TrimSuffix(m.Database, ".sqlite") + ".assets/"
	for _, artifact := range m.Artifacts {
		if !filepath.IsLocal(artifact.Path) || filepath.ToSlash(filepath.Clean(artifact.Path)) != artifact.Path || !strings.HasPrefix(artifact.Path, prefix) {
			return errors.New("storage: invalid prune artifact path")
		}
		if err := check(filepath.Join(directory, artifact.Path), artifact.SHA256, artifact.Bytes); err != nil {
			return err
		}
	}
	if err := os.Remove(database); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(m.Artifacts) > 0 {
		if err := os.RemoveAll(filepath.Join(directory, strings.TrimSuffix(m.Database, ".sqlite")+".assets")); err != nil {
			return err
		}
	}
	if err := syncDirectory(directory); err != nil {
		return err
	}
	if err := os.Remove(tombstone); err != nil {
		return err
	}
	return syncDirectory(directory)
}
