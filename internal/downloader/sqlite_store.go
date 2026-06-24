package downloader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SQLiteTaskStore struct {
	db *sql.DB
}

func NewSQLiteTaskStore(path string) (*SQLiteTaskStore, error) {
	db, err := OpenSQLiteDatabase(path)
	if err != nil {
		return nil, err
	}
	return &SQLiteTaskStore{db: db}, nil
}

func (s *SQLiteTaskStore) Create(ctx context.Context, task *Task) error {
	if task == nil {
		return errors.New("downloader: task is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("downloader: begin create task tx: %w", err)
	}
	defer tx.Rollback()

	if err := upsertTaskRow(ctx, tx, task, false); err != nil {
		return err
	}
	if err := insertTaskEvent(ctx, tx, task.ID, "created", "", string(task.Status), "task created"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("downloader: commit create task tx: %w", err)
	}
	return nil
}

func (s *SQLiteTaskStore) Update(ctx context.Context, task *Task) error {
	if task == nil {
		return errors.New("downloader: task is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("downloader: begin update task tx: %w", err)
	}
	defer tx.Rollback()

	var previousStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, task.ID).Scan(&previousStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("downloader: task %q not found", task.ID)
		}
		return fmt.Errorf("downloader: load task %q before update: %w", task.ID, err)
	}

	if err := upsertTaskRow(ctx, tx, task, true); err != nil {
		return err
	}

	message := "task updated"
	if previousStatus != string(task.Status) {
		message = fmt.Sprintf("task status %s -> %s", previousStatus, task.Status)
	}
	if err := insertTaskEvent(ctx, tx, task.ID, "updated", previousStatus, string(task.Status), message); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("downloader: commit update task tx: %w", err)
	}
	return nil
}

func (s *SQLiteTaskStore) Find(ctx context.Context, id string) (*Task, error) {
	row := s.db.QueryRowContext(ctx, taskSelectSQL+` WHERE id = ?`, id)
	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("downloader: task %q not found", id)
		}
		return nil, fmt.Errorf("downloader: find task %q: %w", id, err)
	}
	return task, nil
}

func (s *SQLiteTaskStore) List(ctx context.Context) ([]*Task, error) {
	rows, err := s.db.QueryContext(ctx, taskSelectSQL+` ORDER BY created_at DESC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("downloader: list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("downloader: scan listed task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("downloader: iterate listed tasks: %w", err)
	}

	sortTasks(tasks)
	return tasks, nil
}

const taskSelectSQL = `
SELECT
  id,
  source,
  query,
  status,
  torrent_url,
  save_path,
  category,
  tags,
  torrent_hash,
  torrent_name,
  progress,
  qbittorrent_state,
  content_path,
  completed_at,
  stash_mode,
  stash_source_path,
  stash_transfer_action,
  stash_transfer_path,
  stash_transfer_status,
  stash_transfer_error,
  stash_job_id,
  stash_scan_path,
  stash_scan_status,
  stash_scan_error,
  stash_scan_hint,
  stash_scan_started_at,
  error,
  candidate_title,
  candidate_tracker,
  candidate_info_hash,
  candidate_link,
  candidate_magnet_uri,
  candidate_size,
  candidate_seeders,
  candidate_peers,
  created_at,
  updated_at
FROM tasks`

func upsertTaskRow(ctx context.Context, tx *sql.Tx, task *Task, isUpdate bool) error {
	source := task.Source
	if source == "" {
		source = TaskSourceManual
	}
	query := `
INSERT INTO tasks (
  id, source, query, status, torrent_url, save_path, category, tags,
  torrent_hash, torrent_name, progress, qbittorrent_state, content_path,
  completed_at, stash_mode, stash_source_path, stash_transfer_action, stash_transfer_path,
  stash_transfer_status, stash_transfer_error, stash_job_id, stash_scan_path, stash_scan_status,
  stash_scan_error, stash_scan_hint, stash_scan_started_at, error,
  candidate_title, candidate_tracker, candidate_info_hash, candidate_link, candidate_magnet_uri,
  candidate_size, candidate_seeders, candidate_peers, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if isUpdate {
		query += `
ON CONFLICT(id) DO UPDATE SET
  source = excluded.source,
  query = excluded.query,
  status = excluded.status,
  torrent_url = excluded.torrent_url,
  save_path = excluded.save_path,
  category = excluded.category,
  tags = excluded.tags,
  torrent_hash = excluded.torrent_hash,
  torrent_name = excluded.torrent_name,
  progress = excluded.progress,
  qbittorrent_state = excluded.qbittorrent_state,
  content_path = excluded.content_path,
  completed_at = excluded.completed_at,
  stash_mode = excluded.stash_mode,
  stash_source_path = excluded.stash_source_path,
  stash_transfer_action = excluded.stash_transfer_action,
  stash_transfer_path = excluded.stash_transfer_path,
  stash_transfer_status = excluded.stash_transfer_status,
  stash_transfer_error = excluded.stash_transfer_error,
  stash_job_id = excluded.stash_job_id,
  stash_scan_path = excluded.stash_scan_path,
  stash_scan_status = excluded.stash_scan_status,
  stash_scan_error = excluded.stash_scan_error,
  stash_scan_hint = excluded.stash_scan_hint,
  stash_scan_started_at = excluded.stash_scan_started_at,
  error = excluded.error,
  candidate_title = excluded.candidate_title,
  candidate_tracker = excluded.candidate_tracker,
  candidate_info_hash = excluded.candidate_info_hash,
  candidate_link = excluded.candidate_link,
  candidate_magnet_uri = excluded.candidate_magnet_uri,
  candidate_size = excluded.candidate_size,
  candidate_seeders = excluded.candidate_seeders,
  candidate_peers = excluded.candidate_peers,
  updated_at = excluded.updated_at`
	}

	if _, err := tx.ExecContext(ctx, query,
		task.ID,
		string(source),
		task.Query,
		string(task.Status),
		task.TorrentURL,
		task.SavePath,
		task.Category,
		task.Tags,
		task.TorrentHash,
		task.TorrentName,
		task.Progress,
		task.QBittorrentState,
		task.ContentPath,
		formatOptionalSQLiteTimestamp(task.CompletedAt),
		task.StashMode,
		task.StashSourcePath,
		task.StashTransferAction,
		task.StashTransferPath,
		task.StashTransferStatus,
		task.StashTransferError,
		task.StashJobID,
		task.StashScanPath,
		task.StashScanStatus,
		task.StashScanError,
		task.StashScanHint,
		formatOptionalSQLiteTimestamp(task.StashScanStartedAt),
		task.Error,
		task.Candidate.Title,
		task.Candidate.Tracker,
		task.Candidate.InfoHash,
		task.Candidate.Link,
		task.Candidate.MagnetURI,
		task.Candidate.Size,
		task.Candidate.Seeders,
		task.Candidate.Peers,
		formatSQLiteTimestamp(task.CreatedAt),
		formatSQLiteTimestamp(task.UpdatedAt),
	); err != nil {
		op := "create"
		if isUpdate {
			op = "update"
		}
		return fmt.Errorf("downloader: %s task %q: %w", op, task.ID, err)
	}

	return nil
}

func insertTaskEvent(ctx context.Context, tx *sql.Tx, taskID, eventType, oldStatus, newStatus, message string) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO task_events (task_id, event_type, level, message, old_status, new_status, payload_json, created_at)
		 VALUES (?, ?, 'info', ?, ?, ?, '{}', ?)`,
		taskID,
		eventType,
		message,
		oldStatus,
		newStatus,
		formatSQLiteTimestamp(nowUTC()),
	); err != nil {
		return fmt.Errorf("downloader: insert task event for %q: %w", taskID, err)
	}
	return nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*Task, error) {
	var (
		task              Task
		source            string
		status            string
		completedAtRaw    sql.NullString
		stashStartedAtRaw sql.NullString
		createdAtRaw      string
		updatedAtRaw      string
	)

	if err := scanner.Scan(
		&task.ID,
		&source,
		&task.Query,
		&status,
		&task.TorrentURL,
		&task.SavePath,
		&task.Category,
		&task.Tags,
		&task.TorrentHash,
		&task.TorrentName,
		&task.Progress,
		&task.QBittorrentState,
		&task.ContentPath,
		&completedAtRaw,
		&task.StashMode,
		&task.StashSourcePath,
		&task.StashTransferAction,
		&task.StashTransferPath,
		&task.StashTransferStatus,
		&task.StashTransferError,
		&task.StashJobID,
		&task.StashScanPath,
		&task.StashScanStatus,
		&task.StashScanError,
		&task.StashScanHint,
		&stashStartedAtRaw,
		&task.Error,
		&task.Candidate.Title,
		&task.Candidate.Tracker,
		&task.Candidate.InfoHash,
		&task.Candidate.Link,
		&task.Candidate.MagnetURI,
		&task.Candidate.Size,
		&task.Candidate.Seeders,
		&task.Candidate.Peers,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return nil, err
	}

	task.Source = TaskSource(source)
	if task.Source == "" {
		task.Source = TaskSourceManual
	}
	task.Status = TaskStatus(status)

	var err error
	if task.CompletedAt, err = parseOptionalSQLiteTimestamp(completedAtRaw); err != nil {
		return nil, fmt.Errorf("downloader: parse completed_at for task %q: %w", task.ID, err)
	}
	if task.StashScanStartedAt, err = parseOptionalSQLiteTimestamp(stashStartedAtRaw); err != nil {
		return nil, fmt.Errorf("downloader: parse stash_scan_started_at for task %q: %w", task.ID, err)
	}
	if task.CreatedAt, err = parseSQLiteTimestamp(createdAtRaw); err != nil {
		return nil, fmt.Errorf("downloader: parse created_at for task %q: %w", task.ID, err)
	}
	if task.UpdatedAt, err = parseSQLiteTimestamp(updatedAtRaw); err != nil {
		return nil, fmt.Errorf("downloader: parse updated_at for task %q: %w", task.ID, err)
	}

	return &task, nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
