package taskruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func TestMonitoringKeepsCompletedStageAndRecoversDownloadErrors(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	for _, task := range []*Task{{ID: "complete", TorrentHash: "complete", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, IngestCompletedAt: &now, Progress: 1}, {ID: "active", TorrentHash: "active", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, Progress: .5}} {
		task.CreatedAt = now
		task.UpdatedAt = now
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	q := &fakeTorrentAdder{torrents: []qbittorrent.Torrent{{Hash: "complete", Progress: 1, State: qbittorrent.TorrentStateUploading}}}
	s, err := NewService(fakeTracker{}, q, store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SyncProgress(ctx); err != nil {
		t.Fatal(err)
	}
	completed, _ := store.Find(ctx, "complete")
	active, _ := store.Find(ctx, "active")
	if completed.Stage != TaskStageCompleted || completed.IngestCompletedAt == nil || completed.QBittorrentState != "uploading" {
		t.Fatal("completed task regressed")
	}
	if active.StageStatus != TaskStageStatusBlocked || active.StageErrorCode != "TORRENT_MISSING" {
		t.Fatal("missing torrent not blocked")
	}
	q.torrents = []qbittorrent.Torrent{{Hash: "active", Progress: .5, State: qbittorrent.TorrentStateError}}
	if _, err := s.SyncProgress(ctx); err != nil {
		t.Fatal(err)
	}
	active, _ = store.Find(ctx, "active")
	if active.StageErrorCode != "TORRENT_ERROR" {
		t.Fatal("qB error not exposed")
	}
	q.err = errors.New("offline")
	if _, err := s.SyncProgress(ctx); err == nil {
		t.Fatal("outage ignored")
	}
	active, _ = store.Find(ctx, "active")
	if active.StageErrorCode != "TORRENT_ERROR" {
		t.Fatal("outage marked missing")
	}
	q.err = nil
	q.torrents = []qbittorrent.Torrent{{Hash: "active", Progress: .7, State: qbittorrent.TorrentStateDownloading}}
	if _, err := s.SyncProgress(ctx); err != nil {
		t.Fatal(err)
	}
	active, _ = store.Find(ctx, "active")
	if active.StageStatus != TaskStageStatusRunning || active.StageErrorCode != "" || active.Progress != .7 {
		t.Fatal("download did not recover")
	}
	completed, _ = store.Find(ctx, "complete")
	if completed.Stage != TaskStageCompleted || completed.QBittorrentState != "missing" {
		t.Fatal("completed removal not monitored")
	}
}
func TestBoundTorrentNeverFallsBackToMatchingName(t *testing.T) {
	_, ok := matchTaskTorrent(&Task{TorrentHash: "original", Candidate: Candidate{Title: "ABCD-123"}}, []qbittorrent.Torrent{{Hash: "unrelated", Name: "ABCD-123"}})
	if ok {
		t.Fatal("bound torrent reattached by name")
	}
}

func TestMonitorObservationsUpdateAllTaskStatesWithoutAdvancingIngest(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	now := time.Now().UTC()
	for _, task := range []*Task{{ID: "complete", TorrentHash: "complete", Stage: TaskStageCompleted, StageStatus: TaskStageStatusDone, IngestCompletedAt: &now}, {ID: "download", TorrentHash: "download", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning}, {ID: "review", TorrentHash: "review", Stage: TaskStageDownloading, StageStatus: TaskStageStatusBlocked, StageErrorCode: TaskStageErrorFileReview}} {
		task.CreatedAt = now
		task.UpdatedAt = now
		store.Create(ctx, task)
	}
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ObserveTorrentStates(ctx, nil); err != nil {
		t.Fatal(err)
	}
	completed, _ := store.Find(ctx, "complete")
	active, _ := store.Find(ctx, "download")
	review, _ := store.Find(ctx, "review")
	if completed.Stage != TaskStageCompleted || completed.StageStatus != TaskStageStatusDone || completed.IngestCompletedAt == nil || completed.QBittorrentState != "missing" {
		t.Fatal("completed task changed stage")
	}
	if active.StageErrorCode != "TORRENT_MISSING" || review.StageErrorCode != TaskStageErrorFileReview {
		t.Fatal("monitor failed to preserve independent blockers")
	}
	if err := service.ObserveTorrentStates(ctx, []qbittorrent.Torrent{{Hash: "download", Progress: 1, State: qbittorrent.TorrentStateUploading}, {Hash: "complete", State: qbittorrent.TorrentStateMissingFiles}}); err != nil {
		t.Fatal(err)
	}
	completed, _ = store.Find(ctx, "complete")
	active, _ = store.Find(ctx, "download")
	if completed.StageStatus != TaskStageStatusDone || completed.QBittorrentState != "missingFiles" {
		t.Fatal("completed missing files became a task error")
	}
	if active.Progress != 1 || active.Stage != TaskStageDownloading || active.StageStatus != TaskStageStatusRunning || active.StageErrorCode != "" {
		t.Fatal("monitor must recover progress without starting ingest")
	}
}
