package stashboxcache

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

//go:embed sqlite_schema.sql
var sqliteSchema string

const sqliteSchemaVersion = "2"

type sqliteStore struct {
	db   *sqlx.DB
	path string
}

func openSQLiteStore(path string) (*sqliteStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("stashboxcache: database path is required")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("stashboxcache: create database directory: %w", err)
		}
	}
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("stashboxcache: open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA journal_mode = WAL", "PRAGMA busy_timeout = 5000"} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("stashboxcache: configure database: %w", err)
		}
	}
	store := &sqliteStore{db: db, path: path}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *sqliteStore) initSchema() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS stashbox_cache_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		return err
	}
	var version string
	err := s.db.Get(&version, `SELECT value FROM stashbox_cache_meta WHERE key = 'schema_version'`)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if version != "" && version != sqliteSchemaVersion {
		if err := s.reset(); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("stashboxcache: initialize schema: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO stashbox_cache_meta(key,value) VALUES('schema_version',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, sqliteSchemaVersion)
	return err
}

func (s *sqliteStore) reset() error {
	for _, table := range []string{"stashbox_cache_snapshot_items", "stashbox_cache_snapshot_pages", "stashbox_cache_snapshots", "stashbox_cache_scenes", "stashbox_cache_performers", "stashbox_cache_meta"} {
		if _, err := s.db.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) getPerformer(ctx context.Context, key PerformerKey) (performerDTO, time.Time, time.Time, bool, error) {
	var row struct {
		Payload      []byte `db:"payload_json"`
		FetchedAt    string `db:"fetched_at"`
		LastAccessed string `db:"last_accessed_at"`
	}
	err := s.db.GetContext(ctx, &row, `SELECT payload_json,fetched_at,last_accessed_at FROM stashbox_cache_performers WHERE endpoint=? AND performer_id=?`, key.Endpoint, key.PerformerID)
	if errors.Is(err, sql.ErrNoRows) {
		return performerDTO{}, time.Time{}, time.Time{}, false, nil
	}
	if err != nil {
		return performerDTO{}, time.Time{}, time.Time{}, false, err
	}
	var value performerDTO
	if err := json.Unmarshal(row.Payload, &value); err != nil {
		return performerDTO{}, time.Time{}, time.Time{}, false, err
	}
	fetchedAt, err := time.Parse(time.RFC3339Nano, row.FetchedAt)
	if err != nil {
		return performerDTO{}, time.Time{}, time.Time{}, false, err
	}
	lastAccessed, err := time.Parse(time.RFC3339Nano, row.LastAccessed)
	if err != nil {
		return performerDTO{}, time.Time{}, time.Time{}, false, err
	}
	return value, fetchedAt, lastAccessed, true, nil
}

func (s *sqliteStore) touchPerformer(ctx context.Context, key PerformerKey, now time.Time) {
	_, _ = s.db.ExecContext(ctx, `UPDATE stashbox_cache_performers SET last_accessed_at=? WHERE endpoint=? AND performer_id=?`, formatTime(now), key.Endpoint, key.PerformerID)
}

func (s *sqliteStore) putPerformer(ctx context.Context, key PerformerKey, value performerDTO, now time.Time) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO stashbox_cache_performers(endpoint,performer_id,payload_json,fetched_at,last_accessed_at) VALUES(?,?,?,?,?)
		ON CONFLICT(endpoint,performer_id) DO UPDATE SET payload_json=excluded.payload_json,fetched_at=excluded.fetched_at,last_accessed_at=excluded.last_accessed_at`, key.Endpoint, key.PerformerID, payload, formatTime(now), formatTime(now))
	return err
}

func (s *sqliteStore) getSnapshot(ctx context.Context, key PerformerKey, active bool) (*snapshot, error) {
	query := `SELECT generation,remote_count,complete,updated_at,last_accessed_at FROM stashbox_cache_snapshots WHERE endpoint=? AND performer_id=? AND active=? ORDER BY generation DESC LIMIT 1`
	var row struct {
		Generation   int64  `db:"generation"`
		RemoteCount  int    `db:"remote_count"`
		Complete     bool   `db:"complete"`
		UpdatedAt    string `db:"updated_at"`
		LastAccessed string `db:"last_accessed_at"`
	}
	if err := s.db.GetContext(ctx, &row, query, key.Endpoint, key.PerformerID, active); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	out := &snapshot{Key: key, Generation: row.Generation, RemoteCount: row.RemoteCount, Complete: row.Complete, Scenes: map[string]sceneDTO{}}
	var err error
	out.UpdatedAt, err = time.Parse(time.RFC3339Nano, row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("stashboxcache: parse snapshot updated time: %w", err)
	}
	out.LastAccessed, err = time.Parse(time.RFC3339Nano, row.LastAccessed)
	if err != nil {
		return nil, fmt.Errorf("stashboxcache: parse snapshot access time: %w", err)
	}
	var pages []struct {
		Number    int    `db:"page_number"`
		FetchedAt string `db:"fetched_at"`
	}
	if err := s.db.SelectContext(ctx, &pages, `SELECT page_number,fetched_at FROM stashbox_cache_snapshot_pages WHERE endpoint=? AND performer_id=? AND generation=? ORDER BY page_number`, key.Endpoint, key.PerformerID, row.Generation); err != nil {
		return nil, err
	}
	pageByNumber := map[int]int{}
	for _, item := range pages {
		at, err := time.Parse(time.RFC3339Nano, item.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("stashboxcache: parse page fetch time: %w", err)
		}
		out.Pages = append(out.Pages, cachedPage{Number: item.Number, FetchedAt: at})
		pageByNumber[item.Number] = len(out.Pages) - 1
	}
	var items []struct {
		Position int    `db:"position"`
		SceneID  string `db:"scene_id"`
		Payload  []byte `db:"payload_json"`
	}
	if err := s.db.SelectContext(ctx, &items, `SELECT i.position,i.scene_id,s.payload_json FROM stashbox_cache_snapshot_items i JOIN stashbox_cache_scenes s ON s.endpoint=i.endpoint AND s.scene_id=i.scene_id WHERE i.endpoint=? AND i.performer_id=? AND i.generation=? ORDER BY i.position`, key.Endpoint, key.PerformerID, row.Generation); err != nil {
		return nil, err
	}
	for _, item := range items {
		var scene sceneDTO
		if err := json.Unmarshal(item.Payload, &scene); err != nil {
			return nil, err
		}
		out.Scenes[item.SceneID] = scene
		pageNumber := item.Position/PageSize + 1
		if pageIndex, ok := pageByNumber[pageNumber]; ok {
			out.Pages[pageIndex].SceneIDs = append(out.Pages[pageIndex].SceneIDs, item.SceneID)
		}
	}
	return out, nil
}

func (s *sqliteStore) touchSnapshot(ctx context.Context, value *snapshot, now time.Time) {
	_, _ = s.db.ExecContext(ctx, `UPDATE stashbox_cache_snapshots SET last_accessed_at=? WHERE endpoint=? AND performer_id=? AND generation=?`, formatTime(now), value.Key.Endpoint, value.Key.PerformerID, value.Generation)
}

func (s *sqliteStore) putSnapshot(ctx context.Context, value *snapshot, scenes []sceneDTO, activate bool) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	nowText := formatTime(value.UpdatedAt)
	if activate {
		if _, err := tx.ExecContext(ctx, `UPDATE stashbox_cache_snapshots SET active=0 WHERE endpoint=? AND performer_id=? AND active=1 AND generation<>?`, value.Key.Endpoint, value.Key.PerformerID, value.Generation); err != nil {
			return err
		}
	}
	active := 0
	if activate {
		active = 1
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO stashbox_cache_snapshots(endpoint,performer_id,generation,remote_count,complete,active,updated_at,last_accessed_at) VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(endpoint,performer_id,generation) DO UPDATE SET remote_count=excluded.remote_count,complete=excluded.complete,active=excluded.active,updated_at=excluded.updated_at,last_accessed_at=excluded.last_accessed_at`, value.Key.Endpoint, value.Key.PerformerID, value.Generation, value.RemoteCount, value.Complete, active, nowText, formatTime(value.LastAccessed)); err != nil {
		return err
	}
	for _, scene := range scenes {
		payload, err := json.Marshal(scene)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO stashbox_cache_scenes(endpoint,scene_id,payload_json) VALUES(?,?,?) ON CONFLICT(endpoint,scene_id) DO UPDATE SET payload_json=excluded.payload_json`, value.Key.Endpoint, scene.ID, payload); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stashbox_cache_snapshot_pages WHERE endpoint=? AND performer_id=? AND generation=?`, value.Key.Endpoint, value.Key.PerformerID, value.Generation); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stashbox_cache_snapshot_items WHERE endpoint=? AND performer_id=? AND generation=?`, value.Key.Endpoint, value.Key.PerformerID, value.Generation); err != nil {
		return err
	}
	position := 0
	for _, page := range value.Pages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO stashbox_cache_snapshot_pages(endpoint,performer_id,generation,page_number,fetched_at) VALUES(?,?,?,?,?)`, value.Key.Endpoint, value.Key.PerformerID, value.Generation, page.Number, formatTime(page.FetchedAt)); err != nil {
			return err
		}
		for _, id := range page.SceneIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO stashbox_cache_snapshot_items(endpoint,performer_id,generation,position,scene_id) VALUES(?,?,?,?,?)`, value.Key.Endpoint, value.Key.PerformerID, value.Generation, position, id); err != nil {
				return err
			}
			position++
		}
	}
	return tx.Commit()
}

func (s *sqliteStore) cleanup(ctx context.Context, cutoff time.Time) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	cutoffText := formatTime(cutoff)
	if _, err := tx.ExecContext(ctx, `DELETE FROM stashbox_cache_snapshots WHERE last_accessed_at < ?`, cutoffText); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stashbox_cache_performers WHERE last_accessed_at < ?`, cutoffText); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stashbox_cache_scenes WHERE NOT EXISTS (SELECT 1 FROM stashbox_cache_snapshot_items i WHERE i.endpoint=stashbox_cache_scenes.endpoint AND i.scene_id=stashbox_cache_scenes.scene_id)`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *sqliteStore) clear(ctx context.Context) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"stashbox_cache_snapshot_items", "stashbox_cache_snapshot_pages", "stashbox_cache_snapshots", "stashbox_cache_scenes", "stashbox_cache_performers"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `VACUUM`)
	return err
}

func (s *sqliteStore) status(ctx context.Context) (Status, error) {
	status := Status{DatabasePath: s.path}
	queries := []struct {
		dest *int
		sql  string
	}{{&status.SceneCount, `SELECT COUNT(*) FROM stashbox_cache_scenes`}, {&status.PerformerCount, `SELECT COUNT(*) FROM stashbox_cache_performers`}, {&status.SnapshotCount, `SELECT COUNT(*) FROM stashbox_cache_snapshots`}}
	for _, query := range queries {
		if err := s.db.GetContext(ctx, query.dest, query.sql); err != nil {
			return status, err
		}
	}
	if err := s.db.GetContext(ctx, &status.UsedBytes, `SELECT COALESCE((SELECT SUM(length(payload_json)) FROM stashbox_cache_scenes),0)+COALESCE((SELECT SUM(length(payload_json)) FROM stashbox_cache_performers),0)`); err != nil {
		return status, err
	}
	return status, nil
}

func (s *sqliteStore) invalidateEndpoint(ctx context.Context, endpoint string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"stashbox_cache_snapshots", "stashbox_cache_scenes", "stashbox_cache_performers"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE endpoint=?`, endpoint); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
