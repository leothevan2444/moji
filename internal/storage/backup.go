package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqlite "modernc.org/sqlite"
)

type BackupManifest struct {
	Format    int              `json:"format"`
	Artifacts []BackupArtifact `json:"artifacts,omitempty"`
	Database  string           `json:"database"`
	SHA256    string           `json:"sha256"`
	Bytes     int64            `json:"bytes"`
	CreatedAt time.Time        `json:"created_at"`
	Reason    string           `json:"reason"`
	Engine    string           `json:"sqlite_version"`
	SourceID  string           `json:"sqlite_source_id"`
	Tables    map[string]int64 `json:"tables"`
	Protected bool             `json:"protected"`
}

type backupConnection interface {
	NewBackup(string) (*sqlite.Backup, error)
}

// Backup uses SQLite's incremental online backup API, not a filesystem copy of
// an open WAL database. Only verified standalone snapshots receive manifests.
func (d *Database) Backup(ctx context.Context, directory, reason string, protect bool, extra ...BackupFile) (manifest BackupManifest, err error) {
	if d.closed.Load() {
		return manifest, ErrClosed
	}
	if err = os.MkdirAll(directory, 0o700); err != nil {
		return manifest, err
	}
	token := make([]byte, 8)
	if _, err = rand.Read(token); err != nil {
		return manifest, err
	}
	stem := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(token)
	temporary, err := os.CreateTemp(directory, ".backup-*.partial")
	if err != nil {
		return manifest, err
	}
	tempPath := temporary.Name()
	if err = temporary.Close(); err != nil {
		os.Remove(tempPath)
		return manifest, err
	}
	defer func() { os.Remove(tempPath) }()
	connection, err := d.Reader.Connx(ctx)
	if err != nil {
		return manifest, err
	}
	err = connection.Raw(func(raw any) error {
		source, ok := raw.(backupConnection)
		if !ok {
			return errors.New("storage: driver has no online backup API")
		}
		backup, err := source.NewBackup(tempPath)
		if err != nil {
			return err
		}
		var copyErr error
		for {
			if copyErr = ctx.Err(); copyErr != nil {
				break
			}
			var more bool
			more, copyErr = backup.Step(128)
			if copyErr != nil {
				var code interface{ Code() int }
				if errors.As(copyErr, &code) && (code.Code()&255 == 5 || code.Code()&255 == 6) {
					select {
					case <-ctx.Done():
						copyErr = ctx.Err()
					case <-time.After(25 * time.Millisecond):
						copyErr = nil
					}
					if copyErr == nil {
						continue
					}
				}
				break
			}
			if !more {
				break
			}
			// Yield between bounded chunks rather than holding a database-wide read.
			select {
			case <-ctx.Done():
				copyErr = ctx.Err()
			case <-time.After(time.Millisecond):
			}
			if copyErr != nil {
				break
			}
		}
		return errors.Join(copyErr, backup.Finish())
	})
	connection.Close()
	if err != nil {
		return manifest, err
	}
	manifest, err = inspectBackup(ctx, tempPath)
	if err != nil {
		return manifest, err
	}
	assets := filepath.Join(directory, stem+".assets")
	complete := false
	defer func() {
		if !complete {
			os.RemoveAll(assets)
		}
	}()
	manifest.Artifacts, err = d.backupArtifacts(ctx, tempPath, assets, extra)
	if err != nil {
		return manifest, err
	}
	manifest.Format = 1
	manifest.Database = stem + ".sqlite"
	manifest.CreatedAt = time.Now().UTC()
	manifest.Reason = reason
	manifest.Protected = protect
	// Sync the SQLite API's finished file before publishing its name/manifest.
	if err = syncFile(tempPath); err != nil {
		return manifest, err
	}
	finalPath := filepath.Join(directory, manifest.Database)
	if err = os.Rename(tempPath, finalPath); err != nil {
		return manifest, err
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return manifest, err
	}
	if err = atomicWrite(finalPath+".json", payload, 0o600); err != nil {
		return manifest, err
	}
	complete = true
	return manifest, syncDirectory(directory)
}

func fileDigest(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil)), n, err
}
func inspectBackup(ctx context.Context, path string) (BackupManifest, error) {
	var result BackupManifest
	db, err := sql.Open("sqlite", readOnlyURI(path))
	if err != nil {
		return result, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			rows.Close()
			return result, err
		}
		if value != "ok" {
			rows.Close()
			return result, fmt.Errorf("storage: backup integrity: %s", value)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return result, err
	}
	violation := rows.Next()
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if violation {
		return result, errors.New("storage: backup has foreign key violations")
	}
	if err = db.QueryRowContext(ctx, "SELECT sqlite_version(),sqlite_source_id()").Scan(&result.Engine, &result.SourceID); err != nil {
		return result, err
	}
	rows, err = db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return result, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return result, err
		}
		names = append(names, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	result.Tables = make(map[string]int64)
	for _, name := range names {
		var count int64
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM \""+strings.ReplaceAll(name, "\"", "\"\"")+"\"").Scan(&count); err != nil {
			return result, err
		}
		result.Tables[name] = count
	}
	result.SHA256, result.Bytes, err = fileDigest(path)
	return result, err
}

// VerifyBackup validates both checksum and relational integrity before restore.
func VerifyBackup(ctx context.Context, manifestPath string) (BackupManifest, error) {
	var saved BackupManifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return saved, err
	}
	if err = json.Unmarshal(data, &saved); err != nil {
		return saved, err
	}
	if saved.Format != 1 || filepath.Base(saved.Database) != saved.Database || !strings.HasSuffix(saved.Database, ".sqlite") {
		return saved, errors.New("storage: invalid backup manifest")
	}
	actual, err := inspectBackup(ctx, filepath.Join(filepath.Dir(manifestPath), saved.Database))
	if err != nil {
		return saved, err
	}
	if actual.SHA256 != saved.SHA256 || actual.Bytes != saved.Bytes {
		return saved, errors.New("storage: backup checksum mismatch")
	}
	if len(actual.Tables) != len(saved.Tables) {
		return saved, errors.New("storage: backup table manifest mismatch")
	}
	for table, count := range actual.Tables {
		if saved.Tables[table] != count {
			return saved, fmt.Errorf("storage: backup table count mismatch: %s", table)
		}
	}
	if err := verifyArtifacts(manifestPath, saved); err != nil {
		return saved, err
	}
	return saved, nil
}
