package taskruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/storage"
)

var ErrTaskRevisionConflict = errors.New("taskruntime: task changed since it was read")

type RelationalTaskStore struct {
	mu           sync.Mutex
	core         *storage.Database
	observations *storage.ObservationStore
}

func NewRelationalTaskStore(core *storage.Database, observations *storage.ObservationStore) *RelationalTaskStore {
	return &RelationalTaskStore{core: core, observations: observations}
}

func (s *RelationalTaskStore) read(ctx context.Context, db sqlx.ExtContext, where string, args ...any) (*Task, error) {
	var row sqliteTaskRow
	if err := sqlx.GetContext(ctx, db, &row, relationalTaskSelectSQL+where, args...); err != nil {
		return nil, err
	}
	task, err := row.toTask()
	if err != nil {
		return nil, err
	}
	return s.overlay(task), nil
}
func (s *RelationalTaskStore) overlay(task *Task) *Task {
	if observed, ok := s.observations.Get(task.ID); ok {
		task.TorrentName, task.Progress, task.QBittorrentState = observed.TorrentName, observed.Progress, observed.QBittorrentState
		if observed.ContentPath != "" {
			task.ContentPath = observed.ContentPath
		}
		if observed.LastProgressAt != nil {
			task.LastProgressAt, _ = timePointer(*observed.LastProgressAt)
		}
		if at, err := time.Parse(time.RFC3339Nano, observed.ObservedAt); err == nil && at.After(task.UpdatedAt) {
			task.UpdatedAt = at
		}
	}
	return task
}
func timePointer(value string) (*time.Time, error) {
	v, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *RelationalTaskStore) Find(ctx context.Context, id string) (*Task, error) {
	task, err := s.read(ctx, s.core.Reader, " WHERE id=?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("taskruntime: task %q not found", id)
	}
	return task, err
}
func (s *RelationalTaskStore) FindByCode(ctx context.Context, code string) (*Task, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, nil
	}
	task, err := s.read(ctx, s.core.Reader, " WHERE code=?", code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return task, err
}
func (s *RelationalTaskStore) FindByTorrentIdentity(ctx context.Context, hash, magnet string) (*Task, error) {
	hash, magnet = normalizeInfoHash(hash), normalizeMagnetURI(magnet)
	if hash == "" && magnet == "" {
		return nil, nil
	}
	task, err := s.read(ctx, s.core.Reader, " WHERE torrent_identity_hash=? OR torrent_identity_magnet=? ORDER BY created_at DESC,id LIMIT 1", hash, magnet)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return task, err
}
func (s *RelationalTaskStore) List(ctx context.Context) ([]*Task, error) {
	var rows []sqliteTaskRow
	if err := s.core.Reader.SelectContext(ctx, &rows, relationalTaskSelectSQL+" ORDER BY created_at DESC,id"); err != nil {
		return nil, err
	}
	out := make([]*Task, 0, len(rows))
	for _, row := range rows {
		task, err := row.toTask()
		if err != nil {
			return nil, err
		}
		out = append(out, s.overlay(task))
	}
	sortTasks(out)
	return out, nil
}
func sameDurableTask(a, b *Task) bool {
	left, right := cloneTask(a), cloneTask(b)
	for _, t := range []*Task{left, right} {
		t.Revision = 0
		if t.FilePlanJSON != "" {
			t.setFilePlan(t.FilePlan())
		}
		t.Progress = 0
		t.TorrentName = ""
		t.QBittorrentState = ""
		t.LastProgressAt = nil
		t.UpdatedAt = time.Time{}
	}
	return reflect.DeepEqual(left, right)
}
func (s *RelationalTaskStore) remember(task *Task) {
	value := storage.TaskObservation{TaskID: task.ID, TorrentName: task.TorrentName, Progress: task.Progress, QBittorrentState: task.QBittorrentState, ContentPath: task.ContentPath, ObservedAt: formatDurableTime(task.UpdatedAt)}
	if task.LastProgressAt != nil {
		v := formatDurableTime(*task.LastProgressAt)
		value.LastProgressAt = &v
	}
	s.observations.Put(value)
}
func formatDurableTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000000Z") }
func (s *RelationalTaskStore) Create(ctx context.Context, task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task == nil {
		return errors.New("taskruntime: task is nil")
	}
	if task.Source == "" {
		task.Source = TaskSourceManual
	}
	refreshTaskStageFields(task)
	task.Revision = 1
	trackTaskProgress(nil, task)
	err := s.core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if err := writeRelationalTask(ctx, tx, task, false); err != nil {
			return err
		}
		return writeAudit(ctx, tx, task, "created", "", "task created")
	})
	if err != nil {
		return translateTaskConstraintError(err)
	}
	s.remember(task)
	return nil
}
func (s *RelationalTaskStore) Update(ctx context.Context, task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task == nil {
		return errors.New("taskruntime: task is nil")
	}
	if task.FilePlanJSON != "" {
		plan, err := strictFilePlan(task.FilePlanJSON)
		if err != nil {
			return err
		}
		task.setFilePlan(plan)
	}
	refreshTaskStageFields(task)
	previous, err := s.Find(ctx, task.ID)
	if err != nil {
		return err
	}
	if previous.Revision != task.Revision {
		return ErrTaskRevisionConflict
	}
	trackTaskProgress(previous, task)
	if sameDurableTask(previous, task) {
		// Observations never reserve the core writer or append a durable audit event.
		s.remember(task)
		return nil
	}
	task.Revision++
	err = s.core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if err := writeRelationalTask(ctx, tx, task, true); err != nil {
			return err
		}
		kind := "updated"
		if !reflect.DeepEqual(task.FilePlan(), previous.FilePlan()) {
			kind = "file-selection"
		}
		if task.RestartStateJSON != previous.RestartStateJSON {
			kind = "restart-checkpoint"
		}
		return writeAudit(ctx, tx, task, kind, string(previous.Stage), "task state changed")
	})
	if err != nil {
		task.Revision--
		return translateTaskConstraintError(err)
	}
	s.remember(task)
	return nil
}
func (s *RelationalTaskStore) Delete(ctx context.Context, id string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var task *Task
	err := s.core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var err error
		task, err = s.read(ctx, tx, " WHERE id=?", id)
		if err != nil {
			return err
		}
		if err = writeAudit(ctx, tx, task, "deleted", string(task.Stage), "task deleted"); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM tasks WHERE id=?", id)
		return err
	})
	if err == nil {
		s.observations.Delete(id)
	}
	return task, err
}
func writeAudit(ctx context.Context, tx *sqlx.Tx, task *Task, kind, oldStage, message string) error {
	payload, err := json.Marshal(map[string]any{"revision": task.Revision, "generation": task.Generation, "error_code": task.StageErrorCode, "file_plan": json.RawMessage(defaultJSON(task.FilePlanJSON)), "restart": json.RawMessage(defaultJSON(task.RestartStateJSON))})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(task_id,subject_id,event_type,level,message,old_stage,new_stage,payload_json,created_at) VALUES(?,?,?,'info',?,?,?,?,?)`, task.ID, task.ID, kind, message, oldStage, task.Stage, string(payload), formatDurableTime(task.UpdatedAt))
	return err
}
func defaultJSON(value string) string {
	if value == "" {
		return "null"
	}
	return value
}

func writeRelationalTask(ctx context.Context, tx *sqlx.Tx, task *Task, update bool) error {
	raw := reflect.ValueOf(taskToSQLiteParams(task))
	params := map[string]any{}
	for i := 0; i < raw.NumField(); i++ {
		params[raw.Type().Field(i).Tag.Get("db")] = raw.Field(i).Interface()
	}
	params["revision"], params["generation"] = task.Revision, task.Generation
	params["created_at"], params["updated_at"] = formatDurableTime(task.CreatedAt), formatDurableTime(task.UpdatedAt)
	for _, key := range []string{"sourcing_last_checked_at", "sourcing_next_check_at", "sourcing_completed_at", "download_completed_at", "ingest_completed_at", "stash_scan_started_at"} {
		if v, ok := params[key].(string); ok {
			at, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return err
			}
			params[key] = formatDurableTime(at)
		}
	}
	core := []string{"id", "source", "code", "stage", "stage_status", "stage_error_code", "stage_error_message", "revision", "generation", "created_at", "updated_at"}
	if update {
		// The expected revision is checked inside the reserved writer transaction.
		set := make([]string, 0, len(core))
		for _, col := range core {
			if col != "id" && col != "created_at" {
				set = append(set, col+"=:"+col)
			}
		}
		result, err := tx.NamedExecContext(ctx, "UPDATE tasks SET "+strings.Join(set, ",")+" WHERE id=:id AND revision=:previous_revision", withPreviousRevision(params, task.Revision-1))
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrTaskRevisionConflict
		}
	} else {
		if err := upsertFields(ctx, tx, "tasks", core, params, false); err != nil {
			return err
		}
	}
	params["task_id"] = task.ID
	groups := []struct {
		table   string
		columns []string
	}{
		{"task_downloads", strings.Fields("task_id torrent_url save_path category tags torrent_identity_hash torrent_identity_magnet torrent_hash content_path")},
		{"task_checkpoints", strings.Fields("task_id sourcing_request_json restart_state_json sourcing_last_checked_at sourcing_next_check_at sourcing_completed_at download_completed_at ingest_completed_at")},
		{"task_ingest", strings.Fields("task_id moji_source_path transfer_action moji_transfer_path transfer_error stash_scan_job_id stash_ingest_phase stash_generate_job_id stash_identify_job_id stash_scene_ids_json stash_scene_recovery_count stash_scan_path stash_scan_error stash_scan_hint stash_scan_started_at")},
	}
	for _, g := range groups {
		for _, col := range g.columns {
			if params[col] == nil && !strings.HasSuffix(col, "_json") && !strings.HasSuffix(col, "_at") && col != "torrent_identity_hash" && col != "torrent_identity_magnet" {
				params[col] = ""
			}
		}
		if err := upsertFields(ctx, tx, g.table, g.columns, params, true); err != nil {
			return err
		}
	}
	candidate := map[string]any{"task_id": task.ID, "title": task.Candidate.Title, "tracker": task.Candidate.Tracker, "info_hash": task.Candidate.InfoHash, "link": task.Candidate.Link, "magnet_uri": task.Candidate.MagnetURI, "size": task.Candidate.Size, "seeders": task.Candidate.Seeders, "peers": task.Candidate.Peers}
	if err := upsertFields(ctx, tx, "task_candidates", strings.Fields("task_id title tracker info_hash link magnet_uri size seeders peers"), candidate, true); err != nil {
		return err
	}
	if len(task.TorrentFileData) > 0 {
		if err := upsertFields(ctx, tx, "task_attachments", []string{"task_id", "torrent_file_data"}, params, true); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, "DELETE FROM task_attachments WHERE task_id=?", task.ID); err != nil {
			return err
		}
	}
	return writeRelationalFilePlan(ctx, tx, task)
}
func withPreviousRevision(params map[string]any, revision int64) map[string]any {
	params["previous_revision"] = revision
	return params
}
func upsertFields(ctx context.Context, tx *sqlx.Tx, table string, columns []string, params map[string]any, update bool) error {
	bindings, set := make([]string, len(columns)), make([]string, 0, len(columns)-1)
	for i, col := range columns {
		bindings[i] = ":" + col
		if i > 0 {
			set = append(set, col+"=excluded."+col)
		}
	}
	query := "INSERT INTO " + table + "(" + strings.Join(columns, ",") + ") VALUES(" + strings.Join(bindings, ",") + ")"
	if update {
		changes := make([]string, 0, len(columns)-1)
		for _, col := range columns[1:] {
			changes = append(changes, table+"."+col+" IS NOT excluded."+col)
		}
		query += " ON CONFLICT(" + columns[0] + ") DO UPDATE SET " + strings.Join(set, ",") + " WHERE " + strings.Join(changes, " OR ")
	}
	_, err := tx.NamedExecContext(ctx, query, params)
	return err
}
func writeRelationalFilePlan(ctx context.Context, tx *sqlx.Tx, task *Task) error {
	if task.FilePlanJSON == "" {
		_, err := tx.ExecContext(ctx, "DELETE FROM task_file_plans WHERE task_id=?", task.ID)
		return err
	}
	if !json.Valid([]byte(task.FilePlanJSON)) {
		return errors.New("taskruntime: invalid file plan checkpoint")
	}
	plan, planErr := strictFilePlan(task.FilePlanJSON)
	if planErr != nil {
		return planErr
	}
	var existing sqliteTaskRow
	if err := tx.GetContext(ctx, &existing, relationalTaskSelectSQL+" WHERE id=?", task.ID); err != nil {
		return err
	}
	var planGeneration int64
	generationErr := tx.GetContext(ctx, &planGeneration, "SELECT generation FROM task_file_plans WHERE task_id=?", task.ID)
	if generationErr != nil && !errors.Is(generationErr, sql.ErrNoRows) {
		return generationErr
	}
	if existing.FilePlanJSON.Valid && generationErr == nil && planGeneration == task.Generation {
		previous, err := existing.toTask()
		if err != nil {
			return err
		}
		if reflect.DeepEqual(previous.FilePlan(), plan) {
			return nil
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO task_file_plans(task_id,generation,confirmed,keep_subtitles,pause_requested,applied) VALUES(?,?,?,?,?,?) ON CONFLICT(task_id) DO UPDATE SET generation=excluded.generation,confirmed=excluded.confirmed,keep_subtitles=excluded.keep_subtitles,pause_requested=excluded.pause_requested,applied=excluded.applied`, task.ID, task.Generation, plan.Confirmed, plan.KeepSubtitles, plan.PauseRequested, plan.Applied)
	if err != nil {
		return err
	}
	// File plans change only at selection/restart, not on routine progress writes.
	if _, err := tx.ExecContext(ctx, "DELETE FROM task_files WHERE task_id=?", task.ID); err != nil {
		return err
	}
	for position, file := range plan.Files {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_files(task_id,file_index,position,path,size,kind,part,selected,reason) VALUES(?,?,?,?,?,?,?,?,?)`, task.ID, file.Index, position, file.Path, file.Size, file.Kind, file.Part, file.Selected, file.Reason); err != nil {
			return err
		}
	}
	return nil
}

// TaskStats reads only the indexed lifecycle columns, never checkpoints or blobs.
func (s *RelationalTaskStore) TaskStats(ctx context.Context) (TaskStats, error) {
	var total, active, completed, downloading, pending, failed int
	err := s.core.Reader.QueryRowContext(ctx, `SELECT count(*),
 COALESCE(sum(stage<>'COMPLETED'),0),COALESCE(sum(stage='COMPLETED'),0),
 COALESCE(sum(stage='DOWNLOADING' AND stage_status='RUNNING'),0),
 COALESCE(sum(stage IN ('PENDING_INGEST','TRANSFERRING','SCANNING')),0),
 COALESCE(sum(stage_status='BLOCKED'),0) FROM tasks`).Scan(&total, &active, &completed, &downloading, &pending, &failed)
	return TaskStats{Total: total, Active: active, Completed: completed, Downloading: downloading, PendingScans: pending, Failed: failed}, err
}

func strictFilePlan(value string) (mediafiles.Plan, error) {
	var plan mediafiles.Plan
	if !json.Valid([]byte(value)) {
		return plan, errors.New("taskruntime: invalid file plan checkpoint JSON")
	}
	if strings.TrimSpace(value) == "null" {
		return plan, errors.New("taskruntime: null file plan checkpoint")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return plan, fmt.Errorf("taskruntime: invalid file plan checkpoint: %w", err)
	}
	if plan.Files == nil {
		plan.Files = []mediafiles.File{}
	}
	return plan, nil
}
