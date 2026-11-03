package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/jmoiron/sqlx"
)

var definitionRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)
var definitionChecksum = regexp.MustCompile(`^[0-9a-f]{64}$`)
var definitionFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*\.yml$`)

// Inventory is read from the completed SQLite snapshot, never a later live
// pointer. Before v4 there is no bundle table and therefore no required assets.
func indexerDefinitionAssets(ctx context.Context, db *sqlx.DB, generation string) ([]BackupFile, error) {
	var exists int
	if err := db.GetContext(ctx, &exists, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='indexer_definition_bundles'`); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, nil
	}
	var bundles []struct {
		Revision string
		Manifest string `db:"manifest_json"`
		SHA256   string `db:"manifest_sha256"`
	}
	if err := db.SelectContext(ctx, &bundles, `SELECT revision,manifest_json,manifest_sha256 FROM indexer_definition_bundles ORDER BY revision`); err != nil {
		return nil, err
	}
	files := []BackupFile{}
	for _, bundle := range bundles {
		if !definitionRevision.MatchString(bundle.Revision) || !definitionChecksum.MatchString(bundle.SHA256) {
			return nil, errors.New("storage: invalid indexer bundle identity")
		}
		sum := sha256.Sum256([]byte(bundle.Manifest))
		if hex.EncodeToString(sum[:]) != bundle.SHA256 {
			return nil, errors.New("storage: indexer bundle manifest checksum mismatch")
		}
		var manifest struct {
			Revision string `json:"revision"`
			Files    []struct {
				Name, SHA256 string
				Bytes        int64
			} `json:"files"`
		}
		if err := json.Unmarshal([]byte(bundle.Manifest), &manifest); err != nil || manifest.Revision != bundle.Revision || len(manifest.Files) == 0 || len(manifest.Files) > 4096 {
			return nil, errors.New("storage: invalid indexer bundle manifest")
		}
		var inventory []struct {
			Name, SHA256 string
			Bytes        int64
		}
		if err := db.SelectContext(ctx, &inventory, `SELECT name,sha256,bytes FROM indexer_definition_files WHERE revision=? ORDER BY name`, bundle.Revision); err != nil {
			return nil, err
		}
		if len(inventory) != len(manifest.Files) {
			return nil, errors.New("storage: incomplete indexer bundle inventory")
		}
		expected := map[string]string{}
		for _, file := range manifest.Files {
			if _, duplicate := expected[file.Name]; duplicate {
				return nil, errors.New("storage: duplicate indexer bundle file")
			}
			expected[file.Name] = fmt.Sprintf("%s:%d", file.SHA256, file.Bytes)
		}
		prefix := filepath.Join("indexers", "bundles", bundle.Revision)
		files = append(files, BackupFile{Name: filepath.Join(prefix, "manifest.json"), Path: filepath.Join(generation, prefix, "manifest.json"), ExpectedSHA256: bundle.SHA256})
		for _, file := range inventory {
			if file.Name != "schema.json" && !definitionFile.MatchString(file.Name) || !definitionChecksum.MatchString(file.SHA256) || file.Bytes < 0 || file.Bytes > 1<<20 || expected[file.Name] != fmt.Sprintf("%s:%d", file.SHA256, file.Bytes) {
				return nil, errors.New("storage: invalid indexer bundle inventory")
			}
			name := filepath.Join(prefix, file.Name)
			files = append(files, BackupFile{Name: name, Path: filepath.Join(generation, name), ExpectedSHA256: file.SHA256})
		}
	}
	return files, nil
}
