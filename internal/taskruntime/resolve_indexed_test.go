package taskruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
	"strings"
	"testing"
	"time"
)

func indexedResolutionFixture(t *testing.T) (*storage.Manager, *RelationalTaskStore, *selectedSource, *Service, *selectedArtifactAdder, *Task) {
	t.Helper()
	ctx := context.Background()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	seedSelectionBundle(t, m)
	store := NewRelationalTaskStore(m.Core, m.Observations)
	now := time.Now().UTC()
	task := &Task{ID: "selected-task", Code: "ABCD-123", Source: TaskSourceSubscription, SavePath: "original-save", Category: "original-category", Tags: "original-tags", CreatedAt: now, UpdatedAt: now}
	blockTask(task, TaskStageErrorNoCandidate, "no candidate", now)
	if err = store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	source := &selectedSource{record: selectionRecord(), data: []byte(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"}))}
	adder := &selectedArtifactAdder{store: store, data: source.data, t: t}
	service, err := NewService(nil, adder, store, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	return m, store, source, service, adder, task
}
func TestExistingTaskChoicePersistsBeforeAcquisitionSurvivesExpiryAndReplaysOnce(t *testing.T) {
	m, store, source, service, adder, task := indexedResolutionFixture(t)
	ctx := context.Background()
	expected := task.Revision
	selected, err := service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ResourceID: "server-reference", ExpectedRevision: expected})
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != task.ID || selected.Source != TaskSourceSubscription || selected.StageStatus != TaskStageStatusPending || source.fetches != 0 || adder.addCount != 0 || selectionPins(t, m) != 1 {
		t.Fatal("existing-task choice acquired/submitted before durable admission")
	}
	var request DownloadRequest
	if json.Unmarshal([]byte(selected.SourcingRequestJSON), &request) != nil || request.SavePath != "original-save" || request.Category != "original-category" || request.Tags != "original-tags" {
		t.Fatal("existing options were lost")
	}
	source.expired = true
	source.record.Instance.ConfigRevision = "changed-account"
	if err = service.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Find(ctx, task.ID)
	if err != nil || saved.Stage != TaskStageDownloading || !bytes.Equal(saved.TorrentFileData, source.data) || adder.addCount != 1 || source.fetches != 1 || source.searches != 0 {
		t.Fatal("selected existing task did not use its durable source", err)
	}
	replay, err := service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ResourceID: "server-reference", ExpectedRevision: expected})
	if err != nil || replay.Stage != TaskStageDownloading || source.resolves != 1 || source.fetches != 1 || adder.addCount != 1 {
		t.Fatal("lost-response replay repeated resolution or acquisition", err)
	}
}
func TestExistingTaskChoiceRechecksRevisionStateIdentityAndUncertainCommands(t *testing.T) {
	for _, scenario := range []string{"revision", "stage", "restart", "code", "uncertain", "future-command", "old-generation", "waiting", "paused"} {
		t.Run(scenario, func(t *testing.T) {
			m, store, source, service, _, task := indexedResolutionFixture(t)
			ctx := context.Background()
			expected := task.Revision
			switch scenario {
			case "revision":
				expected++
			case "stage":
				setTaskStage(task, TaskStageDownloading, TaskStageStatusBlocked)
			case "restart":
				task.RestartStateJSON = `{"version":1}`
			case "code":
				source.record.Ref.Locator.Title = "OTHER-999"
			case "waiting":
				next := time.Now().Add(time.Hour)
				setTaskStage(task, TaskStageSourcing, TaskStageStatusPending)
				task.SourcingNextCheckAt = &next
			case "paused":
				task.SourcingRequestJSON = `{"Paused":true}`
			}
			if scenario == "stage" || scenario == "restart" || scenario == "waiting" || scenario == "paused" {
				if err := store.Update(ctx, task); err != nil {
					t.Fatal(err)
				}
				expected = task.Revision
			}
			if scenario == "uncertain" || scenario == "old-generation" || scenario == "future-command" {
				generation, version := int64(0), 1
				if scenario == "old-generation" {
					generation = -1
				}
				if scenario == "future-command" {
					version = 2
				}
				if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
					_, err := tx.Exec(`INSERT INTO external_commands(id,task_id,target_hash,kind,state,payload_version,payload_json,created_at,updated_at) VALUES('uncertain',?,'hash','TASK_ADD','UNCERTAIN',?,?, 'now','now')`, task.ID, version, `{"generation":`+func() string {
						if generation < 0 {
							return "-1"
						}
						return "0"
					}()+`}`)
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			paused := false
			selected, err := service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ResourceID: "server-reference", ExpectedRevision: expected, Paused: &paused})
			succeeds := scenario == "old-generation" || scenario == "waiting" || scenario == "paused"
			if succeeds {
				if err != nil || selected == nil || selected.StageStatus != TaskStageStatusPending {
					t.Fatal("eligible source choice rejected", err)
				}
				if scenario == "paused" {
					var req DownloadRequest
					if json.Unmarshal([]byte(selected.SourcingRequestJSON), &req) != nil || req.Paused == nil || *req.Paused {
						t.Fatal("explicit false did not replace saved pause")
					}
				}
				return
			}
			if err == nil {
				t.Fatal("ineligible choice admitted")
			}
			if (scenario == "uncertain" || scenario == "future-command") && !errors.Is(err, ErrSubmissionUncertain) {
				t.Fatal("unreconciled submission reason was hidden", err)
			}
			if scenario == "revision" && !errors.Is(err, ErrTaskRevisionConflict) {
				t.Fatal(err)
			}
			persisted, findErr := store.Find(ctx, task.ID)
			if findErr != nil || persisted.SelectedResourceID != "" || selectionPins(t, m) != 0 || source.fetches != 0 {
				t.Fatal("rejection mutated task, pin or acquisition", findErr)
			}
		})
	}
}
func TestExistingTaskChoiceFailureRollsBackSelectionAuditAndPin(t *testing.T) {
	m, store, _, service, _, task := indexedResolutionFixture(t)
	ctx := context.Background()
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER reject_indexed_choice BEFORE INSERT ON task_indexer_selections BEGIN SELECT RAISE(ABORT,'synthetic failure'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var before, after int
	if err := m.Core.Reader.Get(&before, `SELECT count(*) FROM audit_events WHERE task_id=?`, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ResourceID: "server-reference", ExpectedRevision: task.Revision}); err == nil {
		t.Fatal("failed transaction acknowledged")
	}
	saved, err := store.Find(ctx, task.ID)
	if err != nil || saved.Revision != task.Revision || saved.SelectedResourceID != "" || saved.StageStatus != TaskStageStatusBlocked || selectionPins(t, m) != 0 {
		t.Fatal("choice transaction was partially committed", err)
	}
	if err = m.Core.Reader.Get(&after, `SELECT count(*) FROM audit_events WHERE task_id=?`, task.ID); err != nil || before != after {
		t.Fatal("failed choice published durable audit", err)
	}
}
func TestExistingTaskChoiceRejectsUnsupportedStores(t *testing.T) {
	source := &selectedSource{record: selectionRecord()}
	service, err := NewService(nil, &fakeTorrentAdder{}, NewMemoryTaskStore(), WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ResolveIndexedResourceContext(context.Background(), "task", ResolveIndexedResourceRequest{ExpectedRevision: 1, ResourceID: strings.Repeat("a", 8)}); !errors.Is(err, indexer.ErrUnsupported) {
		t.Fatal("non-atomic legacy store acknowledged selection", err)
	}
}

func TestExistingTaskChoiceEventIsPublishedOnlyAfterPrivateSourceCommits(t *testing.T) {
	m, store, source, _, adder, task := indexedResolutionFixture(t)
	ctx := context.Background()
	bus := NewTaskEventBus(8)
	events := bus.Subscribe(ctx)
	wrapped, err := NewEventingTaskStore(store, bus)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(nil, adder, wrapped, WithIndexers(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ExpectedRevision: task.Revision, ResourceID: "server-reference"}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Type != TaskEventUpdated || event.Task == nil || event.Task.IndexerResource != nil {
			t.Fatal("private selection escaped the published task")
		}
		saved, err := store.Find(ctx, task.ID)
		if err != nil || saved.IndexerResource == nil || selectionPins(t, m) != 1 {
			t.Fatal("event preceded durable selection", err)
		}
	default:
		t.Fatal("choice did not publish updated task")
	}
}

func TestReplacingFailedSelectedSourceAtomicallySwapsBundlePin(t *testing.T) {
	m, store, source, service, _, task := indexedResolutionFixture(t)
	ctx := context.Background()
	old, err := service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ExpectedRevision: task.Revision, ResourceID: "server-reference"})
	if err != nil {
		t.Fatal(err)
	}
	blockTask(old, TaskStageErrorTorrentMetadata, "synthetic acquisition failure", time.Now().UTC())
	if err = store.Update(ctx, old); err != nil {
		t.Fatal(err)
	}
	replacement := selectionRecord()
	replacement.Ref.ID = "replacement-reference"
	replacement.Ref.BundleRevision = strings.Repeat("c", 40)
	replacement.Instance.BundleRevision = replacement.Ref.BundleRevision
	source.record = replacement
	// The unavailable new bundle must not unpin or overwrite the old accepted source.
	if _, err = service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ExpectedRevision: old.Revision, ResourceID: replacement.Ref.ID}); err == nil {
		t.Fatal("missing replacement bundle was acknowledged")
	}
	saved, err := store.Find(ctx, task.ID)
	if err != nil || saved.SelectedResourceID != "server-reference" || saved.IndexerResource == nil || saved.Revision != old.Revision || selectionPins(t, m) != 1 {
		t.Fatal("failed replacement destroyed old source/pin", err)
	}
	if err = m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO indexer_definition_bundles(revision,manifest_json,manifest_sha256,installed_at) VALUES(?,?,?,?)`, replacement.Ref.BundleRevision, `{}`, strings.Repeat("d", 64), "now")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	accepted, err := service.ResolveIndexedResourceContext(ctx, task.ID, ResolveIndexedResourceRequest{ExpectedRevision: old.Revision, ResourceID: replacement.Ref.ID})
	if err != nil || accepted.SelectedResourceID != replacement.Ref.ID || accepted.StageStatus != TaskStageStatusPending || source.fetches != 0 {
		t.Fatal("replacement did not persist its own frozen source", err)
	}
	var pinned string
	if err = m.Core.Reader.Get(&pinned, `SELECT revision FROM indexer_definition_refs WHERE owner_kind='task-resource' AND owner_id=?`, task.ID); err != nil || pinned != replacement.Ref.BundleRevision || selectionPins(t, m) != 1 {
		t.Fatal("source and rule pin did not change atomically", err)
	}
}
