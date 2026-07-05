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
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/subscription"
)

func TestDownloadMediaCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		downloadTask: &downloader.Task{
			ID:         "task-1",
			Query:      "ABCD-123",
			Code:       "ABCD-123",
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
			code
			status
			torrentUrl
			candidate { title tracker seeders }
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	task := resp.Data.DownloadMedia
	if task.ID != "task-1" || task.Code != "ABCD-123" || task.Status != "added" || task.Candidate.Title != "ABCD-123" {
		t.Fatalf("unexpected download task response: %+v", task)
	}
	if downloader.downloadRequest.Query != "ABCD-123" || downloader.downloadRequest.Limit != 1 {
		t.Fatalf("unexpected download request: %+v", downloader.downloadRequest)
	}
	if string(downloader.downloadRequest.Source) != "MANUAL" {
		t.Fatalf("expected manual task source, got %+v", downloader.downloadRequest)
	}
}

func TestAddTorrentCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		addTask: &downloader.Task{
			ID:         "task-manual",
			Query:      "magnet:?xt=urn:btih:manual",
			Code:       "ABCD-123",
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
			code
			status
			torrentUrl
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.AddTorrent.ID != "task-manual" || resp.Data.AddTorrent.Code != "ABCD-123" || resp.Data.AddTorrent.Status != "added" {
		t.Fatalf("unexpected add torrent response: %+v", resp.Data.AddTorrent)
	}
	if downloader.addRequest.URL != "magnet:?xt=urn:btih:manual" || downloader.addRequest.Category != "moji" {
		t.Fatalf("unexpected add torrent request: %+v", downloader.addRequest)
	}
	if string(downloader.addRequest.Source) != "MANUAL" {
		t.Fatalf("expected manual task source, got %+v", downloader.addRequest)
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

	resp := executeGraphQL(t, resolver, `{ tasks { id query code status } }`)
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

func TestDeleteTaskMutation(t *testing.T) {
	downloader := &fakeDownloader{
		deleteTask: &downloader.Task{
			ID:        "task-delete",
			Status:    downloader.TaskStatusCompleted,
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		deleteTask(id: "task-delete") {
			id
			status
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if downloader.deleteTaskID != "task-delete" {
		t.Fatalf("expected delete request for task-delete, got %q", downloader.deleteTaskID)
	}
	if resp.Data.DeleteTask.ID != "task-delete" || resp.Data.DeleteTask.Status != "completed" {
		t.Fatalf("unexpected delete task response: %+v", resp.Data.DeleteTask)
	}
}

func TestStashPerformersQueryPaginatesResults(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.Subscription = &fakeSubscriptionService{
		performers: []subscription.Performer{
			{ID: "performer-1", Name: "Alice", Subscribed: true},
			{ID: "performer-2", Name: "Beth", Subscribed: false},
			{ID: "performer-3", Name: "Clara", Subscribed: false},
		},
	}

	resp := executeGraphQL(t, resolver, `{
		stashPerformers(page: 2, pageSize: 2) {
			items { id name subscribed }
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
			URL:              "http://stash.invalid",
			APIKeyConfigured: true,
		},
		Ingest: IngestSettingsSnapshot{
			DeliveryMode: "PATH_MAP",
			Downloads: DownloadsIngestSettingsSnapshot{
				QBRoot: "/downloads",
			},
			Library: LibraryIngestSettingsSnapshot{
				StashRoot: "/library",
			},
		},
		Jackett: JackettSettingsSnapshot{
			Configured:       true,
			URL:              "http://jackett.invalid",
			APIKeyConfigured: true,
		},
		QBittorrent: QBittorrentSettingsSnapshot{
			Configured:         false,
			URL:                "http://qbittorrent.invalid",
			Username:           "operator",
			UsernameConfigured: true,
			PasswordConfigured: false,
			DefaultSavePath:    "/downloads",
			Category:           "moji",
			Tags:               "auto",
		},
		Automation: AutomationSettingsSnapshot{
			TaskProgressSyncIntervalSeconds: 60,
			SubscriptionPollIntervalHours:   6,
			StashBoxEndpoints:               []string{"https://javstash.example.org/graphql"},
			TorrentSelection: TorrentSelectionSettingsSnapshot{
				Enabled: true,
				Rules: []TorrentSelectionRuleSnapshot{
					{
						ID:        "seeders",
						Name:      "Seeders",
						Type:      "SEEDERS",
						Enabled:   true,
						Direction: "DESC",
					},
				},
			},
		},
		System: SystemSettingsSnapshot{
			TaskDeletePolicy: "KEEP_ONLY",
		},
	}
	resolver.RuntimeStatus = &SettingsStatusSnapshot{
		Automation: AutomationStatusSnapshot{
			TaskProgressSyncIntervalSeconds: 60,
			TaskProgressSyncEnabled:         true,
			SubscriptionPollIntervalHours:   6,
			SubscriptionPollEnabled:         true,
		},
	}

	resp := executeGraphQL(t, resolver, `{
		settings {
			stash { configured url apiKeyConfigured }
			ingest { deliveryMode downloads { qbRoot } library { stashRoot } }
			jackett { configured url apiKeyConfigured }
			qbittorrent { configured url username usernameConfigured passwordConfigured defaultSavePath category tags }
			automation { taskProgressSyncIntervalSeconds subscriptionPollIntervalHours stashBoxEndpoints torrentSelection { enabled rules { id name type direction } } }
			system { taskDeletePolicy }
		}
		settingsStatus {
			automation { taskProgressSyncEnabled }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if !resp.Data.Settings.Stash.Configured || resp.Data.Settings.Stash.URL != "http://stash.invalid" {
		t.Fatalf("unexpected stash settings: %+v", resp.Data.Settings.Stash)
	}
	if !resp.Data.Settings.Automation.TorrentSelection.Enabled || len(resp.Data.Settings.Automation.TorrentSelection.Rules) != 1 {
		t.Fatalf("unexpected automation torrent selection: %+v", resp.Data.Settings.Automation.TorrentSelection)
	}
	if resp.Data.Settings.Ingest.Downloads.QBRoot != "/downloads" || resp.Data.Settings.Ingest.Library.StashRoot != "/library" {
		t.Fatalf("unexpected ingest settings: %+v", resp.Data.Settings.Ingest)
	}
	if resp.Data.Settings.Qbittorrent.PasswordConfigured {
		t.Fatalf("expected passwordConfigured false, got %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.Qbittorrent.Username != "operator" {
		t.Fatalf("unexpected qbittorrent username: %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.Automation.TaskProgressSyncIntervalSeconds != 60 {
		t.Fatalf("unexpected automation settings: %+v", resp.Data.Settings.Automation)
	}
	if len(resp.Data.Settings.Automation.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected automation stash-box endpoints: %+v", resp.Data.Settings.Automation)
	}
	if resp.Data.Settings.System.TaskDeletePolicy != "KEEP_ONLY" {
		t.Fatalf("unexpected system settings: %+v", resp.Data.Settings.System)
	}
	if !resp.Data.SettingsStatus.Automation.TaskProgressSyncEnabled {
		t.Fatalf("unexpected automation status: %+v", resp.Data.SettingsStatus.Automation)
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
	resolver.SettingsEditor = &fakeSettingsEditor{
		snapshot: &SettingsSnapshot{
			Automation: AutomationSettingsSnapshot{
				TaskProgressSyncIntervalSeconds: 60,
				SubscriptionPollIntervalHours:   1,
				StashBoxEndpoints:               []string{"https://javstash.example.org/graphql"},
				TorrentSelection: TorrentSelectionSettingsSnapshot{
					Enabled: true,
					Rules: []TorrentSelectionRuleSnapshot{
						{ID: "seeders", Name: "Seeders", Type: "SEEDERS", Enabled: true, Direction: "DESC"},
					},
				},
			},
			QBittorrent: QBittorrentSettingsSnapshot{
				URL:      "http://qb.invalid",
				Username: "editor-user",
			},
			System: SystemSettingsSnapshot{
				TaskDeletePolicy: "REMOVE_TORRENT",
			},
		},
		statusSnapshot: &SettingsStatusSnapshot{
			Automation: AutomationStatusSnapshot{
				SubscriptionPollIntervalHours: 1,
				SubscriptionPollEnabled:       true,
			},
		},
	}

	resp := executeGraphQL(t, resolver, `{ settings { qbittorrent { url username } automation { subscriptionPollIntervalHours stashBoxEndpoints torrentSelection { enabled rules { id name } } } system { taskDeletePolicy } } settingsStatus { automation { subscriptionPollIntervalHours subscriptionPollEnabled } } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Settings.Automation.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected automation settings: %+v", resp.Data.Settings.Automation)
	}
	if resp.Data.Settings.Qbittorrent.Username != "editor-user" {
		t.Fatalf("unexpected qbittorrent settings: %+v", resp.Data.Settings.Qbittorrent)
	}
	if resp.Data.Settings.System.TaskDeletePolicy != "REMOVE_TORRENT" {
		t.Fatalf("unexpected system settings: %+v", resp.Data.Settings.System)
	}
	if !resp.Data.SettingsStatus.Automation.SubscriptionPollEnabled || resp.Data.SettingsStatus.Automation.SubscriptionPollIntervalHours != 1 {
		t.Fatalf("unexpected automation status: %+v", resp.Data.SettingsStatus.Automation)
	}
	if len(resp.Data.Settings.Automation.TorrentSelection.Rules) != 1 {
		t.Fatalf("unexpected automation torrent selection: %+v", resp.Data.Settings.Automation.TorrentSelection)
	}
}

func TestUpdateStashSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateStashSnapshot: &SettingsSnapshot{
			Stash: StashSettingsSnapshot{
				Configured:       true,
				URL:              "http://stash.updated",
				APIKeyConfigured: true,
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateStashSettings(input: {
			url: "http://stash.updated"
			apiKey: "secret"
		}) {
			stash { url apiKeyConfigured }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.stashInput.URL != "http://stash.updated" {
		t.Fatalf("unexpected stash input: %+v", editor.stashInput)
	}
	if editor.stashInput.APIKey != "secret" {
		t.Fatalf("unexpected stash input details: %+v", editor.stashInput)
	}
	if resp.Data.UpdateStashSettings.Stash.URL != "http://stash.updated" {
		t.Fatalf("unexpected stash response: %+v", resp.Data.UpdateStashSettings.Stash)
	}
}

func TestUpdateIngestSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateIngestSnapshot: &SettingsSnapshot{
			Ingest: IngestSettingsSnapshot{
				DeliveryMode: "TRANSFER",
				Downloads: DownloadsIngestSettingsSnapshot{
					QBRoot:   "/downloads",
					MojiRoot: "/srv/downloads",
				},
				Library: LibraryIngestSettingsSnapshot{
					MojiRoot:  "/mnt/library",
					StashRoot: "/library",
				},
				Transfer: TransferIngestSettingsSnapshot{
					Action: "COPY",
				},
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateIngestSettings(input: {
			deliveryMode: "TRANSFER"
			downloads: {
				qbRoot: "/downloads"
				mojiRoot: "/srv/downloads"
			}
			library: {
				mojiRoot: "/mnt/library"
				stashRoot: "/library"
			}
			transfer: {
				action: "COPY"
			}
		}) {
			ingest { deliveryMode downloads { qbRoot mojiRoot } library { mojiRoot stashRoot } transfer { action } }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.ingestInput.DeliveryMode != "TRANSFER" || editor.ingestInput.Downloads.MojiRoot != "/srv/downloads" || editor.ingestInput.Library.MojiRoot != "/mnt/library" {
		t.Fatalf("unexpected ingest input: %+v", editor.ingestInput)
	}
	if resp.Data.UpdateIngestSettings.Ingest.DeliveryMode != "TRANSFER" {
		t.Fatalf("unexpected ingest response: %+v", resp.Data.UpdateIngestSettings.Ingest)
	}
}

func TestUpdateJackettSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateJackettSnapshot: &SettingsSnapshot{
			Jackett: JackettSettingsSnapshot{
				Configured:       true,
				URL:              "http://jackett.updated",
				APIKeyConfigured: true,
			},
			Automation: AutomationSettingsSnapshot{
				StashBoxEndpoints: []string{"https://javstash.example.org/graphql"},
				TorrentSelection: TorrentSelectionSettingsSnapshot{
					Enabled: true,
					Rules: []TorrentSelectionRuleSnapshot{
						{
							ID:        "pref",
							Name:      "Indexer Preference",
							Type:      "INDEXER_PREFERENCE",
							Enabled:   true,
							Direction: "DESC",
							IndexerPreference: IndexerPreferenceRuleSnapshot{
								TrackerIDs: []string{"alpha"},
							},
						},
					},
				},
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateJackettSettings(input: {
			url: "http://jackett.updated"
			apiKey: "secret"
			password: "pw"
		}) {
			jackett { url }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.jackettInput.URL != "http://jackett.updated" || editor.jackettInput.Password != "pw" {
		t.Fatalf("unexpected jackett input: %+v", editor.jackettInput)
	}
	if resp.Data.UpdateJackettSettings.Jackett.URL != "http://jackett.updated" {
		t.Fatalf("unexpected jackett response: %+v", resp.Data.UpdateJackettSettings.Jackett)
	}
}

func TestUpdateAutomationSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateAutomationSnapshot: &SettingsSnapshot{
			Automation: AutomationSettingsSnapshot{
				TaskProgressSyncIntervalSeconds: 60,
				SubscriptionPollIntervalHours:   1,
				StashBoxEndpoints:               []string{"https://javstash.example.org/graphql"},
				TorrentSelection: TorrentSelectionSettingsSnapshot{
					Enabled: true,
					Rules:   []TorrentSelectionRuleSnapshot{{ID: "seeders", Name: "Seeders", Type: "SEEDERS", Enabled: true, Direction: "DESC"}},
				},
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateAutomationSettings(input: {
			taskProgressSyncIntervalSeconds: 60
			subscriptionPollIntervalHours: 1
			stashBoxEndpoints: ["https://javstash.example.org/graphql"]
			torrentSelection: {
				enabled: true
				rules: [{
					id: "seeders"
					name: "My Seeders Rule"
					type: SEEDERS
					enabled: true
					direction: DESC
					indexerPreference: { trackerIds: [] }
					titleMatch: { clauses: [] }
				}]
			}
		}) {
			automation {
				stashBoxEndpoints
				torrentSelection { enabled rules { id name } }
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(editor.automationInput.StashBoxEndpoints) != 1 || editor.automationInput.StashBoxEndpoints[0] != "https://javstash.example.org/graphql" {
		t.Fatalf("unexpected automation input: %+v", editor.automationInput)
	}
	if len(editor.automationInput.TorrentSelection.Rules) != 1 || editor.automationInput.TorrentSelection.Rules[0].Name != "My Seeders Rule" {
		t.Fatalf("unexpected automation rule input: %+v", editor.automationInput.TorrentSelection.Rules)
	}
	if len(resp.Data.UpdateAutomationSettings.Automation.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected automation response: %+v", resp.Data.UpdateAutomationSettings.Automation)
	}
	if len(resp.Data.UpdateAutomationSettings.Automation.TorrentSelection.Rules) != 1 || resp.Data.UpdateAutomationSettings.Automation.TorrentSelection.Rules[0].Name != "Seeders" {
		t.Fatalf("unexpected automation response rules: %+v", resp.Data.UpdateAutomationSettings.Automation.TorrentSelection.Rules)
	}
}

func TestUpdateSystemSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateSystemSnapshot: &SettingsSnapshot{
			System: SystemSettingsSnapshot{
				TaskDeletePolicy: "REMOVE_TORRENT_AND_FILES",
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateSystemSettings(input: {
			taskDeletePolicy: REMOVE_TORRENT_AND_FILES
		}) {
			system {
				taskDeletePolicy
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.systemInput.TaskDeletePolicy != "REMOVE_TORRENT_AND_FILES" {
		t.Fatalf("unexpected system input: %+v", editor.systemInput)
	}
	if resp.Data.UpdateSystemSettings.System.TaskDeletePolicy != "REMOVE_TORRENT_AND_FILES" {
		t.Fatalf("unexpected system response: %+v", resp.Data.UpdateSystemSettings.System)
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
	deleteTaskID        string
	deleteTask          *downloader.Task
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

func (f *fakeDownloader) DeleteTask(_ context.Context, id string) (*downloader.Task, error) {
	f.deleteTaskID = id
	return f.deleteTask, nil
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
			Code       string `json:"code"`
			Status     string `json:"status"`
			TorrentURL string `json:"torrentUrl"`
		} `json:"addTorrent"`
		DownloadMedia struct {
			ID         string `json:"id"`
			Query      string `json:"query"`
			Code       string `json:"code"`
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
			Code   string `json:"code"`
			Status string `json:"status"`
		} `json:"tasks"`
		Task *struct {
			ID string `json:"id"`
		} `json:"task"`
		DeleteTask struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"deleteTask"`
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
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
			} `json:"stash"`
			Ingest struct {
				DeliveryMode string `json:"deliveryMode"`
				Downloads    struct {
					QBRoot   string `json:"qbRoot"`
					MojiRoot string `json:"mojiRoot"`
				} `json:"downloads"`
				Library struct {
					MojiRoot  string `json:"mojiRoot"`
					StashRoot string `json:"stashRoot"`
				} `json:"library"`
				Transfer struct {
					Action string `json:"action"`
				} `json:"transfer"`
			} `json:"ingest"`
			Jackett struct {
				Configured       bool   `json:"configured"`
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
			} `json:"jackett"`
			Qbittorrent struct {
				Configured         bool   `json:"configured"`
				URL                string `json:"url"`
				Username           string `json:"username"`
				UsernameConfigured bool   `json:"usernameConfigured"`
				PasswordConfigured bool   `json:"passwordConfigured"`
				DefaultSavePath    string `json:"defaultSavePath"`
				Category           string `json:"category"`
				Tags               string `json:"tags"`
			} `json:"qbittorrent"`
			Automation struct {
				TaskProgressSyncIntervalSeconds int      `json:"taskProgressSyncIntervalSeconds"`
				SubscriptionPollIntervalHours   int      `json:"subscriptionPollIntervalHours"`
				StashBoxEndpoints               []string `json:"stashBoxEndpoints"`
				TorrentSelection                struct {
					Enabled bool `json:"enabled"`
					Rules   []struct {
						ID        string `json:"id"`
						Name      string `json:"name"`
						Type      string `json:"type"`
						Direction string `json:"direction"`
					} `json:"rules"`
				} `json:"torrentSelection"`
			} `json:"automation"`
			System struct {
				TaskDeletePolicy string `json:"taskDeletePolicy"`
			} `json:"system"`
		} `json:"settings"`
		SettingsStatus struct {
			Automation struct {
				TaskProgressSyncEnabled       bool `json:"taskProgressSyncEnabled"`
				SubscriptionPollIntervalHours int  `json:"subscriptionPollIntervalHours"`
				SubscriptionPollEnabled       bool `json:"subscriptionPollEnabled"`
			} `json:"automation"`
		} `json:"settingsStatus"`
		StashPerformers struct {
			Items []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Subscribed bool   `json:"subscribed"`
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
			} `json:"stash"`
		} `json:"updateStashSettings"`
		UpdateIngestSettings struct {
			Ingest struct {
				DeliveryMode string `json:"deliveryMode"`
				Downloads    struct {
					QBRoot   string `json:"qbRoot"`
					MojiRoot string `json:"mojiRoot"`
				} `json:"downloads"`
				Library struct {
					MojiRoot  string `json:"mojiRoot"`
					StashRoot string `json:"stashRoot"`
				} `json:"library"`
				Transfer struct {
					Action string `json:"action"`
				} `json:"transfer"`
			} `json:"ingest"`
		} `json:"updateIngestSettings"`
		UpdateJackettSettings struct {
			Jackett struct {
				URL string `json:"url"`
			} `json:"jackett"`
		} `json:"updateJackettSettings"`
		UpdateAutomationSettings struct {
			Automation struct {
				TaskProgressSyncIntervalSeconds int      `json:"taskProgressSyncIntervalSeconds"`
				SubscriptionPollIntervalHours   int      `json:"subscriptionPollIntervalHours"`
				StashBoxEndpoints               []string `json:"stashBoxEndpoints"`
				TorrentSelection                struct {
					Enabled bool `json:"enabled"`
					Rules   []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Type string `json:"type"`
					} `json:"rules"`
				} `json:"torrentSelection"`
			} `json:"automation"`
		} `json:"updateAutomationSettings"`
		UpdateSystemSettings struct {
			System struct {
				TaskDeletePolicy string `json:"taskDeletePolicy"`
			} `json:"system"`
		} `json:"updateSystemSettings"`
		QbittorrentAdd bool `json:"qbittorrentAdd"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type fakeSettingsEditor struct {
	snapshot                  *SettingsSnapshot
	statusSnapshot            *SettingsStatusSnapshot
	stashInput                UpdateStashSettingsInput
	updateStashSnapshot       *SettingsSnapshot
	ingestInput               UpdateIngestSettingsInput
	updateIngestSnapshot      *SettingsSnapshot
	jackettInput              UpdateJackettSettingsInput
	updateJackettSnapshot     *SettingsSnapshot
	qbittorrentInput          UpdateQBittorrentSettingsInput
	updateQBittorrentSnapshot *SettingsSnapshot
	automationInput           UpdateAutomationSettingsInput
	updateAutomationSnapshot  *SettingsSnapshot
	systemInput               UpdateSystemSettingsInput
	updateSystemSnapshot      *SettingsSnapshot
}

func (f *fakeSettingsEditor) Snapshot() *SettingsSnapshot {
	return f.snapshot
}

func (f *fakeSettingsEditor) StatusSnapshot() *SettingsStatusSnapshot {
	return f.statusSnapshot
}

func (f *fakeSettingsEditor) UpdateStashSettings(input UpdateStashSettingsInput) (*SettingsSnapshot, error) {
	f.stashInput = input
	return f.updateStashSnapshot, nil
}

func (f *fakeSettingsEditor) UpdateIngestSettings(input UpdateIngestSettingsInput) (*SettingsSnapshot, error) {
	f.ingestInput = input
	return f.updateIngestSnapshot, nil
}

func (f *fakeSettingsEditor) UpdateJackettSettings(input UpdateJackettSettingsInput) (*SettingsSnapshot, error) {
	f.jackettInput = input
	if f.updateJackettSnapshot != nil {
		return f.updateJackettSnapshot, nil
	}
	return f.snapshot, nil
}

func (f *fakeSettingsEditor) UpdateQBittorrentSettings(input UpdateQBittorrentSettingsInput) (*SettingsSnapshot, error) {
	f.qbittorrentInput = input
	return f.updateQBittorrentSnapshot, nil
}

func (f *fakeSettingsEditor) UpdateAutomationSettings(input UpdateAutomationSettingsInput) (*SettingsSnapshot, error) {
	f.automationInput = input
	return f.updateAutomationSnapshot, nil
}

func (f *fakeSettingsEditor) UpdateSystemSettings(input UpdateSystemSettingsInput) (*SettingsSnapshot, error) {
	f.systemInput = input
	return f.updateSystemSnapshot, nil
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

type fakeSubscriptionService struct {
	performers    []subscription.Performer
	discovered    subscription.DiscoverScenePage
	detail        subscription.PerformerDetail
	performerPage subscription.PerformerScenePage
	queueTask     *downloader.Task
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

func (f *fakeSubscriptionService) ListStashPerformers(_ context.Context, _ string) ([]subscription.Performer, error) {
	return f.performers, nil
}

func (f *fakeSubscriptionService) SearchPreferredStashBoxScenes(context.Context, string, int, subscription.DiscoverSort) (subscription.DiscoverScenePage, error) {
	return f.discovered, nil
}

func (f *fakeSubscriptionService) QueueDiscoveredScene(context.Context, string, string) (*downloader.Task, error) {
	return f.queueTask, nil
}

func (f *fakeSubscriptionService) ListSubscribedPerformers(context.Context) ([]subscription.SubscribedPerformer, error) {
	return nil, nil
}

func (f *fakeSubscriptionService) GetPerformerDetail(context.Context, string) (subscription.PerformerDetail, error) {
	return f.detail, nil
}

func (f *fakeSubscriptionService) ListPerformerScenes(context.Context, string, subscription.PerformerSceneQuery) (subscription.PerformerScenePage, error) {
	return f.performerPage, nil
}

func (f *fakeSubscriptionService) SubscribePerformer(context.Context, string) (subscription.SubscribedPerformer, error) {
	return subscription.SubscribedPerformer{}, nil
}

func (f *fakeSubscriptionService) UnsubscribePerformer(context.Context, string) error {
	return nil
}

func (f *fakeSubscriptionService) RefreshSubscribedPerformer(context.Context, string) (subscription.SubscribedPerformer, error) {
	return subscription.SubscribedPerformer{}, nil
}

func (f *fakeSubscriptionService) RefreshAll(context.Context) ([]subscription.SubscribedPerformer, error) {
	return nil, nil
}

func (f *fakeSubscriptionService) RefreshStashBoxes(context.Context) error {
	return nil
}

func (f *fakeSubscriptionService) SnapshotState() ([]subscription.StashBoxEndpoint, subscription.LoadState) {
	return nil, subscription.LoadState{}
}

func (fakeStashService) MetadataScan(context.Context, stashsync.ScanRequest) (string, error) {
	return "job-1", nil
}

func (fakeStashService) FindJob(context.Context, string) (*stashsync.Job, error) {
	return nil, nil
}

func (fakeStashService) CurrentConfig() stashsync.IntegrationConfig {
	return stashsync.IntegrationConfig{}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
