package taskruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

const diagnosticReport = `{"version":2,"parserVersion":7,"policy":{"version":7},"end":"USER_SELECTED","selected":"TEST-123","candidates":[],"sites":[],"counts":{}}`

func TestSelectionReportLivesOnlyInRuntimeAfterCoreCommit(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	task.SelectionReportJSON = diagnosticReport
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	var core sql.NullString
	if err := m.Core.Reader.GetContext(ctx, &core, `SELECT selection_report_json FROM task_checkpoints WHERE task_id=?`, task.ID); err != nil || core.Valid {
		t.Fatal("core contains selection explanation", err)
	}
	saved, err := s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != diagnosticReport {
		t.Fatal("runtime report missing", err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER fail_selection_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'audit failure'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	task.StageErrorMessage = "attempted state change"
	task.SelectionReportJSON = strings.Replace(diagnosticReport, "USER_SELECTED", "INCOMPLETE", 1)
	if err := s.Update(ctx, task); err == nil {
		t.Fatal("core failure ignored")
	}
	saved, err = s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != diagnosticReport {
		t.Fatal("uncommitted report replaced accepted report", err)
	}
}

func TestSelectionReportUpdateDoesNotWriteCore(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	task.SelectionReportJSON = diagnosticReport
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	writes, revision := m.Core.Metrics().Writes, task.Revision
	task.SelectionReportJSON = strings.Replace(diagnosticReport, "USER_SELECTED", "INCOMPLETE", 1)
	if err := s.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if m.Core.Metrics().Writes != writes || task.Revision != revision {
		t.Fatal("diagnostic update changed core")
	}
	saved, err := s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != task.SelectionReportJSON {
		t.Fatal("report-only update lost", err)
	}
}

func TestSelectionReportRuntimeFailureDoesNotBlockTask(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER fail_reports BEFORE INSERT ON diagnostics BEGIN SELECT RAISE(ABORT,'runtime unavailable'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	task := richRelationalTask()
	task.SelectionReportJSON = diagnosticReport
	if err := s.Create(ctx, task); err != nil {
		t.Fatal("diagnostic write blocked task", err)
	}
	saved, err := s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != "" || saved.TorrentURL != task.TorrentURL || string(saved.TorrentFileData) != string(task.TorrentFileData) {
		t.Fatal("core recovery state lost", err)
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.Exec(`DROP TRIGGER fail_reports`); return err }); err != nil {
		t.Fatal(err)
	}
	saved.SelectionReportJSON = diagnosticReport
	if err := s.Update(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.Exec(`DROP TABLE diagnostics`); return err }); err != nil {
		t.Fatal(err)
	}
	saved, err = s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != "" {
		t.Fatal("report read failure hid task", err)
	}
	saved.StageErrorMessage = "recover without diagnostics"
	if err := s.Update(ctx, saved); err != nil {
		t.Fatal("report read failure blocked update", err)
	}
}

func TestSelectionReportPruningPreservesAcceptedResource(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	task.SelectionReportJSON = diagnosticReport
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE diagnostics SET created_at=?`, formatDurableTime(time.Now().Add(-15*24*time.Hour)))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if n, err := m.PruneRuntime(ctx, time.Now()); err != nil || n != 1 {
		t.Fatal("report retention failed", n, err)
	}
	saved, err := s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != "" || saved.TorrentURL != task.TorrentURL {
		t.Fatal("expiry changed accepted resource", err)
	}
	saved.StageErrorMessage = "continued after expiry"
	if err := s.Update(ctx, saved); err != nil {
		t.Fatal(err)
	}
}

func TestSelectionReportForOldSourceIsHiddenWhenNewWriteFails(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	task.SelectionReportJSON = diagnosticReport
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER fail_reports BEFORE INSERT ON diagnostics BEGIN SELECT RAISE(ABORT,'runtime unavailable'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	task.TorrentURL = "https://example.invalid/new-resource"
	if err := s.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	saved, err := s.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != "" || saved.TorrentURL != task.TorrentURL {
		t.Fatal("old source report shown after new source accepted", err)
	}
}

func TestExpiredManualReportIsNotRebuiltDuringAcquisition(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, torrentResponse(testTorrentFile("SGKI-105B", []string{"unknown.mp4"}), 200))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.AddTorrentContext(ctx, AddTorrentRequest{Film: testFilm("SGKI-105B"), URL: "https://download.example/resource"})
	if err != nil {
		t.Fatal(err)
	}
	if task.SelectionReportJSON == "" {
		t.Fatal("explicit choice missing report")
	}
	task.SelectionReportJSON = ""
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := service.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Find(ctx, task.ID)
	if err != nil || saved.SelectionReportJSON != "" || saved.StageStatus != TaskStageStatusRunning {
		t.Fatal("expired report regenerated or acquisition blocked", err)
	}
}

func TestSelectionReportListLoadsRuntimeReportsWithoutCoreReferences(t *testing.T) {
	s, _ := relationalStore(t)
	ctx := context.Background()
	for _, id := range []string{"one", "two"} {
		task := richRelationalTask()
		task.ID = id
		task.Code = id
		task.TorrentIdentityHash = id
		task.TorrentIdentityMagnet = id
		task.SelectionReportJSON = diagnosticReport
		if err := s.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	tasks, err := s.List(ctx)
	if err != nil || len(tasks) != 2 {
		t.Fatal("list failed", err)
	}
	for _, task := range tasks {
		if task.SelectionReportJSON != diagnosticReport {
			t.Fatal("list lost runtime report")
		}
	}
}

func TestSelectionReportFromFutureRevisionIsNotDisplayed(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(storedSelectionReport{Revision: task.Revision + 1, Generation: task.Generation, Source: selectionReportSource(task), Report: json.RawMessage(diagnosticReport)})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO diagnostics(task_id,component,level,message,created_at) VALUES(?,?,'info',?,?)`, task.ID, selectionReportComponent, string(raw), formatDurableTime(time.Now()))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Find(ctx, task.ID)
	if err != nil || loaded.SelectionReportJSON != "" {
		t.Fatal("future task report displayed", err)
	}
}
