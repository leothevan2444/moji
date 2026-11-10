package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
)

func (s *SQLiteStore) GetCheckState(ctx context.Context, id string) (*PerformerState, error) {
	var row sqlitePerformerStateRow
	err := s.db.GetContext(ctx, &row, "SELECT performer_id,last_checked_at,last_error,activity_json,next_check_at,check_failures FROM subscription_performer_state WHERE performer_id=?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toState()
}
func (s *SQLiteStore) SaveActivity(ctx context.Context, id string, activity ActivityState) error {
	raw, err := marshalActivityObservation(activity)
	if err != nil {
		return err
	}
	now := formatTimestamp(time.Now().UTC())
	return s.withWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO subscription_performer_state(performer_id,activity_json,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(performer_id) DO UPDATE SET activity_json=excluded.activity_json,updated_at=excluded.updated_at WHERE activity_json IS NOT excluded.activity_json`, id, string(raw), now, now)
		return err
	})
}
func (s *SQLiteStore) SaveCheckOutcome(ctx context.Context, state *PerformerState) error {
	now := formatTimestamp(time.Now().UTC())
	return s.withWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO subscription_performer_state(performer_id,last_checked_at,last_error,next_check_at,check_failures,created_at,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(performer_id) DO UPDATE SET last_checked_at=excluded.last_checked_at,last_error=excluded.last_error,next_check_at=excluded.next_check_at,check_failures=excluded.check_failures,updated_at=excluded.updated_at`, state.PerformerID, formatOptionalTimestamp(state.LastCheckedAt), nullableStringParam(state.LastError), formatCheckDeadline(state.Activity.NextCheckAt), state.Activity.Failures, now, now)
		return err
	})
}
func (s *SQLiteStore) InitializeCheckSchedule(ctx context.Context, id string, next time.Time) error {
	now := formatTimestamp(time.Now().UTC())
	return s.withWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO subscription_performer_state(performer_id,next_check_at,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(performer_id) DO UPDATE SET next_check_at=excluded.next_check_at,updated_at=excluded.updated_at WHERE subscription_performer_state.next_check_at IS NULL`, id, formatCheckDeadline(&next), now, now)
		return err
	})
}

type sqliteCheckSchedule struct {
	PerformerID   string         `db:"performer_id"`
	LastCheckedAt sql.NullString `db:"last_checked_at"`
	NextCheckAt   sql.NullString `db:"next_check_at"`
}

func decodeCheckSchedules(rows []sqliteCheckSchedule) ([]CheckSchedule, error) {
	out := make([]CheckSchedule, 0, len(rows))
	for _, row := range rows {
		last, err := parseOptionalTimestamp(row.LastCheckedAt)
		if err != nil {
			return nil, err
		}
		next, err := parseOptionalTimestamp(row.NextCheckAt)
		if err != nil {
			return nil, err
		}
		out = append(out, CheckSchedule{PerformerID: row.PerformerID, LastCheckedAt: last, NextCheckAt: next})
	}
	return out, nil
}
func (s *SQLiteStore) ListCheckSchedules(ctx context.Context) ([]CheckSchedule, error) {
	var rows []sqliteCheckSchedule
	if err := s.db.SelectContext(ctx, &rows, "SELECT performer_id,last_checked_at,next_check_at FROM subscription_performer_state ORDER BY performer_id"); err != nil {
		return nil, err
	}
	return decodeCheckSchedules(rows)
}
func (s *SQLiteStore) ListDueCheckSchedules(ctx context.Context, now time.Time, after *CheckSchedule, limit int) ([]CheckSchedule, error) {
	limit = max(1, min(limit, 128))
	query := "SELECT performer_id,last_checked_at,next_check_at FROM subscription_performer_state WHERE next_check_at IS NOT NULL AND next_check_at<=?"
	args := []any{formatCheckDeadline(&now)}
	if after != nil {
		query += " AND (next_check_at,performer_id)>(?,?)"
		args = append(args, formatCheckDeadline(after.NextCheckAt), after.PerformerID)
	}
	query += " ORDER BY next_check_at,performer_id LIMIT ?"
	args = append(args, limit)
	var rows []sqliteCheckSchedule
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return decodeCheckSchedules(rows)
}

func checkSchedules(states map[string]*PerformerState) []CheckSchedule {
	out := make([]CheckSchedule, 0, len(states))
	for id, state := range states {
		out = append(out, CheckSchedule{PerformerID: id, LastCheckedAt: state.LastCheckedAt, NextCheckAt: state.Activity.NextCheckAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PerformerID < out[j].PerformerID })
	return out
}
func dueCheckSchedules(states map[string]*PerformerState, now time.Time, after *CheckSchedule, limit int) []CheckSchedule {
	out := []CheckSchedule{}
	for _, row := range checkSchedules(states) {
		if row.NextCheckAt == nil || row.NextCheckAt.After(now) {
			continue
		}
		if after != nil && (row.NextCheckAt.Before(*after.NextCheckAt) || (row.NextCheckAt.Equal(*after.NextCheckAt) && row.PerformerID <= after.PerformerID)) {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NextCheckAt.Equal(*out[j].NextCheckAt) {
			return out[i].PerformerID < out[j].PerformerID
		}
		return out[i].NextCheckAt.Before(*out[j].NextCheckAt)
	})
	return out[:min(len(out), max(1, min(limit, 128)))]
}
func saveActivityState(states map[string]*PerformerState, id string, activity ActivityState) {
	state := states[id]
	if state == nil {
		state = &PerformerState{PerformerID: id}
		states[id] = state
	}
	next, failures := state.Activity.NextCheckAt, state.Activity.Failures
	state.Activity = activity
	state.Activity.Days = append([]ActivityDay(nil), activity.Days...)
	state.Activity.NextCheckAt, state.Activity.Failures = next, failures
}
func saveOutcomeState(states map[string]*PerformerState, outcome *PerformerState) {
	state := states[outcome.PerformerID]
	if state == nil {
		state = &PerformerState{PerformerID: outcome.PerformerID}
		states[outcome.PerformerID] = state
	}
	state.LastCheckedAt, state.LastError = outcome.LastCheckedAt, outcome.LastError
	state.Activity.NextCheckAt, state.Activity.Failures = outcome.Activity.NextCheckAt, outcome.Activity.Failures
}
func initializeScheduleState(states map[string]*PerformerState, id string, next time.Time) {
	state := states[id]
	if state == nil {
		state = &PerformerState{PerformerID: id}
		states[id] = state
	}
	if state.Activity.NextCheckAt == nil {
		state.Activity.NextCheckAt = &next
	}
}

func marshalActivityObservation(activity ActivityState) ([]byte, error) {
	activity.NextCheckAt = nil
	activity.Failures = 0
	return json.Marshal(activity)
}

// Deadlines use fixed UTC precision so indexed text order equals time order.
func formatCheckDeadline(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format("2006-01-02T15:04:05.000000000Z")
}
func cloneCheckState(state *PerformerState) *PerformerState {
	if state == nil {
		return nil
	}
	out := *state
	out.PendingReleases = nil
	out.ProcessedReleases = nil
	out.Activity.Days = append([]ActivityDay(nil), state.Activity.Days...)
	return &out
}

// GetCheckStates projects only the requested page, without release joins.
func (s *SQLiteStore) GetCheckStates(ctx context.Context, ids []string) ([]*PerformerState, error) {
	out := make([]*PerformerState, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	query, args, err := sqlx.In("SELECT performer_id,last_checked_at,last_error,activity_json,next_check_at,check_failures FROM subscription_performer_state WHERE performer_id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	var rows []sqlitePerformerStateRow
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		state, err := row.toState()
		if err != nil {
			return nil, err
		}
		out = append(out, state)
	}
	return out, nil
}
