package datamigration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/durablefile"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/subscription"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type Result struct {
	Format         int               `json:"format"`
	SourceSHA256   string            `json:"source_sha256"`
	CacheSHA256    string            `json:"cache_sha256,omitempty"`
	VerifiedTables map[string]string `json:"verified_table_sha256"`
	Directory      string            `json:"directory"`
	Source         string            `json:"source"`
	Counts         map[string]int64  `json:"counts"`
	History        Archive           `json:"history"`
	CompletedAt    time.Time         `json:"completed_at"`
}

func readOnly(path string) (*sqlx.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	q := url.Values{}
	q.Set("mode", "ro")
	q.Add("_pragma", "query_only(1)")
	u.RawQuery = q.Encode()
	db, err := sqlx.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// Prepare accepts a standalone consistent legacy backup, never the live source.
// No active-layout pointer is published and no external service is contacted.
func Prepare(ctx context.Context, sourcePath, cachePath, destination string) (result Result, err error) {
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return result, err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return result, err
	}
	result = Result{Format: 1, Directory: destination, Source: sourcePath, Counts: map[string]int64{}, VerifiedTables: map[string]string{}}
	result.SourceSHA256, err = digestFile(sourcePath)
	if err != nil {
		return result, err
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		return result, errors.New("migration: staging destination must not already exist")
	}
	source, err := readOnly(sourcePath)
	if err != nil {
		return result, err
	}
	defer source.Close()
	var version string
	if err := source.GetContext(ctx, &version, "SELECT value FROM task_store_meta WHERE key='schema_version'"); err != nil {
		return result, err
	}
	if version != "16" {
		return result, fmt.Errorf("migration: unsupported legacy version %q", version)
	}
	if err := validateLegacyPayloads(ctx, source); err != nil {
		return result, err
	}
	tasks, err := taskruntime.ReadLegacyTasks(ctx, source)
	if err != nil {
		return result, err
	}
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(destination))
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, manager.Close()) }()
	store := taskruntime.NewRelationalTaskStore(manager.Core, manager.Observations)
	if err := store.ImportLegacyTasks(ctx, tasks); err != nil {
		return result, err
	}
	if err := store.VerifyImportedTasks(ctx, tasks); err != nil {
		return result, err
	}
	for _, table := range []string{"subscription_performer_state", "subscription_release_entities", "subscription_performer_releases"} {
		if err := copyTable(ctx, source, manager.Core, table); err != nil {
			return result, err
		}
		digest, err := compareCopiedTable(ctx, source, manager.Core.Reader, table)
		if err != nil {
			return result, err
		}
		result.VerifiedTables[table] = digest
	}
	legacy := subscription.NewLegacyReader(source)
	current := subscription.NewManagedStore(manager.Core)
	reviews, err := legacy.ListReleaseReviews(ctx)
	if err != nil {
		return result, err
	}
	if err := current.PutReleaseReviews(ctx, reviews); err != nil {
		return result, err
	}
	loadedReviews, err := current.ListReleaseReviews(ctx)
	if err != nil {
		return result, err
	}
	// Typed FKs clear associations to tasks already deleted in the legacy store;
	// the untouched record_json retains the original historical identifier.
	knownTasks := map[string]bool{}
	for _, task := range tasks {
		knownTasks[task.ID] = true
	}
	for i := range reviews {
		if !knownTasks[reviews[i].TaskID] {
			reviews[i].TaskID = ""
		}
	}
	if err := sameJSON(reviews, loadedReviews, "release reviews"); err != nil {
		return result, err
	}
	runs, err := legacy.LoadCheckRuns(ctx)
	if err != nil {
		return result, err
	}
	for _, run := range runs {
		if err := current.SaveCheckRun(ctx, run); err != nil {
			return result, err
		}
	}
	loadedRuns, err := current.LoadCheckRuns(ctx)
	if err != nil {
		return result, err
	}
	// Both readers sort newest first. Ties are normalized below for comparison.
	sort.Slice(runs, func(i, j int) bool { return runs[i].ID < runs[j].ID })
	sort.Slice(loadedRuns, func(i, j int) bool { return loadedRuns[i].ID < loadedRuns[j].ID })
	if err := sameJSON(runs, loadedRuns, "subscription checks"); err != nil {
		return result, err
	}
	var stateRows []struct{ Key, Value string }
	if err := source.SelectContext(ctx, &stateRows, "SELECT key,value FROM downloader_state"); err != nil {
		return result, err
	}
	state := map[string]string{}
	for _, row := range stateRows {
		state[row.Key] = row.Value
	}
	if err := downloader.ImportLegacyState(ctx, manager, state); err != nil {
		return result, err
	}
	if cachePath != "" {
		result.CacheSHA256, err = digestFile(cachePath)
		if err != nil {
			return result, err
		}
		cache, err := readOnly(cachePath)
		if err != nil {
			return result, err
		}
		for _, table := range []string{"stashbox_cache_performers", "stashbox_cache_scenes", "stashbox_cache_snapshots", "stashbox_cache_snapshot_pages", "stashbox_cache_snapshot_items"} {
			if err := copyTable(ctx, cache, manager.Cache, table); err != nil {
				cache.Close()
				return result, err
			}
			digest, err := compareCopiedTable(ctx, cache, manager.Cache.Reader, table)
			if err != nil {
				cache.Close()
				return result, err
			}
			result.VerifiedTables[table] = digest
		}
		if err := cache.Close(); err != nil {
			return result, err
		}
	}
	result.History, err = archiveEvents(ctx, source, filepath.Join(destination, "legacy-task-events.jsonl.gz"))
	if err != nil {
		return result, err
	}
	var sourceEvents int64
	if err := source.GetContext(ctx, &sourceEvents, "SELECT count(*) FROM task_events"); err != nil {
		return result, err
	}
	if sourceEvents != result.History.Rows {
		return result, errors.New("migration: source event count changed during archive")
	}
	if err := manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO history_archives(id,path,sha256,source_rows,retained_rows,created_at) VALUES('legacy-task-events',?,?,?,?,?)`, filepath.Base(result.History.Path), result.History.SHA256, result.History.Rows, 0, time.Now().UTC().Format(time.RFC3339Nano))
		return err
	}); err != nil {
		return result, err
	}
	for _, table := range []string{"tasks", "subscription_performer_state", "subscription_release_entities", "subscription_performer_releases", "subscription_release_reviews", "subscription_check_runs"} {
		var from, to int64
		if err := source.GetContext(ctx, &from, "SELECT count(*) FROM "+table); err != nil {
			return result, err
		}
		if err := manager.Core.Reader.GetContext(ctx, &to, "SELECT count(*) FROM "+table); err != nil {
			return result, err
		}
		if from != to {
			return result, fmt.Errorf("migration: %s rows differ: source=%d destination=%d", table, from, to)
		}
		result.Counts[table] = to
	}
	if err := manager.Observations.FlushAll(ctx); err != nil {
		return result, err
	}
	result.CompletedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	if err := durablefile.Write(filepath.Join(destination, "migration-report.json"), raw, 0o600); err != nil {
		return result, err
	}
	return result, nil
}

func copyTable(ctx context.Context, source *sqlx.DB, destination *storage.Database, table string) error {
	allowed := map[string]bool{"subscription_performer_state": true, "subscription_release_entities": true, "subscription_performer_releases": true, "stashbox_cache_performers": true, "stashbox_cache_scenes": true, "stashbox_cache_snapshots": true, "stashbox_cache_snapshot_pages": true, "stashbox_cache_snapshot_items": true}
	if !allowed[table] {
		return errors.New("migration: table not allowed")
	}
	rows, err := source.QueryxContext(ctx, "SELECT * FROM "+table)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	for i, col := range columns {
		if strings.ContainsAny(col, "\"\x00") {
			return errors.New("migration: invalid column name")
		}
		columns[i] = "\"" + col + "\""
	}
	statement := "INSERT INTO " + table + "(" + strings.Join(columns, ",") + ") VALUES(" + strings.Join(placeholders, ",") + ")"
	batch := make([][]any, 0, 128)
	flush := func() error {
		return destination.WithWrite(ctx, func(tx *sqlx.Tx) error {
			for _, values := range batch {
				if _, err := tx.ExecContext(ctx, statement, values...); err != nil {
					return err
				}
			}
			return nil
		})
	}
	for rows.Next() {
		values, err := rows.SliceScan()
		if err != nil {
			return err
		}
		batch = append(batch, values)
		if len(batch) == 128 {
			if err := flush(); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		return flush()
	}
	return nil
}

func sameJSON(expected, actual any, name string) error {
	left, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	right, err := json.Marshal(actual)
	if err != nil {
		return err
	}
	if !bytes.Equal(left, right) {
		return fmt.Errorf("migration: %s differ after migration", name)
	}
	return nil
}
