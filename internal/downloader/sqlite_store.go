package downloader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type SQLiteTaskStore struct {
	db *sqlx.DB
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

	tx, err := s.db.BeginTxx(ctx, nil)
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

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("downloader: begin update task tx: %w", err)
	}
	defer tx.Rollback()

	var previousStatus string
	if err := tx.GetContext(ctx, &previousStatus, `SELECT status FROM tasks WHERE id = ?`, task.ID); err != nil {
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
	var row sqliteTaskRow
	if err := s.db.GetContext(ctx, &row, taskSelectSQL+` WHERE id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("downloader: task %q not found", id)
		}
		return nil, fmt.Errorf("downloader: find task %q: %w", id, err)
	}
	return row.toTask()
}

func (s *SQLiteTaskStore) FindByCode(ctx context.Context, code string) (*Task, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, nil
	}
	var row sqliteTaskRow
	if err := s.db.GetContext(ctx, &row, taskSelectSQL+` WHERE code = ? ORDER BY created_at DESC, id ASC LIMIT 1`, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("downloader: find task by code %q: %w", code, err)
	}
	return row.toTask()
}

func (s *SQLiteTaskStore) FindByTorrentIdentity(ctx context.Context, infoHash string, magnetURI string) (*Task, error) {
	infoHash = normalizeInfoHash(infoHash)
	magnetURI = normalizeMagnetURI(magnetURI)
	if infoHash == "" && magnetURI == "" {
		return nil, nil
	}

	var (
		query string
		args  []any
	)
	switch {
	case infoHash != "" && magnetURI != "":
		query = taskSelectSQL + ` WHERE torrent_identity_hash = ? OR torrent_identity_magnet = ? ORDER BY created_at DESC, id ASC LIMIT 1`
		args = []any{infoHash, magnetURI}
	case infoHash != "":
		query = taskSelectSQL + ` WHERE torrent_identity_hash = ? ORDER BY created_at DESC, id ASC LIMIT 1`
		args = []any{infoHash}
	default:
		query = taskSelectSQL + ` WHERE torrent_identity_magnet = ? ORDER BY created_at DESC, id ASC LIMIT 1`
		args = []any{magnetURI}
	}

	var row sqliteTaskRow
	if err := s.db.GetContext(ctx, &row, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("downloader: find task by torrent identity: %w", err)
	}
	return row.toTask()
}

func (s *SQLiteTaskStore) List(ctx context.Context) ([]*Task, error) {
	rows := make([]sqliteTaskRow, 0)
	if err := s.db.SelectContext(ctx, &rows, taskSelectSQL+` ORDER BY created_at DESC, id ASC`); err != nil {
		return nil, fmt.Errorf("downloader: list tasks: %w", err)
	}

	tasks := make([]*Task, 0, len(rows))
	for _, row := range rows {
		task, err := row.toTask()
		if err != nil {
			return nil, fmt.Errorf("downloader: scan listed task: %w", err)
		}
		tasks = append(tasks, task)
	}

	sortTasks(tasks)
	return tasks, nil
}

func (s *SQLiteTaskStore) Delete(ctx context.Context, id string) (*Task, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("downloader: begin delete task tx: %w", err)
	}
	defer tx.Rollback()

	var row sqliteTaskRow
	if err := tx.GetContext(ctx, &row, taskSelectSQL+` WHERE id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("downloader: task %q not found", id)
		}
		return nil, fmt.Errorf("downloader: load task %q before delete: %w", id, err)
	}
	task, err := row.toTask()
	if err != nil {
		return nil, fmt.Errorf("downloader: decode task %q before delete: %w", id, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("downloader: delete task %q: %w", id, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("downloader: commit delete task tx: %w", err)
	}
	return task, nil
}

const taskSelectSQL = `
SELECT
  id,
  source,
  query,
  code,
  status,
  torrent_url,
  save_path,
  category,
  tags,
  torrent_identity_hash,
  torrent_identity_magnet,
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

type sqliteTaskRow struct {
	ID                    string         `db:"id"`
	Source                string         `db:"source"`
	Query                 string         `db:"query"`
	Code                  string         `db:"code"`
	Status                string         `db:"status"`
	TorrentURL            string         `db:"torrent_url"`
	SavePath              sql.NullString `db:"save_path"`
	Category              sql.NullString `db:"category"`
	Tags                  sql.NullString `db:"tags"`
	TorrentIdentityHash   sql.NullString `db:"torrent_identity_hash"`
	TorrentIdentityMagnet sql.NullString `db:"torrent_identity_magnet"`
	TorrentHash           sql.NullString `db:"torrent_hash"`
	TorrentName           sql.NullString `db:"torrent_name"`
	Progress              float64        `db:"progress"`
	QBittorrentState      sql.NullString `db:"qbittorrent_state"`
	ContentPath           sql.NullString `db:"content_path"`
	CompletedAt           sql.NullString `db:"completed_at"`
	StashMode             sql.NullString `db:"stash_mode"`
	StashSourcePath       sql.NullString `db:"stash_source_path"`
	StashTransferAction   sql.NullString `db:"stash_transfer_action"`
	StashTransferPath     sql.NullString `db:"stash_transfer_path"`
	StashTransferStatus   sql.NullString `db:"stash_transfer_status"`
	StashTransferError    sql.NullString `db:"stash_transfer_error"`
	StashJobID            sql.NullString `db:"stash_job_id"`
	StashScanPath         sql.NullString `db:"stash_scan_path"`
	StashScanStatus       sql.NullString `db:"stash_scan_status"`
	StashScanError        sql.NullString `db:"stash_scan_error"`
	StashScanHint         sql.NullString `db:"stash_scan_hint"`
	StashScanStartedAt    sql.NullString `db:"stash_scan_started_at"`
	Error                 sql.NullString `db:"error"`
	CandidateTitle        string         `db:"candidate_title"`
	CandidateTracker      string         `db:"candidate_tracker"`
	CandidateInfoHash     string         `db:"candidate_info_hash"`
	CandidateLink         string         `db:"candidate_link"`
	CandidateMagnetURI    string         `db:"candidate_magnet_uri"`
	CandidateSize         int64          `db:"candidate_size"`
	CandidateSeeders      int            `db:"candidate_seeders"`
	CandidatePeers        int            `db:"candidate_peers"`
	CreatedAt             string         `db:"created_at"`
	UpdatedAt             string         `db:"updated_at"`
}

func (r sqliteTaskRow) toTask() (*Task, error) {
	task := &Task{
		ID:                    r.ID,
		Source:                TaskSource(r.Source),
		Query:                 r.Query,
		Code:                  r.Code,
		Status:                TaskStatus(r.Status),
		TorrentURL:            r.TorrentURL,
		SavePath:              nullableStringValue(r.SavePath),
		Category:              nullableStringValue(r.Category),
		Tags:                  nullableStringValue(r.Tags),
		TorrentIdentityHash:   nullableStringValue(r.TorrentIdentityHash),
		TorrentIdentityMagnet: nullableStringValue(r.TorrentIdentityMagnet),
		TorrentHash:           nullableStringValue(r.TorrentHash),
		TorrentName:           nullableStringValue(r.TorrentName),
		Progress:              r.Progress,
		QBittorrentState:      nullableStringValue(r.QBittorrentState),
		ContentPath:           nullableStringValue(r.ContentPath),
		StashMode:             nullableStringValue(r.StashMode),
		StashSourcePath:       nullableStringValue(r.StashSourcePath),
		StashTransferAction:   nullableStringValue(r.StashTransferAction),
		StashTransferPath:     nullableStringValue(r.StashTransferPath),
		StashTransferStatus:   nullableStringValue(r.StashTransferStatus),
		StashTransferError:    nullableStringValue(r.StashTransferError),
		StashJobID:            nullableStringValue(r.StashJobID),
		StashScanPath:         nullableStringValue(r.StashScanPath),
		StashScanStatus:       nullableStringValue(r.StashScanStatus),
		StashScanError:        nullableStringValue(r.StashScanError),
		StashScanHint:         nullableStringValue(r.StashScanHint),
		Error:                 nullableStringValue(r.Error),
		Candidate: Candidate{
			Title:     r.CandidateTitle,
			Tracker:   r.CandidateTracker,
			InfoHash:  r.CandidateInfoHash,
			Link:      r.CandidateLink,
			MagnetURI: r.CandidateMagnetURI,
			Size:      r.CandidateSize,
			Seeders:   r.CandidateSeeders,
			Peers:     r.CandidatePeers,
		},
	}
	if task.Source == "" {
		task.Source = TaskSourceManual
	}

	var err error
	if task.CompletedAt, err = parseOptionalSQLiteTimestamp(r.CompletedAt); err != nil {
		return nil, fmt.Errorf("downloader: parse completed_at for task %q: %w", task.ID, err)
	}
	if task.StashScanStartedAt, err = parseOptionalSQLiteTimestamp(r.StashScanStartedAt); err != nil {
		return nil, fmt.Errorf("downloader: parse stash_scan_started_at for task %q: %w", task.ID, err)
	}
	if task.CreatedAt, err = parseSQLiteTimestamp(r.CreatedAt); err != nil {
		return nil, fmt.Errorf("downloader: parse created_at for task %q: %w", task.ID, err)
	}
	if task.UpdatedAt, err = parseSQLiteTimestamp(r.UpdatedAt); err != nil {
		return nil, fmt.Errorf("downloader: parse updated_at for task %q: %w", task.ID, err)
	}
	return task, nil
}

type sqliteTaskParams struct {
	ID                    string  `db:"id"`
	Source                string  `db:"source"`
	Query                 string  `db:"query"`
	Code                  string  `db:"code"`
	Status                string  `db:"status"`
	TorrentURL            string  `db:"torrent_url"`
	SavePath              any     `db:"save_path"`
	Category              any     `db:"category"`
	Tags                  any     `db:"tags"`
	TorrentIdentityHash   any     `db:"torrent_identity_hash"`
	TorrentIdentityMagnet any     `db:"torrent_identity_magnet"`
	TorrentHash           any     `db:"torrent_hash"`
	TorrentName           any     `db:"torrent_name"`
	Progress              float64 `db:"progress"`
	QBittorrentState      any     `db:"qbittorrent_state"`
	ContentPath           any     `db:"content_path"`
	CompletedAt           any     `db:"completed_at"`
	StashMode             any     `db:"stash_mode"`
	StashSourcePath       any     `db:"stash_source_path"`
	StashTransferAction   any     `db:"stash_transfer_action"`
	StashTransferPath     any     `db:"stash_transfer_path"`
	StashTransferStatus   any     `db:"stash_transfer_status"`
	StashTransferError    any     `db:"stash_transfer_error"`
	StashJobID            any     `db:"stash_job_id"`
	StashScanPath         any     `db:"stash_scan_path"`
	StashScanStatus       any     `db:"stash_scan_status"`
	StashScanError        any     `db:"stash_scan_error"`
	StashScanHint         any     `db:"stash_scan_hint"`
	StashScanStartedAt    any     `db:"stash_scan_started_at"`
	Error                 any     `db:"error"`
	CandidateTitle        string  `db:"candidate_title"`
	CandidateTracker      string  `db:"candidate_tracker"`
	CandidateInfoHash     string  `db:"candidate_info_hash"`
	CandidateLink         string  `db:"candidate_link"`
	CandidateMagnetURI    string  `db:"candidate_magnet_uri"`
	CandidateSize         int64   `db:"candidate_size"`
	CandidateSeeders      int     `db:"candidate_seeders"`
	CandidatePeers        int     `db:"candidate_peers"`
	CreatedAt             string  `db:"created_at"`
	UpdatedAt             string  `db:"updated_at"`
}

func taskToSQLiteParams(task *Task) sqliteTaskParams {
	source := task.Source
	if source == "" {
		source = TaskSourceManual
	}
	return sqliteTaskParams{
		ID:                    task.ID,
		Source:                string(source),
		Query:                 task.Query,
		Code:                  task.Code,
		Status:                string(task.Status),
		TorrentURL:            task.TorrentURL,
		SavePath:              nullableStringParam(task.SavePath),
		Category:              nullableStringParam(task.Category),
		Tags:                  nullableStringParam(task.Tags),
		TorrentIdentityHash:   nullableStringParam(task.TorrentIdentityHash),
		TorrentIdentityMagnet: nullableStringParam(task.TorrentIdentityMagnet),
		TorrentHash:           nullableStringParam(task.TorrentHash),
		TorrentName:           nullableStringParam(task.TorrentName),
		Progress:              task.Progress,
		QBittorrentState:      nullableStringParam(task.QBittorrentState),
		ContentPath:           nullableStringParam(task.ContentPath),
		CompletedAt:           formatOptionalSQLiteTimestamp(task.CompletedAt),
		StashMode:             nullableStringParam(task.StashMode),
		StashSourcePath:       nullableStringParam(task.StashSourcePath),
		StashTransferAction:   nullableStringParam(task.StashTransferAction),
		StashTransferPath:     nullableStringParam(task.StashTransferPath),
		StashTransferStatus:   nullableStringParam(task.StashTransferStatus),
		StashTransferError:    nullableStringParam(task.StashTransferError),
		StashJobID:            nullableStringParam(task.StashJobID),
		StashScanPath:         nullableStringParam(task.StashScanPath),
		StashScanStatus:       nullableStringParam(task.StashScanStatus),
		StashScanError:        nullableStringParam(task.StashScanError),
		StashScanHint:         nullableStringParam(task.StashScanHint),
		StashScanStartedAt:    formatOptionalSQLiteTimestamp(task.StashScanStartedAt),
		Error:                 nullableStringParam(task.Error),
		CandidateTitle:        task.Candidate.Title,
		CandidateTracker:      task.Candidate.Tracker,
		CandidateInfoHash:     task.Candidate.InfoHash,
		CandidateLink:         task.Candidate.Link,
		CandidateMagnetURI:    task.Candidate.MagnetURI,
		CandidateSize:         task.Candidate.Size,
		CandidateSeeders:      task.Candidate.Seeders,
		CandidatePeers:        task.Candidate.Peers,
		CreatedAt:             formatSQLiteTimestamp(task.CreatedAt),
		UpdatedAt:             formatSQLiteTimestamp(task.UpdatedAt),
	}
}

func upsertTaskRow(ctx context.Context, tx *sqlx.Tx, task *Task, isUpdate bool) error {
	params := taskToSQLiteParams(task)
	query := `
INSERT INTO tasks (
  id, source, query, code, status, torrent_url, save_path, category, tags,
  torrent_identity_hash, torrent_identity_magnet, torrent_hash, torrent_name, progress, qbittorrent_state, content_path,
  completed_at, stash_mode, stash_source_path, stash_transfer_action, stash_transfer_path,
  stash_transfer_status, stash_transfer_error, stash_job_id, stash_scan_path, stash_scan_status,
  stash_scan_error, stash_scan_hint, stash_scan_started_at, error,
  candidate_title, candidate_tracker, candidate_info_hash, candidate_link, candidate_magnet_uri,
  candidate_size, candidate_seeders, candidate_peers, created_at, updated_at
) VALUES (
  :id, :source, :query, :code, :status, :torrent_url, :save_path, :category, :tags,
  :torrent_identity_hash, :torrent_identity_magnet, :torrent_hash, :torrent_name, :progress, :qbittorrent_state, :content_path,
  :completed_at, :stash_mode, :stash_source_path, :stash_transfer_action, :stash_transfer_path,
  :stash_transfer_status, :stash_transfer_error, :stash_job_id, :stash_scan_path, :stash_scan_status,
  :stash_scan_error, :stash_scan_hint, :stash_scan_started_at, :error,
  :candidate_title, :candidate_tracker, :candidate_info_hash, :candidate_link, :candidate_magnet_uri,
  :candidate_size, :candidate_seeders, :candidate_peers, :created_at, :updated_at
)`
	if isUpdate {
		query += `
ON CONFLICT(id) DO UPDATE SET
  source = excluded.source,
  query = excluded.query,
  code = excluded.code,
  status = excluded.status,
  torrent_url = excluded.torrent_url,
  save_path = excluded.save_path,
  category = excluded.category,
  tags = excluded.tags,
  torrent_identity_hash = excluded.torrent_identity_hash,
  torrent_identity_magnet = excluded.torrent_identity_magnet,
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

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		op := "create"
		if isUpdate {
			op = "update"
		}
		return fmt.Errorf("downloader: %s task %q: %w", op, task.ID, translateTaskConstraintError(err))
	}
	return nil
}

func insertTaskEvent(ctx context.Context, tx *sqlx.Tx, taskID, eventType, oldStatus, newStatus, message string) error {
	if _, err := tx.NamedExecContext(
		ctx,
		`INSERT INTO task_events (task_id, event_type, level, message, old_status, new_status, created_at)
		 VALUES (:task_id, :event_type, 'info', :message, :old_status, :new_status, :created_at)`,
		map[string]any{
			"task_id":    taskID,
			"event_type": eventType,
			"message":    message,
			"old_status": oldStatus,
			"new_status": newStatus,
			"created_at": formatSQLiteTimestamp(nowUTC()),
		},
	); err != nil {
		return fmt.Errorf("downloader: insert task event for %q: %w", taskID, err)
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func translateTaskConstraintError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "idx_tasks_code_unique"),
		strings.Contains(message, "tasks.code"):
		return ErrDuplicateCodeTask
	case strings.Contains(message, "idx_tasks_torrent_identity_hash_unique"),
		strings.Contains(message, "idx_tasks_torrent_identity_magnet_unique"),
		strings.Contains(message, "tasks.torrent_identity_hash"),
		strings.Contains(message, "tasks.torrent_identity_magnet"):
		return ErrDuplicateTorrentTask
	default:
		return err
	}
}

func nullableStringParam(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
