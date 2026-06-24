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
			DeliveryMode:     "PATH_MAP",
			StashLibraryPath: "/library",
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
			ingest { deliveryMode stashLibraryPath }
			jackett { configured url apiKeyConfigured }
			qbittorrent { configured url username usernameConfigured passwordConfigured defaultSavePath category tags }
			automation { taskProgressSyncIntervalSeconds subscriptionPollIntervalHours }
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
	if resp.Data.Settings.Ingest.StashLibraryPath != "/library" {
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
			},
			Subscription: SubscriptionSettingsSnapshot{
				StashBoxEndpoints: []string{"https://javstash.example.org/graphql"},
			},
			QBittorrent: QBittorrentSettingsSnapshot{
				URL:      "http://qb.invalid",
				Username: "editor-user",
			},
		},
		statusSnapshot: &SettingsStatusSnapshot{
			Automation: AutomationStatusSnapshot{
				SubscriptionPollIntervalHours: 1,
				SubscriptionPollEnabled:       true,
			},
		},
	}

	resp := executeGraphQL(t, resolver, `{ settings { subscription { stashBoxEndpoints } qbittorrent { url username } automation { subscriptionPollIntervalHours } } settingsStatus { automation { subscriptionPollIntervalHours subscriptionPollEnabled } } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Settings.Subscription.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected subscription settings: %+v", resp.Data.Settings.Subscription)
	}
	if resp.Data.Settings.Qbittorrent.Username != "editor-user" {
		t.Fatalf("unexpected qbittorrent settings: %+v", resp.Data.Settings.Qbittorrent)
	}
	if !resp.Data.SettingsStatus.Automation.SubscriptionPollEnabled || resp.Data.SettingsStatus.Automation.SubscriptionPollIntervalHours != 1 {
		t.Fatalf("unexpected automation status: %+v", resp.Data.SettingsStatus.Automation)
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
				DeliveryMode:     "TRANSFER",
				StashLibraryPath: "/library",
				Transfer: TransferIngestSettingsSnapshot{
					Action:         "COPY",
					MojiSourceRoot: "/downloads",
					MojiTargetRoot: "/mnt/library",
				},
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateIngestSettings(input: {
			deliveryMode: "TRANSFER"
			stashLibraryPath: "/library"
			transfer: {
				action: "COPY"
				mojiSourceRoot: "/downloads"
				mojiTargetRoot: "/mnt/library"
			}
		}) {
			ingest { deliveryMode stashLibraryPath transfer { action mojiSourceRoot mojiTargetRoot } }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if editor.ingestInput.DeliveryMode != "TRANSFER" || editor.ingestInput.Transfer.MojiTargetRoot != "/mnt/library" {
		t.Fatalf("unexpected ingest input: %+v", editor.ingestInput)
	}
	if resp.Data.UpdateIngestSettings.Ingest.DeliveryMode != "TRANSFER" {
		t.Fatalf("unexpected ingest response: %+v", resp.Data.UpdateIngestSettings.Ingest)
	}
}

func TestUpdateSubscriptionSettingsMutation(t *testing.T) {
	editor := &fakeSettingsEditor{
		updateSubscriptionSnapshot: &SettingsSnapshot{
			Subscription: SubscriptionSettingsSnapshot{
				StashBoxEndpoints: []string{"https://javstash.example.org/graphql"},
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.SettingsEditor = editor

	resp := executeGraphQL(t, resolver, `mutation {
		updateSubscriptionSettings(input: {
			stashBoxEndpoints: ["https://javstash.example.org/graphql"]
		}) {
			subscription {
				stashBoxEndpoints
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(editor.subscriptionInput.StashBoxEndpoints) != 1 ||
		editor.subscriptionInput.StashBoxEndpoints[0] != "https://javstash.example.org/graphql" {
		t.Fatalf("unexpected selected endpoints: %+v", editor.subscriptionInput.StashBoxEndpoints)
	}
	if len(resp.Data.UpdateSubscriptionSettings.Subscription.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected subscription response: %+v", resp.Data.UpdateSubscriptionSettings.Subscription)
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
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
			} `json:"stash"`
			Ingest struct {
				DeliveryMode     string `json:"deliveryMode"`
				StashLibraryPath string `json:"stashLibraryPath"`
				Transfer         struct {
					Action         string `json:"action"`
					MojiSourceRoot string `json:"mojiSourceRoot"`
					MojiTargetRoot string `json:"mojiTargetRoot"`
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
			Subscription struct {
				StashBoxEndpoints []string `json:"stashBoxEndpoints"`
			} `json:"subscription"`
			Automation struct {
				TaskProgressSyncIntervalSeconds int `json:"taskProgressSyncIntervalSeconds"`
				SubscriptionPollIntervalHours   int `json:"subscriptionPollIntervalHours"`
			} `json:"automation"`
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
				DeliveryMode     string `json:"deliveryMode"`
				StashLibraryPath string `json:"stashLibraryPath"`
				Transfer         struct {
					Action         string `json:"action"`
					MojiSourceRoot string `json:"mojiSourceRoot"`
					MojiTargetRoot string `json:"mojiTargetRoot"`
				} `json:"transfer"`
			} `json:"ingest"`
		} `json:"updateIngestSettings"`
		UpdateSubscriptionSettings struct {
			Subscription struct {
				StashBoxEndpoints []string `json:"stashBoxEndpoints"`
			} `json:"subscription"`
		} `json:"updateSubscriptionSettings"`
		QbittorrentAdd bool `json:"qbittorrentAdd"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type fakeSettingsEditor struct {
	snapshot                   *SettingsSnapshot
	statusSnapshot             *SettingsStatusSnapshot
	stashInput                 UpdateStashSettingsInput
	updateStashSnapshot        *SettingsSnapshot
	ingestInput                UpdateIngestSettingsInput
	updateIngestSnapshot       *SettingsSnapshot
	qbittorrentInput           UpdateQBittorrentSettingsInput
	updateQBittorrentSnapshot  *SettingsSnapshot
	automationInput            UpdateAutomationSettingsInput
	updateAutomationSnapshot   *SettingsSnapshot
	subscriptionInput          UpdateSubscriptionSettingsInput
	updateSubscriptionSnapshot *SettingsSnapshot
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

func (f *fakeSettingsEditor) UpdateJackettSettings(UpdateJackettSettingsInput) (*SettingsSnapshot, error) {
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

func (f *fakeSettingsEditor) UpdateSubscriptionSettings(input UpdateSubscriptionSettingsInput) (*SettingsSnapshot, error) {
	f.subscriptionInput = input
	return f.updateSubscriptionSnapshot, nil
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
	detail        subscription.PerformerDetail
	performerPage subscription.PerformerScenePage
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
