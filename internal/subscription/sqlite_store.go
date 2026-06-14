package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/leothevan2444/moji/internal/downloader"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := downloader.OpenSQLiteDatabase(path)
	if err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Get(ctx context.Context, performerID string) (*PerformerState, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT performer_id, last_checked_at, last_error
FROM subscription_performer_state
WHERE performer_id = ?`, performerID)

	var (
		state            PerformerState
		lastCheckedAtRaw sql.NullString
	)
	if err := row.Scan(&state.PerformerID, &lastCheckedAtRaw, &state.LastError); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription: load performer state %q: %w", performerID, err)
	}

	lastCheckedAt, err := parseOptionalTimestamp(lastCheckedAtRaw)
	if err != nil {
		return nil, fmt.Errorf("subscription: parse last_checked_at for %q: %w", performerID, err)
	}
	state.LastCheckedAt = lastCheckedAt

	rows, err := s.db.QueryContext(ctx, `
SELECT
  sre.status,
  sre.release_key,
  sre.source,
  sre.title,
  sre.code,
  sre.release_date,
  sre.url,
  sre.query,
  sre.task_id,
  sre.seen_at
FROM subscription_performer_releases spr
JOIN subscription_release_entities sre ON sre.id = spr.release_id
WHERE spr.performer_id = ?
ORDER BY sre.seen_at DESC, sre.release_key ASC`, performerID)
	if err != nil {
		return nil, fmt.Errorf("subscription: load releases for %q: %w", performerID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			status string
			item   RecordedRelease
			seenAt string
		)
		if err := rows.Scan(
			&status,
			&item.Key,
			&item.Source,
			&item.Title,
			&item.Code,
			&item.Date,
			&item.URL,
			&item.Query,
			&item.TaskID,
			&seenAt,
		); err != nil {
			return nil, fmt.Errorf("subscription: scan release for %q: %w", performerID, err)
		}
		item.SeenAt, err = parseTimestamp(seenAt)
		if err != nil {
			return nil, fmt.Errorf("subscription: parse seen_at for %q release %q: %w", performerID, item.Key, err)
		}
		if status == "processed" {
			state.ProcessedReleases = append(state.ProcessedReleases, item)
		} else {
			state.PendingReleases = append(state.PendingReleases, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subscription: iterate releases for %q: %w", performerID, err)
	}

	return cloneState(&state), nil
}

func (s *SQLiteStore) Put(ctx context.Context, state *PerformerState) error {
	if state == nil {
		return errors.New("subscription: performer state is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("subscription: begin put state tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO subscription_performer_state (performer_id, last_checked_at, last_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(performer_id) DO UPDATE SET
  last_checked_at = excluded.last_checked_at,
  last_error = excluded.last_error,
  updated_at = excluded.updated_at`,
		state.PerformerID,
		formatOptionalTimestamp(state.LastCheckedAt),
		state.LastError,
		formatTimestamp(now),
		formatTimestamp(now),
	); err != nil {
		return fmt.Errorf("subscription: upsert performer state %q: %w", state.PerformerID, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM subscription_performer_releases WHERE performer_id = ?`, state.PerformerID); err != nil {
		return fmt.Errorf("subscription: clear release links for %q: %w", state.PerformerID, err)
	}

	writeRelease := func(status string, release RecordedRelease) error {
		releaseID, err := upsertReleaseEntity(ctx, tx, status, release, now)
		if err != nil {
			return err
		}
		linkedAt := release.SeenAt
		if linkedAt.IsZero() {
			linkedAt = now
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`,
			state.PerformerID,
			releaseID,
			formatTimestamp(linkedAt),
			formatTimestamp(now),
			formatTimestamp(now),
		); err != nil {
			return fmt.Errorf("subscription: insert performer-release link %q/%q: %w", state.PerformerID, release.Key, err)
		}
		return nil
	}

	for _, release := range state.ProcessedReleases {
		if err := writeRelease("processed", release); err != nil {
			return err
		}
	}
	for _, release := range state.PendingReleases {
		if err := writeRelease("pending", release); err != nil {
			return err
		}
	}

	if err := deleteOrphanReleaseEntities(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("subscription: commit put state tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Delete(ctx context.Context, performerID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("subscription: begin delete state tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM subscription_performer_state WHERE performer_id = ?`, performerID); err != nil {
		return fmt.Errorf("subscription: delete performer state %q: %w", performerID, err)
	}
	if err := deleteOrphanReleaseEntities(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("subscription: commit delete state tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) List(ctx context.Context) ([]*PerformerState, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT performer_id FROM subscription_performer_state ORDER BY last_checked_at DESC, performer_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("subscription: list performer states: %w", err)
	}
	defer rows.Close()

	states := make([]*PerformerState, 0)
	for rows.Next() {
		var performerID string
		if err := rows.Scan(&performerID); err != nil {
			return nil, fmt.Errorf("subscription: scan listed performer id: %w", err)
		}
		state, err := s.Get(ctx, performerID)
		if err != nil {
			return nil, err
		}
		if state != nil {
			states = append(states, state)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subscription: iterate performer states: %w", err)
	}

	sortStates(states)
	return states, nil
}

func upsertReleaseEntity(ctx context.Context, tx *sql.Tx, status string, release RecordedRelease, now time.Time) (int64, error) {
	seenAt := release.SeenAt
	if seenAt.IsZero() {
		seenAt = now
	}

	var id int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, release_date, url, query, task_id, last_error, seen_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)
ON CONFLICT(release_key) DO UPDATE SET
  status = excluded.status,
  source = excluded.source,
  title = excluded.title,
  code = excluded.code,
  release_date = excluded.release_date,
  url = excluded.url,
  query = excluded.query,
  task_id = CASE WHEN excluded.task_id <> '' THEN excluded.task_id ELSE subscription_release_entities.task_id END,
  seen_at = excluded.seen_at,
  updated_at = excluded.updated_at
RETURNING id`,
		release.Key,
		status,
		release.Source,
		release.Title,
		release.Code,
		release.Date,
		release.URL,
		release.Query,
		release.TaskID,
		formatTimestamp(seenAt),
		formatTimestamp(now),
		formatTimestamp(now),
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("subscription: upsert release entity %q: %w", release.Key, err)
	}

	return id, nil
}

func deleteOrphanReleaseEntities(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `
DELETE FROM subscription_release_entities
WHERE NOT EXISTS (
  SELECT 1
  FROM subscription_performer_releases spr
  WHERE spr.release_id = subscription_release_entities.id
)`); err != nil {
		return fmt.Errorf("subscription: delete orphan release entities: %w", err)
	}
	return nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatOptionalTimestamp(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTimestamp(*value)
}

func parseOptionalTimestamp(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	return parseTimestampValue(raw.String)
}

func parseTimestamp(raw string) (time.Time, error) {
	parsed, err := parseTimestampValue(raw)
	if err != nil {
		return time.Time{}, err
	}
	return *parsed, nil
}

func parseTimestampValue(raw string) (*time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func sortRecordedReleases(items []RecordedRelease) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].SeenAt.Equal(items[j].SeenAt) {
			return items[i].Key < items[j].Key
		}
		return items[i].SeenAt.After(items[j].SeenAt)
	})
}
