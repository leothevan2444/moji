package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func (s *SQLiteStore) putTypedReviews(ctx context.Context, records []ReleaseReview) error {
	return s.withWrite(ctx, func(tx *sqlx.Tx) error {
		for _, record := range records {
			raw, err := json.Marshal(record)
			if err != nil {
				return err
			}
			taskID, err := sanitizeReleaseTaskID(ctx, tx, record.Key, record.TaskID)
			if err != nil {
				return err
			}
			// A review must never accidentally inherit an unrelated release association.
			if record.TaskID == "" {
				taskID = nil
			}
			var previous sql.NullString
			var previousVersion int
			err = tx.QueryRowContext(ctx, "SELECT record_json,payload_version FROM subscription_release_reviews WHERE release_key=?", record.Key).Scan(&previous, &previousVersion)
			if err == nil && previousVersion != 1 {
				return errors.New("subscription: unsupported review payload version")
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if previous.Valid && previous.String == string(raw) {
				continue
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO subscription_release_reviews(release_key,status,code,title,source,release_date,task_id,decision,reviewed_at,seen_at,record_json)
   VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(release_key) DO UPDATE SET status=excluded.status,code=excluded.code,title=excluded.title,source=excluded.source,release_date=excluded.release_date,task_id=excluded.task_id,decision=excluded.decision,reviewed_at=excluded.reviewed_at,seen_at=excluded.seen_at,record_json=excluded.record_json`,
				record.Key, record.Status, record.Code, record.Title, record.Source, nullableStringParam(record.Date), taskID, record.Decision, formatOptionalTimestamp(record.ReviewedAt), formatTimestamp(record.SeenAt), string(raw))
			if err != nil {
				return err
			}
			// The performer relation survives unsubscribing: no FK to subscription state.
			var existing []string
			if err := tx.SelectContext(ctx, &existing, "SELECT performer_id FROM subscription_review_performers WHERE release_key=?", record.Key); err != nil {
				return err
			}
			retained := map[string]bool{}
			for position, performer := range record.Performers {
				if retained[performer.ID] {
					return fmt.Errorf("subscription: duplicate review performer %q", performer.ID)
				}
				retained[performer.ID] = true
				if _, err := tx.ExecContext(ctx, `INSERT INTO subscription_review_performers(release_key,performer_id,name,position) VALUES(?,?,?,?) ON CONFLICT(release_key,performer_id) DO UPDATE SET name=excluded.name,position=excluded.position WHERE name IS NOT excluded.name OR position IS NOT excluded.position`, record.Key, performer.ID, performer.Name, position); err != nil {
					return err
				}
			}
			for _, id := range existing {
				if !retained[id] {
					if _, err := tx.ExecContext(ctx, "DELETE FROM subscription_review_performers WHERE release_key=? AND performer_id=?", record.Key, id); err != nil {
						return err
					}
				}
			}
			// Decisions and their durable audit entry commit together.
			changedDecision := true
			if previous.Valid {
				var old ReleaseReview
				if err := json.Unmarshal([]byte(previous.String), &old); err != nil {
					return err
				}
				changedDecision = old.Status != record.Status || old.Decision != record.Decision || !equalReviewTime(old.ReviewedAt, record.ReviewedAt)
			}
			if changedDecision {
				_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(subject_id,event_type,level,message,payload_json,created_at) VALUES(?,'release-review','info','release review decision',?,?)`, record.Key, string(raw), formatTimestamp(time.Now().UTC()))
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func equalReviewTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
func (s *SQLiteStore) listTypedReviews(ctx context.Context) ([]ReleaseReview, error) {
	type row struct {
		Raw     string         `db:"record_json"`
		Version int            `db:"payload_version"`
		TaskID  sql.NullString `db:"task_id"`
	}
	var rows []row
	if err := s.db.SelectContext(ctx, &rows, "SELECT record_json,task_id,payload_version FROM subscription_release_reviews ORDER BY release_key"); err != nil {
		return nil, err
	}
	out := make([]ReleaseReview, 0, len(rows))
	for _, row := range rows {
		if row.Version != 1 {
			return nil, errors.New("subscription: unsupported review payload version")
		}
		var record ReleaseReview
		if err := json.Unmarshal([]byte(row.Raw), &record); err != nil {
			return nil, err
		}
		record.TaskID = nullableStringValue(row.TaskID)
		out = append(out, record)
	}
	return out, nil
}
func (s *SQLiteStore) saveTypedCheckRun(ctx context.Context, run CheckRun) error {
	return s.withWrite(ctx, func(tx *sqlx.Tx) error {
		var version int
		err := tx.GetContext(ctx, &version, "SELECT payload_version FROM subscription_check_runs WHERE id=?", run.ID)
		if err == nil && version != 1 {
			return errors.New("subscription: unsupported check payload version")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscription_check_runs(id,trigger,scope,status,started_at,finished_at,error) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,finished_at=excluded.finished_at,error=excluded.error WHERE status IS NOT excluded.status OR finished_at IS NOT excluded.finished_at OR error IS NOT excluded.error`, run.ID, run.Trigger, run.Scope, run.Status, formatTimestamp(run.StartedAt), formatOptionalTimestamp(run.FinishedAt), run.Error); err != nil {
			return err
		}
		// Check item order is fixed at admission; changes indicate corrupt/stale input.
		var existing []string
		if err := tx.SelectContext(ctx, &existing, "SELECT performer_id FROM subscription_check_items WHERE run_id=? ORDER BY position", run.ID); err != nil {
			return err
		}
		if len(existing) > 0 {
			if len(existing) != len(run.Items) {
				return errors.New("subscription: check membership changed after admission")
			}
			for i, item := range run.Items {
				if existing[i] != item.PerformerID {
					return errors.New("subscription: check membership changed after admission")
				}
			}
		}
		for position, item := range run.Items {
			_, err := tx.ExecContext(ctx, `INSERT INTO subscription_check_items(run_id,performer_id,name,position,status,started_at,finished_at,error) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(run_id,performer_id) DO UPDATE SET name=excluded.name,status=excluded.status,started_at=excluded.started_at,finished_at=excluded.finished_at,error=excluded.error WHERE name IS NOT excluded.name OR status IS NOT excluded.status OR started_at IS NOT excluded.started_at OR finished_at IS NOT excluded.finished_at OR error IS NOT excluded.error`, run.ID, item.PerformerID, item.Name, position, item.Status, formatOptionalTimestamp(item.StartedAt), formatOptionalTimestamp(item.FinishedAt), item.Error)
			if err != nil {
				return err
			}
		}
		// Retain admitted/incomplete work and the latest scheduled run.
		_, err = tx.ExecContext(ctx, `DELETE FROM subscription_check_runs WHERE status!='RUNNING' AND id NOT IN(SELECT id FROM subscription_check_runs ORDER BY started_at DESC LIMIT ?) AND id NOT IN(SELECT id FROM subscription_check_runs WHERE trigger='SCHEDULED' ORDER BY started_at DESC LIMIT 1)`, checkHistoryLimit)
		return err
	})
}
func (s *SQLiteStore) loadTypedCheckRuns(ctx context.Context) ([]CheckRun, error) {
	type header struct {
		Version                           int `db:"payload_version"`
		ID, Trigger, Scope, Status, Error string
		StartedAt                         string         `db:"started_at"`
		FinishedAt                        sql.NullString `db:"finished_at"`
	}
	var headers []header
	if err := s.db.SelectContext(ctx, &headers, `SELECT id,trigger,scope,status,error,started_at,finished_at,payload_version FROM subscription_check_runs ORDER BY started_at DESC,id`); err != nil {
		return nil, err
	}
	type itemRow struct {
		RunID               string `db:"run_id"`
		PerformerID         string `db:"performer_id"`
		Name, Status, Error string
		StartedAt           sql.NullString `db:"started_at"`
		FinishedAt          sql.NullString `db:"finished_at"`
	}
	var items []itemRow
	if err := s.db.SelectContext(ctx, &items, `SELECT run_id,performer_id,name,status,error,started_at,finished_at FROM subscription_check_items ORDER BY run_id,position`); err != nil {
		return nil, err
	}
	byRun := map[string][]CheckItem{}
	for _, row := range items {
		started, err := parseOptionalTimestamp(row.StartedAt)
		if err != nil {
			return nil, err
		}
		finished, err := parseOptionalTimestamp(row.FinishedAt)
		if err != nil {
			return nil, err
		}
		byRun[row.RunID] = append(byRun[row.RunID], CheckItem{PerformerID: row.PerformerID, Name: row.Name, Status: row.Status, Error: row.Error, StartedAt: started, FinishedAt: finished})
	}
	out := make([]CheckRun, 0, len(headers))
	for _, row := range headers {
		if row.Version != 1 {
			return nil, errors.New("subscription: unsupported check payload version")
		}
		started, err := parseTimestamp(row.StartedAt)
		if err != nil {
			return nil, err
		}
		finished, err := parseOptionalTimestamp(row.FinishedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, CheckRun{ID: row.ID, Trigger: row.Trigger, Scope: row.Scope, Status: row.Status, Error: row.Error, StartedAt: started, FinishedAt: finished, Items: byRun[row.ID]})
	}
	return out, nil
}
