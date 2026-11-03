package taskruntime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type selectedSource struct {
	record                      indexer.ResourceRecord
	data                        []byte
	magnet                      string
	expired                     bool
	fetchErr                    error
	resolves, fetches, searches int
	received                    indexer.ResourceRecord
}

func (f *selectedSource) ResolveReference(ctx context.Context, id string) (*indexer.ResourceLease, error) {
	f.resolves++
	if f.expired {
		return nil, indexer.ErrExpired
	}
	if id != f.record.Ref.ID {
		return nil, indexer.ErrReference
	}
	return &indexer.ResourceLease{Record: f.record.Clone()}, nil
}
func (f *selectedSource) FetchRecord(ctx context.Context, record indexer.ResourceRecord) (indexer.Artifact, error) {
	f.fetches++
	f.received = record.Clone()
	if f.fetchErr != nil {
		return indexer.Artifact{}, f.fetchErr
	}
	return indexer.Artifact{TorrentBytes: append([]byte(nil), f.data...), Magnet: f.magnet}, nil
}
func (f *selectedSource) FetchReference(context.Context, string) (indexer.Artifact, error) {
	return indexer.Artifact{}, errors.New("expired runtime reference must not be used")
}
func (f *selectedSource) SearchSequential(context.Context, indexer.Query, func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
	f.searches++
	return indexer.SearchReport{}, errors.New("accepted candidate must not be replaced by a search")
}
func selectionRecord() indexer.ResourceRecord {
	revision := strings.Repeat("a", 40)
	return indexer.ResourceRecord{Version: 1, Instance: indexer.InstanceSnapshot{ID: "builtin-one", Name: "Synthetic site", Kind: indexer.Builtin, DefinitionID: "example", ConfigRevision: "old-account", BundleRevision: revision, BaseURL: "https://old.example", Settings: map[string]indexer.SettingValue{"password": {String: func() *string { s := "synthetic-private-password"; return &s }()}}, Network: indexer.NetworkProfile{ID: "private", Revision: "old-network", ProxyURL: "http://synthetic-user:synthetic-proxy-password@proxy.example"}}, Ref: indexer.ResourceRef{Version: 1, ID: "server-reference", InstanceID: "builtin-one", Kind: indexer.Builtin, ConfigRevision: "old-account", BundleRevision: revision, DefinitionID: "example", DefinitionDigest: strings.Repeat("b", 64), Locator: indexer.Locator{Title: "ABCD-123", DownloadURL: "https://old.example/grab?ticket=synthetic-private-ticket", Fields: map[string]string{"download-id": "original"}}}}
}
func seedSelectionBundle(t *testing.T, m *storage.Manager) {
	t.Helper()
	if err := m.Core.WithWrite(context.Background(), func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO indexer_definition_bundles(revision,manifest_json,manifest_sha256,installed_at) VALUES(?,?,?,?)`, strings.Repeat("a", 40), `{}`, strings.Repeat("b", 64), "now")
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
func selectionPins(t *testing.T, m *storage.Manager) int {
	t.Helper()
	var count int
	if err := m.Core.Reader.Get(&count, `SELECT count(*) FROM indexer_definition_refs WHERE owner_kind='task-resource'`); err != nil {
		t.Fatal(err)
	}
	return count
}

type selectedArtifactAdder struct {
	addCount int
	fakeTorrentAdder
	store *RelationalTaskStore
	data  []byte
	t     *testing.T
}

func (a *selectedArtifactAdder) AddNewTorrent(ctx context.Context, options qbittorrent.AddTorrentOptions) error {
	a.addCount++
	task, err := a.store.Find(ctx, "selected-task")
	if err != nil || task.IndexerResource != nil || task.SelectedResourceID != "server-reference" || !bytes.Equal(task.TorrentFileData, a.data) {
		a.t.Error("source secrets were not atomically replaced by durable bytes before qB", err)
	}
	if len(options.Torrents) != 1 || !bytes.Equal(options.Torrents[0].Data, a.data) {
		a.t.Error("qB did not receive the persisted original bytes")
	}
	return a.fakeTorrentAdder.AddNewTorrent(ctx, options)
}
func TestAcceptedSelectionSurvivesRestartExpiryAndAccountChange(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { m.Close() }()
	seedSelectionBundle(t, m)
	store := NewRelationalTaskStore(m.Core, m.Observations)
	source := &selectedSource{record: selectionRecord(), data: []byte(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"}))}
	adder := &selectedArtifactAdder{store: store, data: source.data, t: t}
	service, err := NewService(nil, adder, store, WithIndexers(source), WithIDGenerator(func() string { return "selected-task" }))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	if task.StageStatus != TaskStageStatusPending || source.fetches != 0 || adder.addCount != 0 || selectionPins(t, m) != 1 {
		t.Fatal("admission fetched/submitted before persistence or lost rule pin")
	}
	// A caller mutating its returned snapshot cannot alter the durable source.
	*task.IndexerResource.Instance.Settings["password"].String = "changed return value"
	task.IndexerResource.Ref.Locator.Fields["download-id"] = "changed"
	stored, err := store.Find(ctx, task.ID)
	if err != nil || *stored.IndexerResource.Instance.Settings["password"].String != "synthetic-private-password" {
		t.Fatal("returned snapshot changed durable task", err)
	}
	source.expired = true
	source.record.Instance.BaseURL = "https://new.example"
	source.record.Instance.ConfigRevision = "new-account"
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	m, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	store = NewRelationalTaskStore(m.Core, m.Observations)
	adder.store = store
	service, err = NewService(nil, adder, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	if err = service.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	stored, err = store.Find(ctx, task.ID)
	if err != nil || stored.Stage != TaskStageDownloading || source.resolves != 1 || source.fetches != 1 || source.searches != 0 || adder.addCount != 1 {
		t.Fatal("restart depended on runtime reference or searched again", err)
	}
	if source.received.Instance.BaseURL != "https://old.example" || source.received.Instance.ConfigRevision != "old-account" || source.received.Ref.Locator.Fields["download-id"] != "original" || selectionPins(t, m) != 0 {
		t.Fatal("frozen account/locator changed or acquired bytes kept rule pin")
	}
	var payload *string
	if err = m.Core.Reader.Get(&payload, `SELECT record_json FROM task_indexer_selections WHERE task_id=?`, task.ID); err != nil || payload != nil {
		t.Fatal("acquired source kept secret payload", err)
	}
	if stored.Candidate.Source == nil || stored.Candidate.Source.DefinitionDigest != strings.Repeat("b", 64) {
		t.Fatal("acquisition removed nonsecret provenance")
	}
}
func TestFailedSelectedCandidateRetainsItsChoiceAndAcquisitionForExplicitRetry(t *testing.T) {
	ctx := context.Background()
	store, m := relationalStore(t)
	seedSelectionBundle(t, m)
	source := &selectedSource{record: selectionRecord(), data: []byte(testTorrentFile("ABCD-999", []string{"ABCD-999.mp4"}))}
	adder := &fakeTorrentAdder{}
	service, err := NewService(nil, adder, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	_ = service.processQueuedSourcing(ctx, task.ID)
	stored, err := store.Find(ctx, task.ID)
	if err != nil || stored.StageStatus != TaskStageStatusBlocked || stored.StageErrorCode != TaskStageErrorCandidateMismatch || stored.SourcingNextCheckAt != nil || !bytes.Equal(stored.TorrentFileData, source.data) || stored.SelectedResourceID != "server-reference" {
		t.Fatal("failed candidate was changed into an automatic resource search", err)
	}
	_, _ = service.RetryTask(ctx, task.ID, nil)
	_ = service.processQueuedSourcing(ctx, task.ID)
	if source.fetches != 1 || source.searches != 0 || len(adder.options.Torrents) != 0 || len(adder.options.URLs) != 0 {
		t.Fatal("retry grabbed/replaced invalid selected bytes or submitted them")
	}
}
func TestSelectionPinsRollBackWithAdmissionAndDisappearWithTaskDeletion(t *testing.T) {
	ctx := context.Background()
	store, m := relationalStore(t)
	source := &selectedSource{record: selectionRecord()}
	service, err := NewService(nil, &fakeTorrentAdder{}, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	// No registered bundle: the source pin fails and the task must roll back too.
	if _, err = service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"}); err == nil {
		t.Fatal("accepted task without required rules")
	}
	tasks, err := store.List(ctx)
	if err != nil || len(tasks) != 0 || selectionPins(t, m) != 0 {
		t.Fatal("failed admission partially committed", err)
	}
	seedSelectionBundle(t, m)
	task, err := service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Delete(ctx, task.ID); err != nil || selectionPins(t, m) != 0 {
		t.Fatal("task deletion left rule pin", err)
	}
	source.expired = true
	if _, err = service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"}); !errors.Is(err, indexer.ErrExpired) {
		t.Fatal("expired candidate admitted", err)
	}
}

func TestCoreV5MigrationRetainsExistingTaskAndRejectsUnknownResourcePayload(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	corePath := m.Core.Path
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	// Build a real old schema in the stopped, isolated fixture generation. Do not
	// fake a migration by deleting a ledger entry from a newer database.
	if err = os.Remove(corePath); err != nil {
		t.Fatal(err)
	}
	oldDB, err := storage.Open(ctx, storage.Options{Path: corePath, Durability: storage.Durable})
	if err != nil {
		t.Fatal(err)
	}
	defer oldDB.Close()
	if err = oldDB.Migrate(ctx, storage.CoreMigrations()[:5]); err != nil {
		t.Fatal(err)
	}
	if err = oldDB.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('old-task','MANUAL','OLD-123','SOURCING','PENDING','2026-11-01T00:00:00.000000000Z','2026-11-01T00:00:00.000000000Z'); INSERT INTO task_downloads(task_id) VALUES('old-task'); INSERT INTO task_ingest(task_id) VALUES('old-task'); INSERT INTO task_checkpoints(task_id) VALUES('old-task'); INSERT INTO task_candidates(task_id,title) VALUES('old-task','OLD-123'); INSERT INTO task_indexer_sources(task_id,payload_version,payload_json) VALUES('old-task',1,'{"Version":1,"Kind":"JACKETT","InstanceID":"old-source"}');`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err = oldDB.Close(); err != nil {
		t.Fatal(err)
	}
	m, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	store := NewRelationalTaskStore(m.Core, m.Observations)
	old, err := store.Find(ctx, "old-task")
	if err != nil || old.Candidate.Source == nil || old.Candidate.Source.InstanceID != "old-source" || old.IndexerResource != nil {
		t.Fatal("v5 migration lost old task/provenance", err)
	}

	seedSelectionBundle(t, m)
	source := &selectedSource{record: selectionRecord()}
	service, err := NewService(nil, &fakeTorrentAdder{}, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE task_indexer_selections SET record_json=json_set(record_json,'$.Version',2) WHERE task_id=?`, task.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Find(ctx, task.ID); err == nil {
		t.Fatal("unknown resource payload was silently accepted")
	}
}

func TestExplicitManualReplacementDropsOldFrozenSelectionAndRulePin(t *testing.T) {
	ctx := context.Background()
	store, m := relationalStore(t)
	seedSelectionBundle(t, m)
	source := &selectedSource{record: selectionRecord(), fetchErr: indexer.ErrAuthentication}
	adder := &fakeTorrentAdder{}
	service, err := NewService(nil, adder, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	_ = service.processQueuedSourcing(ctx, task.ID)
	blocked, err := store.Find(ctx, task.ID)
	if err != nil || blocked.StageStatus != TaskStageStatusBlocked || blocked.IndexerResource == nil || selectionPins(t, m) != 1 {
		t.Fatal("failed acquisition lost its durable source", err)
	}
	data := []byte(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"}))
	resolved, err := service.ResolveBlockedSourcingTask(ctx, task.ID, ResolveBlockedSourcingRequest{TorrentData: data})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.SelectedResourceID != "" || resolved.IndexerResource != nil || selectionPins(t, m) != 0 || source.fetches != 1 {
		t.Fatal("explicit replacement reused or retained old source")
	}
	var count int
	if err = m.Core.Reader.Get(&count, `SELECT count(*) FROM task_indexer_selections WHERE task_id=?`, task.ID); err != nil || count != 0 {
		t.Fatal("explicit replacement kept old selection intent", err)
	}
}

func TestAcceptedMagnetDropsSecretSourceAndPublishesOnlyAnOpaqueReference(t *testing.T) {
	ctx := context.Background()
	store, m := relationalStore(t)
	seedSelectionBundle(t, m)
	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40) + "&dn=ABCD-123&tr=https://tracker.example/announce?passkey=synthetic-private"
	source := &selectedSource{record: selectionRecord(), magnet: magnet}
	adder := &fakeTorrentAdder{}
	service, err := NewService(nil, adder, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.QueueIndexedResourceContext(ctx, QueueIndexedResourceRequest{Code: "ABCD-123", ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	if err = service.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Find(ctx, task.ID)
	if err != nil || stored.IndexerResource != nil || stored.TorrentURL != magnet || selectionPins(t, m) != 0 || len(adder.options.URLs) != 1 || adder.options.URLs[0] != magnet {
		t.Fatal("magnet checkpoint did not preserve exact submission or retire secret source", err)
	}
	public := PublicTaskSnapshot(stored)
	if public.IndexerResource != nil || public.Candidate.MagnetURI != "" || public.TorrentURL != "indexer-resource:server-reference" {
		t.Fatal("public event exposed private tracker URL")
	}
}
