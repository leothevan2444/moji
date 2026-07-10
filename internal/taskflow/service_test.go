package taskflow

import (
	"context"
	"errors"
	"testing"

	"github.com/leothevan2444/moji/internal/downloader"
)

type fakeDownloader struct {
	addRequest      downloader.AddTorrentRequest
	downloadRequest downloader.DownloadRequest
	addTask         *downloader.Task
	downloadTask    *downloader.Task
	err             error
}

type fakeDiscoveredSceneResolver struct {
	resolved ResolvedScene
	err      error
	sceneID  string
	endpoint string
}

func (f *fakeDownloader) AddTorrentContext(_ context.Context, req downloader.AddTorrentRequest) (*downloader.Task, error) {
	f.addRequest = req
	if f.err != nil {
		return nil, f.err
	}
	return f.addTask, nil
}

func (f *fakeDownloader) DownloadMediaContext(_ context.Context, req downloader.DownloadRequest) (*downloader.Task, error) {
	f.downloadRequest = req
	if f.err != nil {
		return nil, f.err
	}
	return f.downloadTask, nil
}

func (f *fakeDiscoveredSceneResolver) ResolveDiscoveredScene(_ context.Context, sceneID string, stashBoxEndpoint string) (ResolvedScene, error) {
	f.sceneID = sceneID
	f.endpoint = stashBoxEndpoint
	if f.err != nil {
		return ResolvedScene{}, f.err
	}
	return f.resolved, nil
}

func TestCreateFromManualTorrentUsesManualSource(t *testing.T) {
	dl := &fakeDownloader{addTask: &downloader.Task{ID: "task-1"}}
	service := NewService(dl)
	paused := true

	task, err := service.CreateFromManualTorrent(context.Background(), CreateFromManualTorrentInput{
		URL:      "magnet:?xt=urn:btih:test",
		Paused:   &paused,
		SavePath: "/downloads",
		Category: "moji",
		Tags:     "a,b",
	})
	if err != nil {
		t.Fatalf("CreateFromManualTorrent failed: %v", err)
	}
	if task == nil || task.ID != "task-1" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if dl.addRequest.Source != downloader.TaskSourceManual {
		t.Fatalf("unexpected source: %s", dl.addRequest.Source)
	}
	if dl.addRequest.URL != "magnet:?xt=urn:btih:test" {
		t.Fatalf("unexpected url: %q", dl.addRequest.URL)
	}
	if dl.addRequest.Paused == nil || !*dl.addRequest.Paused || dl.addRequest.SavePath != "/downloads" || dl.addRequest.Category != "moji" || dl.addRequest.Tags != "a,b" {
		t.Fatalf("unexpected request: %+v", dl.addRequest)
	}
}

func TestCreateFromSearchCodeUsesManualSource(t *testing.T) {
	dl := &fakeDownloader{downloadTask: &downloader.Task{ID: "task-2"}}
	service := NewService(dl)
	paused := true

	task, err := service.CreateFromSearchCode(context.Background(), CreateFromSearchCodeInput{
		Code:       "ABCD-123",
		Trackers:   []string{"t1"},
		Categories: []int{1},
		Limit:      5,
		Paused:     &paused,
		SavePath:   "/downloads",
		Category:   "moji",
		Tags:       "tag",
	})
	if err != nil {
		t.Fatalf("CreateFromSearchCode failed: %v", err)
	}
	if task == nil || task.ID != "task-2" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if dl.downloadRequest.Source != downloader.TaskSourceManual {
		t.Fatalf("unexpected source: %s", dl.downloadRequest.Source)
	}
	if dl.downloadRequest.Code != "ABCD-123" || dl.downloadRequest.Limit != 5 {
		t.Fatalf("unexpected request: %+v", dl.downloadRequest)
	}
}

func TestCreateFromDiscoveredSceneUsesSearchSource(t *testing.T) {
	dl := &fakeDownloader{downloadTask: &downloader.Task{ID: "task-3"}}
	service := NewService(dl)

	task, err := service.CreateFromDiscoveredScene(context.Background(), CreateFromDiscoveredSceneInput{
		Code:  "ABCD-123",
		Title: "ignored",
	})
	if err != nil {
		t.Fatalf("CreateFromDiscoveredScene failed: %v", err)
	}
	if task == nil || task.ID != "task-3" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if dl.downloadRequest.Source != downloader.TaskSourceSearch {
		t.Fatalf("unexpected source: %s", dl.downloadRequest.Source)
	}
	if dl.downloadRequest.Code != "ABCD-123" {
		t.Fatalf("unexpected request: %+v", dl.downloadRequest)
	}
}

func TestCreateFromDiscoveredSceneRefUsesResolver(t *testing.T) {
	dl := &fakeDownloader{downloadTask: &downloader.Task{ID: "task-5"}}
	resolver := &fakeDiscoveredSceneResolver{
		resolved: ResolvedScene{Code: "ABCD-321", Title: "ignored"},
	}
	service := NewService(dl)
	service.SetDiscoveredSceneResolver(resolver)

	task, err := service.CreateFromDiscoveredSceneRef(context.Background(), CreateFromDiscoveredSceneRefInput{
		SceneID:          "scene-1",
		StashBoxEndpoint: "https://box.example/graphql",
	})
	if err != nil {
		t.Fatalf("CreateFromDiscoveredSceneRef failed: %v", err)
	}
	if task == nil || task.ID != "task-5" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if resolver.sceneID != "scene-1" || resolver.endpoint != "https://box.example/graphql" {
		t.Fatalf("unexpected resolver call: %+v", resolver)
	}
	if dl.downloadRequest.Source != downloader.TaskSourceSearch || dl.downloadRequest.Code != "ABCD-321" {
		t.Fatalf("unexpected request: %+v", dl.downloadRequest)
	}
}

func TestCreateFromSubscriptionReleaseUsesCode(t *testing.T) {
	dl := &fakeDownloader{downloadTask: &downloader.Task{ID: "task-4"}}
	service := NewService(dl)

	task, err := service.CreateFromSubscriptionRelease(context.Background(), CreateFromSubscriptionReleaseInput{
		Code: "SSIS-001",
	})
	if err != nil {
		t.Fatalf("CreateFromSubscriptionRelease failed: %v", err)
	}
	if task == nil || task.ID != "task-4" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if dl.downloadRequest.Source != downloader.TaskSourceSubscription {
		t.Fatalf("unexpected source: %s", dl.downloadRequest.Source)
	}
	if dl.downloadRequest.Code != "SSIS-001" {
		t.Fatalf("unexpected request: %+v", dl.downloadRequest)
	}
}

func TestCreateFromSubscriptionReleaseRejectsMissingCode(t *testing.T) {
	service := NewService(&fakeDownloader{})

	task, err := service.CreateFromSubscriptionRelease(context.Background(), CreateFromSubscriptionReleaseInput{})
	if err == nil || err.Error() != "task code is required" {
		t.Fatalf("expected missing code error, got task=%+v err=%v", task, err)
	}
}

func TestCreateFromSearchCodePropagatesDownloaderErrors(t *testing.T) {
	service := NewService(&fakeDownloader{err: errors.New("boom")})

	_, err := service.CreateFromSearchCode(context.Background(), CreateFromSearchCodeInput{Code: "ABCD-123"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected downloader error, got %v", err)
	}
}
