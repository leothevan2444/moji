package datamigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// compareCopiedTable streams every stored value in primary-key order, including
// NULLs and BLOBs. Counts alone do not prove a cache/entity copy was lossless.
func compareCopiedTable(ctx context.Context, source, target *sqlx.DB, table string) (string, error) {
	type column struct {
		Name string `db:"name"`
		PK   int    `db:"pk"`
	}
	var columns []column
	if err := source.SelectContext(ctx, &columns, "SELECT name,pk FROM pragma_table_info(?)", table); err != nil {
		return "", err
	}
	quote := func(value string) (string, error) {
		if strings.ContainsAny(value, "\"\x00") {
			return "", errors.New("migration: invalid SQL identifier")
		}
		return "\"" + value + "\"", nil
	}
	names := make([]string, 0, len(columns))
	var keys []column
	for _, column := range columns {
		quoted, err := quote(column.Name)
		if err != nil {
			return "", err
		}
		names = append(names, quoted)
		if column.PK > 0 {
			keys = append(keys, column)
		}
	}
	if len(keys) == 0 {
		return "", errors.New("migration: copied table lacks a primary key")
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].PK < keys[j].PK })
	order := make([]string, 0, len(keys))
	for _, key := range keys {
		quoted, err := quote(key.Name)
		if err != nil {
			return "", err
		}
		order = append(order, quoted)
	}
	quoted, err := quote(table)
	if err != nil {
		return "", err
	}
	query := "SELECT " + strings.Join(names, ",") + " FROM " + quoted + " ORDER BY " + strings.Join(order, ",")
	digest := func(db *sqlx.DB) (string, error) {
		rows, err := db.QueryxContext(ctx, query)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		h := sha256.New()
		encoder := json.NewEncoder(h)
		if err := encoder.Encode(names); err != nil {
			return "", err
		}
		for rows.Next() {
			values, err := rows.SliceScan()
			if err != nil {
				return "", err
			}
			if err := encoder.Encode(values); err != nil {
				return "", err
			}
		}
		if err := rows.Err(); err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}
	left, err := digest(source)
	if err != nil {
		return "", err
	}
	right, err := digest(target)
	if err != nil {
		return "", err
	}
	if left != right {
		return "", fmt.Errorf("migration: copied %s data differs", table)
	}
	return left, nil
}
