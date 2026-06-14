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
	"github.com/leothevan2444/moji/internal/following"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/logging"
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

func TestStashPerformersQueryPaginatesResults(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.Following = &fakeFollowingService{
		performers: []following.Performer{
			{ID: "performer-1", Name: "Alice", Followed: true},
			{ID: "performer-2", Name: "Beth", Followed: false},
			{ID: "performer-3", Name: "Clara", Followed: false},
		},
	}

	resp := executeGraphQL(t, resolver, `{
		stashPerformers(page: 2, pageSize: 2) {
			items { id name followed }
			page
			pageSize
			totalCount
			totalPages
			hasPrevPage
			hasNextPage
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.StashPerformers.Page != 2 || resp.Data.StashPerformers.PageSize != 2 {
		t.Fatalf("unexpected page metadata: %+v", resp.Data.StashPerformers)
	}
	if resp.Data.StashPerformers.TotalCount != 3 || resp.Data.StashPerformers.TotalPages != 2 {
		t.Fatalf("unexpected total metadata: %+v", resp.Data.StashPerformers)
	}
	if !resp.Data.StashPerformers.HasPrevPage || resp.Data.StashPerformers.HasNextPage {
		t.Fatalf("unexpected pagination flags: %+v", resp.Data.StashPerformers)
	}
	if len(resp.Data.StashPerformers.Items) != 1 || resp.Data.StashPerformers.Items[0].ID != "performer-3" {
		t.Fatalf("unexpected page items: %+v", resp.Data.StashPerformers.Items)
	}
}

func TestLogsQueryReturnsRecentEntries(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.LogReader = &fakeLogReader{
		entries: []logging.Entry{
			{Message: "latest error", Level: "error"},
			{Message: "background info", Level: "info"},
		},
	}

	resp := executeGraphQL(t, resolver, `{
		logs(limit: 10, minLevel: Info) {
			level
			message
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %+v", resp.Data.Logs)
	}
	if resp.Data.Logs[0].Level != "Error" || resp.Data.Logs[0].Message != "latest error" {
		t.Fatalf("unexpected first log: %+v", resp.Data.Logs[0])
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

func TestTriggerTaskStashScan(t *testing.T) {
	dl := &fakeDownloader{
		triggerTaskScanTask: &downloader.Task{
			ID:              "task-single",
			Status:          downloader.TaskStatusCompleted,
			StashJobID:      "job-single",
			StashScanStatus: downloader.StashScanStatusStarted,
			CreatedAt:       time.Unix(100, 0).UTC(),
			UpdatedAt:       time.Unix(300, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, dl, fakeStashService{}, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		triggerTaskStashScan(id: "task-single") {
			id
			stashJobId
			stashScanStatus
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if dl.triggerTaskScanID != "task-single" {
		t.Fatalf("unexpected trigger task id: %q", dl.triggerTaskScanID)
	}
	if resp.Data.TriggerTaskStashScan.ID != "task-single" || resp.Data.TriggerTaskStashScan.StashJobID != "job-single" {
		t.Fatalf("unexpected single stash scan response: %+v", resp.Data.TriggerTaskStashScan)
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
		Logging: LoggingSettingsSnapshot{
			Level:            "debug",
			FilePath:         "runtime/moji.log",
			MaxEntries:       300,
			MaxFileSizeBytes: 2048,
			MaxFileBackups:   4,
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
			logging { level filePath maxEntries maxFileSizeBytes maxFileBackups }
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
	if resp.Data.Settings.Logging.Level != "debug" || resp.Data.Settings.Logging.MaxFileBackups != 4 {
		t.Fatalf("unexpected logging settings: %+v", resp.Data.Settings.Logging)
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

func TestDashboardStatsQueryAggregatesTasks(t *testing.T) {
	downloader := &fakeDownloader{
		listTasks: []*downloader.Task{
			{ID: "task-1", Status: downloader.TaskStatusDownloading},
			{ID: "task-2", Status: downloader.TaskStatusCompleted, StashScanStatus: downloader.StashScanStatusStarted},
			{ID: "task-3", Status: downloader.TaskStatusFailed},
			{ID: "task-4", Status: downloader.TaskStatusCompleted, StashScanError: "stash rejected scan"},
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{
		dashboardStats {
			total
			active
			completed
			downloading
			pendingScans
			failed
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	stats := resp.Data.DashboardStats
	if stats.Total != 4 || stats.Active != 1 || stats.Completed != 2 || stats.Downloading != 1 || stats.PendingScans != 1 || stats.Failed != 2 {
		t.Fatalf("unexpected dashboard stats: %+v", stats)
	}
}

func TestSettingsQueryUsesSettingsEditorSnapshot(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.RuntimeSettings = &SettingsSnapshot{
		System: SystemSettingsSnapshot{AppVersion: "stale-version"},
	}
	resolver.SettingsEditor = &fakeSettingsEditor{
		snapshot: &SettingsSnapshot{
			Following: FollowingSettingsSnapshot{
				Store:               "json",
				JSONPath:            "moji-following.json",
				PollIntervalSeconds: 3600,
				PollEnabled:         true,
				JAVStashEnabled:     true,
			},
			QBittorrent: QBittorrentSettingsSnapshot{
				URL:      "http://qb.invalid",
				Username: "editor-user",
			},
			Logging: LoggingSettingsSnapshot{
				Level:            "warn",
				FilePath:         "custom/moji.log",
				MaxEntries:       200,
				MaxFileSizeBytes: 4096,
				MaxFileBackups:   3,
			},
			System: SystemSettingsSnapshot{AppVersion: "editor-version"},
		},
	}

	resp := executeGraphQL(t, resolver, `{ settings { following { store jsonPath pollIntervalSeconds pollEnabled javstashEnabled } qbittorrent { url username } logging { level filePath maxEntries maxFileBackups } system { appVersion } } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.Settings.Following.Store != "json" || !resp.Data.Settings.Following.PollEnabled {
		t.Fatalf("unexpected following settings: %+v", resp.Data.Settings.Following)
	}
	if resp.Data.Settings.Qbittorrent.Username != "editor-user" {
		t.Fatalf("unexpected qbittorrent settings: %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.Logging.Level != "warn" || resp.Data.Settings.Logging.MaxEntries != 200 {
		t.Fatalf("unexpected logging settings: %+v", resp.Data.Settings.Logging)
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

func TestUpdateFollowingSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateFollowingSnapshot: &SettingsSnapshot{
			Following: FollowingSettingsSnapshot{
				Store:                    "json",
				JSONPath:                 "state/following.json",
				PollIntervalSeconds:      1800,
				PollEnabled:              true,
				JAVStashEnabled:          true,
				JAVStashAPIKeyConfigured: true,
			},
			System: SystemSettingsSnapshot{AppVersion: "test-version"},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateFollowingSettings(input: {
			store: "json"
			jsonPath: "state/following.json"
			pollIntervalSeconds: 1800
			javstashApiKey: "token"
		}) {
			following {
				store
				jsonPath
				pollIntervalSeconds
				pollEnabled
				javstashApiKeyConfigured
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.followingInput.Store != "json" || editor.followingInput.JSONPath != "state/following.json" {
		t.Fatalf("unexpected following input: %+v", editor.followingInput)
	}
	if editor.followingInput.PollIntervalSeconds != 1800 {
		t.Fatalf("unexpected following interval: %+v", editor.followingInput)
	}
	if editor.followingInput.JAVStashAPIKey == nil || *editor.followingInput.JAVStashAPIKey != "token" {
		t.Fatalf("unexpected following api key: %+v", editor.followingInput.JAVStashAPIKey)
	}
	if resp.Data.UpdateFollowingSettings.Following.JSONPath != "state/following.json" || !resp.Data.UpdateFollowingSettings.Following.JavstashAPIKeyConfigured {
		t.Fatalf("unexpected following response: %+v", resp.Data.UpdateFollowingSettings.Following)
	}
}

func TestUpdateLoggingSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateLoggingSnapshot: &SettingsSnapshot{
			Logging: LoggingSettingsSnapshot{
				Level:            "debug",
				FilePath:         "logs/custom.log",
				MaxEntries:       250,
				MaxFileSizeBytes: 8192,
				MaxFileBackups:   6,
			},
			System: SystemSettingsSnapshot{AppVersion: "test-version"},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateLoggingSettings(input: {
			level: "debug"
			filePath: "logs/custom.log"
			maxEntries: 250
			maxFileSizeBytes: 8192
			maxFileBackups: 6
		}) {
			logging {
				level
				filePath
				maxEntries
				maxFileSizeBytes
				maxFileBackups
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.loggingInput.Level != "debug" || editor.loggingInput.FilePath != "logs/custom.log" {
		t.Fatalf("unexpected logging input: %+v", editor.loggingInput)
	}
	if editor.loggingInput.MaxEntries != 250 || editor.loggingInput.MaxFileSizeBytes != 8192 || editor.loggingInput.MaxFileBackups != 6 {
		t.Fatalf("unexpected logging limits: %+v", editor.loggingInput)
	}
	if resp.Data.UpdateLoggingSettings.Logging.Level != "debug" || resp.Data.UpdateLoggingSettings.Logging.MaxEntries != 250 {
		t.Fatalf("unexpected logging response: %+v", resp.Data.UpdateLoggingSettings.Logging)
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
	addRequest          downloader.AddTorrentRequest
	downloadRequest     downloader.DownloadRequest
	addTask             *downloader.Task
	downloadTask        *downloader.Task
	findTask            *downloader.Task
	listTasks           []*downloader.Task
	syncTasks           []*downloader.Task
	stashTasks          []*downloader.Task
	triggerTaskScanID   string
	triggerTaskScanTask *downloader.Task
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

func (f *fakeDownloader) TriggerTaskStashScan(_ context.Context, id string, _ downloader.StashScanner) (*downloader.Task, error) {
	f.triggerTaskScanID = id
	return f.triggerTaskScanTask, nil
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
		TriggerTaskStashScan struct {
			ID              string `json:"id"`
			StashJobID      string `json:"stashJobId"`
			StashScanStatus string `json:"stashScanStatus"`
		} `json:"triggerTaskStashScan"`
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
			Following struct {
				Store               string `json:"store"`
				JSONPath            string `json:"jsonPath"`
				PollIntervalSeconds int    `json:"pollIntervalSeconds"`
				PollEnabled         bool   `json:"pollEnabled"`
				JavstashEnabled     bool   `json:"javstashEnabled"`
			} `json:"following"`
			Logging struct {
				Level            string `json:"level"`
				FilePath         string `json:"filePath"`
				MaxEntries       int    `json:"maxEntries"`
				MaxFileSizeBytes int    `json:"maxFileSizeBytes"`
				MaxFileBackups   int    `json:"maxFileBackups"`
			} `json:"logging"`
			System struct {
				AppVersion string `json:"appVersion"`
			} `json:"system"`
		} `json:"settings"`
		StashPerformers struct {
			Items []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Followed bool   `json:"followed"`
			} `json:"items"`
			Page        int  `json:"page"`
			PageSize    int  `json:"pageSize"`
			TotalCount  int  `json:"totalCount"`
			TotalPages  int  `json:"totalPages"`
			HasPrevPage bool `json:"hasPrevPage"`
			HasNextPage bool `json:"hasNextPage"`
		} `json:"stashPerformers"`
		Logs []struct {
			Level   string `json:"level"`
			Message string `json:"message"`
		} `json:"logs"`
		DashboardStats struct {
			Total        int `json:"total"`
			Active       int `json:"active"`
			Completed    int `json:"completed"`
			Downloading  int `json:"downloading"`
			PendingScans int `json:"pendingScans"`
			Failed       int `json:"failed"`
		} `json:"dashboardStats"`
		UpdateStashSettings struct {
			Stash struct {
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
				LibraryPath      string `json:"libraryPath"`
			} `json:"stash"`
		} `json:"updateStashSettings"`
		UpdateFollowingSettings struct {
			Following struct {
				Store                    string `json:"store"`
				JSONPath                 string `json:"jsonPath"`
				PollIntervalSeconds      int    `json:"pollIntervalSeconds"`
				PollEnabled              bool   `json:"pollEnabled"`
				JavstashAPIKeyConfigured bool   `json:"javstashApiKeyConfigured"`
			} `json:"following"`
		} `json:"updateFollowingSettings"`
		UpdateLoggingSettings struct {
			Logging struct {
				Level            string `json:"level"`
				FilePath         string `json:"filePath"`
				MaxEntries       int    `json:"maxEntries"`
				MaxFileSizeBytes int    `json:"maxFileSizeBytes"`
				MaxFileBackups   int    `json:"maxFileBackups"`
			} `json:"logging"`
		} `json:"updateLoggingSettings"`
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
	followingInput            UpdateFollowingSettingsInput
	updateFollowingSnapshot   *SettingsSnapshot
	loggingInput              UpdateLoggingSettingsInput
	updateLoggingSnapshot     *SettingsSnapshot
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

func (f *fakeSettingsEditor) UpdateFollowingSettings(input UpdateFollowingSettingsInput) (*SettingsSnapshot, error) {
	f.followingInput = input
	return f.updateFollowingSnapshot, nil
}

func (f *fakeSettingsEditor) UpdateLoggingSettings(input UpdateLoggingSettingsInput) (*SettingsSnapshot, error) {
	f.loggingInput = input
	return f.updateLoggingSnapshot, nil
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

type fakeFollowingService struct {
	performers []following.Performer
}

type fakeLogReader struct {
	entries []logging.Entry
}

func (f *fakeLogReader) Entries(limit int, _ string) []logging.Entry {
	if limit <= 0 || limit >= len(f.entries) {
		return append([]logging.Entry(nil), f.entries...)
	}
	return append([]logging.Entry(nil), f.entries[:limit]...)
}

func (f *fakeFollowingService) ListStashPerformers(_ context.Context, _ string) ([]following.Performer, error) {
	return f.performers, nil
}

func (f *fakeFollowingService) ListFollowingPerformers(context.Context) ([]following.FollowingPerformer, error) {
	return nil, nil
}

func (f *fakeFollowingService) FollowPerformer(context.Context, string) (following.FollowingPerformer, error) {
	return following.FollowingPerformer{}, nil
}

func (f *fakeFollowingService) UnfollowPerformer(context.Context, string) error {
	return nil
}

func (f *fakeFollowingService) RefreshPerformer(context.Context, string) (following.FollowingPerformer, error) {
	return following.FollowingPerformer{}, nil
}

func (f *fakeFollowingService) RefreshAll(context.Context) ([]following.FollowingPerformer, error) {
	return nil, nil
}

func (fakeStashService) MetadataScan(context.Context, stashsync.ScanRequest) (string, error) {
	return "job-1", nil
}

func (fakeStashService) FindJob(context.Context, string) (*stashsync.Job, error) {
	return nil, nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
