package taskruntime

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
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
	options      qbittorrent.AddTorrentOptions
	torrents     []qbittorrent.Torrent
	deleteHashes []string
	deleteFiles  bool
	deleteErr    error
	err          error
}

type fakeLibraryCodeChecker struct {
	codes map[string]bool
	err   error
}

func (f fakeLibraryCodeChecker) HasCode(_ context.Context, code string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.codes[normalizeCode(code)], nil
}

func (f *fakeTorrentAdder) AddNewTorrent(_ context.Context, opts qbittorrent.AddTorrentOptions) error {
	f.options = opts
	return f.err
}

func (f *fakeTorrentAdder) GetTorrentList(_ context.Context, _ *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return f.torrents, f.err
}

func (f *fakeTorrentAdder) DeleteTorrents(_ context.Context, hashes []string, deleteFiles bool) error {
	f.deleteHashes = append([]string(nil), hashes...)
	f.deleteFiles = deleteFiles
	return f.deleteErr
}

func TestDownloadMediaContextAddsBestTorrent(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "SONE-000 low seeders", Link: "https://example.test/low.torrent", Seeders: 1, Size: 100},
			{Title: "SONE-000 best", MagnetURI: "magnet:?xt=urn:btih:best", Seeders: 10, Size: 50},
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
		Code:     "SONE-000",
		SavePath: "/downloads/stash",
		Category: "moji",
		Tags:     "moji,stash",
	})
	if err != nil {
		t.Fatalf("DownloadMediaContext failed: %v", err)
	}

	if task.Stage != TaskStageDownloading || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("expected task stage DOWNLOADING/RUNNING, got %s/%s", task.Stage, task.StageStatus)
	}
	if task.Source != TaskSourceManual {
		t.Fatalf("expected task source %q, got %q", TaskSourceManual, task.Source)
	}
	if task.Candidate.Title != "SONE-000 best" {
		t.Fatalf("expected best candidate, got %q", task.Candidate.Title)
	}
	if got := qbt.options.URLs; len(got) != 1 || got[0] != "magnet:?xt=urn:btih:best" {
		t.Fatalf("unexpected qBittorrent URLs: %v", got)
	}

	stored, err := store.Find(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if stored.Stage != TaskStageDownloading || stored.StageStatus != TaskStageStatusRunning {
		t.Fatalf("expected stored task stage DOWNLOADING/RUNNING, got %s/%s", stored.Stage, stored.StageStatus)
	}
	if stored.Code != "SONE-000" {
		t.Fatalf("expected stored code %q, got %q", "SONE-000", stored.Code)
	}
}

func TestResolveBlockedSourcingTaskUsesSelectedTorrentOnExistingTask(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	blocked := &Task{
		ID: "task-blocked", Source: TaskSourceSearch, Code: "ABCD-123",
		Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked,
		StageErrorCode: TaskStageErrorNoCandidate, StageErrorMessage: "no candidate",
		SavePath: "/downloads", Category: "moji", Tags: "stash",
		CreatedAt: time.Unix(100, 0).UTC(), UpdatedAt: time.Unix(100, 0).UTC(),
	}
	if err := store.Create(context.Background(), blocked); err != nil {
		t.Fatalf("create blocked task: %v", err)
	}
	service, err := NewService(fakeTracker{}, qbt, store, WithClock(func() time.Time { return time.Unix(200, 0) }))
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.ResolveBlockedSourcingTask(context.Background(), blocked.ID, ResolveBlockedSourcingRequest{
		URL:   "magnet:?xt=urn:btih:selected123&dn=ABCD-123+manual",
		Title: "ABCD-123 selected", Tracker: "manual-indexer", InfoHash: "selected123",
		Size: 42, Seeders: 7, Peers: 3,
	})
	if err != nil {
		t.Fatalf("ResolveBlockedSourcingTask failed: %v", err)
	}
	if task.ID != blocked.ID || task.Stage != TaskStageDownloading || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("unexpected resolved task state: %+v", task)
	}
	if task.StageErrorCode != "" || task.StageErrorMessage != "" {
		t.Fatalf("expected stage error to be cleared: %+v", task)
	}
	if task.Candidate.Title != "ABCD-123 selected" || task.Candidate.Tracker != "manual-indexer" || task.Candidate.Seeders != 7 {
		t.Fatalf("unexpected selected candidate: %+v", task.Candidate)
	}
	if len(qbt.options.URLs) != 1 || qbt.options.URLs[0] != task.TorrentURL {
		t.Fatalf("unexpected qBittorrent options: %+v", qbt.options)
	}
	stored, err := store.Find(context.Background(), blocked.ID)
	if err != nil || stored.StageStatus != TaskStageStatusRunning || stored.TorrentURL != task.TorrentURL {
		t.Fatalf("resolved task was not persisted: task=%+v err=%v", stored, err)
	}
}

func TestResolveBlockedSourcingTaskRejectsOtherStages(t *testing.T) {
	store := NewMemoryTaskStore()
	task := &Task{ID: "task-running", Code: "ABCD-123", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	if _, err := service.ResolveBlockedSourcingTask(context.Background(), task.ID, ResolveBlockedSourcingRequest{URL: "magnet:?xt=urn:btih:test"}); err == nil {
		t.Fatal("expected non-blocked sourcing task to be rejected")
	}
}

func TestDownloadMediaContextRecordsAddFailure(t *testing.T) {
	qbtErr := errors.New("qbt rejected torrent")
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "SONE-000 candidate", Link: "https://example.test/file.torrent", Seeders: 1},
		}},
		&fakeTorrentAdder{err: qbtErr},
		NewMemoryTaskStore(),
		WithIDGenerator(func() string { return "task-failed" }),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Code: "SONE-000"})
	if err == nil {
		t.Fatal("expected add torrent error")
	}
	if task == nil {
		t.Fatal("expected failed task to be returned")
	}
	if task.Stage != TaskStageDownloading || task.StageStatus != TaskStageStatusBlocked {
		t.Fatalf("expected task stage DOWNLOADING/BLOCKED, got %s/%s", task.Stage, task.StageStatus)
	}
	if task.StageErrorMessage != qbtErr.Error() {
		t.Fatalf("expected task stage error %q, got %q", qbtErr.Error(), task.StageErrorMessage)
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
		URL:      "magnet:?xt=urn:btih:manual123&dn=SONE-000",
		SavePath: "/downloads/stash",
		Category: "moji",
		Tags:     "manual",
	})
	if err != nil {
		t.Fatalf("AddTorrentContext failed: %v", err)
	}

	if task.ID != "task-manual" || task.Stage != TaskStageDownloading || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("unexpected task: %+v", task)
	}
	if task.Source != TaskSourceManual {
		t.Fatalf("expected task source %q, got %q", TaskSourceManual, task.Source)
	}
	if task.TorrentURL != "magnet:?xt=urn:btih:manual123&dn=SONE-000" || task.Candidate.MagnetURI != task.TorrentURL {
		t.Fatalf("unexpected torrent candidate: %+v", task.Candidate)
	}
	if task.Candidate.InfoHash != "manual123" {
		t.Fatalf("expected info hash %q, got %q", "manual123", task.Candidate.InfoHash)
	}
	if task.Code != "SONE-000" {
		t.Fatalf("expected code %q, got %q", "SONE-000", task.Code)
	}
	if got := qbt.options.URLs; len(got) != 1 || got[0] != task.TorrentURL {
		t.Fatalf("unexpected qBittorrent URLs: %v", got)
	}

	stored, err := store.Find(context.Background(), "task-manual")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if stored.Stage != TaskStageDownloading || stored.StageStatus != TaskStageStatusRunning || stored.TorrentURL != task.TorrentURL {
		t.Fatalf("unexpected stored task: %+v", stored)
	}
}

func TestDownloadMediaContextRejectsDuplicateCodeTask(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "existing-task",
		Code:        "SONE-000",
		Stage:       TaskStageDownloading,
		StageStatus: TaskStageStatusRunning,
		CreatedAt:   time.Unix(50, 0).UTC(),
		UpdatedAt:   time.Unix(50, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "SONE-000 best release", MagnetURI: "magnet:?xt=urn:btih:newhash", Seeders: 10},
		}},
		&fakeTorrentAdder{},
		store,
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Code: "SONE-000"})
	if !errors.Is(err, ErrDuplicateCodeTask) {
		t.Fatalf("expected duplicate code error, got task=%+v err=%v", task, err)
	}
	if task != nil {
		t.Fatalf("expected no task to be created, got %+v", task)
	}
}

func TestDownloadMediaContextRejectsExistingStashLibraryCode(t *testing.T) {
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "SONE-000 best release", MagnetURI: "magnet:?xt=urn:btih:newhash", Seeders: 10},
		}},
		&fakeTorrentAdder{},
		NewMemoryTaskStore(),
		WithLibraryCodeChecker(fakeLibraryCodeChecker{
			codes: map[string]bool{"SONE-000": true},
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Code: "SONE-000"})
	if !errors.Is(err, ErrDuplicateLibraryCode) {
		t.Fatalf("expected duplicate library code error, got task=%+v err=%v", task, err)
	}
	if task != nil {
		t.Fatalf("expected no task to be created, got %+v", task)
	}
}

func TestAddTorrentContextRejectsDuplicateTorrentTask(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:                    "existing-task",
		Code:                  "SONE-000",
		Stage:                 TaskStageDownloading,
		StageStatus:           TaskStageStatusRunning,
		TorrentIdentityHash:   "MANUAL123",
		TorrentIdentityMagnet: "magnet:?xt=urn:btih:MANUAL123",
		CreatedAt:             time.Unix(50, 0).UTC(),
		UpdatedAt:             time.Unix(50, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	service, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.AddTorrentContext(context.Background(), AddTorrentRequest{
		URL: "magnet:?xt=urn:btih:manual123&dn=SONE-999&tr=https://tracker.example.test/announce",
	})
	if !errors.Is(err, ErrDuplicateTorrentTask) {
		t.Fatalf("expected duplicate torrent error, got task=%+v err=%v", task, err)
	}
	if task != nil {
		t.Fatalf("expected no task to be created, got %+v", task)
	}
}

func TestDownloadMediaContextRequiresCode(t *testing.T) {
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "plain title without number", MagnetURI: "magnet:?xt=urn:btih:newhash", Seeders: 10},
		}},
		&fakeTorrentAdder{},
		NewMemoryTaskStore(),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Code: "plain title"})
	if !errors.Is(err, ErrTaskCodeRequired) {
		t.Fatalf("expected code required error, got task=%+v err=%v", task, err)
	}
	if task != nil {
		t.Fatalf("expected no task to be created, got %+v", task)
	}
}

func TestAddTorrentContextParsesTorrentMetadataForCode(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	store := NewMemoryTaskStore()
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithIDGenerator(func() string { return "task-http" }),
		WithClock(func() time.Time { return time.Unix(100, 0) }),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(testTorrentFile("SONE-001", []string{"SONE-001.mp4"}))),
					Request:    req,
				}, nil
			}),
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.AddTorrentContext(context.Background(), AddTorrentRequest{
		URL: "https://example.test/release.torrent",
	})
	if err != nil {
		t.Fatalf("AddTorrentContext failed: %v", err)
	}
	if task.Code != "SONE-001" {
		t.Fatalf("expected code %q, got %q", "SONE-001", task.Code)
	}
	if task.Candidate.Title != "SONE-001" {
		t.Fatalf("expected torrent title %q, got %q", "SONE-001", task.Candidate.Title)
	}
	if task.TorrentIdentityHash == "" {
		t.Fatal("expected info hash to be populated from torrent metadata")
	}
	if got := qbt.options.URLs; len(got) != 1 || got[0] != "https://example.test/release.torrent" {
		t.Fatalf("unexpected qBittorrent URLs: %v", got)
	}
}

func TestParseTorrentMetadataIncludesPaths(t *testing.T) {
	metadata, err := parseTorrentMetadata([]byte(testTorrentFile("SONE-001", []string{"disc1/SONE-001.mp4", "disc1/sample.txt"})))
	if err != nil {
		t.Fatalf("parseTorrentMetadata failed: %v", err)
	}
	if len(metadata.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %+v", metadata.Paths)
	}
	if metadata.FilePath != "disc1/SONE-001.mp4" {
		t.Fatalf("unexpected first path: %+v", metadata)
	}
	inspection := inspectTorrentMetadata(metadata)
	if !inspection.SingleVideo {
		t.Fatalf("expected single video inspection, got %+v", inspection)
	}
	if len(inspection.VideoPaths) != 1 || inspection.VideoPaths[0] != "disc1/SONE-001.mp4" {
		t.Fatalf("unexpected video paths: %+v", inspection.VideoPaths)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testTorrentFile(name string, files []string) string {
	var builder strings.Builder
	builder.WriteString("d4:info")
	builder.WriteString("d")
	builder.WriteString(bencodeString("name"))
	builder.WriteString(bencodeString(name))
	builder.WriteString(bencodeString("files"))
	builder.WriteString("l")
	for _, file := range files {
		builder.WriteString("d")
		builder.WriteString(bencodeString("length"))
		builder.WriteString("i1e")
		builder.WriteString(bencodeString("path"))
		builder.WriteString("l")
		for _, segment := range strings.Split(file, "/") {
			builder.WriteString(bencodeString(segment))
		}
		builder.WriteString("e")
		builder.WriteString("e")
	}
	builder.WriteString("e")
	builder.WriteString("e")
	builder.WriteString("e")
	return builder.String()
}

func bencodeString(value string) string {
	return strings.Join([]string{strconv.Itoa(len(value)), ":", value}, "")
}

func TestDeleteTaskRemovesPersistedTask(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-delete",
		Code:        "ABCD-123",
		Stage:       TaskStageCompleted,
		StageStatus: TaskStageStatusDone,
		TorrentHash: "hash-delete",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	qbt := &fakeTorrentAdder{}
	service, err := NewService(fakeTracker{}, qbt, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DeleteTask(context.Background(), "task-delete")
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	if task.ID != "task-delete" {
		t.Fatalf("unexpected deleted task: %+v", task)
	}
	if len(qbt.deleteHashes) != 0 {
		t.Fatalf("expected KEEP_ONLY to skip qBittorrent delete, got %+v", qbt.deleteHashes)
	}
	if _, err := store.Find(context.Background(), "task-delete"); err == nil {
		t.Fatal("expected deleted task to be removed from store")
	}
}

func TestDeleteTaskRemovesQBittorrentTorrentWhenPolicyRequestsIt(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-delete-qbt",
		Code:        "ABCD-123",
		Stage:       TaskStageCompleted,
		StageStatus: TaskStageStatusDone,
		TorrentHash: "hash-delete-qbt",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	qbt := &fakeTorrentAdder{}
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithTaskDeletePolicyProvider(func() config.TaskDeletePolicy {
			return config.TaskDeletePolicyRemoveTorrent
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	if _, err := service.DeleteTask(context.Background(), "task-delete-qbt"); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	if len(qbt.deleteHashes) != 1 || qbt.deleteHashes[0] != "hash-delete-qbt" {
		t.Fatalf("expected qBittorrent delete for hash-delete-qbt, got %+v", qbt.deleteHashes)
	}
	if qbt.deleteFiles {
		t.Fatal("expected REMOVE_TORRENT policy to keep downloaded files")
	}
}

func TestDeleteTaskDeletesDownloadedFilesWhenPolicyRequestsIt(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-delete-files",
		Code:        "ABCD-123",
		Stage:       TaskStageCompleted,
		StageStatus: TaskStageStatusDone,
		TorrentHash: "hash-delete-files",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	qbt := &fakeTorrentAdder{}
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithTaskDeletePolicyProvider(func() config.TaskDeletePolicy {
			return config.TaskDeletePolicyRemoveTorrentAndFiles
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	if _, err := service.DeleteTask(context.Background(), "task-delete-files"); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	if !qbt.deleteFiles {
		t.Fatal("expected REMOVE_TORRENT_AND_FILES policy to delete downloaded files")
	}
}

func TestDeleteTaskKeepsPersistedTaskWhenQBittorrentDeleteFails(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-delete-fail",
		Code:        "ABCD-123",
		Stage:       TaskStageCompleted,
		StageStatus: TaskStageStatusDone,
		TorrentHash: "hash-delete-fail",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	qbt := &fakeTorrentAdder{deleteErr: errors.New("qbt delete failed")}
	service, err := NewService(
		fakeTracker{},
		qbt,
		store,
		WithTaskDeletePolicyProvider(func() config.TaskDeletePolicy {
			return config.TaskDeletePolicyRemoveTorrent
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DeleteTask(context.Background(), "task-delete-fail")
	if err == nil {
		t.Fatal("expected DeleteTask to fail when qBittorrent delete fails")
	}
	if task != nil {
		t.Fatalf("expected no deleted task on qBittorrent failure, got %+v", task)
	}
	if _, err := store.Find(context.Background(), "task-delete-fail"); err != nil {
		t.Fatalf("expected task to remain in store after qBittorrent failure: %v", err)
	}
}

func TestSyncProgressUpdatesTaskFromTorrentList(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-sync",
		Code:        "ABCD-123",
		Stage:       TaskStageDownloading,
		StageStatus: TaskStageStatusRunning,
		TorrentURL:  "magnet:?xt=urn:btih:sync",
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
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
	if task.Stage != TaskStageDownloading || task.StageStatus != TaskStageStatusRunning {
		t.Fatalf("expected stage DOWNLOADING/RUNNING, got %s/%s", task.Stage, task.StageStatus)
	}
	if task.TorrentHash != "hash-sync" || task.Progress != 0.5 || task.QBittorrentState != string(qbittorrent.TorrentStateDownloading) {
		t.Fatalf("unexpected synced task: %+v", task)
	}
}

func TestSyncProgressMarksCompletedTask(t *testing.T) {
	store := NewMemoryTaskStore()
	if err := store.Create(context.Background(), &Task{
		ID:          "task-complete",
		Code:        "ABCD-123",
		Stage:       TaskStageDownloading,
		StageStatus: TaskStageStatusRunning,
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
	if task.Stage != TaskStagePendingIngest || task.StageStatus != TaskStageStatusPending {
		t.Fatalf("expected stage PENDING_INGEST/PENDING, got %s/%s", task.Stage, task.StageStatus)
	}
	if task.DownloadCompletedAt == nil || !task.DownloadCompletedAt.Equal(time.Unix(300, 0).UTC()) {
		t.Fatalf("unexpected download completed at: %v", task.DownloadCompletedAt)
	}
}
