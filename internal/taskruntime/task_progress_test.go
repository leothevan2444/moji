package taskruntime

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func TestStoresTrackProgressAndIgnorePolling(t *testing.T) {
	for _, kind := range []string{"memory", "json", "sqlite"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			var store interface {
				Create(context.Context, *Task) error
				Update(context.Context, *Task) error
				Find(context.Context, string) (*Task, error)
			}
			switch kind {
			case "memory":
				store = NewMemoryTaskStore()
			case "json":
				s, err := NewJSONTaskStore(filepath.Join(t.TempDir(), "tasks.json"))
				if err != nil {
					t.Fatal(err)
				}
				store = s
			case "sqlite":
				s, err := NewSQLiteTaskStore(filepath.Join(t.TempDir(), "tasks.db"))
				if err != nil {
					t.Fatal(err)
				}
				defer s.db.Close()
				store = s
			}
			clock := time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC)
			task := &Task{ID: "progress", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: clock, UpdatedAt: clock}
			if err := store.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			if task.LastProgressAt == nil || !task.LastProgressAt.Equal(clock) {
				t.Fatal("creation was not recorded")
			}
			steps := []struct {
				name    string
				mutate  func(*Task)
				changed bool
			}{
				{"unchanged download poll", func(*Task) {}, false},
				{"download progress", func(x *Task) { x.Progress = .2 }, true},
				{"downloader state", func(x *Task) { x.QBittorrentState = "stalledDL" }, true},
				{"unchanged stalled poll", func(*Task) {}, false},
				{"stage", func(x *Task) { setTaskStage(x, TaskStageScanning, TaskStageStatusRunning) }, true},
				{"normalize initial scan", func(x *Task) { x.StashIngestPhase = StashIngestScan }, false},
				{"ingest phase", func(x *Task) { x.StashIngestPhase = StashIngestGenerate }, true},
				{"unchanged job poll", func(*Task) {}, false},
				{"blocked", func(x *Task) {
					setTaskStage(x, TaskStageScanning, TaskStageStatusBlocked)
					x.StageErrorCode = "FAILED"
					x.StageErrorMessage = "unavailable"
				}, true},
				{"same error again", func(*Task) {}, false},
				{"different failure", func(x *Task) { x.StageErrorMessage = "missing scene" }, true},
				{"retry", func(x *Task) { setTaskStage(x, TaskStageScanning, TaskStageStatusRunning); clearTaskStageError(x) }, true},
				{"complete", func(x *Task) { setTaskStage(x, TaskStageCompleted, TaskStageStatusDone) }, true},
				{"metadata refresh", func(x *Task) { x.TorrentName = "metadata" }, false},
			}
			expected := clock
			for _, step := range steps {
				clock = clock.Add(time.Minute)
				step.mutate(task)
				task.UpdatedAt = clock
				if err := store.Update(ctx, task); err != nil {
					t.Fatal(err)
				}
				if step.changed {
					expected = clock
				}
				if task.LastProgressAt == nil || !task.LastProgressAt.Equal(expected) {
					t.Fatalf("%s: response timestamp %v expected %v", step.name, task.LastProgressAt, expected)
				}
				stored, err := store.Find(ctx, task.ID)
				if err != nil || stored.LastProgressAt == nil || !stored.LastProgressAt.Equal(expected) || !stored.UpdatedAt.Equal(clock) {
					t.Fatalf("%s: persisted task %+v %v", step.name, stored, err)
				}
				copy := cloneTask(stored)
				*copy.LastProgressAt = clock.Add(time.Hour)
				if !stored.LastProgressAt.Equal(expected) {
					t.Fatal("progress timestamp shared across clones")
				}
			}
			switch s := store.(type) {
			case *JSONTaskStore:
				reopened, err := NewJSONTaskStore(s.path)
				if err != nil {
					t.Fatal(err)
				}
				store = reopened
			case *SQLiteTaskStore:
				var path string
				if err := s.db.Get(&path, `SELECT file FROM pragma_database_list WHERE name='main'`); err != nil {
					t.Fatal(err)
				}
				reopened, err := NewSQLiteTaskStore(path)
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.db.Close()
				store = reopened
			}
			restored, err := store.Find(ctx, task.ID)
			if err != nil || restored.LastProgressAt == nil || !restored.LastProgressAt.Equal(expected) {
				t.Fatalf("progress timestamp lost on restart: %+v %v", restored, err)
			}
		})
	}
}

func TestDownloadPollingDoesNotAdvanceProgressClock(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 19, 2, 0, 0, 0, time.UTC)
	store := NewMemoryTaskStore()
	task := &Task{ID: "sync-progress", Code: "ABCD-123", TorrentHash: "hash", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: clock, UpdatedAt: clock}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	qbt := &fakeTorrentAdder{torrents: []qbittorrent.Torrent{{Hash: "hash", Name: "ABCD-123", Progress: .5, State: qbittorrent.TorrentStateDownloading}}}
	service, err := NewService(fakeTracker{}, qbt, store, WithClock(func() time.Time { return clock }))
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(time.Minute)
	first, err := service.SyncProgress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	progressAt := *first[0].LastProgressAt
	clock = clock.Add(time.Minute)
	second, err := service.SyncProgress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !second[0].UpdatedAt.Equal(clock) || !second[0].LastProgressAt.Equal(progressAt) {
		t.Fatal("unchanged downloader poll changed progress clock")
	}
	qbt.torrents[0].Progress = .6
	clock = clock.Add(time.Minute)
	third, err := service.SyncProgress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !third[0].LastProgressAt.Equal(clock) {
		t.Fatal("new downloader progress was not recorded")
	}
}

func TestSQLiteV11ProgressUpgradePreservesTasksAndHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC)
	if err := store.Create(ctx, &Task{ID: "history", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`ALTER TABLE tasks DROP COLUMN last_progress_at`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE task_store_meta SET value='11' WHERE key='schema_version'`); err != nil {
		t.Fatal(err)
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	task, err := store.Find(ctx, "history")
	if err != nil || task.LastProgressAt != nil {
		t.Fatalf("historical timestamp was fabricated: %+v %v", task, err)
	}
	var count int
	if err := store.db.Get(&count, `SELECT COUNT(*) FROM task_events WHERE task_id='history'`); err != nil || count != 1 {
		t.Fatalf("history lost: %d %v", count, err)
	}
	task.UpdatedAt = now.Add(time.Minute)
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if task.LastProgressAt != nil {
		t.Fatal("polling fabricated a historical progress time")
	}
	task.Progress = .5
	task.UpdatedAt = now.Add(2 * time.Minute)
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if task.LastProgressAt == nil || !task.LastProgressAt.Equal(task.UpdatedAt) {
		t.Fatal("new progress after migration was not recorded")
	}
}

func TestIngestPollingOnlyAdvancesProgressOnPhaseChanges(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 19, 3, 0, 0, 0, time.UTC)
	store := NewMemoryTaskStore()
	task := &Task{ID: "ingest-progress", Code: "ABCD-123", Stage: TaskStageScanning, StageStatus: TaskStageStatusRunning, StashIngestPhase: StashIngestScan, StashScanJobID: "scan-1", StashScanPath: "/library/ABCD-123.mp4", CreatedAt: clock, UpdatedAt: clock}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	initial := clock
	scanner := readyIngestScanner()
	scanner.jobs["scan-1"].Status = "RUNNING"
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithClock(func() time.Time { return clock }))
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(time.Minute)
	first := advanceIngest(t, service, scanner)
	if !first.UpdatedAt.Equal(clock) || !first.LastProgressAt.Equal(initial) {
		t.Fatal("unchanged Stash polling refreshed progress time")
	}
	clock = clock.Add(time.Minute)
	scanner.jobs["scan-1"].Status = "FINISHED"
	second := advanceIngest(t, service, scanner)
	if second.StashIngestPhase != StashIngestGenerate || !second.LastProgressAt.Equal(clock) {
		t.Fatal("phase transition did not refresh progress time")
	}
}
