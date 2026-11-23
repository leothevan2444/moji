package taskruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

const selectionReportComponent = "resource-selection-report"

// Association and consistency checks belong to runtime alongside the report.
// Revision fences uncommitted/future state; source prevents a previous resource's
// report from being shown after a source change whose diagnostic write failed.
type storedSelectionReport struct {
	Revision   int64           `json:"revision"`
	Generation int64           `json:"generation"`
	Source     string          `json:"source"`
	Report     json.RawMessage `json:"report"`
}

func selectionReportSource(task *Task) string {
	data, _ := json.Marshal([]any{task.CreatedAt.UTC(), task.Generation, task.SourcingMode, task.SelectedResourceID, task.TorrentURL})
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// Only committed tasks publish reports. Runtime failure cannot undo the core
// operation. Reports use existing diagnostics retention and no core reference.
func (s *RelationalTaskStore) saveSelectionReport(ctx context.Context, previous, task *Task) {
	if task.SelectionReportJSON == "" || (previous != nil && previous.SelectionReportJSON == task.SelectionReportJSON && selectionReportSource(previous) == selectionReportSource(task)) {
		return
	}
	var header struct {
		Version int `json:"version"`
	}
	if len(task.SelectionReportJSON) > 256<<10 || json.Unmarshal([]byte(task.SelectionReportJSON), &header) != nil || (header.Version != 1 && header.Version != 2) {
		return
	}
	data, err := json.Marshal(storedSelectionReport{Revision: task.Revision, Generation: task.Generation, Source: selectionReportSource(task), Report: json.RawMessage(task.SelectionReportJSON)})
	if err != nil {
		return
	}
	work, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	err = s.runtime.WithWrite(work, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(work, `INSERT INTO diagnostics(task_id,component,level,message,created_at) VALUES(?,?,'info',?,?)`, task.ID, selectionReportComponent, string(data), formatDurableTime(time.Now()))
		return err
	})
	if err != nil {
		slog.Warn("resource selection report unavailable", "task_id", task.ID, "operation", "write")
	}
}

func (s *RelationalTaskStore) loadSelectionReport(ctx context.Context, task *Task) {
	task.SelectionReportJSON = ""
	work, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	var raw string
	err := s.runtime.Reader.GetContext(work, &raw, `SELECT message FROM diagnostics WHERE task_id=? AND component=? ORDER BY id DESC LIMIT 1`, task.ID, selectionReportComponent)
	if err != nil {
		return
	}
	applySelectionReport(task, raw)
}

func applySelectionReport(task *Task, raw string) {
	var stored storedSelectionReport
	if json.Unmarshal([]byte(raw), &stored) != nil || stored.Revision > task.Revision || stored.Generation != task.Generation || stored.Source != selectionReportSource(task) || len(stored.Report) > 256<<10 {
		return
	}
	var header struct {
		Version int `json:"version"`
	}
	if json.Unmarshal(stored.Report, &header) == nil && (header.Version == 1 || header.Version == 2) {
		task.SelectionReportJSON = string(stored.Report)
	}
}

// List retains full TaskStore semantics without one diagnostic scan per task.
// Summary/page queries intentionally do not load reports.
func (s *RelationalTaskStore) loadSelectionReports(ctx context.Context, tasks []*Task) {
	for offset := 0; offset < len(tasks); offset += 128 {
		end := min(offset+128, len(tasks))
		ids := make([]string, 0, end-offset)
		byID := make(map[string]*Task, end-offset)
		for _, task := range tasks[offset:end] {
			ids = append(ids, task.ID)
			byID[task.ID] = task
		}
		query, args, err := sqlx.In(`SELECT task_id,message FROM diagnostics WHERE id IN (SELECT MAX(id) FROM diagnostics WHERE component=? AND task_id IN (?) GROUP BY task_id)`, selectionReportComponent, ids)
		if err != nil {
			return
		}
		var rows []struct {
			TaskID  string `db:"task_id"`
			Message string `db:"message"`
		}
		work, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		err = s.runtime.Reader.SelectContext(work, &rows, query, args...)
		cancel()
		if err != nil {
			continue
		}
		for _, row := range rows {
			applySelectionReport(byID[row.TaskID], row.Message)
		}
	}
}
