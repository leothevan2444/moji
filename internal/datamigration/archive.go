// Package datamigration prepares immutable staging generations from legacy data.
// Production activation is a separate explicit operation after verification.
package datamigration

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
)

type Archive struct {
	Format  int      `json:"format"`
	Path    string   `json:"path"`
	SHA256  string   `json:"sha256"`
	Rows    int64    `json:"rows"`
	Columns []string `json:"columns"`
}

func archiveEvents(ctx context.Context, source *sqlx.DB, path string) (result Archive, err error) {
	result = Archive{Format: 1, Path: path}
	rows, err := source.QueryxContext(ctx, "SELECT * FROM task_events ORDER BY id")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Columns, err = rows.Columns()
	if err != nil {
		return result, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return result, err
	}
	published := false
	defer func() {
		file.Close()
		if !published {
			os.Remove(path)
		}
	}()
	digest := sha256.New()
	compressed := gzip.NewWriter(io.MultiWriter(file, digest))
	encoder := json.NewEncoder(compressed)
	if err := encoder.Encode(map[string]any{"format": 1, "columns": result.Columns}); err != nil {
		return result, err
	}
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		values, err := rows.SliceScan()
		if err != nil {
			return result, err
		}
		if err := encoder.Encode(values); err != nil {
			return result, err
		}
		result.Rows++
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if err := compressed.Close(); err != nil {
		return result, err
	}
	if err := file.Sync(); err != nil {
		return result, err
	}
	if err := file.Close(); err != nil {
		return result, err
	}
	result.SHA256 = hex.EncodeToString(digest.Sum(nil))
	if err := VerifyArchive(ctx, result); err != nil {
		return result, err
	}
	published = true
	return result, nil
}

func VerifyArchive(ctx context.Context, archive Archive) error {
	file, err := os.Open(archive.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	digest := sha256.New()
	compressed, err := gzip.NewReader(io.TeeReader(file, digest))
	if err != nil {
		return err
	}
	defer compressed.Close()
	decoder := json.NewDecoder(compressed)
	decoder.UseNumber()
	var header struct {
		Format  int      `json:"format"`
		Columns []string `json:"columns"`
	}
	if err := decoder.Decode(&header); err != nil {
		return err
	}
	if header.Format != 1 || len(header.Columns) != len(archive.Columns) {
		return errors.New("migration: invalid archive header")
	}
	for i, name := range header.Columns {
		if archive.Columns[i] != name {
			return errors.New("migration: changed archive columns")
		}
	}
	var count int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var row []any
		err := decoder.Decode(&row)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if len(row) != len(header.Columns) {
			return errors.New("migration: invalid archive record")
		}
		count++
	}
	// The gzip reader verifies its footer while reaching EOF; account for any
	// remaining compressed stream bytes in the digest too.
	if _, err := io.Copy(digest, file); err != nil {
		return err
	}
	if count != archive.Rows || hex.EncodeToString(digest.Sum(nil)) != archive.SHA256 {
		return fmt.Errorf("migration: archive verification failed: rows=%d expected=%d", count, archive.Rows)
	}
	return nil
}
