package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
)

type BackupFile struct{ Name, Path, ExpectedSHA256 string }
type BackupArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

func copyPrivateFile(ctx context.Context, source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("storage: backup artifact is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	good := false
	defer func() {
		out.Close()
		if !good {
			os.Remove(destination)
		}
	}()
	buffer := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := in.Read(buffer)
		if n > 0 {
			if _, err := out.Write(buffer[:n]); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	good = true
	return syncDirectory(filepath.Dir(destination))
}
func (d *Database) backupArtifacts(ctx context.Context, snapshot, assets string, extra []BackupFile) ([]BackupArtifact, error) {
	db, err := sqlx.Open("sqlite", readOnlyURI(snapshot))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	files := append([]BackupFile(nil), extra...)
	var exists int
	if err := db.GetContext(ctx, &exists, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='history_archives'"); err != nil {
		return nil, err
	}
	if exists > 0 {
		var archives []struct {
			Path   string
			SHA256 string
		}
		if err := db.SelectContext(ctx, &archives, "SELECT path,sha256 FROM history_archives ORDER BY id"); err != nil {
			return nil, err
		}
		for _, archive := range archives {
			if !filepath.IsLocal(archive.Path) {
				return nil, errors.New("storage: invalid history archive path")
			}
			files = append(files, BackupFile{Name: archive.Path, Path: filepath.Join(filepath.Dir(d.Path), archive.Path), ExpectedSHA256: archive.SHA256})
		}
	}
	seen := map[string]bool{}
	out := make([]BackupArtifact, 0, len(files))
	for _, file := range files {
		if !filepath.IsLocal(file.Name) || file.Name == "." || strings.Contains(file.Name, "\\") {
			return nil, errors.New("storage: invalid supplemental backup name")
		}
		clean := filepath.Clean(file.Name)
		if seen[clean] {
			return nil, fmt.Errorf("storage: duplicate backup artifact %q", clean)
		}
		seen[clean] = true
		destination := filepath.Join(assets, clean)
		if err := copyPrivateFile(ctx, file.Path, destination); err != nil {
			return nil, err
		}
		digest, size, err := fileDigest(destination)
		if err != nil {
			return nil, err
		}
		if file.ExpectedSHA256 != "" && digest != file.ExpectedSHA256 {
			return nil, fmt.Errorf("storage: history archive checksum mismatch: %s", file.Name)
		}
		out = append(out, BackupArtifact{Path: filepath.ToSlash(filepath.Join(filepath.Base(assets), clean)), SHA256: digest, Bytes: size})
	}
	if len(files) > 0 {
		if err := syncDirectory(assets); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func verifyArtifacts(manifestPath string, m BackupManifest) error {
	known := map[string]BackupArtifact{}
	prefix := strings.TrimSuffix(m.Database, ".sqlite") + ".assets/"
	for _, artifact := range m.Artifacts {
		if filepath.ToSlash(filepath.Clean(artifact.Path)) != artifact.Path || !filepath.IsLocal(artifact.Path) || !strings.HasPrefix(artifact.Path, strings.TrimSuffix(m.Database, ".sqlite")+".assets/") {
			return errors.New("storage: invalid backup artifact path")
		}
		name := strings.TrimPrefix(artifact.Path, prefix)
		if _, exists := known[name]; exists {
			return errors.New("storage: duplicate backup artifact")
		}
		known[name] = artifact
		digest, size, err := fileDigest(filepath.Join(filepath.Dir(manifestPath), artifact.Path))
		if err != nil {
			return err
		}
		if digest != artifact.SHA256 || size != artifact.Bytes {
			return fmt.Errorf("storage: backup artifact checksum mismatch: %s", artifact.Path)
		}
	}
	db, err := sqlx.Open("sqlite", readOnlyURI(filepath.Join(filepath.Dir(manifestPath), m.Database)))
	if err != nil {
		return err
	}
	defer db.Close()
	var exists int
	if err := db.Get(&exists, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='history_archives'"); err != nil {
		return err
	}
	if exists > 0 {
		var archives []struct{ Path, SHA256 string }
		if err := db.Select(&archives, "SELECT path,sha256 FROM history_archives"); err != nil {
			return err
		}
		for _, archive := range archives {
			artifact, ok := known[filepath.ToSlash(archive.Path)]
			if !ok || artifact.SHA256 != archive.SHA256 {
				return fmt.Errorf("storage: backup is missing referenced history %q", archive.Path)
			}
		}
	}

	return nil
}
