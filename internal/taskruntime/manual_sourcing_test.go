package taskruntime

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/stashsync"
)

func createWaitingTask(t *testing.T, store TaskStore) *Task {
	t.Helper()
	now := time.Now().UTC()
	next := now.Add(6 * time.Hour)
	task := &Task{ID: "waiting", Source: TaskSourceSearch, Code: "OFJE-657", Stage: TaskStageSourcing, StageStatus: TaskStageStatusPending,
		SourcingLastCheckedAt: &now, SourcingNextCheckAt: &next, CreatedAt: now, UpdatedAt: now, SavePath: "/downloads", Category: "moji", Tags: "tracked"}
	if err := ensureSourcingRequest(task); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	return task
}

func TestWaitingTaskAcceptsManualMagnetOrUploadedTorrent(t *testing.T) {
	for _, mode := range []string{"magnet", "file"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryTaskStore()
			task := createWaitingTask(t, store)
			qbt := &fakeTorrentAdder{}
			s, _ := NewService(fakeTracker{}, qbt, store, WithMediaPreparation(func() stashsync.IntegrationConfig { return stashsync.IntegrationConfig{} }, nil))
			body := []byte(testTorrentFile("ofje00657", []string{"OFJE657.mp4"}))
			req := ResolveBlockedSourcingRequest{TorrentData: body}
			if mode == "magnet" {
				req = ResolveBlockedSourcingRequest{URL: "magnet:?xt=urn:btih:manual657&dn=OFJE657"}
			}
			resolved, err := s.ResolveBlockedSourcingTask(ctx, task.ID, req)
			if err != nil {
				t.Fatal(err)
			}
			stored, _ := store.Find(ctx, task.ID)
			if resolved.ID != task.ID || stored.Stage != TaskStageDownloading || stored.StageStatus != TaskStageStatusRunning || stored.SourcingNextCheckAt != nil || stored.SourcingCompletedAt == nil {
				t.Fatalf("manual resource did not advance original task: %+v", stored)
			}
			if *qbt.options.SavePath != "/downloads" || *qbt.options.Category != "moji" || *qbt.options.Tags != "tracked" {
				t.Fatal("task download defaults lost")
			}
			if _, err := s.ResolveBlockedSourcingTask(ctx, task.ID, req); !errors.Is(err, ErrSourcingUnavailable) {
				t.Fatalf("stale manual submit was not rejected: %v", err)
			}
			if mode == "file" {
				if len(qbt.options.URLs) != 0 || len(qbt.options.Torrents) != 1 || !bytes.Equal(qbt.options.Torrents[0].Data, body) || !bytes.Equal(stored.TorrentFileData, body) || qbt.options.Paused == nil || !*qbt.options.Paused {
					t.Fatal("verified file bytes were not uploaded and retained while stopped")
				}
				body[0] = 'x'
				if stored.TorrentFileData[0] != 'd' {
					t.Fatal("stored upload shares the caller's mutable buffer")
				}
			} else if len(qbt.options.URLs) != 1 {
				t.Fatal("magnet was not submitted")
			}
		})
	}
}

func TestBadManualResourcePreservesWaitingScheduleAndNeverSubmits(t *testing.T) {
	for _, req := range []ResolveBlockedSourcingRequest{
		{}, {URL: "magnet:?xt=urn:btih:wrong&dn=OFJE667"}, {TorrentData: []byte("<html>not a torrent</html>")},
		{TorrentData: []byte(testTorrentFile("OFJE657", []string{"OFJE667.mp4"}))},
		{URL: "magnet:?xt=urn:btih:both&dn=OFJE657", TorrentData: []byte("both")},
		{TorrentData: []byte{}}, {TorrentData: make([]byte, MaxTorrentBytes+1)},
	} {
		store := NewMemoryTaskStore()
		task := createWaitingTask(t, store)
		qbt := &fakeTorrentAdder{}
		s, _ := NewService(fakeTracker{}, qbt, store)
		if _, err := s.ResolveBlockedSourcingTask(context.Background(), task.ID, req); err == nil {
			t.Fatal("invalid manual resource was accepted")
		}
		stored, _ := store.Find(context.Background(), task.ID)
		if !waitingForResource(stored) || !stored.SourcingNextCheckAt.Equal(*task.SourcingNextCheckAt) || stored.StageErrorCode != "" || len(qbt.options.URLs) != 0 || len(qbt.options.Torrents) != 0 {
			t.Fatalf("invalid resource changed the tracking plan: %+v", stored)
		}
	}
}

func TestUploadedTorrentSurvivesRestartAndRetriesAsAFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "uploads.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	task := createWaitingTask(t, store)
	body := []byte(testTorrentFile("release", []string{"OFJE657.mp4"}))
	qbt := &fakeTorrentAdder{err: errors.New("downloader unavailable")}
	s, _ := NewService(fakeTracker{}, qbt, store)
	if _, err := s.ResolveBlockedSourcingTask(ctx, task.ID, ResolveBlockedSourcingRequest{TorrentData: body}); err == nil {
		t.Fatal("downloader failure was hidden")
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	qbt = &fakeTorrentAdder{}
	s, _ = NewService(fakeTracker{}, qbt, store)
	resolved, err := s.RetryTask(ctx, task.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Stage != TaskStageDownloading || resolved.StageStatus != TaskStageStatusRunning || len(qbt.options.URLs) != 0 || len(qbt.options.Torrents) != 1 || !bytes.Equal(qbt.options.Torrents[0].Data, body) {
		t.Fatal("retry lost the uploaded torrent or downgraded it to a magnet")
	}
}

func TestSchema14UpgradeRetainsResourceScheduleAndAddsTorrentStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	task := createWaitingTask(t, store)
	for _, query := range []string{"ALTER TABLE tasks DROP COLUMN torrent_file_data", "UPDATE task_store_meta SET value='14' WHERE key='schema_version'"} {
		if _, err := store.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	store.db.Close()
	store, err = NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	stored, err := store.Find(context.Background(), task.ID)
	if err != nil || !waitingForResource(stored) || !stored.SourcingNextCheckAt.Equal(*task.SourcingNextCheckAt) {
		t.Fatalf("upgrade damaged tracking: %+v %v", stored, err)
	}
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if _, err := s.ResolveBlockedSourcingTask(context.Background(), task.ID, ResolveBlockedSourcingRequest{TorrentData: []byte(testTorrentFile("OFJE657", []string{"OFJE657.mp4"}))}); err != nil {
		t.Fatal(err)
	}
}
