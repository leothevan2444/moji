package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type SQLiteStore struct {
	db   *sqlx.DB
	core *storage.Database
}

// NewManagedStore shares the application storage owner; it never opens a pool.
func NewManagedStore(core *storage.Database) *SQLiteStore {
	return &SQLiteStore{db: core.Reader, core: core}
}
func (s *SQLiteStore) withWrite(ctx context.Context, work func(*sqlx.Tx) error) error {
	if s.core != nil {
		return s.core.WithWrite(ctx, work)
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = work(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := taskruntime.OpenSQLiteDatabase(path)
	if err != nil {
		return nil, err
	}
	if err := ensureSubscriptionSQLiteSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Get(ctx context.Context, performerID string) (*PerformerState, error) {
	var stateRow sqlitePerformerStateRow
	if err := s.db.GetContext(ctx, &stateRow, `
SELECT performer_id, last_checked_at, last_error
FROM subscription_performer_state
WHERE performer_id = ?`, performerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription: load performer state %q: %w", performerID, err)
	}

	state, err := stateRow.toState()
	if err != nil {
		return nil, err
	}

	releaseRows := make([]sqliteReleaseRow, 0)
	if err := s.db.SelectContext(ctx, &releaseRows, `
SELECT
  sre.status,
  sre.release_key,
  sre.source,
  sre.title,
  sre.code,
  sre.release_date,
  sre.url,
  sre.task_id,
  sre.performer_count,
  sre.performer_names,
  sre.classification,
  sre.decision,
  sre.decision_reason,
  sre.seen_at
FROM subscription_performer_releases spr
JOIN subscription_release_entities sre ON sre.id = spr.release_id
WHERE spr.performer_id = ?
ORDER BY sre.seen_at DESC, sre.release_key ASC`, performerID); err != nil {
		return nil, fmt.Errorf("subscription: load releases for %q: %w", performerID, err)
	}

	for _, row := range releaseRows {
		status, release, err := row.toRecordedRelease()
		if err != nil {
			return nil, fmt.Errorf("subscription: decode release for %q: %w", performerID, err)
		}
		if status == "processed" {
			state.ProcessedReleases = append(state.ProcessedReleases, release)
		} else {
			state.PendingReleases = append(state.PendingReleases, release)
		}
	}

	return cloneState(state), nil
}

func (s *SQLiteStore) Put(ctx context.Context, state *PerformerState) error {
	if state == nil {
		return errors.New("subscription: performer state is nil")
	}

	return s.withWrite(ctx, func(tx *sqlx.Tx) error {

		now := time.Now().UTC()
		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO subscription_performer_state (performer_id, last_checked_at, last_error, created_at, updated_at)
VALUES (:performer_id, :last_checked_at, :last_error, :created_at, :updated_at)
ON CONFLICT(performer_id) DO UPDATE SET
  last_checked_at = excluded.last_checked_at,
  last_error = excluded.last_error,
  updated_at = excluded.updated_at
WHERE subscription_performer_state.last_checked_at IS NOT excluded.last_checked_at OR subscription_performer_state.last_error IS NOT excluded.last_error`,
			map[string]any{
				"performer_id":    state.PerformerID,
				"last_checked_at": formatOptionalTimestamp(state.LastCheckedAt),
				"last_error":      nullableStringParam(state.LastError),
				"created_at":      formatTimestamp(now),
				"updated_at":      formatTimestamp(now),
			},
		); err != nil {
			return fmt.Errorf("subscription: upsert performer state %q: %w", state.PerformerID, err)
		}

		existingLinks := []int64{}
		if err := tx.SelectContext(ctx, &existingLinks, "SELECT release_id FROM subscription_performer_releases WHERE performer_id=?", state.PerformerID); err != nil {
			return err
		}
		retained := map[int64]bool{}

		writeRelease := func(status string, release RecordedRelease) error {
			releaseID, err := upsertReleaseEntity(ctx, tx, status, release, now)
			if err != nil {
				return err
			}
			retained[releaseID] = true
			linkedAt := release.SeenAt
			if linkedAt.IsZero() {
				linkedAt = now
			}
			if _, err := tx.NamedExecContext(ctx, `
INSERT INTO subscription_performer_releases (performer_id, release_id, linked_at)
VALUES (:performer_id, :release_id, :linked_at) ON CONFLICT(performer_id,release_id) DO NOTHING`,
				map[string]any{
					"performer_id": state.PerformerID,
					"release_id":   releaseID,
					"linked_at":    formatTimestamp(linkedAt),
				},
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

		for _, id := range existingLinks {
			if !retained[id] {
				if _, err := tx.ExecContext(ctx, "DELETE FROM subscription_performer_releases WHERE performer_id=? AND release_id=?", state.PerformerID, id); err != nil {
					return err
				}
			}
		}

		if s.core == nil {
			if err := deleteOrphanReleaseEntities(ctx, tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *SQLiteStore) Delete(ctx context.Context, performerID string) error {
	return s.withWrite(ctx, func(tx *sqlx.Tx) error {

		if _, err := tx.ExecContext(ctx, `DELETE FROM subscription_performer_state WHERE performer_id = ?`, performerID); err != nil {
			return fmt.Errorf("subscription: delete performer state %q: %w", performerID, err)
		}
		if s.core == nil {
			if err := deleteOrphanReleaseEntities(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SQLiteStore) List(ctx context.Context) ([]*PerformerState, error) {
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var rows []sqlitePerformerStateRow
	if err := tx.SelectContext(ctx, &rows, "SELECT performer_id,last_checked_at,last_error FROM subscription_performer_state ORDER BY performer_id"); err != nil {
		return nil, err
	}
	states := make([]*PerformerState, 0, len(rows))
	byID := map[string]*PerformerState{}
	for _, row := range rows {
		state, err := row.toState()
		if err != nil {
			return nil, err
		}
		states = append(states, state)
		byID[state.PerformerID] = state
	}
	var releases []struct {
		PerformerID string `db:"performer_id"`
		sqliteReleaseRow
	}
	if err := tx.SelectContext(ctx, &releases, `SELECT spr.performer_id,sre.status,sre.release_key,sre.source,sre.title,sre.code,sre.release_date,sre.url,sre.task_id,sre.performer_count,sre.performer_names,sre.classification,sre.decision,sre.decision_reason,sre.seen_at FROM subscription_performer_releases spr JOIN subscription_release_entities sre ON sre.id=spr.release_id ORDER BY spr.performer_id,sre.seen_at DESC,sre.release_key`); err != nil {
		return nil, err
	}
	for _, row := range releases {
		state := byID[row.PerformerID]
		if state == nil {
			continue
		}
		status, release, err := row.toRecordedRelease()
		if err != nil {
			return nil, err
		}
		if status == "processed" {
			state.ProcessedReleases = append(state.ProcessedReleases, release)
		} else {
			state.PendingReleases = append(state.PendingReleases, release)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	sortStates(states)
	return states, nil
}

type sqlitePerformerStateRow struct {
	PerformerID   string         `db:"performer_id"`
	LastCheckedAt sql.NullString `db:"last_checked_at"`
	LastError     sql.NullString `db:"last_error"`
}

func (r sqlitePerformerStateRow) toState() (*PerformerState, error) {
	lastCheckedAt, err := parseOptionalTimestamp(r.LastCheckedAt)
	if err != nil {
		return nil, fmt.Errorf("subscription: parse last_checked_at for %q: %w", r.PerformerID, err)
	}
	return &PerformerState{
		PerformerID:   r.PerformerID,
		LastCheckedAt: lastCheckedAt,
		LastError:     nullableStringValue(r.LastError),
	}, nil
}

type sqliteReleaseRow struct {
	Status         string         `db:"status"`
	Key            string         `db:"release_key"`
	Source         string         `db:"source"`
	Title          string         `db:"title"`
	Code           string         `db:"code"`
	Date           sql.NullString `db:"release_date"`
	URL            sql.NullString `db:"url"`
	TaskID         sql.NullString `db:"task_id"`
	PerformerCount int            `db:"performer_count"`
	PerformerNames string         `db:"performer_names"`
	Classification string         `db:"classification"`
	Decision       string         `db:"decision"`
	DecisionReason string         `db:"decision_reason"`
	SeenAt         string         `db:"seen_at"`
}

func (r sqliteReleaseRow) toRecordedRelease() (string, RecordedRelease, error) {
	item := RecordedRelease{
		Key:            r.Key,
		Source:         r.Source,
		Title:          r.Title,
		Code:           r.Code,
		Date:           nullableStringValue(r.Date),
		URL:            nullableStringValue(r.URL),
		TaskID:         nullableStringValue(r.TaskID),
		PerformerCount: r.PerformerCount,
		Classification: ReleaseClassification(r.Classification),
		Decision:       ReleaseDecision(r.Decision),
		DecisionReason: r.DecisionReason,
	}
	if err := json.Unmarshal([]byte(r.PerformerNames), &item.PerformerNames); err != nil {
		return "", RecordedRelease{}, fmt.Errorf("subscription: parse performer_names for release %q: %w", r.Key, err)
	}
	seenAt, err := parseTimestamp(r.SeenAt)
	if err != nil {
		return "", RecordedRelease{}, fmt.Errorf("subscription: parse seen_at for release %q: %w", r.Key, err)
	}
	item.SeenAt = seenAt
	return r.Status, item, nil
}

func upsertReleaseEntity(ctx context.Context, tx *sqlx.Tx, status string, release RecordedRelease, now time.Time) (int64, error) {
	seenAt := release.SeenAt
	if seenAt.IsZero() {
		seenAt = now
	}
	taskID, err := sanitizeReleaseTaskID(ctx, tx, release.Key, release.TaskID)
	if err != nil {
		return 0, err
	}

	params := map[string]any{
		"release_key":     release.Key,
		"status":          status,
		"source":          release.Source,
		"title":           release.Title,
		"code":            release.Code,
		"release_date":    nullableStringParam(release.Date),
		"url":             nullableStringParam(release.URL),
		"task_id":         taskID,
		"performer_count": release.PerformerCount,
		"performer_names": marshalPerformerNames(release.PerformerNames),
		"classification":  string(release.Classification),
		"decision":        string(release.Decision),
		"decision_reason": release.DecisionReason,
		"seen_at":         formatTimestamp(seenAt),
		"created_at":      formatTimestamp(now),
		"updated_at":      formatTimestamp(now),
	}

	rows, err := sqlx.NamedQueryContext(ctx, tx, `
INSERT INTO subscription_release_entities (
  release_key, status, source, title, code, release_date, url, task_id, performer_count, performer_names, classification, decision, decision_reason, seen_at, created_at, updated_at
) VALUES (
  :release_key, :status, :source, :title, :code, :release_date, :url, :task_id, :performer_count, :performer_names, :classification, :decision, :decision_reason, :seen_at, :created_at, :updated_at
)
ON CONFLICT(release_key) DO UPDATE SET
  status = excluded.status,
  source = excluded.source,
  title = excluded.title,
  code = excluded.code,
  release_date = excluded.release_date,
  url = excluded.url,
  task_id = COALESCE(excluded.task_id, subscription_release_entities.task_id),
  performer_count = excluded.performer_count,
  performer_names = excluded.performer_names,
  classification = excluded.classification,
  decision = excluded.decision,
  decision_reason = excluded.decision_reason,
  seen_at = excluded.seen_at,
  updated_at = excluded.updated_at
WHERE subscription_release_entities.status IS NOT excluded.status OR subscription_release_entities.source IS NOT excluded.source OR subscription_release_entities.title IS NOT excluded.title OR subscription_release_entities.code IS NOT excluded.code OR subscription_release_entities.release_date IS NOT excluded.release_date OR subscription_release_entities.url IS NOT excluded.url OR subscription_release_entities.performer_count IS NOT excluded.performer_count OR subscription_release_entities.performer_names IS NOT excluded.performer_names OR subscription_release_entities.classification IS NOT excluded.classification OR subscription_release_entities.decision IS NOT excluded.decision OR subscription_release_entities.decision_reason IS NOT excluded.decision_reason OR subscription_release_entities.seen_at IS NOT excluded.seen_at OR (excluded.task_id IS NOT NULL AND subscription_release_entities.task_id IS NOT excluded.task_id)
RETURNING id`, params)
	if err != nil {
		return 0, fmt.Errorf("subscription: upsert release entity %q: %w", release.Key, err)
	}
	defer rows.Close()

	var id int64
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("subscription: read release entity id for %q: %w", release.Key, err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("subscription: finalize release entity upsert %q: %w", release.Key, err)
	}
	if id == 0 {
		if err := rows.Close(); err != nil {
			return 0, err
		}
		if err := tx.GetContext(ctx, &id, "SELECT id FROM subscription_release_entities WHERE release_key=?", release.Key); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func sanitizeReleaseTaskID(ctx context.Context, tx *sqlx.Tx, releaseKey, candidate string) (any, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" {
		exists, err := sqliteTaskExists(ctx, tx, candidate)
		if err != nil {
			return nil, fmt.Errorf("subscription: verify task %q for release %q: %w", candidate, releaseKey, err)
		}
		if exists {
			return candidate, nil
		}
		return nil, nil
	}

	var existing sql.NullString
	err := tx.GetContext(ctx, &existing, `SELECT task_id FROM subscription_release_entities WHERE release_key = ?`, releaseKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("subscription: load existing task for release %q: %w", releaseKey, err)
	}
	if !existing.Valid || strings.TrimSpace(existing.String) == "" {
		return nil, nil
	}
	exists, err := sqliteTaskExists(ctx, tx, existing.String)
	if err != nil {
		return nil, fmt.Errorf("subscription: verify existing task %q for release %q: %w", existing.String, releaseKey, err)
	}
	if exists {
		return existing.String, nil
	}
	return nil, nil
}

func marshalPerformerNames(names []string) string {
	if len(names) == 0 {
		return "[]"
	}
	data, err := json.Marshal(names)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func deleteOrphanReleaseEntities(ctx context.Context, tx *sqlx.Tx) error {
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
	return value.UTC().Format("2006-01-02T15:04:05.000000000Z")
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

func nullableStringParam(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

// NewLegacyReader is used only by explicit migrations, with a mode=ro,
// query_only connection. It does not initialize or alter the source schema.
func NewLegacyReader(db *sqlx.DB) *SQLiteStore { return &SQLiteStore{db: db} }
