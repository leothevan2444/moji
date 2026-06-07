package downloader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type fakeTracker struct {
	results []jackett.SearchResult
	err     error
}

func (f fakeTracker) Search(_ string, _ ...tracker.SearchOption) ([]jackett.SearchResult, error) {
	return f.results, f.err
}

type fakeTorrentAdder struct {
	options  qbittorrent.AddTorrentOptions
	torrents []qbittorrent.Torrent
	err      error
}

func (f *fakeTorrentAdder) AddNewTorrent(_ context.Context, opts qbittorrent.AddTorrentOptions) error {
	f.options = opts
	return f.err
}

func (f *fakeTorrentAdder) GetTorrentList(_ context.Context, _ *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return f.torrents, f.err
}

func TestDownloadMediaContextAddsBestTorrent(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "low seeders", Link: "https://example.test/low.torrent", Seeders: 1, Size: 100},
			{Title: "best", MagnetURI: "magnet:?xt=urn:btih:best", Seeders: 10, Size: 50},
		}},
		qbt,
		store,
		WithIDGenerator(func() string { return "task-test" }),
		WithClock(func() time.Time { return time.Unix(100, 0) }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{
		Query:    "SONE-000",
		SavePath: "/downloads/stash",
		Category: "moji",
		Tags:     "moji,stash",
	})
	if err != nil {
		t.Fatalf("DownloadMediaContext failed: %v", err)
	}

	if task.Status != TaskStatusAdded {
		t.Fatalf("expected task status %q, got %q", TaskStatusAdded, task.Status)
	}
	if task.Candidate.Title != "best" {
		t.Fatalf("expected best candidate, got %q", task.Candidate.Title)
	}
	if got := qbt.options.URLs; len(got) != 1 || got[0] != "magnet:?xt=urn:btih:best" {
		t.Fatalf("unexpected qBittorrent URLs: %v", got)
	}

	stored, err := store.Find(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if stored.Status != TaskStatusAdded {
		t.Fatalf("expected stored task status %q, got %q", TaskStatusAdded, stored.Status)
	}
}

func TestDownloadMediaContextRecordsAddFailure(t *testing.T) {
	qbtErr := errors.New("qbt rejected torrent")
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "candidate", Link: "https://example.test/file.torrent", Seeders: 1},
		}},
		&fakeTorrentAdder{err: qbtErr},
		NewMemoryTaskStore(),
		WithIDGenerator(func() string { return "task-failed" }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Query: "SONE-000"})
	if err == nil {
		t.Fatal("expected add torrent error")
	}
	if task == nil {
		t.Fatal("expected failed task to be returned")
	}
	if task.Status != TaskStatusFailed {
		t.Fatalf("expected task status %q, got %q", TaskStatusFailed, task.Status)
	}
	if task.Error != qbtErr.Error() {
		t.Fatalf("expected task error %q, got %q", qbtErr.Error(), task.Error)
	}
}

func TestAddTorrentContextCreatesPersistedTask(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithIDGenerator(func() string { return "task-manual" }),
		WithClock(func() time.Time { return time.Unix(100, 0) }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.AddTorrentContext(context.Background(), AddTorrentRequest{
		URL:      "magnet:?xt=urn:btih:manual",
		SavePath: "/downloads/stash",
		Category: "moji",
		Tags:     "manual",
	})
	if err != nil {
		t.Fatalf("AddTorrentContext failed: %v", err)
	}

	if task.ID != "task-manual" || task.Status != TaskStatusAdded {
		t.Fatalf("unexpected task: %+v", task)
	}
	if task.TorrentURL != "magnet:?xt=urn:btih:manual" || task.Candidate.MagnetURI != task.TorrentURL {
		t.Fatalf("unexpected torrent candidate: %+v", task.Candidate)
	}
	if task.Candidate.InfoHash != "manual" {
		t.Fatalf("expected info hash %q, got %q", "manual", task.Candidate.InfoHash)
	}
	if got := qbt.options.URLs; len(got) != 1 || got[0] != task.TorrentURL {
		t.Fatalf("unexpected qBittorrent URLs: %v", got)
	}

	stored, err := store.Find(context.Background(), "task-manual")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if stored.Status != TaskStatusAdded || stored.TorrentURL != task.TorrentURL {
		t.Fatalf("unexpected stored task: %+v", stored)
	}
}

func TestSyncProgressUpdatesTaskFromTorrentList(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:         "task-sync",
		Query:      "ABCD-123",
		Status:     TaskStatusAdded,
		TorrentURL: "magnet:?xt=urn:btih:sync",
		CreatedAt:  time.Unix(100, 0).UTC(),
		UpdatedAt:  time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	qbt := &fakeTorrentAdder{
		torrents: []qbittorrent.Torrent{
			{
				Hash:        "hash-sync",
				Name:        "ABCD-123",
				MagnetURI:   "magnet:?xt=urn:btih:sync",
				Progress:    0.5,
				State:       qbittorrent.TorrentStateDownloading,
				ContentPath: "/downloads/ABCD-123.mp4",
			},
		},
	}
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithClock(func() time.Time { return time.Unix(200, 0).UTC() }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	tasks, err := service.SyncProgress(context.Background())
	if err != nil {
		t.Fatalf("SyncProgress failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	task := tasks[0]
	if task.Status != TaskStatusDownloading {
		t.Fatalf("expected status %q, got %q", TaskStatusDownloading, task.Status)
	}
	if task.TorrentHash != "hash-sync" || task.Progress != 0.5 || task.QBittorrentState != string(qbittorrent.TorrentStateDownloading) {
		t.Fatalf("unexpected synced task: %+v", task)
	}
}

func TestSyncProgressMarksCompletedTask(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-complete",
		Query:       "ABCD-123",
		Status:      TaskStatusDownloading,
		TorrentHash: "hash-complete",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	qbt := &fakeTorrentAdder{
		torrents: []qbittorrent.Torrent{
			{
				Hash:         "hash-complete",
				Name:         "ABCD-123",
				Progress:     1,
				State:        qbittorrent.TorrentStateUploading,
				CompletionOn: 300,
			},
		},
	}
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithClock(func() time.Time { return time.Unix(400, 0).UTC() }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	tasks, err := service.SyncProgress(context.Background())
	if err != nil {
		t.Fatalf("SyncProgress failed: %v", err)
	}
	task := tasks[0]
	if task.Status != TaskStatusCompleted {
		t.Fatalf("expected status %q, got %q", TaskStatusCompleted, task.Status)
	}
	if task.CompletedAt == nil || !task.CompletedAt.Equal(time.Unix(300, 0).UTC()) {
		t.Fatalf("unexpected completed at: %v", task.CompletedAt)
	}
}
