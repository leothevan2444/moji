package indexer

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type StatisticsReader interface {
	Statistics(context.Context, time.Time, int) (ExecutionStatistics, error)
	RecentExecutions(context.Context, []string) (map[string]RecentExecutions, error)
}
type RecentExecutions struct {
	Count, Failures int
	LatestFailed    bool
}
type ExecutionStatistics struct {
	WindowStartedAt, WindowEndedAt     time.Time
	Searches, Succeeded, TimedSearches int
	TotalElapsedMs                     int64
	Records                            int
}

func (s *DiagnosticStore) Statistics(ctx context.Context, now time.Time, hours int) (ExecutionStatistics, error) {
	if hours != 24 && hours != 72 && hours != 336 {
		return ExecutionStatistics{}, ErrInvalidQuery
	}
	result := ExecutionStatistics{WindowStartedAt: now.UTC().Add(-time.Duration(hours) * time.Hour), WindowEndedAt: now.UTC()}
	tx, err := s.db.Reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err = tx.GetContext(ctx, &result.Records, `SELECT count(*) FROM indexer_executions`); err != nil {
		return result, err
	}
	err = tx.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(outcome='SUCCEEDED'),0),coalesce(sum(json_extract(message,'$.ElapsedKnown')=1),0),coalesce(sum(CASE WHEN json_extract(message,'$.ElapsedKnown')=1 THEN json_extract(message,'$.Report.Elapsed') ELSE 0 END),0)/1000000 FROM indexer_executions WHERE eligible=1 AND operation='SEARCH' AND finished_at>=? AND finished_at<=?`, instant(result.WindowStartedAt), instant(result.WindowEndedAt)).Scan(&result.Searches, &result.Succeeded, &result.TimedSearches, &result.TotalElapsedMs)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s *DiagnosticStore) RecentExecutions(ctx context.Context, ids []string) (map[string]RecentExecutions, error) {
	out := map[string]RecentExecutions{}
	if len(ids) > 2048 {
		return nil, ErrResponseBudget
	}
	if len(ids) == 0 {
		return out, nil
	}
	for _, id := range ids {
		if id == "" || len(id) > 256 {
			return nil, ErrInvalidQuery
		}
	}
	query, args, err := sqlx.In(`SELECT instance_id,count(*),sum(outcome='FAILED'),max(CASE WHEN rank=1 THEN outcome='FAILED' ELSE 0 END) FROM (SELECT instance_id,outcome,row_number() OVER(PARTITION BY instance_id ORDER BY started_at DESC,id DESC) AS rank FROM indexer_executions WHERE eligible=1 AND instance_id IN (?)) WHERE rank<=10 GROUP BY instance_id`, ids)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var state RecentExecutions
		if err = rows.Scan(&id, &state.Count, &state.Failures, &state.LatestFailed); err != nil {
			return nil, err
		}
		out[id] = state
	}
	return out, rows.Err()
}

// Prune retains every record in the statistics window, plus each configured
// site's ten newest effective executions even when those executions are older.
// No global row budget is applied. Callers supply a locally confirmed site list.
func (s *DiagnosticStore) Prune(ctx context.Context, configured []string, preserveJackett bool, now time.Time) error {
	protected := `SELECT id FROM (SELECT id,row_number() OVER(PARTITION BY instance_id ORDER BY started_at DESC,id DESC) AS rank FROM indexer_executions WHERE eligible=1 AND (? AND instance_id LIKE 'jackett/%')) WHERE rank<=10`
	args := []any{instant(now.UTC().Add(-14 * 24 * time.Hour)), preserveJackett}
	if len(configured) > 0 {
		query, siteArgs, err := sqlx.In(`SELECT id FROM (SELECT id,row_number() OVER(PARTITION BY instance_id ORDER BY started_at DESC,id DESC) AS rank FROM indexer_executions WHERE eligible=1 AND ((? AND instance_id LIKE 'jackett/%') OR instance_id IN (?))) WHERE rank<=10`, preserveJackett, configured)
		if err != nil {
			return err
		}
		protected = query
		args = append([]any{instant(now.UTC().Add(-14 * 24 * time.Hour))}, siteArgs...)
	}
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM indexer_executions WHERE finished_at<? AND id NOT IN (`+protected+`)`, args...)
		return err
	})
}
