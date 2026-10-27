package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"sort"
	"strings"
)

const workBatchSize = 128

var errWorkBatchFull = errors.New("taskruntime: work queue full")

type taskWorkPageProvider interface {
	TaskWorkPage(context.Context, string, string, int) ([]*Task, error)
}

type taskWorkPager interface {
	TaskWorkIDs(context.Context, string, string, int) ([]string, error)
}

func walkTaskWork(ctx context.Context, store TaskStore, kind string, visit func(*Task) error) error {
	if pager, ok := store.(taskWorkPageProvider); ok {
		after := ""
		for {
			tasks, err := pager.TaskWorkPage(ctx, kind, after, workBatchSize)
			if err != nil {
				return err
			}
			for _, task := range tasks {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := visit(task); err != nil {
					return err
				}
				after = task.ID
			}
			if len(tasks) < workBatchSize {
				return nil
			}
		}
	}
	if pager, ok := store.(taskWorkPager); ok {
		after := ""
		for {
			ids, err := pager.TaskWorkIDs(ctx, kind, after, workBatchSize)
			if err != nil {
				return err
			}
			for _, id := range ids {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := visit(&Task{ID: id}); err != nil {
					return err
				}
				after = id
			}
			if len(ids) < workBatchSize {
				return nil
			}
		}
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := visit(task); err != nil {
			return err
		}
	}
	return nil
}
func (s *RelationalTaskStore) TaskWorkIDs(ctx context.Context, kind, after string, limit int) ([]string, error) {
	if limit < 1 || limit > workBatchSize {
		return nil, fmt.Errorf("taskruntime: invalid work batch limit")
	}
	predicate, err := workPredicate(kind)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	err = s.core.Reader.SelectContext(ctx, &ids, `SELECT t.id FROM tasks t LEFT JOIN task_downloads d ON d.task_id=t.id LEFT JOIN task_checkpoints k ON k.task_id=t.id WHERE t.id>? AND (`+predicate+`) ORDER BY t.id LIMIT ?`, after, limit)
	return ids, err
}
func (s *EventingTaskStore) TaskWorkIDs(ctx context.Context, kind, after string, limit int) ([]string, error) {
	if pager, ok := s.store.(taskWorkPager); ok {
		return pager.TaskWorkIDs(ctx, kind, after, limit)
	}
	// Development stores retain their existing ordering/behavior through a stable
	// keyset adapter. Production never enters this path.
	tasks, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task != nil && task.ID > after {
			ids = append(ids, task.ID)
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

// SyncActiveProgress is the bounded worker path. Explicit API requests retain
// SyncProgress's existing complete response contract.
func (s *Service) SyncActiveProgress(ctx context.Context) error {
	torrents, err := s.qbt.GetTorrentList(ctx, nil)
	if err != nil {
		return err
	}
	byHash := map[string]qbittorrent.Torrent{}
	for _, torrent := range torrents {
		byHash[strings.ToLower(torrent.Hash)] = torrent
	}
	return walkTaskWork(ctx, s.store, "MONITOR", func(snapshot *Task) error {
		if snapshot.Stage != "" && snapshot.Stage != TaskStageDownloading {
			next := applyTorrentObservation(snapshot, byHash, s.now().UTC())
			if sameTaskProgress(snapshot, next) && snapshot.ContentPath == next.ContentPath {
				return nil
			}
		}
		_, err := s.syncTaskProgress(ctx, snapshot, torrents)
		return err
	})
}

func workPredicate(kind string) (string, error) {
	predicate := ""
	switch kind {
	case "SOURCING":
		predicate = "t.stage='SOURCING' AND t.stage_status<>'DONE'"
	case "MONITOR":
		predicate = "t.stage<>'SOURCING' AND (d.torrent_hash<>'' OR d.torrent_identity_hash IS NOT NULL OR t.stage='DOWNLOADING') AND k.restart_state_json IS NULL"
	case "INGEST":
		predicate = "(t.stage='PENDING_INGEST' AND t.stage_status='PENDING') OR (t.stage IN ('TRANSFERRING','SCANNING') AND t.stage_status IN ('RUNNING','PENDING'))"
	default:
		return "", fmt.Errorf("taskruntime: unknown work kind %q", kind)
	}
	return predicate, nil
}

// TaskWorkPage loads lifecycle/checkpoint headers only. Consumers must Find
// before any mutation; attachments and file plans are never queue payloads.
func (s *RelationalTaskStore) TaskWorkPage(ctx context.Context, kind, after string, limit int) ([]*Task, error) {
	if limit < 1 || limit > workBatchSize {
		return nil, errors.New("taskruntime: invalid work batch limit")
	}
	predicate, err := workPredicate(kind)
	if err != nil {
		return nil, err
	}
	var rows []sqliteTaskRow
	err = s.core.Reader.SelectContext(ctx, &rows, `SELECT COALESCE(k.payload_version,1) AS checkpoint_version,t.id,t.source,t.code,t.stage,t.stage_status,t.stage_error_code,t.stage_error_message,t.revision,t.generation,t.created_at,t.updated_at,
 d.torrent_hash,d.torrent_identity_hash,d.content_path,k.sourcing_request_json,k.sourcing_last_checked_at,k.sourcing_next_check_at
 FROM tasks t LEFT JOIN task_downloads d ON d.task_id=t.id LEFT JOIN task_checkpoints k ON k.task_id=t.id WHERE t.id>? AND (`+predicate+`) ORDER BY t.id LIMIT ?`, after, limit)
	if err != nil {
		return nil, err
	}
	tasks := make([]*Task, 0, len(rows))
	for _, row := range rows {
		task, err := row.toTask()
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, s.overlay(task))
	}
	return tasks, nil
}
func (s *EventingTaskStore) TaskWorkPage(ctx context.Context, kind, after string, limit int) ([]*Task, error) {
	if pager, ok := s.store.(taskWorkPageProvider); ok {
		return pager.TaskWorkPage(ctx, kind, after, limit)
	}
	ids, err := s.TaskWorkIDs(ctx, kind, after, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*Task, 0, len(ids))
	for _, id := range ids {
		task, err := s.store.Find(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, nil
}

type downloaderTaskProvider interface {
	ListDownloaderTasks(context.Context) ([]*Task, error)
}

func (s *RelationalTaskStore) ListDownloaderTasks(ctx context.Context) ([]*Task, error) {
	var rows []sqliteTaskRow
	if err := s.core.Reader.SelectContext(ctx, &rows, `SELECT t.id,t.source,t.code,t.stage,t.stage_status,t.created_at,t.updated_at,d.torrent_hash,d.torrent_identity_hash,i.transfer_action,i.moji_source_path,i.moji_transfer_path FROM tasks t JOIN task_downloads d ON d.task_id=t.id JOIN task_ingest i ON i.task_id=t.id WHERE d.torrent_hash<>'' OR d.torrent_identity_hash IS NOT NULL ORDER BY t.id`); err != nil {
		return nil, err
	}
	tasks := make([]*Task, 0, len(rows))
	for _, row := range rows {
		task, err := row.toTask()
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, s.overlay(task))
	}
	return tasks, nil
}
func (s *EventingTaskStore) ListDownloaderTasks(ctx context.Context) ([]*Task, error) {
	if provider, ok := s.store.(downloaderTaskProvider); ok {
		return provider.ListDownloaderTasks(ctx)
	}
	return s.store.List(ctx)
}
func (s *Service) ListDownloaderTasks(ctx context.Context) ([]*Task, error) {
	if provider, ok := s.store.(downloaderTaskProvider); ok {
		return provider.ListDownloaderTasks(ctx)
	}
	return s.store.List(ctx)
}

type reviewTaskProvider interface {
	ListReviewTasks(context.Context) ([]*Task, error)
}

func (s *RelationalTaskStore) ListReviewTasks(ctx context.Context) ([]*Task, error) {
	type row struct{ ID, Code string }
	var rows []row
	if err := s.core.Reader.SelectContext(ctx, &rows, "SELECT id,code FROM tasks WHERE code<>'' ORDER BY created_at DESC,id"); err != nil {
		return nil, err
	}
	out := make([]*Task, 0, len(rows))
	for _, row := range rows {
		out = append(out, &Task{ID: row.ID, Code: row.Code})
	}
	return out, nil
}
func (s *EventingTaskStore) ListReviewTasks(ctx context.Context) ([]*Task, error) {
	if provider, ok := s.store.(reviewTaskProvider); ok {
		return provider.ListReviewTasks(ctx)
	}
	return s.store.List(ctx)
}
func (s *Service) ListReviewTasks(ctx context.Context) ([]*Task, error) {
	if provider, ok := s.store.(reviewTaskProvider); ok {
		return provider.ListReviewTasks(ctx)
	}
	return s.store.List(ctx)
}
