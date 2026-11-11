package taskruntime

import (
	"context"
	"fmt"
	"time"
)

// HomeStats is the current task-count projection shared by Home and widgets.
type HomeStats struct {
	Total, Attention, Active, Waiting, CompletedLast24Hours int
}

type homeStatsStore interface {
	HomeStats(context.Context, time.Time) (HomeStats, error)
}

func (s *Service) HomeStats(ctx context.Context, now time.Time) (HomeStats, error) {
	if store, ok := s.store.(homeStatsStore); ok {
		return store.HomeStats(ctx, now)
	}
	return HomeStats{}, fmt.Errorf("task store does not support home statistics")
}

type homeTaskCategory int

const (
	homeCompleted homeTaskCategory = iota
	homeAttention
	homeWaiting
	homeActive
)

func classifyHomeTask(task *Task) homeTaskCategory {
	switch {
	case task.Stage == TaskStageCompleted:
		return homeCompleted
	case task.StageStatus == TaskStageStatusBlocked:
		return homeAttention
	case task.Stage == TaskStageSourcing && task.StageStatus == TaskStageStatusPending && task.SourcingNextCheckAt != nil:
		return homeWaiting
	default:
		return homeActive
	}
}

func summarizeHomeStats(tasks []*Task, now time.Time) HomeStats {
	var result HomeStats
	for _, task := range tasks {
		if task == nil {
			continue
		}
		result.Total++
		switch classifyHomeTask(task) {
		case homeCompleted:
			if at := task.IngestCompletedAt; at != nil && !at.Before(now.Add(-24*time.Hour)) && !at.After(now) {
				result.CompletedLast24Hours++
			}
		case homeAttention:
			result.Attention++
		case homeWaiting:
			result.Waiting++
		case homeActive:
			result.Active++
		}
	}
	return result
}

type homeStatsReader interface {
	GetContext(context.Context, interface{}, string, ...interface{}) error
}

// from is supplied only by the store implementations, never by an API caller.
func queryHomeStats(ctx context.Context, reader homeStatsReader, from string, now time.Time) (HomeStats, error) {
	var counts struct {
		Total     int `db:"total"`
		Attention int `db:"attention"`
		Waiting   int `db:"waiting"`
		Completed int `db:"completed"`
		Recent    int `db:"recent"`
	}
	err := reader.GetContext(ctx, &counts, `SELECT COUNT(*) total,
		COUNT(CASE WHEN stage != 'COMPLETED' AND stage_status = 'BLOCKED' THEN 1 END) attention,
		COUNT(CASE WHEN stage = 'SOURCING' AND stage_status = 'PENDING' AND sourcing_next_check_at IS NOT NULL THEN 1 END) waiting,
		COUNT(CASE WHEN stage = 'COMPLETED' THEN 1 END) completed,
		COUNT(CASE WHEN stage = 'COMPLETED' AND julianday(ingest_completed_at) BETWEEN julianday(?) AND julianday(?) THEN 1 END) recent FROM `+from,
		now.Add(-24*time.Hour).UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return HomeStats{}, err
	}
	return HomeStats{Total: counts.Total, Attention: counts.Attention, Waiting: counts.Waiting, Active: counts.Total - counts.Attention - counts.Waiting - counts.Completed, CompletedLast24Hours: counts.Recent}, nil
}

func (s *SQLiteTaskStore) HomeStats(ctx context.Context, now time.Time) (HomeStats, error) {
	return queryHomeStats(ctx, s.db, "tasks", now)
}

func (s *RelationalTaskStore) HomeStats(ctx context.Context, now time.Time) (HomeStats, error) {
	return queryHomeStats(ctx, s.core.Reader, "tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id", now)
}

func (s *EventingTaskStore) HomeStats(ctx context.Context, now time.Time) (HomeStats, error) {
	if store, ok := s.store.(homeStatsStore); ok {
		return store.HomeStats(ctx, now)
	}
	return HomeStats{}, fmt.Errorf("task store does not support home statistics")
}

func (s *MemoryTaskStore) HomeStats(ctx context.Context, now time.Time) (HomeStats, error) {
	if err := ctx.Err(); err != nil {
		return HomeStats{}, err
	}
	tasks, err := s.List(ctx)
	if err != nil {
		return HomeStats{}, err
	}
	return summarizeHomeStats(tasks, now), nil
}
