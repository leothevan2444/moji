package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func builtinPipelineTorrent(code string, sample bool) []byte {
	name := code + ".mp4"
	if sample {
		name = code + ".sample.mp4"
	}
	return []byte(fmt.Sprintf("d4:infod6:lengthi1e4:name%d:%see", len(name), name))
}

type builtinPipelineDownloader struct {
	t        *testing.T
	store    taskruntime.TaskStore
	sourceID string
	mu       sync.Mutex
	adds     map[string]int
}

func (d *builtinPipelineDownloader) AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error {
	if len(opts.Torrents) != 1 {
		return fmt.Errorf("expected one original torrent")
	}
	code := strings.TrimSuffix(opts.Torrents[0].Filename, ".torrent")
	task, err := d.store.FindByCode(ctx, code)
	if err != nil || task == nil {
		return fmt.Errorf("task must exist before qB: %s: %w", code, err)
	}
	want := builtinPipelineTorrent(code, false)
	if !bytes.Equal(task.TorrentFileData, want) || !bytes.Equal(opts.Torrents[0].Data, want) || task.Candidate.Source == nil || task.Candidate.Source.Kind != indexer.Builtin || task.Candidate.Source.InstanceID != d.sourceID || task.Candidate.Source.BundleRevision == "" {
		d.t.Error("builtin source and filtered original bytes must be durable before qB")
	}
	d.mu.Lock()
	d.adds[code]++
	d.mu.Unlock()
	return nil
}
func (*builtinPipelineDownloader) GetTorrentList(context.Context, *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return nil, nil
}
func (*builtinPipelineDownloader) DeleteTorrents(context.Context, []string, bool) error { return nil }

func TestBuiltinManagedPipelineFiltersAndPersistsBeforeDownloadUnderQueuedLoad(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	var requestMu sync.Mutex
	requests := map[string]int{}
	// This server is an explicitly configured proxy, not a relay. It answers
	// every request itself and never dials the synthetic public target.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Host != "1.1.1.1" || !r.URL.IsAbs() {
			t.Error("unexpected proxy destination or method")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("q")
		requestMu.Lock()
		requests[r.URL.Path+"?"+r.URL.RawQuery]++
		requestMu.Unlock()
		switch r.URL.Path {
		case "/search":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<article><h2>%s</h2><a href="/sample?q=%s">get</a></article><article><h2>%s</h2><a href="/torrent?q=%s">get</a></article>`, code, code, code, code)
		case "/sample", "/torrent":
			w.Header().Set("Content-Type", "application/x-bittorrent")
			w.Write(builtinPipelineTorrent(code, r.URL.Path == "/sample"))
		default:
			t.Error("unexpected builtin request path")
			http.NotFound(w, r)
		}
	}))
	defer proxy.Close()
	store := indexerTestStore(t)
	if _, err = store.ApplyPatch(map[string]any{
		"indexers": map[string]any{"jackett": map[string]any{"enabled": false}},
		"tasks":    map[string]any{"torrent_selection": map[string]any{"enabled": true, "torrent_rules": map[string]any{"torrent_file_name_match": map[string]any{"enabled": true, "clauses": []any{map[string]any{"pattern": "sample", "pattern_mode": "PLAIN", "effect": "EXCLUDE"}}}}}},
	}); err != nil {
		t.Fatal(err)
	}
	rule := strings.ReplaceAll(runtimeIndexerRule, "https://example.invalid/", "http://1.1.1.1/")
	rule = strings.Replace(rule, "paths: [{path: /}]", "paths: [{path: /search}]\n  inputs: {q: '{{ .Keywords }}'}", 1)
	source := &runtimeRuleSource{revision: strings.Repeat("a", 40), archive: runtimeRuleArchive(t, rule)}
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, source)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err = runtime.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = runtime.definitions.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err := runtime.definitions.Begin(ctx, "INSTALL", source.revision, "")
	if err != nil {
		t.Fatal(err)
	}
	if installed := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID); installed.State != indexerdefs.Completed {
		t.Fatal("rule installation failed", installed.ErrorCode)
	}
	profile := "synthetic-proxy"
	if _, err = runtime.admin.SaveNetworkProfile(ctx, store.Revision(), profile, true, config.IndexerNetworkProfilePatch{ProxyURL: &proxy.URL}); err != nil {
		t.Fatal(err)
	}
	definition, enabled := "example", true
	saved, err := runtime.admin.SaveBuiltin(ctx, "", store.Revision(), indexeradmin.BuiltinPatch{Definition: &definition, Enabled: &enabled, NetworkProfile: &profile})
	if err != nil {
		t.Fatal(err)
	}
	core := taskruntime.NewRelationalTaskStore(manager.Core, manager.Observations)
	downloader := &builtinPipelineDownloader{t: t, store: core, sourceID: saved.Instance.ID, adds: map[string]int{}}
	service := configureTaskRuntimeWithIndexers(store.Config(), store, nil, runtime.service, downloader, nil, taskruntime.NewTaskEventBus(8), manager)
	flow := taskflow.NewService(service.(taskflow.TaskRuntime))
	const count = 8
	var group sync.WaitGroup
	samples := make([]time.Duration, count)
	starts := make([]time.Time, count)
	ids := make([]string, count)
	for i := range count {
		group.Go(func() {
			code := fmt.Sprintf("ABCD-%d", 100+i)
			starts[i] = time.Now()
			request, endRequest := context.WithCancel(ctx)
			defer endRequest()
			task, err := flow.CreateFromSearchCode(request, taskflow.CreateFromSearchCodeInput{Code: code})
			if err != nil || task == nil || task.Stage != taskruntime.TaskStageSourcing || task.StageStatus != taskruntime.TaskStageStatusPending {
				t.Errorf("builtin task %s was not durably queued: %+v %v", code, task, err)
				return
			}
			ids[i] = task.ID
		})
	}
	group.Wait()
	if t.Failed() {
		return
	}
	if len(requests) != 0 || len(downloader.adds) != 0 {
		t.Fatal("taskflow admission performed external work before the worker ran")
	}
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.(interface{ RunSourcingWorker(context.Context) }).RunSourcingWorker(workerCtx)
	}()
	defer func() { stop(); <-done }()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for completed := 0; completed < count; {
		for i, id := range ids {
			if samples[i] != 0 {
				continue
			}
			task, err := core.Find(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if task.Stage == taskruntime.TaskStageDownloading {
				samples[i] = time.Since(starts[i])
				completed++
			} else if task.StageStatus == taskruntime.TaskStageStatusBlocked {
				t.Fatalf("builtin task %s blocked: %s", task.Code, task.StageErrorCode)
			}
		}
		if completed == count {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("builtin queued pipeline did not finish", ctx.Err())
		case <-tick.C:
		}
	}
	stop()
	<-done
	for i := range count {
		code := fmt.Sprintf("ABCD-%d", 100+i)
		if downloader.adds[code] != 1 {
			t.Errorf("%s: qB submissions = %d", code, downloader.adds[code])
		}
		for _, path := range []string{"/search", "/sample", "/torrent"} {
			if requests[path+"?q="+code] != 1 {
				t.Errorf("%s: %s requests = %d", code, path, requests[path+"?q="+code])
			}
		}
	}
	if len(requests) != count*3 || source.downloads.Load() != 1 {
		t.Fatal("pipeline repeated or omitted network work", len(requests), source.downloads.Load())
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	t.Logf("8 concurrently admitted builtin tasks through production 2-worker queue: p50=%s p95=%s, HTTP=%d, qB=%d (synthetic local proxy, real pacing, includes queue/poll latency)", samples[3], samples[7], len(requests), count)
}
