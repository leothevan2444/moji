package subscription

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func upcomingHarness(t *testing.T, store Store) (*Service, *fakeStashboxClient, *reviewTestTasks, *time.Time) {
	t.Helper()
	clock := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	actor := &stashboxgraphql.PerformerFragment{ID: "box-p1", Name: "Actor"}
	scene := releasePolicyScene("future", actor, actor)
	scene.Title = stringPtr("Future film")
	scene.Date = stringPtr("2026-11-01")
	scene.Code = stringPtr("ABC-123")
	box := &fakeStashboxClient{performer: actor, scenes: []*stashboxgraphql.SceneFragment{scene}}
	registry := metadata.NewRegistry(stubFactory{client: box})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "test", Endpoint: refreshTaskEndpoint}})
	tasks := &reviewTestTasks{}
	service, err := NewService(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{"p1": {ID: "p1", Name: "Actor", CustomFields: map[string]any{DefaultCustomFieldKey: true}, StashIds: []*stashgraphql.StashIDFragment{{Endpoint: refreshTaskEndpoint, StashID: "box-p1"}}}}}, metadata.NewService(nil, registry), tasks, store)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return clock }
	return service, box, tasks, &clock
}
func TestUpcomingFuturePostponementOutageAndRelease(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	service, box, tasks, clock := upcomingHarness(t, store)
	for i := 0; i < 2; i++ {
		if _, err := service.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
			t.Fatal(err)
		}
	}
	page, err := service.ListUpcomingReleases(ctx, UpcomingQuery{})
	if err != nil || page.TotalCount != 1 || tasks.calls != 0 {
		t.Fatalf("future page=%+v tasks=%d err=%v", page, tasks.calls, err)
	}
	reviews, _ := store.ListReleaseReviews(ctx)
	if len(reviews) != 0 {
		t.Fatal("future release created review")
	}
	key := page.Items[0].Release.Key
	*clock = clock.Add(24 * time.Hour)
	box.scenes[0].Date = stringPtr("2026-11-03")
	if err = service.ProcessDueUpcoming(ctx); err != nil {
		t.Fatal(err)
	}
	record, _ := store.GetUpcoming(ctx, key)
	if record.Release.Date != "2026-11-03" || record.State != "WAITING" || tasks.calls != 0 {
		t.Fatalf("postponement %+v", record)
	}
	box.findSceneErr = errors.New("offline")
	*clock = clock.Add(48 * time.Hour)
	if err = service.ProcessDueUpcoming(ctx); err != nil {
		t.Fatal(err)
	}
	record, _ = store.GetUpcoming(ctx, key)
	if record.LastErrorCode != "METADATA_UNAVAILABLE" || record.Release.Title == "" || tasks.calls != 0 {
		t.Fatalf("outage %+v", record)
	}
	box.findSceneErr = nil
	*clock = clock.Add(time.Hour)
	if err = service.ProcessDueUpcoming(ctx); err != nil {
		t.Fatal(err)
	}
	record, _ = store.GetUpcoming(ctx, key)
	if record.State != "HANDED_OFF" || tasks.calls != 1 {
		t.Fatalf("release %+v tasks=%d", record, tasks.calls)
	}
	if err = service.ProcessDueUpcoming(ctx); err != nil {
		t.Fatal(err)
	}
	if tasks.calls != 1 {
		t.Fatal("duplicate task")
	}
	page, _ = service.ListUpcomingReleases(ctx, UpcomingQuery{})
	if page.TotalCount != 0 {
		t.Fatal("released item remains upcoming")
	}
}
func TestUpcomingUsesLatestPolicyAndSubscription(t *testing.T) {
	for _, behavior := range []config.SubscriptionReleaseBehavior{config.SubscriptionReleaseBehaviorReview, config.SubscriptionReleaseBehaviorBlock} {
		t.Run(string(behavior), func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryStore()
			s, _, tasks, clock := upcomingHarness(t, store)
			if _, err := s.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
				t.Fatal(err)
			}
			if err := s.UnsubscribePerformer(ctx, "p1"); err != nil {
				t.Fatal(err)
			}
			*clock = clock.Add(24 * time.Hour)
			if err := s.ProcessDueUpcoming(ctx); err != nil {
				t.Fatal(err)
			}
			p, _ := s.ListUpcomingReleases(ctx, UpcomingQuery{})
			if p.TotalCount != 0 || tasks.calls != 0 {
				t.Fatal("unsubscribed item active")
			}
			if _, err := s.SubscribePerformer(ctx, "p1"); err != nil {
				t.Fatal(err)
			}
			policy := DefaultReleasePolicyConfig()
			policy.SoloBehavior = behavior
			s.SetReleasePolicy(policy)
			due, _ := store.DueUpcoming(ctx, clock.Add(48*time.Hour), 32)
			if len(due) != 1 {
				t.Fatal("unsubscribe deleted record")
			}
			if _, err := s.RefreshUpcomingRelease(ctx, due[0].Release.Key); err != nil {
				t.Fatal(err)
			}
			reviews, _ := store.ListReleaseReviews(ctx)
			want := ReviewPending
			if behavior == config.SubscriptionReleaseBehaviorBlock {
				want = ReviewSkipped
			}
			if len(reviews) != 1 || reviews[0].Status != want || tasks.calls != 0 {
				t.Fatalf("reviews=%+v calls=%d", reviews, tasks.calls)
			}
		})
	}
}
func TestUpcomingRestartReconcilesTaskAndUnknownDateReviews(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "restart", true: "unknown"}[unknown], func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryStore()
			s, box, tasks, clock := upcomingHarness(t, store)
			if _, err := s.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
				t.Fatal(err)
			}
			page, _ := s.ListUpcomingReleases(ctx, UpcomingQuery{})
			key := page.Items[0].Release.Key
			*clock = clock.Add(24 * time.Hour)
			if unknown {
				box.scenes[0].Date = nil
			} else {
				tasks.tasks = []*taskruntime.Task{{ID: "already-created", Code: "ABC-123"}}
			}
			if _, err := s.RefreshUpcomingRelease(ctx, key); err != nil {
				t.Fatal(err)
			}
			reviews, _ := store.ListReleaseReviews(ctx)
			if len(reviews) != 1 || tasks.calls != 0 {
				t.Fatalf("reviews=%+v calls=%d", reviews, tasks.calls)
			}
			if unknown && reviews[0].DecisionReason != "release_date_unknown_review" {
				t.Fatal("unknown date not reviewed")
			}
			if !unknown && reviews[0].TaskID != "already-created" {
				t.Fatal("task not reconciled")
			}
		})
	}
}
func TestUpcomingManagedMigrationReopenPaginationAndCAS(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "data")
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(dir))
	if err != nil {
		t.Fatal(err)
	}
	// Construct a genuine v2 core in this isolated fixture. Removing a middle
	// migration ledger row leaves a gap once later migrations exist.
	corePath := manager.Core.Path
	if err = manager.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(corePath); err != nil {
		t.Fatal(err)
	}
	previous, err := storage.Open(ctx, storage.Options{Path: corePath, Durability: storage.Durable})
	if err != nil {
		t.Fatal(err)
	}
	if err := previous.Migrate(ctx, storage.CoreMigrations()[:2]); err != nil {
		previous.Close()
		t.Fatal(err)
	}
	if err := previous.Close(); err != nil {
		t.Fatal(err)
	}
	manager, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(dir))
	if err != nil {
		t.Fatal(err)
	}
	store := NewManagedStore(manager.Core)
	service, _, _, _ := upcomingHarness(t, store)
	if _, err = service.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	page, err := store.QueryUpcoming(ctx, UpcomingQuery{PageSize: 1}, []string{"p1"})
	if err != nil || page.TotalCount != 1 {
		t.Fatalf("page %+v %v", page, err)
	}
	record := page.Items[0]
	stale := record
	record.Performers = append(record.Performers, ReviewPerformer{ID: "p2", Name: "Second"})
	if err = store.PutUpcoming(ctx, &record); err != nil {
		t.Fatal(err)
	}
	if err = store.PutUpcoming(ctx, &stale); !errors.Is(err, errUpcomingRevision) {
		t.Fatalf("stale write accepted %v", err)
	}
	if err = store.Delete(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	page, err = store.QueryUpcoming(ctx, UpcomingQuery{Search: "ABC", PerformerID: "p2"}, []string{"p2"})
	if err != nil || page.TotalCount != 1 {
		t.Fatalf("shared item lost %+v %v", page, err)
	}
	backup, err := manager.Core.Backup(ctx, manager.BackupDirectory, "upcoming-test", true)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(manager.BackupDirectory, backup.Database+".json")
	record.State = "HANDED_OFF"
	if err = store.PutUpcoming(ctx, &record); err != nil {
		t.Fatal(err)
	}
	if err = manager.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.RestoreCore(ctx, dir, manifest); err != nil {
		t.Fatal(err)
	}
	manager, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	reopened := NewManagedStore(manager.Core)
	recordPtr, err := reopened.GetUpcoming(ctx, record.Release.Key)
	if err != nil || recordPtr == nil || len(recordPtr.Performers) != 2 || recordPtr.State != "WAITING" {
		t.Fatalf("reopen %+v %v", recordPtr, err)
	}
}

type cancelAtUpcomingLibraryCheck struct{ *fakeStashClient }

func (s *cancelAtUpcomingLibraryCheck) FindScenes(context.Context, *stashgraphql.SceneFilterType, *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error) {
	s.performers["p1"].CustomFields[DefaultCustomFieldKey] = false
	return nil, nil
}
func TestUpcomingCancellationDuringLibraryLookupDoesNotCreateTask(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	service, _, tasks, clock := upcomingHarness(t, store)
	service.stash = &cancelAtUpcomingLibraryCheck{service.stash.(*fakeStashClient)}
	if _, err := service.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(24 * time.Hour)
	if err := service.ProcessDueUpcoming(ctx); err != nil {
		t.Fatal(err)
	}
	if tasks.calls != 0 {
		t.Fatal("created task after cancellation during library read")
	}
}

func TestSubscriptionCheckUpdatesExistingUpcomingBeforeDueWorker(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	service, box, tasks, clock := upcomingHarness(t, store)
	catalog, err := performer.NewService(service.stash, service.metadata, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog.SetUpcomingStateReader(service)
	scenePage := func() performer.ScenePage {
		t.Helper()
		page, err := catalog.ListPerformerScenes(ctx, "p1", performer.SceneQuery{Source: performer.SceneSourceFilterStashBox, PageSize: 24})
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("catalog: %+v %v", page, err)
		}
		return page
	}
	if state := scenePage().Items[0].UpcomingState; state != "" {
		t.Fatalf("future date alone must not mark persisted state: %s", state)
	}
	if _, err := service.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListUpcomingReleases(ctx, UpcomingQuery{})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("capture: %+v %v", page, err)
	}
	first := page.Items[0]
	if state := scenePage().Items[0].UpcomingState; state != "WAITING" {
		t.Fatalf("persisted state missing: %s", state)
	}
	box.scenes[0].Title = stringPtr("Updated future title")
	box.scenes[0].Date = stringPtr("2026-11-02")
	if _, err = service.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	record, err := store.GetUpcoming(ctx, first.Release.Key)
	if err != nil || record.Release.Title != "Updated future title" || record.Release.Date != "2026-11-02" || !record.FirstSeenAt.Equal(first.FirstSeenAt) || len(record.Performers) != 1 || tasks.calls != 0 {
		t.Fatalf("repeat capture: %+v tasks=%d err=%v", record, tasks.calls, err)
	}
	*clock = clock.Add(48 * time.Hour)
	if _, err = service.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	record, err = store.GetUpcoming(ctx, first.Release.Key)
	reviews, reviewErr := store.ListReleaseReviews(ctx)
	if err != nil || reviewErr != nil || tasks.calls != 0 || len(reviews) != 0 || record.NextCheckAt.After(*clock) {
		t.Fatalf("check must defer due handoff: %+v tasks=%d reviews=%d err=%v/%v", record, tasks.calls, len(reviews), err, reviewErr)
	}
	if err = service.ProcessDueUpcoming(ctx); err != nil {
		t.Fatal(err)
	}
	if tasks.calls != 1 {
		t.Fatalf("expected one task after worker, got %d", tasks.calls)
	}
	if state := scenePage().Items[0].UpcomingState; state != "" {
		t.Fatalf("handed-off record still marked upcoming: %s", state)
	}
}

func TestUpcomingReleaseStateLookupMemoryAndSQLite(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	for name, store := range map[string]UpcomingStore{"memory": NewMemoryStore(), "sqlite": NewManagedStore(manager.Core)} {
		t.Run(name, func(t *testing.T) {
			for key, state := range map[string]string{"source-a:film": "WAITING", "source-b:film": "UNCONFIRMED", "pending": "HANDOFF_PENDING", "finished": "HANDED_OFF", "unrequested": "WAITING"} {
				record := &UpcomingRelease{Release: Release{Key: key, Code: "ABC-123"}, State: state}
				if err := store.PutUpcoming(ctx, record); err != nil {
					t.Fatal(err)
				}
			}
			// Same code and remote ID do not imply the same source identity. Paused
			// records have no subscribed-actor filter in this read-only projection.
			states, err := store.UpcomingReleaseStates(ctx, []string{"source-a:film", "source-b:film", "source-c:film", "pending", "finished"})
			if err != nil || len(states) != 3 || states["source-a:film"] != "WAITING" || states["source-b:film"] != "UNCONFIRMED" || states["pending"] != "HANDOFF_PENDING" {
				t.Fatalf("states=%v err=%v", states, err)
			}
			states, err = store.UpcomingReleaseStates(ctx, nil)
			if err != nil || len(states) != 0 {
				t.Fatalf("empty=%v err=%v", states, err)
			}
		})
	}
}
