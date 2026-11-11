package taskruntime

import (
	"context"
	"database/sql"
	"sort"
	"time"
)

// HomeOverview is a bounded operational snapshot, independent of task history.
type HomeOverview struct {
	HomeStats
	AttentionTasks, ActiveTasks, RecentTasks []*Task
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
	result := HomeOverview{HomeStats: summarizeHomeStats(tasks, now)}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		switch classifyHomeTask(task) {
		case homeCompleted:
			if task.IngestCompletedAt != nil {
				result.RecentTasks = append(result.RecentTasks, task)
			}
		case homeAttention:
			result.AttentionTasks = append(result.AttentionTasks, task)
		case homeWaiting:
		default:
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
	counts, err := queryHomeStats(ctx, tx, "tasks", now)
	if err != nil {
		return HomeOverview{}, err
	}
	result := HomeOverview{HomeStats: counts}
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
	counts, err := queryHomeStats(ctx, tx, "tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id", now)
	if err != nil {
		return HomeOverview{}, err
	}
	result := HomeOverview{HomeStats: counts}
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
