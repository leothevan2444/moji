package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/stashsync"
)

func TestDownloadMediaCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		downloadTask: &downloader.Task{
			ID:         "task-1",
			Query:      "ABCD-123",
			Status:     downloader.TaskStatusAdded,
			TorrentURL: "magnet:?xt=urn:btih:test",
			Candidate: downloader.Candidate{
				Title:   "ABCD-123",
				Tracker: "demo",
				Seeders: 5,
			},
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		downloadMedia(input: { query: "ABCD-123", limit: 1 }) {
			id
			query
			status
			torrentUrl
			candidate { title tracker seeders }
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	task := resp.Data.DownloadMedia
	if task.ID != "task-1" || task.Status != "added" || task.Candidate.Title != "ABCD-123" {
		t.Fatalf("unexpected download task response: %+v", task)
	}
	if downloader.downloadRequest.Query != "ABCD-123" || downloader.downloadRequest.Limit != 1 {
		t.Fatalf("unexpected download request: %+v", downloader.downloadRequest)
	}
}

func TestAddTorrentCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		addTask: &downloader.Task{
			ID:         "task-manual",
			Query:      "magnet:?xt=urn:btih:manual",
			Status:     downloader.TaskStatusAdded,
			TorrentURL: "magnet:?xt=urn:btih:manual",
			Candidate: downloader.Candidate{
				Title:     "magnet:?xt=urn:btih:manual",
				MagnetURI: "magnet:?xt=urn:btih:manual",
			},
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		addTorrent(input: { url: "magnet:?xt=urn:btih:manual", category: "moji" }) {
			id
			status
			torrentUrl
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.AddTorrent.ID != "task-manual" || resp.Data.AddTorrent.Status != "added" {
		t.Fatalf("unexpected add torrent response: %+v", resp.Data.AddTorrent)
	}
	if downloader.addRequest.URL != "magnet:?xt=urn:btih:manual" || downloader.addRequest.Category != "moji" {
		t.Fatalf("unexpected add torrent request: %+v", downloader.addRequest)
	}
}

func TestDeprecatedQBittorrentAddCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		addTask: &downloader.Task{
			ID:         "task-manual",
			Status:     downloader.TaskStatusAdded,
			TorrentURL: "magnet:?xt=urn:btih:manual",
			CreatedAt:  time.Unix(100, 0).UTC(),
			UpdatedAt:  time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		qbittorrentAdd(input: { url: "magnet:?xt=urn:btih:manual" })
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if !resp.Data.QbittorrentAdd {
		t.Fatal("expected qbittorrentAdd to return true")
	}
	if downloader.addRequest.URL != "magnet:?xt=urn:btih:manual" {
		t.Fatalf("unexpected add torrent request: %+v", downloader.addRequest)
	}
}

func TestTasksQueryListsTasks(t *testing.T) {
	downloader := &fakeDownloader{
		listTasks: []*downloader.Task{
			{ID: "task-2", Query: "BBBB-222", Status: downloader.TaskStatusAdded, CreatedAt: time.Unix(200, 0).UTC(), UpdatedAt: time.Unix(200, 0).UTC()},
			{ID: "task-1", Query: "AAAA-111", Status: downloader.TaskStatusFailed, CreatedAt: time.Unix(100, 0).UTC(), UpdatedAt: time.Unix(100, 0).UTC()},
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{ tasks { id query status } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Tasks) != 2 || resp.Data.Tasks[0].ID != "task-2" || resp.Data.Tasks[1].ID != "task-1" {
		t.Fatalf("unexpected tasks response: %+v", resp.Data.Tasks)
	}
}

func TestTasksQueryWithoutDownloaderReturnsEmptyList(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{ tasks { id } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.Tasks == nil {
		t.Fatal("expected empty tasks list, got nil")
	}
	if len(resp.Data.Tasks) != 0 {
		t.Fatalf("expected empty tasks list, got %+v", resp.Data.Tasks)
	}
}

func TestTaskQueryWithoutDownloaderReturnsNull(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{ task(id: "task-1") { id } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.Task != nil {
		t.Fatalf("expected null task, got %+v", resp.Data.Task)
	}
}

func TestSyncTaskProgress(t *testing.T) {
	downloader := &fakeDownloader{
		syncTasks: []*downloader.Task{
			{
				ID:               "task-sync",
				Query:            "ABCD-123",
				Status:           downloader.TaskStatusDownloading,
				TorrentHash:      "hash-sync",
				TorrentName:      "ABCD-123",
				Progress:         0.5,
				QBittorrentState: "downloading",
				ContentPath:      "/downloads/ABCD-123.mp4",
				CreatedAt:        time.Unix(100, 0).UTC(),
				UpdatedAt:        time.Unix(200, 0).UTC(),
			},
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		syncTaskProgress {
			id
			status
			torrentHash
			progress
			qbittorrentState
			contentPath
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.SyncTaskProgress) != 1 {
		t.Fatalf("expected one synced task, got %+v", resp.Data.SyncTaskProgress)
	}
	task := resp.Data.SyncTaskProgress[0]
	if task.ID != "task-sync" || task.TorrentHash != "hash-sync" || task.Progress != 0.5 {
		t.Fatalf("unexpected synced task: %+v", task)
	}
}

func TestTriggerStashScans(t *testing.T) {
	dl := &fakeDownloader{
		stashTasks: []*downloader.Task{
			{
				ID:                 "task-stash",
				Status:             downloader.TaskStatusCompleted,
				StashJobID:         "job-1",
				StashScanStatus:    downloader.StashScanStatusStarted,
				StashScanStartedAt: ptrTime(time.Unix(300, 0).UTC()),
				CreatedAt:          time.Unix(100, 0).UTC(),
				UpdatedAt:          time.Unix(300, 0).UTC(),
			},
		},
	}
	resolver := NewResolver(nil, nil, dl, fakeStashService{}, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		triggerStashScans {
			id
			stashJobId
			stashScanStatus
			stashScanStartedAt
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.TriggerStashScans) != 1 {
		t.Fatalf("expected one stash scan task, got %+v", resp.Data.TriggerStashScans)
	}
	task := resp.Data.TriggerStashScans[0]
	if task.StashJobID != "job-1" || task.StashScanStatus != downloader.StashScanStatusStarted {
		t.Fatalf("unexpected stash scan task: %+v", task)
	}
}

func TestDownloadMediaRequiresDownloader(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		downloadMedia(input: { query: "ABCD-123" }) { id }
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected downloader configuration error")
	}
	if got := resp.Errors[0].Message; got != "downloader is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestTriggerStashScansRequiresStash(t *testing.T) {
	resolver := NewResolver(nil, nil, &fakeDownloader{}, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		triggerStashScans { id }
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected stash configuration error")
	}
	if got := resp.Errors[0].Message; got != "stash client is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestStashMetadataScanRequiresStash(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		stashMetadataScan(input: { paths: ["/library"] })
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected stash configuration error")
	}
	if got := resp.Errors[0].Message; got != "stash client is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestStashJobRequiresStash(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{ stashJob(id: "job-1") { id } }`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected stash configuration error")
	}
	if got := resp.Errors[0].Message; got != "stash client is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestSettingsQueryReturnsRuntimeSnapshot(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.RuntimeSettings = &SettingsSnapshot{
		Stash: StashSettingsSnapshot{
			Configured:       true,
			Enabled:          true,
			URL:              "http://stash.invalid",
			APIKeyConfigured: true,
			LibraryPath:      "/data/library",
		},
		Jackett: JackettSettingsSnapshot{
			Configured:       true,
			Enabled:          true,
			URL:              "http://jackett.invalid",
			APIKeyConfigured: true,
		},
		QBittorrent: QBittorrentSettingsSnapshot{
			Configured:         false,
			Enabled:            false,
			URL:                "http://qbittorrent.invalid",
			Username:           "operator",
			UsernameConfigured: true,
			PasswordConfigured: false,
			DefaultSavePath:    "/downloads",
			Category:           "moji",
			Tags:               "auto",
		},
		Tasks: TaskSettingsSnapshot{
			Store:                       "json",
			JSONPath:                    "moji-tasks.json",
			ProgressSyncIntervalSeconds: 60,
			ProgressSyncEnabled:         true,
		},
		System: SystemSettingsSnapshot{
			AppVersion: "test-version",
		},
	}

	resp := executeGraphQL(t, resolver, `{
		settings {
			stash { configured enabled url apiKeyConfigured libraryPath }
			jackett { configured enabled url apiKeyConfigured }
			qbittorrent { configured enabled url username usernameConfigured passwordConfigured defaultSavePath category tags }
			tasks { store jsonPath progressSyncIntervalSeconds progressSyncEnabled }
			system { appVersion }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if !resp.Data.Settings.Stash.Configured || resp.Data.Settings.Stash.URL != "http://stash.invalid" {
		t.Fatalf("unexpected stash settings: %+v", resp.Data.Settings.Stash)
	}
	if resp.Data.Settings.Qbittorrent.PasswordConfigured {
		t.Fatalf("expected passwordConfigured false, got %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.Qbittorrent.Username != "operator" {
		t.Fatalf("unexpected qbittorrent username: %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.Tasks.ProgressSyncIntervalSeconds != 60 || !resp.Data.Settings.Tasks.ProgressSyncEnabled {
		t.Fatalf("unexpected task settings: %+v", resp.Data.Settings.Tasks)
	}
	if resp.Data.Settings.System.AppVersion != "test-version" {
		t.Fatalf("unexpected system settings: %+v", resp.Data.Settings.System)
	}
}

func TestSettingsQueryFallsBackToAppVersion(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "fallback-version")

	resp := executeGraphQL(t, resolver, `{ settings { system { appVersion } } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.Settings.System.AppVersion != "fallback-version" {
		t.Fatalf("unexpected app version: %+v", resp.Data.Settings.System)
	}
}

func TestSettingsQueryUsesSettingsEditorSnapshot(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.RuntimeSettings = &SettingsSnapshot{
		System: SystemSettingsSnapshot{AppVersion: "stale-version"},
	}
	resolver.SettingsEditor = &fakeSettingsEditor{
		snapshot: &SettingsSnapshot{
			QBittorrent: QBittorrentSettingsSnapshot{
				URL:      "http://qb.invalid",
				Username: "editor-user",
			},
			System: SystemSettingsSnapshot{AppVersion: "editor-version"},
		},
	}

	resp := executeGraphQL(t, resolver, `{ settings { qbittorrent { url username } system { appVersion } } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.Settings.Qbittorrent.Username != "editor-user" {
		t.Fatalf("unexpected qbittorrent settings: %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.System.AppVersion != "editor-version" {
		t.Fatalf("unexpected system settings: %+v", resp.Data.Settings.System)
	}
}

func TestUpdateStashSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateStashSnapshot: &SettingsSnapshot{
			Stash: StashSettingsSnapshot{
				Configured:       true,
				Enabled:          false,
				URL:              "http://stash.updated",
				APIKeyConfigured: true,
				LibraryPath:      "/library/updated",
			},
			System: SystemSettingsSnapshot{AppVersion: "test-version"},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateStashSettings(input: {
			url: "http://stash.updated"
			apiKey: "secret"
			libraryPath: "/library/updated"
		}) {
			stash { url apiKeyConfigured libraryPath }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.stashInput.URL != "http://stash.updated" || editor.stashInput.LibraryPath != "/library/updated" {
		t.Fatalf("unexpected stash input: %+v", editor.stashInput)
	}
	if editor.stashInput.APIKey == nil || *editor.stashInput.APIKey != "secret" {
		t.Fatalf("unexpected stash api key: %+v", editor.stashInput.APIKey)
	}
	if resp.Data.UpdateStashSettings.Stash.URL != "http://stash.updated" {
		t.Fatalf("unexpected stash response: %+v", resp.Data.UpdateStashSettings.Stash)
	}
}

func TestUpdateQBittorrentSettingsRequiresEditor(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		updateQBittorrentSettings(input: {
			url: "http://localhost:8080"
			username: "admin"
			defaultSavePath: "/downloads"
			category: "moji"
			tags: "auto"
		}) {
			qbittorrent { url }
		}
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected settings editor configuration error")
	}
	if got := resp.Errors[0].Message; got != "settings editor is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

type fakeDownloader struct {
	addRequest      downloader.AddTorrentRequest
	downloadRequest downloader.DownloadRequest
	addTask         *downloader.Task
	downloadTask    *downloader.Task
	findTask        *downloader.Task
	listTasks       []*downloader.Task
	syncTasks       []*downloader.Task
	stashTasks      []*downloader.Task
}

func (f *fakeDownloader) AddTorrentContext(_ context.Context, req downloader.AddTorrentRequest) (*downloader.Task, error) {
	f.addRequest = req
	return f.addTask, nil
}

func (f *fakeDownloader) DownloadMediaContext(_ context.Context, req downloader.DownloadRequest) (*downloader.Task, error) {
	f.downloadRequest = req
	return f.downloadTask, nil
}

func (f *fakeDownloader) FindTask(_ context.Context, _ string) (*downloader.Task, error) {
	return f.findTask, nil
}

func (f *fakeDownloader) ListTasks(_ context.Context) ([]*downloader.Task, error) {
	return f.listTasks, nil
}

func (f *fakeDownloader) SyncProgress(_ context.Context) ([]*downloader.Task, error) {
	return f.syncTasks, nil
}

func (f *fakeDownloader) TriggerStashScans(_ context.Context, _ downloader.StashScanner) ([]*downloader.Task, error) {
	return f.stashTasks, nil
}

type graphQLTaskResponse struct {
	Data struct {
		AddTorrent struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			TorrentURL string `json:"torrentUrl"`
		} `json:"addTorrent"`
		DownloadMedia struct {
			ID         string `json:"id"`
			Query      string `json:"query"`
			Status     string `json:"status"`
			TorrentURL string `json:"torrentUrl"`
			Candidate  struct {
				Title   string `json:"title"`
				Tracker string `json:"tracker"`
				Seeders int    `json:"seeders"`
			} `json:"candidate"`
		} `json:"downloadMedia"`
		Tasks []struct {
			ID     string `json:"id"`
			Query  string `json:"query"`
			Status string `json:"status"`
		} `json:"tasks"`
		Task *struct {
			ID string `json:"id"`
		} `json:"task"`
		SyncTaskProgress []struct {
			ID               string  `json:"id"`
			Status           string  `json:"status"`
			TorrentHash      string  `json:"torrentHash"`
			Progress         float64 `json:"progress"`
			QbittorrentState string  `json:"qbittorrentState"`
			ContentPath      string  `json:"contentPath"`
		} `json:"syncTaskProgress"`
		TriggerStashScans []struct {
			ID                 string  `json:"id"`
			StashJobID         string  `json:"stashJobId"`
			StashScanStatus    string  `json:"stashScanStatus"`
			StashScanStartedAt *string `json:"stashScanStartedAt"`
		} `json:"triggerStashScans"`
		Settings struct {
			Stash struct {
				Configured       bool   `json:"configured"`
				Enabled          bool   `json:"enabled"`
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
				LibraryPath      string `json:"libraryPath"`
			} `json:"stash"`
			Jackett struct {
				Configured       bool   `json:"configured"`
				Enabled          bool   `json:"enabled"`
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
			} `json:"jackett"`
			Qbittorrent struct {
				Configured         bool   `json:"configured"`
				Enabled            bool   `json:"enabled"`
				URL                string `json:"url"`
				Username           string `json:"username"`
				UsernameConfigured bool   `json:"usernameConfigured"`
				PasswordConfigured bool   `json:"passwordConfigured"`
				DefaultSavePath    string `json:"defaultSavePath"`
				Category           string `json:"category"`
				Tags               string `json:"tags"`
			} `json:"qbittorrent"`
			Tasks struct {
				Store                       string `json:"store"`
				JSONPath                    string `json:"jsonPath"`
				ProgressSyncIntervalSeconds int    `json:"progressSyncIntervalSeconds"`
				ProgressSyncEnabled         bool   `json:"progressSyncEnabled"`
			} `json:"tasks"`
			System struct {
				AppVersion string `json:"appVersion"`
			} `json:"system"`
		} `json:"settings"`
		UpdateStashSettings struct {
			Stash struct {
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
				LibraryPath      string `json:"libraryPath"`
			} `json:"stash"`
		} `json:"updateStashSettings"`
		QbittorrentAdd bool `json:"qbittorrentAdd"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type fakeSettingsEditor struct {
	snapshot                  *SettingsSnapshot
	stashInput                UpdateStashSettingsInput
	updateStashSnapshot       *SettingsSnapshot
	qbittorrentInput          UpdateQBittorrentSettingsInput
	updateQBittorrentSnapshot *SettingsSnapshot
}

func (f *fakeSettingsEditor) Snapshot() *SettingsSnapshot {
	return f.snapshot
}

func (f *fakeSettingsEditor) UpdateStashSettings(input UpdateStashSettingsInput) (*SettingsSnapshot, error) {
	f.stashInput = input
	return f.updateStashSnapshot, nil
}

func (f *fakeSettingsEditor) UpdateJackettSettings(UpdateJackettSettingsInput) (*SettingsSnapshot, error) {
	return f.snapshot, nil
}

func (f *fakeSettingsEditor) UpdateQBittorrentSettings(input UpdateQBittorrentSettingsInput) (*SettingsSnapshot, error) {
	f.qbittorrentInput = input
	return f.updateQBittorrentSnapshot, nil
}

func executeGraphQL(t *testing.T, resolver *Resolver, query string) graphQLTaskResponse {
	t.Helper()

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp graphQLTaskResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

type fakeStashService struct{}

func (fakeStashService) MetadataScan(context.Context, stashsync.ScanRequest) (string, error) {
	return "job-1", nil
}

func (fakeStashService) FindJob(context.Context, string) (*stashsync.Job, error) {
	return nil, nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
