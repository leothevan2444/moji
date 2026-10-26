package graphqlapi

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type apiDownloadClient struct {
	*qbittorrent.Client
	torrents []qbittorrent.Torrent
	paused   string
}

func (c *apiDownloadClient) GetTorrentList(context.Context, *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return c.torrents, nil
}
func (c *apiDownloadClient) GetDownloadServerState(context.Context) (*qbittorrent.DownloadServerState, error) {
	space := int64(5000)
	return &qbittorrent.DownloadServerState{FreeSpace: &space}, nil
}
func (c *apiDownloadClient) PauseTorrents(_ context.Context, hashes []string) error {
	c.paused = hashes[0]
	return nil
}
func TestDownloaderGraphQLFilteringPolicyAndCommands(t *testing.T) {
	ctx := context.Background()
	hash := strings.Repeat("a", 40)
	q := &apiDownloadClient{torrents: []qbittorrent.Torrent{{Hash: hash, Name: "ABCD-123", Progress: 1, State: qbittorrent.TorrentStateUploading, Ratio: 2.5, SeedingTime: 7200}}}
	tasks := &fakeTaskRuntime{listTasks: []*taskruntime.Task{{ID: "linked", Code: "ABCD-123", TorrentHash: hash, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone}, {ID: "missing", Code: "EFGH-456", TorrentHash: strings.Repeat("b", 40), Stage: taskruntime.TaskStageDownloading}}}
	service, err := downloader.New(q, tasks, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(nil, nil, tasks, nil, "test")
	resolver.Downloader = service
	var response struct {
		Data struct {
			Downloader             model.DownloaderOverview
			ControlTorrent         bool
			UpdateDownloaderPolicy model.DownloaderPolicy
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `query {downloader(taskId:"linked") {total attentionCount stale freeSpace torrents {hash taskIds ratio seedingSeconds present} policy {enabled}}}`, &response)
	v := response.Data.Downloader
	if len(response.Errors) > 0 || v.Total != 1 || v.AttentionCount != 0 || v.Stale || v.FreeSpace == nil || len(v.Torrents) != 1 || v.Torrents[0].Ratio != 2.5 || v.Torrents[0].SeedingSeconds != 7200 {
		t.Fatalf("wrong contract: %+v", response)
	}
	executeGraphQLInto(t, resolver, `query {downloader(attentionOnly:true) {total torrents {state taskIds}}}`, &response)
	if response.Data.Downloader.Total != 0 || response.Data.Downloader.AttentionCount != 0 {
		t.Fatalf("wrong issue filtering: %+v", response)
	}
	executeGraphQLInto(t, resolver, `mutation {updateDownloaderPolicy(input:{enabled:true,ratio:3,seedHours:24,matchAll:true,deleteFiles:false}) {enabled ratio seedHours matchAll deleteFiles}}`, &response)
	if len(response.Errors) > 0 || !response.Data.UpdateDownloaderPolicy.MatchAll || response.Data.UpdateDownloaderPolicy.Ratio != 3 {
		t.Fatalf("policy failed: %+v", response)
	}
	executeGraphQLInto(t, resolver, `mutation {controlTorrent(hash:"`+hash+`",action:PAUSE)}`, &response)
	if len(response.Errors) > 0 || !response.Data.ControlTorrent || q.paused != hash {
		t.Fatalf("command failed: %+v", response)
	}
	if classifyError(downloader.ErrUnsafeFiles) != "TORRENT_FILES_PROTECTED" {
		t.Fatal("unsafe removal should be actionable")
	}
}

func TestDownloaderCompletedMissingSourcesAreExcludedFromAttention(t *testing.T) {
	ctx := context.Background()
	done, active := strings.Repeat("c", 40), strings.Repeat("d", 40)
	q := &apiDownloadClient{torrents: []qbittorrent.Torrent{{Hash: done, Name: "ABCD-123", State: qbittorrent.TorrentStateMissingFiles}}}
	tasks := &fakeTaskRuntime{listTasks: []*taskruntime.Task{{ID: "done", TorrentHash: done, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone}, {ID: "active", TorrentHash: active, Stage: taskruntime.TaskStageDownloading}}}
	service, err := downloader.New(q, tasks, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(nil, nil, tasks, nil, "test")
	r.Downloader = service
	var result struct {
		Data   struct{ Downloader model.DownloaderOverview }
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, r, `{downloader(taskId:"done"){attentionCount total torrents {state needsAttention allCompleted} policy {deleteFiles}}}`, &result)
	v := result.Data.Downloader
	if len(result.Errors) > 0 || v.AttentionCount != 0 || v.Total != 1 || v.Torrents[0].NeedsAttention || !v.Policy.DeleteFiles {
		t.Fatalf("completed source classified incorrectly: %+v", result)
	}
	q.torrents = nil
	if err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	executeGraphQLInto(t, r, `{downloader(attentionOnly:true){attentionCount total torrents {taskIds state needsAttention}}}`, &result)
	v = result.Data.Downloader
	if v.AttentionCount != 0 || v.Total != 0 {
		t.Fatalf("missing task observations leaked into torrent list: %+v", result)
	}
	executeGraphQLInto(t, r, `{downloader(taskId:"active"){attentionCount total torrents {taskIds state needsAttention}}}`, &result)
	v = result.Data.Downloader
	if v.AttentionCount != 1 || v.Total != 1 || v.Torrents[0].State != "missing" {
		t.Fatalf("task detail lost its missing source: %+v", result)
	}
}

func TestDownloaderListUsesCurrentQBTorrentsBeforeFilteringAndPagination(t *testing.T) {
	ctx := context.Background()
	matched, orphan, unlinked := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	q := &apiDownloadClient{torrents: []qbittorrent.Torrent{{Hash: matched, Name: "Matched seed", State: qbittorrent.TorrentStateDownloading}, {Hash: unlinked, Name: "Independent seed", State: qbittorrent.TorrentStateError}}}
	tasks := &fakeTaskRuntime{listTasks: []*taskruntime.Task{{ID: "match", Code: "ABCD-123", TorrentHash: matched, Stage: taskruntime.TaskStageDownloading}, {ID: "orphan", Code: "EFGH-456", TorrentHash: orphan, Stage: taskruntime.TaskStageDownloading}}}
	service, err := downloader.New(q, tasks, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(nil, nil, tasks, nil, "test")
	resolver.Downloader = service
	var result struct {
		Data   struct{ Downloader model.DownloaderOverview }
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `{downloader {total attentionCount torrents {hash taskIds taskCodes present}}}`, &result)
	v := result.Data.Downloader
	if len(result.Errors) > 0 || v.Total != 2 || v.AttentionCount != 1 || len(v.Torrents) != 2 || v.Torrents[0].TaskIds[0] != "match" || len(v.Torrents[1].TaskIds) != 0 {
		t.Fatalf("qB list and task association diverged: %+v", result)
	}
	executeGraphQLInto(t, resolver, `{downloader(offset:1,limit:1){total torrents {hash taskIds}}}`, &result)
	if result.Data.Downloader.Total != 2 || result.Data.Downloader.Torrents[0].Hash != unlinked {
		t.Fatalf("pagination included missing task: %+v", result)
	}
	executeGraphQLInto(t, resolver, `{downloader(search:"EFGH-456"){total torrents {hash}}}`, &result)
	if result.Data.Downloader.Total != 0 {
		t.Fatal("orphan task was searchable in qB list")
	}
	q.torrents = q.torrents[1:]
	if err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	executeGraphQLInto(t, resolver, `{downloader{total torrents{hash}}}`, &result)
	if result.Data.Downloader.Total != 1 || result.Data.Downloader.Torrents[0].Hash != unlinked {
		t.Fatal("removed seed remained in the list")
	}
	executeGraphQLInto(t, resolver, `{downloader(taskId:"match"){total torrents{state present}}}`, &result)
	if result.Data.Downloader.Total != 1 || result.Data.Downloader.Torrents[0].Present {
		t.Fatal("task monitoring stopped after qB removal")
	}
}
