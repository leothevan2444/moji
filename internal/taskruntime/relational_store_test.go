package taskruntime

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/storage"
)

func relationalStore(t *testing.T) (*RelationalTaskStore, *storage.Manager) {
	t.Helper()
	m, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	return NewRelationalTaskStore(m.Core, m.Observations), m
}
func richRelationalTask() *Task {
	now := time.Date(2026, 10, 27, 5, 2, 3, 123456789, time.UTC)
	task := &Task{ID: "task-one", Source: TaskSourceSubscription, Code: "TEST-123", Stage: TaskStageScanning, StageStatus: TaskStageStatusBlocked,
		StageErrorCode: "SCAN_FAILURE", StageErrorMessage: "retry required", Generation: 3,
		RestartStateJSON: `{"version":1,"paused":true}`, SourcingRequestJSON: `{"code":"TEST-123"}`,
		TorrentFileData: []byte("torrent attachment"), SourcingLastCheckedAt: &now, SourcingNextCheckAt: &now,
		Candidate:  Candidate{Title: "title", Tracker: "tracker", InfoHash: "candidate-hash", Link: "https://example.invalid/torrent", MagnetURI: "magnet:?xt=urn:btih:abc", Size: 999, Seeders: 5, Peers: 7},
		TorrentURL: "https://example.invalid/file", SavePath: "/downloads", Category: "moji", Tags: "one,two", TorrentIdentityHash: "identity-hash", TorrentIdentityMagnet: "magnet:?xt=urn:btih:xyz", TorrentHash: "actual-hash",
		TorrentName: "download name", Progress: 0.8, QBittorrentState: "stoppedDL", ContentPath: "/downloads/content",
		SourcingCompletedAt: &now, DownloadCompletedAt: &now, IngestCompletedAt: &now,
		MojiSourcePath: "/source", TransferAction: "COPY", MojiTransferPath: "/target", TransferError: "transfer error",
		StashScanJobID: "scan", StashIngestPhase: "IDENTIFY", StashGenerateJobID: "generate", StashIdentifyJobID: "identify", StashSceneIDsJSON: `["s1","s2"]`, StashSceneRecoveryCount: 2,
		StashScanPath: "/scan", StashScanError: "scan error", StashScanHint: "hint", StashScanStartedAt: &now, CreatedAt: now, UpdatedAt: now}
	task.setFilePlan(mediafiles.Plan{Confirmed: true, KeepSubtitles: true, PauseRequested: true, Applied: true, Files: []mediafiles.File{
		{Index: 8, Path: "part-b.mp4", Size: 600, Kind: mediafiles.Main, Part: 2, Selected: true, Reason: "selected"},
		{Index: 2, Path: "part-a.mp4", Size: 300, Kind: mediafiles.Main, Part: 1, Selected: false},
		{Index: 9, Path: "caption.srt", Size: 99, Kind: mediafiles.Subtitle, Selected: true},
	}})
	return task
}
func TestRelationalTaskRoundTripAndAuditAtomicity(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Find(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(task, loaded) {
		t.Fatalf("roundtrip differs:\noriginal=%+v\nloaded=%+v", task, loaded)
	}
	// Failure after updating the task must roll back every child and the revision.
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`CREATE TRIGGER fail_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'simulated audit failure'); END`)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	task.StageErrorMessage = "new message"
	task.UpdatedAt = task.UpdatedAt.Add(time.Second)
	if err := s.Update(ctx, task); err == nil {
		t.Fatal("expected injected audit error")
	}
	after, err := s.Find(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, after) {
		t.Fatal("failed transaction changed task or observation")
	}
}
func TestRelationalProgressDoesNotWriteCoreAndRejectsStaleState(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	before := m.Core.Metrics().Writes
	for i := 0; i < 100; i++ {
		task.Progress = float64(i) / 100
		task.UpdatedAt = task.UpdatedAt.Add(time.Second)
		if err := s.Update(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	if m.Core.Metrics().Writes != before {
		t.Fatal("progress reserved core writer")
	}
	if m.Observations.Pending() != 1 {
		t.Fatal("progress was not coalesced")
	}
	stale := cloneTask(task)
	task.StageErrorMessage = "new business decision"
	if err := s.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	stale.StageErrorMessage = "old decision"
	if err := s.Update(ctx, stale); !errors.Is(err, ErrTaskRevisionConflict) {
		t.Fatalf("stale write accepted: %v", err)
	}
	var count int
	if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM audit_events"); err != nil || count != 2 {
		t.Fatalf("audit count=%d err=%v", count, err)
	}
	if err := m.Observations.FlushAll(ctx); err != nil {
		t.Fatal(err)
	}
	reloaded, err := storage.NewObservationStore(ctx, m.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	latest, ok := reloaded.Get(task.ID)
	if !ok || latest.Progress != task.Progress {
		t.Fatal("observation did not survive flush")
	}
}
func TestRelationalUnchangedColdDataDoesNotRewrite(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	// A normal state transition must not rewrite choices or the torrent attachment.
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`CREATE TRIGGER reject_file_delete BEFORE DELETE ON task_files BEGIN SELECT RAISE(ABORT,'file churn'); END;
 CREATE TRIGGER reject_attachment_update BEFORE UPDATE ON task_attachments BEGIN SELECT RAISE(ABORT,'attachment churn'); END;`)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	task.StageErrorMessage = "another decision"
	if err := s.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
}

func TestRelationalStatsMatchExistingSemantics(t *testing.T) {
	s, _ := relationalStore(t)
	ctx := context.Background()
	states := []struct {
		stage  TaskStage
		status TaskStageStatus
	}{
		{TaskStageDownloading, TaskStageStatusRunning}, {TaskStageDownloading, TaskStageStatusBlocked},
		{TaskStagePendingIngest, TaskStageStatusPending}, {TaskStageTransferring, TaskStageStatusRunning},
		{TaskStageScanning, TaskStageStatusBlocked}, {TaskStageCompleted, TaskStageStatusDone},
	}
	for i, state := range states {
		task := richRelationalTask()
		task.ID = string(rune('a' + i))
		task.Code = ""
		task.TorrentIdentityHash = ""
		task.TorrentIdentityMagnet = ""
		task.Stage = state.stage
		task.StageStatus = state.status
		if err := s.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	tasks, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.TaskStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := CalculateTaskStats(tasks); got != want {
		t.Fatalf("stats=%+v want=%+v", got, want)
	}
}
