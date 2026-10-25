package subscription

import (
	"context"
	"fmt"
	"testing"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

const refreshTaskEndpoint = "https://example.org/graphql"

func refreshTaskService(t *testing.T, store Store, tasks TaskCreator, codes ...string) *Service {
	t.Helper()
	scenes := make([]*stashboxgraphql.SceneFragment, 0, len(codes))
	for _, code := range codes {
		scenes = append(scenes, &stashboxgraphql.SceneFragment{ID: code, Code: stringPtr(code), Title: stringPtr("Movie")})
	}
	registry := metadata.NewRegistry(stubFactory{client: &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "box-p1", Name: "Actor"}, scenes: scenes,
	}})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "metadata", Endpoint: refreshTaskEndpoint}})
	service, err := NewService(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "Actor", CustomFields: map[string]any{DefaultCustomFieldKey: true},
			StashIds: []*stashgraphql.StashIDFragment{{Endpoint: refreshTaskEndpoint, StashID: "box-p1"}}},
	}}, metadata.NewService(nil, registry), tasks, store)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestRefreshLinksExistingTasksAndRepairsHistoricalFailure(t *testing.T) {
	for _, snapshot := range []string{"new", "legacy-pending", "review-already-linked"} {
		t.Run(snapshot, func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryStore()
			existing := &taskruntime.Task{ID: "search-task", Code: "URE-114", Source: taskruntime.TaskSourceSearch}
			record := reviewTestRecord("stashbox:https___example.org_graphql:URE-114", ReviewFailed)
			record.Code, record.Decision = "URE-114", ReleaseDecisionDownloaded
			record.Error = fmt.Sprintf("%v: task already uses code URE-114", taskruntime.ErrDuplicateCodeTask)
			if snapshot != "new" {
				review := reviewTestRecord("manual-review", ReviewPending)
				review.Code = "ABCVR-123"
				skipped := reviewTestRecord("policy-skipped", ReviewSkipped)
				skipped.Code, skipped.Decision = "ABCVR-456", ReleaseDecisionBlocked
				if err := store.Put(ctx, &PerformerState{PerformerID: "p1", LastError: record.Error,
					PendingReleases: []RecordedRelease{record.RecordedRelease, review.RecordedRelease, skipped.RecordedRelease}}); err != nil {
					t.Fatal(err)
				}
				if snapshot == "review-already-linked" {
					record.TaskID, record.Status, record.Error = existing.ID, ReviewDownloaded, ""
					if err := store.PutReleaseReviews(ctx, []ReleaseReview{record}); err != nil {
						t.Fatal(err)
					}
				}
			}
			tasks := &reviewTestTasks{tasks: []*taskruntime.Task{existing}}
			service := refreshTaskService(t, store, tasks, "URE-114")
			for check := 0; check < 2; check++ {
				item, err := service.RefreshSubscribedPerformer(ctx, "p1")
				if err != nil || item.LastError != "" || item.ProcessedReleaseCount != 1 {
					t.Fatalf("check %d: result=%+v err=%v", check, item, err)
				}
				state, err := store.Get(ctx, "p1")
				if err != nil || state.LastError != "" || len(state.ProcessedReleases) != 1 || state.ProcessedReleases[0].TaskID != existing.ID {
					t.Fatalf("snapshot not repaired: %+v err=%v", state, err)
				}
				wantPending := 0
				if snapshot != "new" {
					wantPending = 2
				}
				if len(state.PendingReleases) != wantPending {
					t.Fatalf("review or skipped records lost: %+v", state)
				}
			}
			if tasks.calls != 0 || len(tasks.tasks) != 1 {
				t.Fatalf("attempted duplicate creation: %+v", tasks)
			}
			rows, err := store.ListReleaseReviews(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				if row.Code == "URE-114" && (row.Status != ReviewDownloaded || row.TaskID != existing.ID || row.Error != "") {
					t.Fatalf("review not repaired: %+v", row)
				}
			}
		})
	}
}

type racingRefreshTasks struct {
	reviewTestTasks
	queueErr error
}

func (f *racingRefreshTasks) QueueSubscriptionRelease(_ context.Context, code, _ string) (*taskruntime.Task, error) {
	f.calls++
	if f.queueErr != nil {
		return nil, f.queueErr
	}
	f.tasks = append(f.tasks, &taskruntime.Task{ID: "concurrent-task", Code: code})
	return nil, fmt.Errorf("concurrent creation: %w", taskruntime.ErrDuplicateCodeTask)
}

func TestRefreshLinksTaskCreatedConcurrently(t *testing.T) {
	tasks := &racingRefreshTasks{}
	store := NewMemoryStore()
	service := refreshTaskService(t, store, tasks, "URE-114")
	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil || item.LastError != "" || item.ProcessedReleaseCount != 1 || tasks.calls != 1 || len(tasks.tasks) != 1 {
		t.Fatalf("concurrent task not reused: %+v tasks=%+v err=%v", item, tasks, err)
	}
	state, err := store.Get(context.Background(), "p1")
	if err != nil || len(state.PendingReleases) != 0 || len(state.ProcessedReleases) != 1 || state.ProcessedReleases[0].TaskID != "concurrent-task" {
		t.Fatalf("concurrent task not persisted: %+v err=%v", state, err)
	}
}

func TestRefreshPreservesUnresolvedFailureAndClearsItAfterTaskAppears(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tasks := &reviewTestTasks{failCode: "FAIL-123", tasks: []*taskruntime.Task{{ID: "existing-task", Code: "URE-114"}}}
	service := refreshTaskService(t, store, tasks, "URE-114", "FAIL-123")
	for check := 0; check < 2; check++ {
		item, err := service.RefreshSubscribedPerformer(ctx, "p1")
		if err != nil || item.LastError != "resource service unavailable" || item.PendingReleaseCount != 1 || item.ProcessedReleaseCount != 1 {
			t.Fatalf("unresolved error lost: %+v err=%v", item, err)
		}
	}
	if tasks.calls != 1 {
		t.Fatalf("retried failed release automatically: %d calls", tasks.calls)
	}
	tasks.tasks = append(tasks.tasks, &taskruntime.Task{ID: "manual-recovery", Code: "FAIL-123"})
	item, err := service.RefreshSubscribedPerformer(ctx, "p1")
	if err != nil || item.LastError != "" || item.PendingReleaseCount != 0 || item.ProcessedReleaseCount != 2 || tasks.calls != 1 {
		t.Fatalf("resolved failure retained: %+v calls=%d err=%v", item, tasks.calls, err)
	}
}

func TestRefreshHandlesLibraryRaceAndMissingTask(t *testing.T) {
	for _, test := range []struct {
		name       string
		err        error
		wantFailed bool
	}{
		{"already-in-library", taskruntime.ErrDuplicateLibraryCode, false},
		{"no-task-returned", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var tasks TaskCreator = &racingRefreshTasks{queueErr: test.err}
			if test.err == nil {
				tasks = &fakeTaskCreator{}
			}
			service := refreshTaskService(t, NewMemoryStore(), tasks, "URE-114")
			item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
			if err != nil || (item.LastError != "") != test.wantFailed {
				t.Fatalf("unexpected state: %+v err=%v", item, err)
			}
			if !test.wantFailed && item.ProcessedReleaseCount != 1 {
				t.Fatalf("library item not handled: %+v", item)
			}
		})
	}
}
