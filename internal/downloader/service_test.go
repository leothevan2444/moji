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
	options qbittorrent.AddTorrentOptions
	err     error
}

func (f *fakeTorrentAdder) AddNewTorrent(_ context.Context, opts qbittorrent.AddTorrentOptions) error {
	f.options = opts
	return f.err
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
