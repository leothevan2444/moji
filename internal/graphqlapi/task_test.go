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
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/subscription"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestDownloadMediaCreatesTask(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		downloadTask: &taskruntime.Task{
			ID:          "task-1",
			Code:        "ABCD-123",
			Stage:       taskruntime.TaskStageDownloading,
			StageStatus: taskruntime.TaskStageStatusRunning,
			TorrentURL:  "magnet:?xt=urn:btih:test",
			Candidate: taskruntime.Candidate{
				Title:   "ABCD-123",
				Tracker: "demo",
				Seeders: 5,
			},
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		downloadMedia(input: { code: "ABCD-123", limit: 1 }) {
			id
			code
			stage
			stageStatus
			torrentUrl
			candidate { title tracker seeders }
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	task := resp.Data.DownloadMedia
	if task.ID != "task-1" || task.Code != "ABCD-123" || task.Stage != "DOWNLOADING" || task.StageStatus != "RUNNING" || task.Candidate.Title != "ABCD-123" {
		t.Fatalf("unexpected download task response: %+v", task)
	}
	if taskRuntime.downloadRequest.Code != "ABCD-123" || taskRuntime.downloadRequest.Limit != 1 {
		t.Fatalf("unexpected download request: %+v", taskRuntime.downloadRequest)
	}
	if string(taskRuntime.downloadRequest.Source) != "MANUAL" {
		t.Fatalf("expected manual task source, got %+v", taskRuntime.downloadRequest)
	}
}

func TestAddTorrentCreatesTask(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		addTask: &taskruntime.Task{
			ID:          "task-manual",
			Code:        "ABCD-123",
			Stage:       taskruntime.TaskStageDownloading,
			StageStatus: taskruntime.TaskStageStatusRunning,
			TorrentURL:  "magnet:?xt=urn:btih:manual",
			Candidate: taskruntime.Candidate{
				Title:     "magnet:?xt=urn:btih:manual",
				MagnetURI: "magnet:?xt=urn:btih:manual",
			},
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		addTorrent(input: { url: "magnet:?xt=urn:btih:manual", category: "moji" }) {
			id
			code
			stage
			stageStatus
			torrentUrl
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.AddTorrent.ID != "task-manual" || resp.Data.AddTorrent.Code != "ABCD-123" || resp.Data.AddTorrent.Stage != "DOWNLOADING" || resp.Data.AddTorrent.StageStatus != "RUNNING" {
		t.Fatalf("unexpected add torrent response: %+v", resp.Data.AddTorrent)
	}
	if taskRuntime.addRequest.URL != "magnet:?xt=urn:btih:manual" || taskRuntime.addRequest.Category != "moji" {
		t.Fatalf("unexpected add torrent request: %+v", taskRuntime.addRequest)
	}
	if string(taskRuntime.addRequest.Source) != "MANUAL" {
		t.Fatalf("expected manual task source, got %+v", taskRuntime.addRequest)
	}
}

func TestResolveBlockedSourcingTaskUsesSelectedCandidate(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		resolveSourcingTask: &taskruntime.Task{
			ID: "task-blocked", Code: "ABCD-123", Stage: taskruntime.TaskStageDownloading,
			StageStatus: taskruntime.TaskStageStatusRunning,
			Candidate:   taskruntime.Candidate{Title: "ABCD-123 selected", Tracker: "demo", InfoHash: "abc123"},
			TorrentURL:  "magnet:?xt=urn:btih:abc123", CreatedAt: time.Unix(100, 0), UpdatedAt: time.Unix(200, 0),
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")
	resp := executeGraphQL(t, resolver, `mutation {
		resolveBlockedSourcingTask(id: "task-blocked", input: {
			torrentUrl: "magnet:?xt=urn:btih:abc123"
			title: "ABCD-123 selected"
			tracker: "demo"
			infoHash: "abc123"
			seeders: 8
		}) { id stage stageStatus torrentUrl candidate { title tracker infoHash } }
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.ResolveBlockedSourcingTask.ID != "task-blocked" || resp.Data.ResolveBlockedSourcingTask.Stage != "DOWNLOADING" {
		t.Fatalf("unexpected response: %+v", resp.Data.ResolveBlockedSourcingTask)
	}
	if taskRuntime.resolveSourcingID != "task-blocked" || taskRuntime.resolveSourcingReq.Title != "ABCD-123 selected" || taskRuntime.resolveSourcingReq.Seeders != 8 {
		t.Fatalf("unexpected resolution request: %+v", taskRuntime.resolveSourcingReq)
	}
}

func TestBlockedTaskTorrentCandidatesSearchesTaskCode(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{findTask: &taskruntime.Task{
		ID: "task-blocked", Code: "ABCD-123", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusBlocked,
	}}
	trackerClient := &fakeGraphQLTracker{results: []jackett.SearchResult{{Title: "ABCD-123 candidate", Tracker: "demo", MagnetURI: "magnet:?xt=urn:btih:test"}}}
	resolver := NewResolver(trackerClient, nil, taskRuntime, nil, "test-version")
	resp := executeGraphQL(t, resolver, `query {
		blockedTaskTorrentCandidates(id: "task-blocked", limit: 20) { title tracker magnetUri }
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if trackerClient.query != "ABCD-123" || len(resp.Data.BlockedTaskTorrentCandidates) != 1 {
		t.Fatalf("unexpected candidate search: query=%q response=%+v", trackerClient.query, resp.Data.BlockedTaskTorrentCandidates)
	}
}

func TestPreviewJackettSelectionReturnsPreviewedResults(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		previewSelection: &taskruntime.CandidateSelectionPreview{
			Results: []jackett.SearchResult{
				{Title: "preferred", TrackerID: "alpha", Link: "https://example.test/a.torrent"},
				{Title: "fallback", TrackerID: "beta", Link: "https://example.test/b.torrent"},
			},
			Meta: taskruntime.CandidateSelectionPreviewMeta{
				AppliedFastRules: true,
				AppliedFileRules: false,
				InspectedCount:   0,
				InspectableCount: 0,
			},
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `query {
		previewJackettSelection(input: {
			query: "ABCD-123"
			applyFastRules: true
			applyFileRules: false
			results: [
				{
					title: "fallback"
					size: 1
					seeders: 1
					peers: 1
					tracker: "beta"
					trackerId: "beta"
					categoryDesc: ""
					publishDate: ""
					details: ""
					link: "https://example.test/b.torrent"
					magnetUri: ""
					infoHash: ""
				},
				{
					title: "preferred"
					size: 1
					seeders: 2
					peers: 1
					tracker: "alpha"
					trackerId: "alpha"
					categoryDesc: ""
					publishDate: ""
					details: ""
					link: "https://example.test/a.torrent"
					magnetUri: ""
					infoHash: ""
				}
			]
		}) {
			results { title trackerId }
			previewMeta { appliedFastRules appliedFileRules inspectedCount inspectableCount }
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if taskRuntime.previewRequest.Query != "ABCD-123" || !taskRuntime.previewRequest.ApplyFastRules || taskRuntime.previewRequest.ApplyFileRules {
		t.Fatalf("unexpected preview request: %+v", taskRuntime.previewRequest)
	}
	if len(resp.Data.PreviewJackettSelection.Results) != 2 || resp.Data.PreviewJackettSelection.Results[0].Title != "preferred" {
		t.Fatalf("unexpected preview response: %+v", resp.Data.PreviewJackettSelection)
	}
	if !resp.Data.PreviewJackettSelection.PreviewMeta.AppliedFastRules || resp.Data.PreviewJackettSelection.PreviewMeta.AppliedFileRules {
		t.Fatalf("unexpected preview meta: %+v", resp.Data.PreviewJackettSelection.PreviewMeta)
	}
}

func TestDeprecatedQBittorrentAddCreatesTask(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		addTask: &taskruntime.Task{
			ID:          "task-manual",
			Stage:       taskruntime.TaskStageDownloading,
			StageStatus: taskruntime.TaskStageStatusRunning,
			TorrentURL:  "magnet:?xt=urn:btih:manual",
			CreatedAt:   time.Unix(100, 0).UTC(),
			UpdatedAt:   time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		qbittorrentAdd(input: { url: "magnet:?xt=urn:btih:manual" })
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if !resp.Data.QbittorrentAdd {
		t.Fatal("expected qbittorrentAdd to return true")
	}
	if taskRuntime.addRequest.URL != "magnet:?xt=urn:btih:manual" {
		t.Fatalf("unexpected add torrent request: %+v", taskRuntime.addRequest)
	}
}

func TestTasksQueryListsTasks(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		listTasks: []*taskruntime.Task{
			{ID: "task-2", Code: "BBBB-222", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning, CreatedAt: time.Unix(200, 0).UTC(), UpdatedAt: time.Unix(200, 0).UTC()},
			{ID: "task-1", Code: "AAAA-111", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusBlocked, CreatedAt: time.Unix(100, 0).UTC(), UpdatedAt: time.Unix(100, 0).UTC()},
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{ tasks { id code stage } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Tasks) != 2 || resp.Data.Tasks[0].ID != "task-2" || resp.Data.Tasks[1].ID != "task-1" {
		t.Fatalf("unexpected tasks response: %+v", resp.Data.Tasks)
	}
}

func TestTasksQueryWithoutTaskRuntimeReturnsEmptyList(t *testing.T) {
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

func TestTasksQueryReturnsNullForUnsetOptionalFields(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		listTasks: []*taskruntime.Task{
			{ID: "task-1", Code: "AAAA-111", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusRunning, CreatedAt: time.Unix(100, 0).UTC(), UpdatedAt: time.Unix(100, 0).UTC()},
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	var resp struct {
		Data struct {
			Tasks []struct {
				ID             string  `json:"id"`
				SavePath       *string `json:"savePath"`
				StashScanJobID *string `json:"stashScanJobId"`
				TransferError  *string `json:"transferError"`
			} `json:"tasks"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	executeGraphQLInto(t, resolver, `{ tasks { id savePath stashScanJobId transferError } }`, &resp)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Tasks) != 1 {
		t.Fatalf("expected one task, got %+v", resp.Data.Tasks)
	}
	if resp.Data.Tasks[0].SavePath != nil || resp.Data.Tasks[0].StashScanJobID != nil || resp.Data.Tasks[0].TransferError != nil {
		t.Fatalf("expected unset optional task fields to serialize as null, got %+v", resp.Data.Tasks[0])
	}
}

func TestQueuePerformerScenesMutationMapsBatchResult(t *testing.T) {
	subscriptionService := &fakeSubscriptionService{
		queuePerformerResult: subscription.QueuePerformerScenesResult{
			QueuedTasks: []*taskruntime.Task{
				{ID: "task-1", Source: taskruntime.TaskSourceSearch, Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning},
			},
			Results: []subscription.QueuePerformerSceneResult{
				{
					Key:          "scene-a",
					Status:       subscription.QueuePerformerSceneStatusQueued,
					ReasonCode:   "QUEUED",
					Message:      "已创建下载任务",
					Task:         &taskruntime.Task{ID: "task-1", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning},
					ResolvedCode: "ABCD-123",
				},
				{
					Key:        "scene-b",
					Status:     subscription.QueuePerformerSceneStatusSkipped,
					ReasonCode: "ALREADY_IN_LIBRARY",
					Message:    "作品已在库中，跳过创建任务",
				},
			},
			Summary: subscription.QueuePerformerScenesSummary{
				RequestedCount: 2,
				QueuedCount:    1,
				SkippedCount:   1,
				FailedCount:    0,
			},
		},
	}
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	resolver.Subscription = subscriptionService

	var resp struct {
		Data struct {
			QueuePerformerScenes struct {
				QueuedTasks []struct {
					ID string `json:"id"`
				} `json:"queuedTasks"`
				Results []struct {
					Key          string  `json:"key"`
					Status       string  `json:"status"`
					ReasonCode   string  `json:"reasonCode"`
					Message      string  `json:"message"`
					ResolvedCode *string `json:"resolvedCode"`
					Task         *struct {
						ID string `json:"id"`
					} `json:"task"`
				} `json:"results"`
				Summary struct {
					RequestedCount int `json:"requestedCount"`
					QueuedCount    int `json:"queuedCount"`
					SkippedCount   int `json:"skippedCount"`
					FailedCount    int `json:"failedCount"`
				} `json:"summary"`
			} `json:"queuePerformerScenes"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	executeGraphQLInto(t, resolver, `mutation {
		queuePerformerScenes(input: {
			performerId: "p1"
			scenes: [
				{
					key: "scene-a"
					sourceSceneId: "scene-a"
					stashBoxSceneId: "scene-a"
					stashBoxEndpoint: "https://box.example/graphql"
					code: "ABCD-123"
					title: "Title A"
					inLibrary: false
				},
				{
					key: "scene-b"
					sourceSceneId: "scene-b"
					inLibrary: true
				}
			]
		}) {
			queuedTasks { id }
			results { key status reasonCode message resolvedCode task { id } }
			summary { requestedCount queuedCount skippedCount failedCount }
		}
	}`, &resp)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if subscriptionService.queuePerformerID != "p1" || len(subscriptionService.queueSelections) != 2 {
		t.Fatalf("unexpected service call: performer=%q selections=%+v", subscriptionService.queuePerformerID, subscriptionService.queueSelections)
	}
	if resp.Data.QueuePerformerScenes.Summary.RequestedCount != 2 || resp.Data.QueuePerformerScenes.Summary.QueuedCount != 1 || resp.Data.QueuePerformerScenes.Summary.SkippedCount != 1 {
		t.Fatalf("unexpected summary: %+v", resp.Data.QueuePerformerScenes.Summary)
	}
	if len(resp.Data.QueuePerformerScenes.Results) != 2 || resp.Data.QueuePerformerScenes.Results[0].Status != "QUEUED" || resp.Data.QueuePerformerScenes.Results[1].ReasonCode != "ALREADY_IN_LIBRARY" {
		t.Fatalf("unexpected results: %+v", resp.Data.QueuePerformerScenes.Results)
	}
}

func TestDeleteTaskMutation(t *testing.T) {
	taskRuntime := &fakeTaskRuntime{
		deleteTask: &taskruntime.Task{
			ID:          "task-delete",
			Stage:       taskruntime.TaskStagePendingIngest,
			StageStatus: taskruntime.TaskStageStatusPending,
			CreatedAt:   time.Unix(100, 0).UTC(),
			UpdatedAt:   time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		deleteTask(id: "task-delete") {
			id
			stage
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if taskRuntime.deleteTaskID != "task-delete" {
		t.Fatalf("expected delete request for task-delete, got %q", taskRuntime.deleteTaskID)
	}
	if resp.Data.DeleteTask.ID != "task-delete" || resp.Data.DeleteTask.Stage != "PENDING_INGEST" {
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

func TestTaskQueryWithoutTaskRuntimeReturnsNull(t *testing.T) {
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
	taskRuntime := &fakeTaskRuntime{
		syncTasks: []*taskruntime.Task{
			{
				ID:               "task-sync",
				Code:             "ABCD-123",
				Stage:            taskruntime.TaskStageDownloading,
				StageStatus:      taskruntime.TaskStageStatusRunning,
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
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		syncTaskProgress {
			id
			stage
			stageStatus
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
	if task.ID != "task-sync" || task.Stage != "DOWNLOADING" || task.StageStatus != "RUNNING" || task.TorrentHash != "hash-sync" || task.Progress != 0.5 {
		t.Fatalf("unexpected synced task: %+v", task)
	}
}

func TestTriggerStashScans(t *testing.T) {
	dl := &fakeTaskRuntime{
		stashTasks: []*taskruntime.Task{
			{
				ID:                 "task-stash",
				Stage:              taskruntime.TaskStagePendingIngest,
				StageStatus:        taskruntime.TaskStageStatusPending,
				StashScanJobID:     "job-1",
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
			stashScanJobId
			stage
			stageStatus
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
	if task.StashScanJobID != "job-1" || task.Stage != "PENDING_INGEST" || task.StageStatus != "PENDING" {
		t.Fatalf("unexpected stash scan task: %+v", task)
	}
}

func TestTriggerTaskStashScan(t *testing.T) {
	dl := &fakeTaskRuntime{
		triggerTaskScanTask: &taskruntime.Task{
			ID:             "task-single",
			Stage:          taskruntime.TaskStagePendingIngest,
			StageStatus:    taskruntime.TaskStageStatusPending,
			StashScanJobID: "job-single",
			CreatedAt:      time.Unix(100, 0).UTC(),
			UpdatedAt:      time.Unix(300, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, dl, fakeStashService{}, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		triggerTaskStashScan(id: "task-single") {
			id
			stashScanJobId
			stage
			stageStatus
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if dl.triggerTaskScanID != "task-single" {
		t.Fatalf("unexpected trigger task id: %q", dl.triggerTaskScanID)
	}
	if resp.Data.TriggerTaskStashScan.ID != "task-single" || resp.Data.TriggerTaskStashScan.StashScanJobID != "job-single" {
		t.Fatalf("unexpected single stash scan response: %+v", resp.Data.TriggerTaskStashScan)
	}
}

func TestDownloadMediaRequiresTaskRuntime(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		downloadMedia(input: { code: "ABCD-123" }) { id }
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected taskRuntime configuration error")
	}
	if got := resp.Errors[0].Message; got != "task runtime is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestTriggerStashScansRequiresStash(t *testing.T) {
	resolver := NewResolver(nil, nil, &fakeTaskRuntime{}, nil, "test-version")

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
				FastRules: []TorrentSelectionRuleSnapshot{
					{
						Type:    "SEEDERS",
						Enabled: true,
						Seeders: DirectionRuleSnapshot{Direction: "DESC"},
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
			automation { taskProgressSyncIntervalSeconds subscriptionPollIntervalHours stashBoxEndpoints torrentSelection { enabled fastRules { type seeders { direction } } torrentRules { type } } }
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
	if !resp.Data.Settings.Automation.TorrentSelection.Enabled || len(resp.Data.Settings.Automation.TorrentSelection.FastRules) != 1 {
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
	taskRuntime := &fakeTaskRuntime{
		listTasks: []*taskruntime.Task{
			{ID: "task-1", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning},
			{ID: "task-2", Stage: taskruntime.TaskStageScanning, StageStatus: taskruntime.TaskStageStatusRunning, StashScanJobID: "job-2"},
			{ID: "task-3", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusBlocked},
			{ID: "task-4", Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone},
		},
	}
	resolver := NewResolver(nil, nil, taskRuntime, nil, "test-version")

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
	if stats.Total != 4 || stats.Active != 3 || stats.Completed != 1 || stats.Downloading != 1 || stats.PendingScans != 1 || stats.Failed != 1 {
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
					FastRules: []TorrentSelectionRuleSnapshot{
						{Type: "SEEDERS", Enabled: true, Seeders: DirectionRuleSnapshot{Direction: "DESC"}},
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

	resp := executeGraphQL(t, resolver, `{ settings { qbittorrent { url username } automation { subscriptionPollIntervalHours stashBoxEndpoints torrentSelection { enabled fastRules { type } torrentRules { type } } } system { taskDeletePolicy } } settingsStatus { automation { subscriptionPollIntervalHours subscriptionPollEnabled } } }`)
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
	if len(resp.Data.Settings.Automation.TorrentSelection.FastRules) != 1 {
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
					FastRules: []TorrentSelectionRuleSnapshot{
						{
							Type:    "INDEXER_PREFERENCE",
							Enabled: true,
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
				SubscriptionReleasePolicy: SubscriptionReleasePolicySnapshot{
					SoloBehavior:           "DOWNLOAD",
					GroupBehavior:          "REVIEW",
					CompilationBehavior:    "BLOCK",
					MaxGroupPerformerCount: 3,
					ReleaseDateRange:       "ALL",
				},
				TorrentSelection: TorrentSelectionSettingsSnapshot{
					Enabled:                  true,
					InspectionCandidateLimit: 5,
					FastRules:                []TorrentSelectionRuleSnapshot{{Type: "SEEDERS", Enabled: true, Seeders: DirectionRuleSnapshot{Direction: "DESC"}}},
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
				subscriptionReleasePolicy: {
					soloBehavior: DOWNLOAD
					groupBehavior: REVIEW
					compilationBehavior: BLOCK
					maxGroupPerformerCount: 3
					releaseDateRange: ALL
				}
				torrentSelection: {
					enabled: true
					inspectionCandidateLimit: 5
					fastRules: [{
						type: SEEDERS
						enabled: true
						seeders: { direction: DESC }
						indexerPreference: { trackerIds: [] }
						titleMatch: { clauses: [] }
					}]
					torrentRules: []
				}
		}) {
			automation {
				stashBoxEndpoints
				subscriptionReleasePolicy { soloBehavior groupBehavior compilationBehavior maxGroupPerformerCount releaseDateRange }
				torrentSelection { enabled fastRules { type } torrentRules { type } }
			}
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(editor.automationInput.StashBoxEndpoints) != 1 || editor.automationInput.StashBoxEndpoints[0] != "https://javstash.example.org/graphql" {
		t.Fatalf("unexpected automation input: %+v", editor.automationInput)
	}
	if len(editor.automationInput.TorrentSelection.FastRules) != 1 || editor.automationInput.TorrentSelection.FastRules[0].Type != "SEEDERS" {
		t.Fatalf("unexpected automation rule input: %+v", editor.automationInput.TorrentSelection.FastRules)
	}
	if len(resp.Data.UpdateAutomationSettings.Automation.StashBoxEndpoints) != 1 {
		t.Fatalf("unexpected automation response: %+v", resp.Data.UpdateAutomationSettings.Automation)
	}
	if len(resp.Data.UpdateAutomationSettings.Automation.TorrentSelection.FastRules) != 1 || resp.Data.UpdateAutomationSettings.Automation.TorrentSelection.FastRules[0].Type != "SEEDERS" {
		t.Fatalf("unexpected automation response rules: %+v", resp.Data.UpdateAutomationSettings.Automation.TorrentSelection.FastRules)
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

func TestUpdateSystemSettingsRejectsInvalidImageCacheBounds(t *testing.T) {
	for _, imageCache := range []string{
		`{ enabled: true, maxSizeMb: 63, retentionDays: 30 }`,
		`{ enabled: true, maxSizeMb: 1024, retentionDays: 366 }`,
	} {
		resolver := NewResolver(nil, nil, nil, nil, "test-version")
		resolver.SettingsEditor = &fakeSettingsEditor{}
		resp := executeGraphQL(t, resolver, `mutation {
			updateSystemSettings(input: {
				taskDeletePolicy: KEEP_ONLY
				imageCache: `+imageCache+`
			}) { system { taskDeletePolicy } }
		}`)
		if len(resp.Errors) == 0 {
			t.Fatalf("expected image cache validation error for %s", imageCache)
		}
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

type fakeTaskRuntime struct {
	addRequest          taskruntime.AddTorrentRequest
	downloadRequest     taskruntime.DownloadRequest
	previewRequest      taskruntime.PreviewJackettSelectionRequest
	addTask             *taskruntime.Task
	downloadTask        *taskruntime.Task
	previewSelection    *taskruntime.CandidateSelectionPreview
	findTask            *taskruntime.Task
	listTasks           []*taskruntime.Task
	deleteTaskID        string
	deleteTask          *taskruntime.Task
	syncTasks           []*taskruntime.Task
	stashTasks          []*taskruntime.Task
	triggerTaskScanID   string
	triggerTaskScanTask *taskruntime.Task
	resolveSourcingID   string
	resolveSourcingReq  taskruntime.ResolveBlockedSourcingRequest
	resolveSourcingTask *taskruntime.Task
}

type fakeGraphQLTracker struct {
	query   string
	results []jackett.SearchResult
}

func (f *fakeGraphQLTracker) Search(query string, _ ...tracker.SearchOption) ([]jackett.SearchResult, error) {
	f.query = query
	return f.results, nil
}

func (f *fakeTaskRuntime) AddTorrentContext(_ context.Context, req taskruntime.AddTorrentRequest) (*taskruntime.Task, error) {
	f.addRequest = req
	return f.addTask, nil
}

func (f *fakeTaskRuntime) DownloadMediaContext(_ context.Context, req taskruntime.DownloadRequest) (*taskruntime.Task, error) {
	f.downloadRequest = req
	return f.downloadTask, nil
}

func (f *fakeTaskRuntime) PreviewJackettSelectionContext(_ context.Context, req taskruntime.PreviewJackettSelectionRequest) (*taskruntime.CandidateSelectionPreview, error) {
	f.previewRequest = req
	return f.previewSelection, nil
}

func (f *fakeTaskRuntime) FindTask(_ context.Context, _ string) (*taskruntime.Task, error) {
	return f.findTask, nil
}

func (f *fakeTaskRuntime) ListTasks(_ context.Context) ([]*taskruntime.Task, error) {
	return f.listTasks, nil
}

func (f *fakeTaskRuntime) DeleteTask(_ context.Context, id string) (*taskruntime.Task, error) {
	f.deleteTaskID = id
	return f.deleteTask, nil
}

func (f *fakeTaskRuntime) RetryTask(_ context.Context, _ string, _ taskruntime.StashScanner) (*taskruntime.Task, error) {
	return nil, nil
}

func (f *fakeTaskRuntime) ResolveBlockedSourcingTask(_ context.Context, id string, req taskruntime.ResolveBlockedSourcingRequest) (*taskruntime.Task, error) {
	f.resolveSourcingID = id
	f.resolveSourcingReq = req
	return f.resolveSourcingTask, nil
}

func (f *fakeTaskRuntime) SyncProgress(_ context.Context) ([]*taskruntime.Task, error) {
	return f.syncTasks, nil
}

func (f *fakeTaskRuntime) TriggerTaskStashScan(_ context.Context, id string, _ taskruntime.StashScanner) (*taskruntime.Task, error) {
	f.triggerTaskScanID = id
	return f.triggerTaskScanTask, nil
}

func (f *fakeTaskRuntime) TriggerStashScans(_ context.Context, _ taskruntime.StashScanner) ([]*taskruntime.Task, error) {
	return f.stashTasks, nil
}

type graphQLTaskResponse struct {
	Data struct {
		AddTorrent struct {
			ID          string `json:"id"`
			Code        string `json:"code"`
			Stage       string `json:"stage"`
			StageStatus string `json:"stageStatus"`
			TorrentURL  string `json:"torrentUrl"`
		} `json:"addTorrent"`
		DownloadMedia struct {
			ID          string `json:"id"`
			Code        string `json:"code"`
			Stage       string `json:"stage"`
			StageStatus string `json:"stageStatus"`
			TorrentURL  string `json:"torrentUrl"`
			Candidate   struct {
				Title   string `json:"title"`
				Tracker string `json:"tracker"`
				Seeders int    `json:"seeders"`
			} `json:"candidate"`
		} `json:"downloadMedia"`
		PreviewJackettSelection struct {
			Results []struct {
				Title     string `json:"title"`
				TrackerID string `json:"trackerId"`
			} `json:"results"`
			PreviewMeta struct {
				AppliedFastRules bool `json:"appliedFastRules"`
				AppliedFileRules bool `json:"appliedFileRules"`
				InspectedCount   int  `json:"inspectedCount"`
				InspectableCount int  `json:"inspectableCount"`
			} `json:"previewMeta"`
		} `json:"previewJackettSelection"`
		Tasks []struct {
			ID    string `json:"id"`
			Code  string `json:"code"`
			Stage string `json:"stage"`
		} `json:"tasks"`
		Task *struct {
			ID string `json:"id"`
		} `json:"task"`
		DeleteTask struct {
			ID    string `json:"id"`
			Stage string `json:"stage"`
		} `json:"deleteTask"`
		ResolveBlockedSourcingTask struct {
			ID          string `json:"id"`
			Stage       string `json:"stage"`
			StageStatus string `json:"stageStatus"`
			TorrentURL  string `json:"torrentUrl"`
		} `json:"resolveBlockedSourcingTask"`
		BlockedTaskTorrentCandidates []struct {
			Title     string `json:"title"`
			Tracker   string `json:"tracker"`
			MagnetURI string `json:"magnetUri"`
		} `json:"blockedTaskTorrentCandidates"`
		SyncTaskProgress []struct {
			ID               string  `json:"id"`
			Stage            string  `json:"stage"`
			StageStatus      string  `json:"stageStatus"`
			TorrentHash      string  `json:"torrentHash"`
			Progress         float64 `json:"progress"`
			QbittorrentState string  `json:"qbittorrentState"`
			ContentPath      string  `json:"contentPath"`
		} `json:"syncTaskProgress"`
		TriggerStashScans []struct {
			ID                 string  `json:"id"`
			StashScanJobID     string  `json:"stashScanJobId"`
			Stage              string  `json:"stage"`
			StageStatus        string  `json:"stageStatus"`
			StashScanStartedAt *string `json:"stashScanStartedAt"`
		} `json:"triggerStashScans"`
		TriggerTaskStashScan struct {
			ID             string `json:"id"`
			StashScanJobID string `json:"stashScanJobId"`
			Stage          string `json:"stage"`
			StageStatus    string `json:"stageStatus"`
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
					Enabled   bool `json:"enabled"`
					FastRules []struct {
						Type string `json:"type"`
					} `json:"fastRules"`
					TorrentRules []struct {
						Type string `json:"type"`
					} `json:"torrentRules"`
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
					Enabled   bool `json:"enabled"`
					FastRules []struct {
						Type string `json:"type"`
					} `json:"fastRules"`
					TorrentRules []struct {
						Type string `json:"type"`
					} `json:"torrentRules"`
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
	var resp graphQLTaskResponse
	executeGraphQLInto(t, resolver, query, &resp)
	return resp
}

func executeGraphQLInto(t *testing.T, resolver *Resolver, query string, target any) {
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

	if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

type fakeStashService struct{}

type fakeSubscriptionService struct {
	performers           []subscription.Performer
	discovered           subscription.DiscoverScenePage
	detail               subscription.PerformerDetail
	performerPage        subscription.PerformerScenePage
	queueTask            *taskruntime.Task
	queuePerformerResult subscription.QueuePerformerScenesResult
	queuePerformerID     string
	queueSelections      []subscription.QueuePerformerSceneSelection
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

func (f *fakeSubscriptionService) QueueDiscoveredScene(context.Context, string, string) (*taskruntime.Task, error) {
	return f.queueTask, nil
}

func (f *fakeSubscriptionService) QueuePerformerScenes(_ context.Context, performerID string, selections []subscription.QueuePerformerSceneSelection) (subscription.QueuePerformerScenesResult, error) {
	f.queuePerformerID = performerID
	f.queueSelections = append([]subscription.QueuePerformerSceneSelection(nil), selections...)
	return f.queuePerformerResult, nil
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
