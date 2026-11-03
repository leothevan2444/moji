package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/controller/api"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/torrentsearch"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

const runtimeIndexerRule = `id: example
name: Example
description: Synthetic application fixture
language: en-US
type: public
encoding: UTF-8
links: [https://example.invalid/]
caps:
  categories: {Movies: Movies}
  modes: {search: [q]}
search:
  paths: [{path: /}]
  rows: {selector: article}
  fields:
    title: {selector: h2}
    category: {text: Movies}
    size: {text: 1GB}
    seeders: {text: 1}
    download: {selector: a, attribute: href}
`

func TestRuntimePublishesCommittedIndexerConfigurationChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := indexerTestStore(t)
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, &runtimeRuleSource{})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err = runtime.Start(ctx); err != nil {
		t.Fatal(err)
	}
	events, stop, err := runtime.events.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	<-events
	done := make(chan struct{})
	go func() { defer close(done); runtime.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("configuration worker did not stop")
		}
	}()
	wait := func() {
		t.Helper()
		select {
		case event := <-events:
			if event.Kind != indexer.EventConfiguration {
				t.Fatal("wrong invalidation kind", event.Kind)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("committed configuration did not invalidate indexer snapshots")
		}
	}
	wait()
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); err != nil {
		t.Fatal(err)
	}
	wait()
	snapshot, err := runtime.admin.Summary(ctx)
	if err != nil || snapshot.Jackett != nil || snapshot.Total == nil || *snapshot.Total != 0 {
		t.Fatal("event preceded committed source configuration", err)
	}
}

type runtimeRuleSource struct {
	revision          string
	archive           []byte
	checks, downloads atomic.Int32
}

func (s *runtimeRuleSource) Latest(context.Context, string) (indexerdefs.Revision, bool, error) {
	s.checks.Add(1)
	return indexerdefs.Revision{Commit: s.revision}, false, nil
}
func (s *runtimeRuleSource) Download(ctx context.Context, revision string, w io.Writer) error {
	s.downloads.Add(1)
	_, err := w.Write(s.archive)
	return err
}
func runtimeRuleArchive(t *testing.T, rule string) []byte {
	t.Helper()
	if _, err := cardigann.Load([]byte(rule)); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	tw := tar.NewWriter(gz)
	for _, entry := range []struct{ name, body string }{{"schema.json", "{}"}, {"example.yml", rule}} {
		if err := tw.WriteHeader(&tar.Header{Name: "Indexers-fixed/definitions/v11/" + entry.name, Mode: 0600, Size: int64(len(entry.body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
func awaitRuntimeRuleJob(t *testing.T, ctx context.Context, service *indexerdefs.Service, id string) *indexerdefs.Job {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		job, err := service.Job(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if job.State.Terminal() {
			return job
		}
		select {
		case <-timer.C:
			t.Fatal("definition job did not finish")
		case <-ticker.C:
		}
	}
}

type runtimeIndexedDownloader struct {
	store taskruntime.TaskStore
	t     *testing.T
	data  []byte
	adds  int
}

func (d *runtimeIndexedDownloader) AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error {
	d.adds++
	task, err := d.store.FindByCode(ctx, "ABCD-123")
	if err != nil || !bytes.Equal(task.TorrentFileData, d.data) || task.Candidate.Source == nil || task.Candidate.Source.InstanceID != "jackett/one" {
		d.t.Error("source and acquired bytes were not durable before qB effect")
	}
	if len(opts.Torrents) != 1 || !bytes.Equal(opts.Torrents[0].Data, d.data) {
		d.t.Error("qB did not receive acquired bytes")
	}
	return nil
}
func (*runtimeIndexedDownloader) GetTorrentList(context.Context, *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return nil, nil
}
func (*runtimeIndexedDownloader) DeleteTorrents(context.Context, []string, bool) error { return nil }

func TestIndexerRuntimeInstallsRulesAndUsesUnifiedAcquisitionInManagedTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	data := []byte("d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee")
	var grabs atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.0/indexers/all/results/torznab/api":
			io.WriteString(w, `<indexers><indexer id="one" configured="true"><title>One</title></indexer></indexers>`)
		case "/api/v2.0/indexers/all/results":
			if r.URL.Query().Get("Tracker[]") != "one" {
				t.Error("search was not scoped to one site")
			}
			json.NewEncoder(w).Encode(map[string]any{"Results": []map[string]any{{"Title": "ABCD-123", "Link": server.URL + "/dl/one", "Size": 1}}, "Indexers": []map[string]any{{"ID": "one", "Status": 200}}})
		case "/dl/one":
			grabs.Add(1)
			w.Write(data)
		default:
			t.Errorf("unexpected management endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store := indexerTestStore(t)
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"url": server.URL}}}); err != nil {
		t.Fatal(err)
	}
	source := &runtimeRuleSource{revision: strings.Repeat("a", 40), archive: runtimeRuleArchive(t, runtimeIndexerRule)}
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, source)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err = runtime.Start(ctx); err != nil {
		t.Fatal(err)
	}
	runtime.checkDefinitions(ctx)
	if source.checks.Load() != 0 || source.downloads.Load() != 0 {
		t.Fatal("startup silently fetched an uninstalled library")
	}
	if _, err = runtime.definitions.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err := runtime.definitions.Begin(ctx, "INSTALL", source.revision, "")
	if err != nil {
		t.Fatal(err)
	}
	if result := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID); result.State != indexerdefs.Completed {
		t.Fatalf("install failed: %s", result.ErrorCode)
	}
	coreStore := taskruntime.NewRelationalTaskStore(manager.Core, manager.Observations)
	downloader := &runtimeIndexedDownloader{store: coreStore, t: t, data: data}
	service := configureTaskRuntimeWithIndexers(store.Config(), store, nil, runtime.service, downloader, nil, taskruntime.NewTaskEventBus(8), manager)
	task, err := service.DownloadMediaContext(ctx, taskruntime.DownloadRequest{Code: "ABCD-123", Trackers: []string{"one"}})
	if err != nil {
		t.Fatal(err)
	}
	if downloader.adds != 1 || grabs.Load() != 1 || task.Candidate.Source == nil {
		t.Fatal("unified pipeline did not reuse acquisition")
	}
	observed, err := runtime.admin.Instances(ctx)
	if err != nil || observed.HealthErrorCode != "" || len(observed.Items) != 1 || observed.Items[0].Health.Search == nil || observed.Items[0].Health.Fetch == nil || observed.Items[0].Health.Search.ErrorCode != "" || observed.Items[0].Health.Fetch.ErrorCode != "" {
		t.Fatal("managed search/acquisition did not publish persisted observations", err)
	}
	definition, enabled := "example", true
	registered, err := runtime.service.Search(ctx, indexer.Query{Keywords: "ABCD-123", InstanceIDs: []string{"jackett/one"}})
	if err != nil || len(registered.Releases) != 1 {
		t.Fatal("reference setup failed", err)
	}
	if _, err = runtime.admin.ClearSearchCache(ctx, ""); err != nil {
		t.Fatal(err)
	}
	lease, err := runtime.resources.Resolve(ctx, registered.Releases[0].Resource.ID)
	if err != nil {
		t.Fatal("cache clearing deleted a registered resource", err)
	}
	lease.Release()
	acquired, err := runtime.service.FetchReference(ctx, registered.Releases[0].Resource.ID)
	if err != nil || !bytes.Equal(acquired.TorrentBytes, data) || grabs.Load() != 1 {
		t.Fatal("another opaque reference or clearing search cache repeated the same acquisition", err)
	}
	persisted, err := coreStore.FindByCode(ctx, "ABCD-123")
	if err != nil || !bytes.Equal(persisted.TorrentFileData, data) || downloader.adds != 1 {
		t.Fatal("cache clearing altered durable accepted task data", err)
	}
	if _, err = runtime.admin.SaveBuiltin(ctx, "", store.Revision(), config.BuiltinInstancePatch{Definition: &definition, Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	source.revision = strings.Repeat("b", 40)
	source.archive = runtimeRuleArchive(t, runtimeIndexerRule+"settings:\n  - {name: passkey, type: password, label: Passkey}\n")
	if _, err = runtime.definitions.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err = runtime.definitions.Begin(ctx, "UPDATE", source.revision, strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	result := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID)
	status, err := runtime.definitions.Status(ctx)
	if err != nil || result.State != indexerdefs.Failed || status.State.Active != strings.Repeat("a", 40) || !status.Available {
		t.Fatal("incompatible update replaced usable rules", err)
	}
}

func TestPickedCandidateUsesTaskflowAndProductionWorkerAfterRuntimeReferenceRemoval(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "new-task"
		if existing {
			name = "existing-task"
		}
		t.Run(name, func(t *testing.T) { testPickedCandidatePipeline(t, existing, false) })
	}
}

func TestPreparedDownloadUsesProductionSourceAndCanBeReusedByTaskWorker(t *testing.T) {
	testPickedCandidatePipeline(t, false, true)
}

func testPickedCandidatePipeline(t *testing.T, existing, prepareDownload bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	data := []byte("d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee")
	var grabs atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.0/indexers/all/results/torznab/api":
			io.WriteString(w, `<indexers><indexer id="one" configured="true"><title>One</title></indexer></indexers>`)
		case "/api/v2.0/indexers/all/results":
			json.NewEncoder(w).Encode(map[string]any{"Results": []map[string]any{{"Title": "ABCD-123", "Link": server.URL + "/dl/one", "Size": 1}}, "Indexers": []map[string]any{{"ID": "one", "Status": 200}}})
		case "/dl/one":
			grabs.Add(1)
			w.Write(data)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	configStore := indexerTestStore(t)
	if _, err = configStore.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"url": server.URL}}}); err != nil {
		t.Fatal(err)
	}
	runtime, err := newIndexerRuntime(ctx, configStore.Config(), configStore, manager, &runtimeRuleSource{})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err = runtime.Start(ctx); err != nil {
		t.Fatal(err)
	}
	core := taskruntime.NewRelationalTaskStore(manager.Core, manager.Observations)
	downloader := &runtimeIndexedDownloader{store: core, t: t, data: data}
	events := taskruntime.NewTaskEventBus(8)
	subscription := events.Subscribe(ctx)
	service := configureTaskRuntimeWithIndexers(configStore.Config(), configStore, nil, runtime.service, downloader, nil, events, manager)
	flow := taskflow.NewService(service.(taskflow.TaskRuntime))
	var prior *taskruntime.Task
	taskInput := ""
	if existing {
		now := time.Now().UTC()
		prior = &taskruntime.Task{ID: "existing-task", Code: "ABCD-123", Source: taskruntime.TaskSourceSubscription, Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusBlocked, CreatedAt: now, UpdatedAt: now}
		if err = core.Create(ctx, prior); err != nil {
			t.Fatal(err)
		}
		taskInput = `,taskId:"existing-task"`
	}

	resolver := graphqlapi.NewResolver(nil, service, nil, "test")
	resolver.SearchRuns = runtime.searches
	resolver.TorrentDownloads = runtime.downloads
	resolver.TaskFlow = flow
	handler := graphqlapi.NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	execute := func(query string, target any) {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"query": query})
		request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatal("GraphQL status", response.Code)
		}
		var envelope struct {
			Data   json.RawMessage
			Errors []json.RawMessage
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || len(envelope.Errors) != 0 {
			t.Fatal("GraphQL pipeline", err, string(response.Body.Bytes()))
		}
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			t.Fatal(err)
		}
	}
	var accepted struct {
		Search struct{ ID, State string } `json:"startTorrentSearch"`
	}
	execute(fmt.Sprintf(`mutation{startTorrentSearch(input:{keywords:"ABCD-123",instanceIds:["jackett/one"]%s}){id state}}`, taskInput), &accepted)
	if accepted.Search.State != "QUEUED" || grabs.Load() != 0 {
		t.Fatal("GraphQL admission did not persist before external work")
	}
	changes, unsubscribe, err := runtime.searches.Subscribe(ctx, accepted.Search.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	runtimeCtx, stopRuntime := context.WithCancel(ctx)
	runtimeDone := make(chan struct{})
	go func() { defer close(runtimeDone); runtime.Run(runtimeCtx) }()
	defer func() {
		stopRuntime()
		select {
		case <-runtimeDone:
		case <-time.After(3 * time.Second):
			t.Error("indexer runtime did not stop")
		}
	}()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		snapshot, err := runtime.searches.Get(ctx, accepted.Search.ID, torrentsearch.Page{})
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.State == "COMPLETED" {
			break
		}
		select {
		case <-changes:
		case <-timeout.C:
			t.Fatal("persisted API search failed to complete", snapshot.State)
		}
	}
	var result struct {
		Search struct{ Candidates []struct{ ID string } } `json:"torrentSearch"`
	}
	execute(fmt.Sprintf(`{torrentSearch(id:%q){candidates{id}}}`, accepted.Search.ID), &result)
	if len(result.Search.Candidates) != 1 {
		t.Fatal("persistent API candidate snapshot lost release")
	}
	if prepareDownload {
		var prepared struct {
			Download struct{ Kind, DownloadURL string } `json:"prepareTorrentDownload"`
		}
		execute(fmt.Sprintf(`mutation{prepareTorrentDownload(input:{searchId:%q,candidateId:%q}){kind downloadUrl}}`, accepted.Search.ID, result.Search.Candidates[0].ID), &prepared)
		mux := http.NewServeMux()
		api.NewHandler(nil, api.WithIndexerDownloads(runtime.downloads)).Register(mux)
		for i := 0; i < 2; i++ {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, prepared.Download.DownloadURL, nil))
			if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), data) || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("production prepared attachment did not preserve acquired bytes", response.Code)
			}
		}
		if prepared.Download.Kind != "TORRENT" || grabs.Load() != 1 || downloader.adds != 0 {
			t.Fatal("prepared download repeated acquisition or submitted qB")
		}
		select {
		case <-subscription:
			t.Fatal("prepared download published a task mutation")
		default:
		}
	}
	var queued struct {
		Task struct{ ID string } `json:"queueTorrentCandidate"`
	}

	if existing {
		var resolved struct {
			Task struct{ ID string } `json:"resolveTaskSourcingCandidate"`
		}
		execute(fmt.Sprintf(`mutation{resolveTaskSourcingCandidate(input:{taskId:"existing-task",searchId:%q,candidateId:%q,expectedRevision:%d}){id}}`, accepted.Search.ID, result.Search.Candidates[0].ID, prior.Revision), &resolved)
		queued.Task.ID = resolved.Task.ID
		if queued.Task.ID != prior.ID {
			t.Fatal("source choice created a different task")
		}
	} else {
		execute(fmt.Sprintf(`mutation{queueTorrentCandidate(input:{code:"ABCD-123",searchId:%q,candidateId:%q}){id}}`, accepted.Search.ID, result.Search.Candidates[0].ID), &queued)
	}
	task, err := core.Find(ctx, queued.Task.ID)
	if err != nil || task == nil {
		t.Fatal("candidate intent was not committed through taskflow", err)
	}
	expectedGrabs := int32(0)
	if prepareDownload {
		expectedGrabs = 1
	}
	if grabs.Load() != expectedGrabs || task.IndexerResource == nil {
		t.Fatal("admission performed network acquisition or lost source")
	}
	select {
	case event := <-subscription:
		if event.Task == nil || event.Task.IndexerResource != nil {
			t.Fatal("task event retained secret configuration")
		}
	default:
		t.Fatal("durable admission did not publish creation")
	}
	if err = manager.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.Exec(`DELETE FROM indexer_resources`); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err = runtime.resources.Resolve(ctx, task.SelectedResourceID); !errors.Is(err, indexer.ErrExpired) {
		t.Fatal("runtime reference was not actually removed", err)
	}
	if _, err = configStore.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"url": "http://new-account.invalid", "api_key": "synthetic-new-account"}}}); err != nil {
		t.Fatal(err)
	}
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.(interface{ RunSourcingWorker(context.Context) }).RunSourcingWorker(workerCtx)
	}()
	defer func() {
		stop()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("sourcing worker did not stop")
		}
	}()
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		saved, err := core.Find(ctx, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Stage == taskruntime.TaskStageDownloading {
			break
		}
		select {
		case <-deadline.C:
			t.Fatalf("queued candidate did not submit: %s/%s", saved.Stage, saved.StageErrorCode)
		case <-ticker.C:
		}
	}
	stop()
	<-done
	if grabs.Load() != 1 || downloader.adds != 1 {
		t.Fatal("production pipeline did not acquire and submit exactly once")
	}
}

func TestDefinitionPolicyControlsFutureScheduledChecksWithoutImplicitInstallation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store := indexerTestStore(t)
	source := &policyPausedRuleSource{runtimeRuleSource: runtimeRuleSource{revision: strings.Repeat("a", 40), archive: runtimeRuleArchive(t, runtimeIndexerRule)}, entered: make(chan struct{}), release: make(chan struct{}), pauseRevision: strings.Repeat("b", 40)}
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, source)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err = runtime.Start(ctx); err != nil {
		t.Fatal(err)
	}
	enabled := true
	if _, err = runtime.admin.UpdateDefinitionPolicy(ctx, store.Revision(), config.IndexerDefinitionPolicyPatch{AutoUpdate: &enabled}); err != nil {
		t.Fatal(err)
	}
	runtime.checkDefinitions(ctx)
	if source.checks.Load() != 0 || source.downloads.Load() != 0 {
		t.Fatal("enabling policy implicitly downloaded an uninstalled library")
	}
	if _, err = runtime.definitions.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err := runtime.definitions.Begin(ctx, "INSTALL", source.revision, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID); got.State != indexerdefs.Completed {
		t.Fatal(got.ErrorCode)
	}
	enabled = false
	if _, err = runtime.admin.UpdateDefinitionPolicy(ctx, store.Revision(), config.IndexerDefinitionPolicyPatch{AutoUpdate: &enabled}); err != nil {
		t.Fatal(err)
	}
	source.revision = strings.Repeat("b", 40)
	checks := source.checks.Load()
	runtime.checkDefinitions(ctx)
	if source.checks.Load() != checks {
		t.Fatal("interval did not suppress premature network check")
	}
	age := func() {
		t.Helper()
		if err = manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`UPDATE indexer_definition_state SET last_checked_at=? WHERE singleton=1`, time.Now().UTC().Add(-25*time.Hour).Format(time.RFC3339Nano))
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	age()
	runtime.checkDefinitions(ctx)
	state, err := runtime.definitions.Status(ctx)
	if err != nil || state.State.Active != strings.Repeat("a", 40) || state.State.Available != source.revision || source.downloads.Load() != 1 {
		t.Fatal("manual-apply policy installed a discovered revision", err, state.State)
	}
	enabled = true
	if _, err = runtime.admin.UpdateDefinitionPolicy(ctx, store.Revision(), config.IndexerDefinitionPolicyPatch{AutoUpdate: &enabled}); err != nil {
		t.Fatal(err)
	}
	runtime.checkDefinitions(ctx)
	if source.downloads.Load() != 1 {
		t.Fatal("policy edit bypassed scheduled interval")
	}
	age()
	runtime.checkDefinitions(ctx)
	select {
	case <-source.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("automatic update never entered acquisition")
	}
	// Turning off future automatic application does not cancel an accepted job.
	enabled = false
	if _, err = runtime.admin.UpdateDefinitionPolicy(ctx, store.Revision(), config.IndexerDefinitionPolicyPatch{AutoUpdate: &enabled}); err != nil {
		t.Fatal(err)
	}
	close(source.release)
	state, err = runtime.definitions.Status(ctx)
	if err != nil || state.Job == nil || state.Job.Kind != "UPDATE" {
		t.Fatal("scheduled policy failed to create update intent", err)
	}
	if got := awaitRuntimeRuleJob(t, ctx, runtime.definitions, state.Job.ID); got.State != indexerdefs.Completed {
		t.Fatal("auto update did not complete validation/activation", got.ErrorCode)
	}
	state, err = runtime.definitions.Status(ctx)
	if err != nil || state.State.Active != source.revision || source.downloads.Load() != 2 {
		t.Fatal("latest saved policy was not used by scheduled runtime", err, state.State)
	}
}

type policyPausedRuleSource struct {
	runtimeRuleSource
	entered, release chan struct{}
	pauseRevision    string
}

func (s *policyPausedRuleSource) Download(ctx context.Context, revision string, w io.Writer) error {
	if revision == s.pauseRevision {
		close(s.entered)
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.runtimeRuleSource.Download(ctx, revision, w)
}

func TestIndexerDiagnosticsDraftConfigurationUsesFrozenRulesWithoutPersistence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store := indexerTestStore(t)
	source := &runtimeRuleSource{revision: strings.Repeat("a", 40), archive: runtimeRuleArchive(t, runtimeIndexerRule)}
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
	if got := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID); got.State != indexerdefs.Completed {
		t.Fatal(got.ErrorCode)
	}
	before, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	revision := store.Revision()
	definition := "example"
	resolver := graphqlapi.NewResolver(nil, nil, nil, "test")
	resolver.IndexerChecks = runtime.checks
	resolver.IndexerLogins = runtime.logins
	report, err := resolver.Mutation().TestIndexer(ctx, model.TestIndexerInput{ExpectedRevision: revision, DefinitionID: &definition})
	if err != nil || report.RuleRevision != source.revision || report.ConfigRevision != revision || report.InstanceID == "" || len(report.Stages) != 4 || report.Stages[0].State != "PASSED" || report.Stages[1].State != "NOT_REQUIRED" || report.Stages[2].State != "NOT_REQUESTED" || report.Stages[3].State != "NOT_REQUESTED" {
		t.Fatal("diagnostic did not preserve actual stage scope/rule snapshot", report, err)
	}
	after, err := os.ReadFile(store.Path())
	if err != nil || !bytes.Equal(before, after) || store.Revision() != revision || len(store.Config().Indexers.Builtin.Instances) != 0 || source.downloads.Load() != 1 {
		t.Fatal("diagnostic wrote a configuration/instance or fetched rules", err)
	}
	for _, table := range []string{"indexer_sessions", "indexer_resources"} {
		var count int
		if err = manager.Runtime.Reader.Get(&count, "SELECT count(*) FROM "+table); err != nil || count != 0 {
			t.Fatal("diagnostic leaked ephemeral state to runtime storage", table, count, err)
		}
	}
	t.Run("saved site login uses the same installed rule and does not create session state for a public site", func(t *testing.T) {
		saved, err := runtime.admin.SaveBuiltin(ctx, "", store.Revision(), indexeradmin.BuiltinPatch{Definition: &definition})
		if err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(store.Path())
		if err != nil {
			t.Fatal(err)
		}
		revision := store.Revision()
		login, err := resolver.Mutation().BeginIndexerLogin(ctx, model.IndexerLoginInput{InstanceID: saved.Instance.ID, ExpectedRevision: revision})
		if err != nil || login.State != "NOT_REQUIRED" || login.RuleRevision != source.revision || login.Challenge != nil {
			t.Fatal("login did not use actual application preparation", login, err)
		}
		after, err := os.ReadFile(store.Path())
		if err != nil || !bytes.Equal(before, after) || store.Revision() != revision || source.downloads.Load() != 1 {
			t.Fatal("login wrote configuration or downloaded rules", err)
		}
		var count int
		if err = manager.Runtime.Reader.Get(&count, "SELECT count(*) FROM indexer_sessions"); err != nil || count != 0 {
			t.Fatal("public site login opened a persistent account session", count, err)
		}
	})
}

func TestManagedIndexerHealthKeepsSearchFetchCatalogAndAccountScopesIndependent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	var requests atomic.Int32
	var canFetch, failCatalog atomic.Bool
	data := []byte("d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/api/v2.0/indexers/all/results/torznab/api":
			if failCatalog.Load() {
				http.Error(w, "directory unavailable", 403)
				return
			}
			io.WriteString(w, `<indexers><indexer id="one" configured="true"><title>One</title></indexer></indexers>`)
		case "/api/v2.0/indexers/all/results":
			json.NewEncoder(w).Encode(map[string]any{"Results": []map[string]any{{"Title": "ABCD-123", "Link": server.URL + "/dl/one", "Size": 1}}, "Indexers": []map[string]any{{"ID": "one", "Status": 200}}})
		case "/dl/one":
			if !canFetch.Load() {
				http.Error(w, "acquisition unavailable", 403)
				return
			}
			w.Write(data)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store := indexerTestStore(t)
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"url": server.URL}}}); err != nil {
		t.Fatal(err)
	}
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, &runtimeRuleSource{revision: strings.Repeat("a", 40)})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	search := func() indexer.SearchResult {
		t.Helper()
		result, e := runtime.service.Search(ctx, indexer.Query{Keywords: "ABCD-123", InstanceIDs: []string{"jackett/one"}})
		if e != nil || len(result.Releases) != 1 {
			t.Fatal("managed search", e)
		}
		return result
	}
	result := search()
	if _, err = runtime.service.FetchReference(ctx, result.Releases[0].Resource.ID); err == nil {
		t.Fatal("failed acquisition reported success")
	}
	before, err := runtime.admin.Instances(ctx)
	if err != nil || len(before.Items) != 1 || before.Items[0].Health.Search == nil || before.Items[0].Health.Fetch == nil || before.Items[0].Health.Search.ErrorCode != "" || before.Items[0].Health.Fetch.ErrorCode == "" {
		t.Fatal("managed operation health was collapsed", err)
	}
	failedFetch := *before.Items[0].Health.Fetch
	_ = search()
	after, err := runtime.admin.Instances(ctx)
	if err != nil || after.Items[0].Health.Search.ErrorCode != "" || after.Items[0].Health.Fetch.CorrelationID != failedFetch.CorrelationID {
		t.Fatal("search success erased acquisition failure", err)
	}
	calls := requests.Load()
	summary, err := runtime.admin.Summary(ctx)
	if err != nil || requests.Load() != calls || len(summary.Issues) != 1 || summary.Issues[0].Operation != "FETCH" || summary.Issues[0].Observation.CorrelationID != failedFetch.CorrelationID || summary.Jackett == nil || summary.Jackett.Sites == nil || *summary.Jackett.Sites != 1 {
		t.Fatal("Home did not project existing failure without network IO", err)
	}
	canFetch.Store(true)
	if _, err = runtime.service.FetchReference(ctx, result.Releases[0].Resource.ID); err != nil {
		t.Fatal("explicit acquisition retry", err)
	}
	current, err := runtime.admin.Instances(ctx)
	if err != nil || current.Items[0].Health.Fetch.ErrorCode != "" || current.Items[0].Health.Search.CorrelationID != after.Items[0].Health.Search.CorrelationID {
		t.Fatal("fetch success changed independent search observation", err)
	}
	// Directory failure preserves the last confirmed count and does not create a
	// search or fetch observation. No page read implicitly refreshes the directory.
	failCatalog.Store(true)
	runtime.registry.mu.Lock()
	runtime.registry.catalogAt = time.Time{}
	runtime.registry.attemptAt = time.Now().Add(-time.Minute)
	runtime.registry.mu.Unlock()
	if _, err = runtime.registry.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	calls = requests.Load()
	summary, err = runtime.admin.Summary(ctx)
	if err != nil || requests.Load() != calls || summary.Jackett == nil || !summary.Jackett.Stale || summary.Jackett.Sites == nil || *summary.Jackett.Sites != 1 || summary.Jackett.ErrorCode == "" {
		t.Fatal("directory failure became empty or caused Home IO", err)
	}
	// A new API account has no confirmed directory and inherits no old health.
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"api_key": "different-synthetic-account"}}}); err != nil {
		t.Fatal(err)
	}
	summary, err = runtime.admin.Summary(ctx)
	if err != nil || requests.Load() != calls || summary.Jackett == nil || summary.Jackett.Sites != nil || summary.Total != nil || len(summary.Issues) != 0 {
		t.Fatal("account switch reused confirmed sites or health", err)
	}
}

func TestHomeRestoresPersistedFailedRuleJobWithoutCheckingOrDownloading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	directory := t.TempDir()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(directory))
	if err != nil {
		t.Fatal(err)
	}
	store := indexerTestStore(t)
	if _, err = store.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); err != nil {
		t.Fatal(err)
	}
	source := &runtimeRuleSource{revision: strings.Repeat("a", 40), archive: []byte("synthetic-invalid-archive")}
	runtime, err := newIndexerRuntime(ctx, store.Config(), store, manager, source)
	if err != nil {
		t.Fatal(err)
	}
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
	failed := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID)
	if failed.State != indexerdefs.Failed {
		t.Fatal("failure fixture did not reach persisted terminal state", failed.State)
	}
	checks, downloads := source.checks.Load(), source.downloads.Load()
	summary, err := runtime.admin.Summary(ctx)
	if err != nil || summary.DefinitionJob == nil || summary.DefinitionJob.ID != job.ID || source.checks.Load() != checks || source.downloads.Load() != downloads {
		t.Fatal("Home did not read failed job locally", err)
	}
	runtime.Close()
	if err = manager.Close(); err != nil {
		t.Fatal(err)
	}
	manager, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	secondSource := &runtimeRuleSource{revision: strings.Repeat("b", 40)}
	restarted, err := newIndexerRuntime(ctx, store.Config(), store, manager, secondSource)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	// The first Home read does not need a prior subscription or a running worker.
	summary, err = restarted.admin.Summary(ctx)
	if err != nil || summary.DefinitionJob == nil || summary.DefinitionJob.ID != job.ID || summary.DefinitionJob.State != indexerdefs.Failed || summary.DefinitionJob.CorrelationID != failed.CorrelationID || secondSource.checks.Load() != 0 || secondSource.downloads.Load() != 0 {
		t.Fatal("restart lost failure or Home initiated external work", err)
	}
}
