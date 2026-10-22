package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type restartScanner struct {
	fakeIngestScanner
	removeErr   error
	removedPath string
	removedIDs  []string
}

func (s *restartScanner) RemoveIngestScenes(_ context.Context, path string, ids []string) error {
	s.removedPath, s.removedIDs = path, append([]string(nil), ids...)
	return s.removeErr
}

func restartFixture(t *testing.T) (*Service, *SQLiteTaskStore, *fakeTorrentAdder, *Task, string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	source, library := filepath.Join(root, "downloads"), filepath.Join(root, "library")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SNOS-184.mp4"), []byte("wrong short video"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "sample.mp4"), []byte("discarded"), 0600); err != nil {
		t.Fatal(err)
	}
	plan := mediafiles.Classify("SNOS-184", []mediafiles.File{{Index: 0, Path: "SNOS-184.mp4", Size: 17}, {Index: 1, Path: "sample.mp4", Size: 9}}, false)
	result, err := (mediafiles.FFmpeg{}).Prepare(ctx, mediafiles.Request{TaskID: "restart", Code: "SNOS-184", SourceRoot: source, TargetRoot: library, Action: "COPY", Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteTaskStore(filepath.Join(root, "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.db.Close() })
	qbt := &fakeTorrentAdder{}
	s, err := NewService(fakeTracker{}, qbt, store, WithTaskDeletePolicyProvider(func() config.TaskDeletePolicy { return config.TaskDeletePolicyKeepOnly }))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	task := &Task{ID: "restart", Code: "SNOS-184", Source: TaskSourceSubscription,
		Stage: TaskStageScanning, StageStatus: TaskStageStatusBlocked, StageErrorCode: TaskStageErrorIngestVerify,
		TorrentHash: "old-hash", TorrentIdentityHash: "OLD-HASH", TorrentURL: "magnet:old", Candidate: Candidate{Title: "old"},
		Progress: 1, DownloadCompletedAt: &now, SourcingCompletedAt: &now, TorrentFileData: []byte("old torrent"),
		MojiTransferPath: result.Directory, MojiSourcePath: source, StashScanPath: "/stash/SNOS-184", StashSceneIDsJSON: `["4525"]`,
		StashIngestPhase: StashIngestIdentify, StashScanJobID: "scan", StashGenerateJobID: "generate",
		SavePath: "/downloads", Category: "moji", Tags: "tracked", CreatedAt: now, UpdatedAt: now}
	task.setFilePlan(plan)
	if err := ensureSourcingRequest(task); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	return s, store, qbt, task, source
}

func TestRestartSourcingDeletesArtifactsAndReusesOriginalTask(t *testing.T) {
	ctx := context.Background()
	s, store, qbt, old, source := restartFixture(t)
	scanner := &restartScanner{}
	next, err := s.RestartTaskSourcing(ctx, old.ID, scanner)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID != old.ID || next.Source != old.Source || !next.CreatedAt.Equal(old.CreatedAt) || next.SavePath != old.SavePath || next.Category != old.Category || next.Tags != old.Tags {
		t.Fatal("restart lost task identity or download defaults")
	}
	if next.Stage != TaskStageSourcing || next.StageStatus != TaskStageStatusBlocked || next.StageErrorCode != TaskStageErrorManualSource {
		t.Fatalf("not waiting for manual selection: %+v", next)
	}
	if next.Progress != 0 || next.TorrentHash != "" || next.TorrentURL != "" || next.Candidate.Title != "" || next.FilePlanJSON != "" || next.TorrentFileData != nil || next.MojiTransferPath != "" || next.StashScanJobID != "" || next.StashSceneIDsJSON != "" || next.DownloadCompletedAt != nil || next.SourcingCompletedAt != nil || next.IngestCompletedAt != nil || next.SourcingNextCheckAt != nil || next.RestartStateJSON != "" {
		t.Fatalf("stale checkpoint retained: %+v", next)
	}
	if len(qbt.deleteHashes) != 1 || qbt.deleteHashes[0] != "old-hash" || !qbt.deleteFiles {
		t.Fatal("restart must delete downloaded files even when ordinary task deletion keeps them")
	}
	if _, err := os.Stat(filepath.Join(source, "SNOS-184.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("download source was retained", err)
	}
	if _, err := os.Stat(filepath.Join(source, "sample.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("discarded download was retained")
	}
	if _, err := os.Stat(old.MojiTransferPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old delivery still occupies target")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(old.MojiTransferPath), ".moji-restarts")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("restart created a backup")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(old.MojiTransferPath), ".moji-receipts", old.ID+".json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old delivery receipt retained")
	}
	if scanner.removedPath != old.StashScanPath || len(scanner.removedIDs) != 1 || scanner.removedIDs[0] != "4525" {
		t.Fatal("incorrect Stash cleanup scope")
	}
	if err := s.processQueuedSourcing(ctx, old.ID); err != nil {
		t.Fatal(err)
	}
	stored, _ := store.Find(ctx, old.ID)
	if stored.StageErrorCode != TaskStageErrorManualSource {
		t.Fatal("worker auto-selected an old resource")
	}
	if _, err := s.RetryTask(ctx, old.ID, scanner); !errors.Is(err, ErrSourcingUnavailable) {
		t.Fatal("retry bypassed manual torrent selection")
	}
	batch, err := s.RetryTasks(ctx, []string{old.ID}, scanner)
	if err != nil || batch.Summary.SkippedCount != 1 {
		t.Fatal("batch retry bypassed manual torrent selection")
	}
	// Duplicate clicks are harmless while waiting for a selection.
	if _, err := s.RestartTaskSourcing(ctx, old.ID, scanner); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.ResolveBlockedSourcingTask(ctx, old.ID, ResolveBlockedSourcingRequest{URL: "magnet:?xt=urn:btih:new184&dn=SNOS184"})
	if err != nil || resolved.Stage != TaskStageDownloading || resolved.ID != old.ID {
		t.Fatalf("new selection did not advance original task: %+v %v", resolved, err)
	}
}

func TestRestartCleanupResumesAfterSQLiteReload(t *testing.T) {
	ctx := context.Background()
	s, store, _, old, source := restartFixture(t)
	scanner := &restartScanner{removeErr: errors.New("Stash offline")}
	if _, err := s.RestartTaskSourcing(ctx, old.ID, scanner); !errors.Is(err, ErrRestartCleanup) {
		t.Fatalf("expected cleanup failure: %v", err)
	}
	pending, _ := store.Find(ctx, old.ID)
	if pending.RestartStateJSON == "" || pending.StageErrorCode != TaskStageErrorRestartPending {
		t.Fatal("cleanup intent not persisted")
	}
	if _, err := os.Stat(filepath.Join(source, "SNOS-184.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("download should already have been deleted")
	}
	if _, err := os.Stat(old.MojiTransferPath); err != nil {
		t.Fatal("failed Stash cleanup should leave delivery for retry")
	}
	if _, err := s.TriggerTaskStashScan(ctx, old.ID, scanner); !errors.Is(err, ErrRestartCleanup) {
		t.Fatal("ingest raced restart cleanup")
	}
	if _, err := s.DeleteTask(ctx, old.ID); !errors.Is(err, ErrRestartCleanup) {
		t.Fatal("deletion lost restart journal")
	}
	path := filepath.Join(filepath.Dir(filepath.Dir(old.MojiTransferPath)), "tasks.db")
	store.db.Close()
	reloaded, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.db.Close()
	s, _ = NewService(fakeTracker{}, &fakeTorrentAdder{}, reloaded)
	scanner.removeErr = nil
	// Source files are already gone. Retry must finish Stash and delivery cleanup.
	next, err := s.RetryTask(ctx, old.ID, scanner)
	if err != nil || next.StageErrorCode != TaskStageErrorManualSource || next.RestartStateJSON != "" {
		t.Fatalf("restart did not resume: %+v %v", next, err)
	}
	if _, err := os.Stat(old.MojiTransferPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("delivery retained after cleanup retry")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(old.MojiTransferPath), ".moji-restarts")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("retry created a backup")
	}
}

func TestRestartRejectsCompletedActiveAndModifiedDelivery(t *testing.T) {
	for _, mode := range []string{"completed", "processing", "modified", "foreign"} {
		t.Run(mode, func(t *testing.T) {
			s, store, qbt, task, _ := restartFixture(t)
			switch mode {
			case "completed":
				task.Stage = TaskStageCompleted
			case "processing":
				task.Stage = TaskStageTransferring
				task.StageStatus = TaskStageStatusRunning
			case "modified":
				if err := os.WriteFile(filepath.Join(task.MojiTransferPath, "SNOS-184.mp4"), []byte("user changed video"), 0600); err != nil {
					t.Fatal(err)
				}
			case "foreign":
				if err := os.Remove(filepath.Join(filepath.Dir(task.MojiTransferPath), ".moji-receipts", task.ID+".json")); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.Update(context.Background(), task); err != nil {
				t.Fatal(err)
			}
			scanner := &restartScanner{}
			if _, err := s.RestartTaskSourcing(context.Background(), task.ID, scanner); err == nil {
				t.Fatal("unsafe restart accepted")
			}
			if scanner.removedPath != "" {
				t.Fatal("unsafe restart deleted Stash scenes")
			}
			if _, err := os.Stat(task.MojiTransferPath); err != nil {
				t.Fatal("existing delivery lost")
			}
			if len(qbt.deleteHashes) != 0 {
				t.Fatal("rejected restart touched downloader")
			}
		})
	}
}

func TestRestartSchemaUpgradePreservesVersion15TasksAndHistory(t *testing.T) {
	ctx := context.Background()
	_, store, _, old, _ := restartFixture(t)
	path := filepath.Join(filepath.Dir(filepath.Dir(old.MojiTransferPath)), "tasks.db")
	if _, err := store.db.Exec("ALTER TABLE tasks DROP COLUMN restart_state_json"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("UPDATE task_store_meta SET value='15' WHERE key='schema_version'"); err != nil {
		t.Fatal(err)
	}
	store.db.Close()
	upgraded, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.db.Close()
	got, err := upgraded.Find(ctx, old.ID)
	if err != nil || got.Code != old.Code || got.MojiTransferPath != old.MojiTransferPath || got.RestartStateJSON != "" {
		t.Fatalf("schema upgrade lost task: %+v %v", got, err)
	}
	var events int
	if err := upgraded.db.Get(&events, "SELECT COUNT(*) FROM task_events WHERE task_id=?", old.ID); err != nil || events == 0 {
		t.Fatal("schema upgrade lost history", err)
	}
	got.RestartStateJSON = "{\"Version\":1}"
	if err := upgraded.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	got, err = upgraded.Find(ctx, old.ID)
	if err != nil || got.RestartStateJSON == "" {
		t.Fatal("new cleanup checkpoint did not persist", err)
	}
}

func TestRestartBeforeFirstDownloadPollRemovesValidatedTorrentIdentity(t *testing.T) {
	ctx := context.Background()
	qbt, store := &fakeTorrentAdder{}, NewMemoryTaskStore()
	now := time.Now().UTC()
	task := &Task{ID: "new-download", Code: "SNOS-184", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning,
		TorrentIdentityHash: "VALIDATED-HASH", TorrentURL: "magnet:?xt=urn:btih:VALIDATED-HASH", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	s, _ := NewService(fakeTracker{}, qbt, store)
	if _, err := s.RestartTaskSourcing(ctx, task.ID, nil); err != nil {
		t.Fatal(err)
	}
	if len(qbt.deleteHashes) != 1 || qbt.deleteHashes[0] != "VALIDATED-HASH" || !qbt.deleteFiles {
		t.Fatal("unpolled torrent and downloaded files must be removed")
	}
}

func TestRestartRecoversDeliveryPublishedBeforeCheckpoint(t *testing.T) {
	ctx := context.Background()
	_, store, qbt, task, _ := restartFixture(t)
	target := task.MojiTransferPath
	cfg := stashsync.IntegrationConfig{Library: stashsync.LibraryPathConfig{MojiRoot: filepath.Dir(target), StashRoot: "/stash"}}
	task.MojiTransferPath, task.StashScanPath, task.StashSceneIDsJSON = "", "", ""
	task.Stage, task.StageStatus = TaskStageTransferring, TaskStageStatusBlocked
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	s, _ := NewService(fakeTracker{}, qbt, store, WithMediaPreparation(func() stashsync.IntegrationConfig { return cfg }, nil))
	if _, err := s.RestartTaskSourcing(ctx, task.ID, &restartScanner{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("published output was abandoned")
	}
	if _, err := os.Stat(filepath.Join(cfg.Library.MojiRoot, ".moji-receipts", task.ID+".json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("uncheckpointed delivery receipt was not deleted")
	}
}

type delayedRemovalTorrent struct {
	fakeTorrentAdder
	checks int
}

func (q *delayedRemovalTorrent) GetTorrentList(context.Context, *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	q.checks++
	if q.checks == 1 {
		return []qbittorrent.Torrent{{Hash: "old-hash"}}, nil
	}
	return nil, nil
}

func TestRestartWaitsForDownloaderRemovalBeforeResetting(t *testing.T) {
	ctx := context.Background()
	_, store, _, task, _ := restartFixture(t)
	qbt := &delayedRemovalTorrent{}
	s, _ := NewService(fakeTracker{}, qbt, store)
	next, err := s.RestartTaskSourcing(ctx, task.ID, &restartScanner{})
	if err != nil {
		t.Fatal(err)
	}
	if qbt.checks != 2 || !qbt.deleteFiles || next.StageErrorCode != TaskStageErrorManualSource {
		t.Fatal("task reset before downloader removal completed")
	}
}

func TestRestartDownloaderFailurePreservesFilesAndCanResume(t *testing.T) {
	ctx := context.Background()
	s, store, qbt, old, source := restartFixture(t)
	qbt.deleteErr = errors.New("qBittorrent offline")
	scanner := &restartScanner{}
	if _, err := s.RestartTaskSourcing(ctx, old.ID, scanner); !errors.Is(err, ErrRestartCleanup) {
		t.Fatal("expected downloader failure", err)
	}
	for _, path := range []string{filepath.Join(source, "SNOS-184.mp4"), filepath.Join(old.MojiTransferPath, "SNOS-184.mp4")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("files deleted before downloader accepted removal")
		}
	}
	if scanner.removedPath != "" {
		t.Fatal("Stash scene deleted before downloader accepted removal")
	}
	qbt.deleteErr = nil
	if _, err := s.RetryTask(ctx, old.ID, scanner); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Find(ctx, old.ID)
	if got.StageErrorCode != TaskStageErrorManualSource {
		t.Fatal("cleanup did not resume")
	}
}

func TestRestartResumesAfterDeliveryFilesAndReceiptWereDeleted(t *testing.T) {
	ctx := context.Background()
	s, store, qbt, task, source := restartFixture(t)
	delivery, err := mediafiles.PlanDeliveryRemoval(ctx, task.ID, task.Code, task.MojiTransferPath)
	if err != nil {
		t.Fatal(err)
	}
	downloads, err := s.planRestartDownloads(ctx, task)
	if err != nil {
		t.Fatal(err)
	}
	state := restartState{Version: 1, Delivery: delivery, Downloads: downloads, DownloaderRemoved: true}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	task.RestartStateJSON = string(data)
	blockTask(task, TaskStageErrorRestartPending, "interrupted cleanup", time.Now().UTC())
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	// Simulate a crash after the filesystem side effects but before task reset.
	if err := mediafiles.RemoveDownloads(ctx, *downloads); err != nil {
		t.Fatal(err)
	}
	if err := mediafiles.RemoveDelivery(ctx, *delivery); err != nil {
		t.Fatal(err)
	}
	next, err := s.RetryTask(ctx, task.ID, &restartScanner{})
	if err != nil || next.StageErrorCode != TaskStageErrorManualSource {
		t.Fatalf("deleted files prevented recovery: %+v %v", next, err)
	}
	if len(qbt.deleteHashes) != 0 {
		t.Fatal("completed downloader cleanup ran twice")
	}
	if _, err := os.Stat(filepath.Join(source, "SNOS-184.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old download recreated")
	}
}

func TestRestartUsesDownloaderFileListBeforeFileSelection(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	save := filepath.Join(root, "subfolder")
	if err := os.MkdirAll(filepath.Join(save, "torrent"), 0700); err != nil {
		t.Fatal(err)
	}
	movie := filepath.Join(save, "torrent", "SNOS-184.mp4")
	unrelated := filepath.Join(root, "another-task.mp4")
	for _, path := range []string{movie, unrelated} {
		if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	qbt := &fileTorrent{files: []qbittorrent.TorrentContentFile{{Index: 0, Name: "torrent/SNOS-184.mp4", Size: 4}}}
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	task := &Task{ID: "unconfirmed", Code: "SNOS-184", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning,
		TorrentHash: "old-hash", SavePath: "/q/downloads/subfolder", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	cfg := stashsync.IntegrationConfig{Downloads: stashsync.DownloadsPathConfig{QBRoot: "/q/downloads", MojiRoot: root}}
	s, _ := NewService(fakeTracker{}, qbt, store, WithMediaPreparation(func() stashsync.IntegrationConfig { return cfg }, nil))
	if _, err := s.RestartTaskSourcing(ctx, task.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(movie); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unconfirmed downloaded file retained")
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatal("another task's download was removed")
	}
	if !qbt.deleteFiles {
		t.Fatal("downloader was not asked to remove files")
	}
}
