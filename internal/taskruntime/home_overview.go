package taskruntime

import (
	"context"
	"database/sql"
	"sort"
	"time"
)

// HomeOverview is a bounded operational snapshot, independent of task history.
type HomeOverview struct {
	Total, Attention, Active, Waiting, CompletedLast24Hours int
	AttentionTasks, ActiveTasks, RecentTasks                []*Task
}

type homeOverviewStore interface {
	HomeOverview(context.Context, time.Time) (HomeOverview, error)
}

func (s *Service) HomeOverview(ctx context.Context, now time.Time) (HomeOverview, error) {
	if store, ok := s.store.(homeOverviewStore); ok {
		return store.HomeOverview(ctx, now)
	}
	// JSON and in-memory stores are development alternatives to SQLite.
	tasks, err := s.store.List(ctx)
	if err != nil {
		return HomeOverview{}, err
	}
	return summarizeHomeTasks(tasks, now), nil
}

func summarizeHomeTasks(tasks []*Task, now time.Time) HomeOverview {
	result := HomeOverview{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		result.Total++
		switch {
		case task.Stage == TaskStageCompleted:
			if task.IngestCompletedAt != nil {
				result.RecentTasks = append(result.RecentTasks, task)
				if !task.IngestCompletedAt.Before(now.Add(-24*time.Hour)) && !task.IngestCompletedAt.After(now) {
					result.CompletedLast24Hours++
				}
			}
		case task.StageStatus == TaskStageStatusBlocked:
			result.Attention++
			result.AttentionTasks = append(result.AttentionTasks, task)
		case task.Stage == TaskStageSourcing && task.StageStatus == TaskStageStatusPending && task.SourcingNextCheckAt != nil:
			result.Waiting++
		default:
			result.Active++
			result.ActiveTasks = append(result.ActiveTasks, task)
		}
	}
	lessCreated := func(a, b *Task) bool {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return a.ID < b.ID
		}
		return a.CreatedAt.Before(b.CreatedAt)
	}
	sort.Slice(result.AttentionTasks, func(i, j int) bool { return lessCreated(result.AttentionTasks[i], result.AttentionTasks[j]) })
	sort.Slice(result.ActiveTasks, func(i, j int) bool { return lessCreated(result.ActiveTasks[i], result.ActiveTasks[j]) })
	sort.Slice(result.RecentTasks, func(i, j int) bool {
		a, b := result.RecentTasks[i], result.RecentTasks[j]
		if a.IngestCompletedAt.Equal(*b.IngestCompletedAt) {
			return a.ID < b.ID
		}
		return a.IngestCompletedAt.After(*b.IngestCompletedAt)
	})
	result.AttentionTasks = result.AttentionTasks[:min(3, len(result.AttentionTasks))]
	result.ActiveTasks = result.ActiveTasks[:min(5, len(result.ActiveTasks))]
	result.RecentTasks = result.RecentTasks[:min(3, len(result.RecentTasks))]
	return result
}

func (s *SQLiteTaskStore) HomeOverview(ctx context.Context, now time.Time) (HomeOverview, error) {
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return HomeOverview{}, err
	}
	defer tx.Rollback()
	var counts struct {
		Total     int `db:"total"`
		Attention int `db:"attention"`
		Waiting   int `db:"waiting"`
		Completed int `db:"completed"`
		Recent    int `db:"recent"`
	}
	err = tx.GetContext(ctx, &counts, `SELECT COUNT(*) total,
		COUNT(CASE WHEN stage != 'COMPLETED' AND stage_status = 'BLOCKED' THEN 1 END) attention,
		COUNT(CASE WHEN stage = 'SOURCING' AND stage_status = 'PENDING' AND sourcing_next_check_at IS NOT NULL THEN 1 END) waiting,
		COUNT(CASE WHEN stage = 'COMPLETED' THEN 1 END) completed,
		COUNT(CASE WHEN stage = 'COMPLETED' AND julianday(ingest_completed_at) BETWEEN julianday(?) AND julianday(?) THEN 1 END) recent FROM tasks`, now.Add(-24*time.Hour).UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return HomeOverview{}, err
	}
	result := HomeOverview{Total: counts.Total, Attention: counts.Attention, Waiting: counts.Waiting, Active: counts.Total - counts.Attention - counts.Waiting - counts.Completed, CompletedLast24Hours: counts.Recent}
	queries := []struct {
		suffix string
		target *[]*Task
	}{
		{` WHERE stage != 'COMPLETED' AND stage_status = 'BLOCKED' ORDER BY created_at ASC, id ASC LIMIT 3`, &result.AttentionTasks},
		{` WHERE stage != 'COMPLETED' AND stage_status != 'BLOCKED' AND NOT (stage = 'SOURCING' AND stage_status = 'PENDING' AND sourcing_next_check_at IS NOT NULL) ORDER BY created_at ASC, id ASC LIMIT 5`, &result.ActiveTasks},
		{` WHERE stage = 'COMPLETED' AND ingest_completed_at IS NOT NULL ORDER BY julianday(ingest_completed_at) DESC, id ASC LIMIT 3`, &result.RecentTasks},
	}
	for _, query := range queries {
		var rows []sqliteTaskRow
		if err := tx.SelectContext(ctx, &rows, taskSelectSQL+query.suffix); err != nil {
			return HomeOverview{}, err
		}
		for _, row := range rows {
			task, err := row.toTask()
			if err != nil {
				return HomeOverview{}, err
			}
			*query.target = append(*query.target, task)
		}
	}
	if err := tx.Commit(); err != nil {
		return HomeOverview{}, err
	}
	return result, nil
}

func (s *RelationalTaskStore) HomeOverview(ctx context.Context, now time.Time) (HomeOverview, error) {
	tx, err := s.core.Reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return HomeOverview{}, err
	}
	defer tx.Rollback()
	var counts struct {
		Total     int `db:"total"`
		Attention int `db:"attention"`
		Waiting   int `db:"waiting"`
		Completed int `db:"completed"`
		Recent    int `db:"recent"`
	}
	err = tx.GetContext(ctx, &counts, `SELECT COUNT(*) total,
		COUNT(CASE WHEN stage != 'COMPLETED' AND stage_status = 'BLOCKED' THEN 1 END) attention,
		COUNT(CASE WHEN stage = 'SOURCING' AND stage_status = 'PENDING' AND sourcing_next_check_at IS NOT NULL THEN 1 END) waiting,
		COUNT(CASE WHEN stage = 'COMPLETED' THEN 1 END) completed,
		COUNT(CASE WHEN stage = 'COMPLETED' AND julianday(ingest_completed_at) BETWEEN julianday(?) AND julianday(?) THEN 1 END) recent FROM tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id`, now.Add(-24*time.Hour).UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return HomeOverview{}, err
	}
	result := HomeOverview{Total: counts.Total, Attention: counts.Attention, Waiting: counts.Waiting, Active: counts.Total - counts.Attention - counts.Waiting - counts.Completed, CompletedLast24Hours: counts.Recent}
	queries := []struct {
		suffix string
		target *[]*Task
	}{
		{` WHERE stage != 'COMPLETED' AND stage_status = 'BLOCKED' ORDER BY created_at ASC, id ASC LIMIT 3`, &result.AttentionTasks},
		{` WHERE stage != 'COMPLETED' AND stage_status != 'BLOCKED' AND NOT (stage = 'SOURCING' AND stage_status = 'PENDING' AND sourcing_next_check_at IS NOT NULL) ORDER BY created_at ASC, id ASC LIMIT 5`, &result.ActiveTasks},
		{` WHERE stage = 'COMPLETED' AND ingest_completed_at IS NOT NULL ORDER BY julianday(ingest_completed_at) DESC, id ASC LIMIT 3`, &result.RecentTasks},
	}
	for _, query := range queries {
		var rows []sqliteTaskRow
		if err := tx.SelectContext(ctx, &rows, relationalTaskSelectSQL+query.suffix); err != nil {
			return HomeOverview{}, err
		}
		for _, row := range rows {
			task, err := row.toTask()
			if err != nil {
				return HomeOverview{}, err
			}
			*query.target = append(*query.target, s.overlay(task))
		}
	}
	if err := tx.Commit(); err != nil {
		return HomeOverview{}, err
	}
	return result, nil
}

func (s *EventingTaskStore) HomeOverview(ctx context.Context, now time.Time) (HomeOverview, error) {
	if provider, ok := s.store.(homeOverviewStore); ok {
		return provider.HomeOverview(ctx, now)
	}
	tasks, err := s.store.List(ctx)
	if err != nil {
		return HomeOverview{}, err
	}
	return summarizeHomeTasks(tasks, now), nil
}
