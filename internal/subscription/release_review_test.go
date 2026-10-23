package subscription

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type reviewTestTasks struct {
	tasks    []*taskruntime.Task
	calls    int
	failCode string
}

func (f *reviewTestTasks) ListTasks(context.Context) ([]*taskruntime.Task, error) {
	return f.tasks, nil
}
func (f *reviewTestTasks) QueueSubscriptionRelease(_ context.Context, code, _ string) (*taskruntime.Task, error) {
	f.calls++
	if code == f.failCode {
		return nil, errors.New("resource service unavailable")
	}
	task := &taskruntime.Task{ID: fmt.Sprintf("task-%d", f.calls), Code: code, Source: taskruntime.TaskSourceSubscription}
	f.tasks = append(f.tasks, task)
	return task, nil
}
func reviewTestRecord(key string, status ReviewStatus) ReleaseReview {
	return ReleaseReview{RecordedRelease: RecordedRelease{Key: key, Code: "ABC-123", Source: "stash-box:https://example.org/graphql", Title: "Movie", Decision: ReleaseDecisionQueued, DecisionReason: "vr_behavior_review", SeenAt: time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)}, Status: status, Performers: []ReviewPerformer{{ID: "p1", Name: "Actor"}}}
}
func reviewTestService(t *testing.T, store Store, tasks *reviewTestTasks) *Service {
	t.Helper()
	service, err := NewService(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{"p1": {ID: "p1", Name: "Actor", CustomFields: map[string]any{DefaultCustomFieldKey: true}}, "p2": {ID: "p2", Name: "Other", CustomFields: map[string]any{DefaultCustomFieldKey: true}}}}, metadata.NewService(nil, nil), tasks, store)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestReviewHistorySurvivesSnapshotLimitsUnsubscribeAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "reviews.db")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	state := &PerformerState{PerformerID: "p1"}
	for i := 0; i < 40; i++ {
		record := reviewTestRecord(fmt.Sprintf("r-%02d", i), ReviewPending)
		state.PendingReleases = append(state.PendingReleases, record.RecordedRelease)
	}
	if err := store.Put(ctx, state); err != nil {
		t.Fatal(err)
	}
	tasks := &reviewTestTasks{}
	service := reviewTestService(t, store, tasks)
	page, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{PageSize: 100})
	if err != nil || page.PendingCount != 40 {
		t.Fatalf("migration: %+v %v", page, err)
	}
	if _, err := service.ProcessReleaseReviews(ctx, []string{"r-00"}, ReviewReject); err != nil {
		t.Fatal(err)
	}
	state.PendingReleases = state.PendingReleases[15:]
	if err := store.Put(ctx, state); err != nil {
		t.Fatal(err)
	}
	if err := service.UnsubscribePerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	store.db.Close()
	reopened, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.db.Close()
	service = reviewTestService(t, reopened, tasks)
	page, err = service.ListReleaseReviews(ctx, ReleaseReviewQuery{Status: "ALL", PageSize: 100})
	if err != nil || page.TotalCount != 40 || page.PendingCount != 39 {
		t.Fatalf("history lost: total=%d pending=%d err=%v", page.TotalCount, page.PendingCount, err)
	}
	rejected, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{Status: ReviewRejected})
	if err != nil || len(rejected.Items) != 1 || rejected.Items[0].ReviewedAt == nil || tasks.calls != 0 {
		t.Fatalf("rejection lost or created task: %+v %v calls=%d", rejected, err, tasks.calls)
	}
}

func TestReviewDeduplicatesActorsAndConcurrentDownload(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	record := reviewTestRecord("same-scene", ReviewPending)
	for _, id := range []string{"p1", "p2"} {
		if err := store.Put(ctx, &PerformerState{PerformerID: id, PendingReleases: []RecordedRelease{record.RecordedRelease}}); err != nil {
			t.Fatal(err)
		}
	}
	tasks := &reviewTestTasks{}
	service := reviewTestService(t, store, tasks)
	page, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{})
	if err != nil || page.TotalCount != 1 || len(page.Items[0].Performers) != 2 {
		t.Fatalf("not deduplicated: %+v %v", page, err)
	}
	for _, actor := range page.Performers {
		if actor.PendingCount != 1 {
			t.Fatalf("incorrect actor review count: %+v", actor)
		}
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := service.ProcessReleaseReviews(ctx, []string{record.Key}, ReviewDownload); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	if tasks.calls != 1 || len(tasks.tasks) != 1 || tasks.tasks[0].Source != taskruntime.TaskSourceSubscription {
		t.Fatalf("created duplicate or incorrect task: %+v", tasks)
	}
	for _, id := range []string{"p1", "p2"} {
		page, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{Status: "ALL", PerformerID: id})
		if err != nil || page.PendingCount != 0 || page.Items[0].TaskID == "" {
			t.Fatalf("actor state did not update: %+v %v", page, err)
		}
	}
}

func TestReviewPartialResultsRetryAndReconsider(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tasks := &reviewTestTasks{failCode: "FAIL-123"}
	service := reviewTestService(t, store, tasks)
	ok := reviewTestRecord("ok", ReviewPending)
	failed := reviewTestRecord("failed", ReviewPending)
	failed.Code = "FAIL-123"
	skipped := reviewTestRecord("skipped", ReviewSkipped)
	skipped.Code = "SKIP-123"
	if err := store.PutReleaseReviews(ctx, []ReleaseReview{ok, failed, skipped}); err != nil {
		t.Fatal(err)
	}
	payload, err := service.ProcessReleaseReviews(ctx, []string{"ok", "failed", "missing", "skipped"}, ReviewDownload)
	if err != nil || payload.SucceededCount != 1 || payload.FailedCount != 2 || payload.SkippedCount != 1 {
		t.Fatalf("unexpected partial results: %+v %v", payload, err)
	}
	page, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{Status: ReviewFailed})
	if err != nil || page.PendingCount != 0 || page.FailedCount != 1 || len(page.Items) != 1 || page.Items[0].Error == "" {
		t.Fatalf("failures mixed into review: %+v %v", page, err)
	}
	if _, err := service.ProcessReleaseReviews(ctx, []string{"skipped"}, ReviewReconsider); err != nil {
		t.Fatal(err)
	}
	page, err = service.ListReleaseReviews(ctx, ReleaseReviewQuery{})
	if err != nil || page.PendingCount != 1 {
		t.Fatalf("reconsider failed: %+v %v", page, err)
	}
	if _, err := service.ProcessReleaseReviews(ctx, []string{"skipped"}, ReviewReject); err != nil {
		t.Fatal(err)
	}
	if tasks.calls != 2 {
		t.Fatalf("reconsider/reject created a task: calls=%d", tasks.calls)
	}
	tasks.failCode = ""
	if _, err := service.ProcessReleaseReviews(ctx, []string{"failed"}, ReviewDownload); err != nil {
		t.Fatal(err)
	}
	if tasks.calls != 3 {
		t.Fatal("failed task was not retried")
	}
}

type reviewFailSaveStore struct {
	*MemoryStore
	fail bool
}

func (s *reviewFailSaveStore) PutReleaseReviews(ctx context.Context, records []ReleaseReview) error {
	for _, record := range records {
		if s.fail && record.Status == ReviewDownloaded {
			s.fail = false
			return errors.New("temporary write failure")
		}
	}
	return s.MemoryStore.PutReleaseReviews(ctx, records)
}

func TestReviewRecoversTaskCreatedBeforeDecisionSave(t *testing.T) {
	ctx := context.Background()
	store := &reviewFailSaveStore{MemoryStore: NewMemoryStore()}
	record := reviewTestRecord("r1", ReviewPending)
	if err := store.PutReleaseReviews(ctx, []ReleaseReview{record}); err != nil {
		t.Fatal(err)
	}
	store.fail = true
	tasks := &reviewTestTasks{}
	service := reviewTestService(t, store, tasks)
	first, err := service.ProcessReleaseReviews(ctx, []string{record.Key}, ReviewDownload)
	if err != nil || first.FailedCount != 1 {
		t.Fatalf("save failure not returned: %+v %v", first, err)
	}
	if _, err := service.ProcessReleaseReviews(ctx, []string{record.Key}, ReviewDownload); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{Status: ReviewDownloaded})
	if err != nil || tasks.calls != 1 || len(page.Items) != 1 || page.Items[0].TaskID == "" {
		t.Fatalf("recovery created duplicate or lost task: %+v calls=%d err=%v", page, tasks.calls, err)
	}
}

func TestReviewLinksManualDownloadsAndFiltersPersistedRecords(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	record := reviewTestRecord("r1", ReviewRejected)
	record.Code = "ABCVR-123"
	if err := store.PutReleaseReviews(ctx, []ReleaseReview{record}); err != nil {
		t.Fatal(err)
	}
	tasks := &reviewTestTasks{tasks: []*taskruntime.Task{{ID: "manual-task", Code: "ABCVR-123", Source: taskruntime.TaskSourceManual}}}
	service := reviewTestService(t, store, tasks)
	yes := true
	page, err := service.ListReleaseReviews(ctx, ReleaseReviewQuery{Status: ReviewDownloaded, VR: &yes, Search: "abcvr", PerformerID: "p1"})
	if err != nil || len(page.Items) != 1 || page.Items[0].TaskID != "manual-task" || tasks.calls != 0 {
		t.Fatalf("manual download not linked: %+v %v", page, err)
	}
}

func TestReviewDoesNotCreateTaskForLibraryScene(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	record := reviewTestRecord("library-scene", ReviewPending)
	record.SceneID = "scene-1"
	if err := store.PutReleaseReviews(ctx, []ReleaseReview{record}); err != nil {
		t.Fatal(err)
	}
	tasks := &reviewTestTasks{}
	service := reviewTestService(t, store, tasks)
	service.stash.(*fakeStashClient).scenes = []*stashgraphql.SceneFragment{{ID: "local-1", StashIds: []*stashgraphql.StashIDFragment{{Endpoint: "https://example.org/graphql", StashID: "scene-1"}}}}
	result, err := service.ProcessReleaseReviews(ctx, []string{record.Key}, ReviewDownload)
	if err != nil || result.SucceededCount != 1 || result.Results[0].Release.Status != ReviewInLibrary || tasks.calls != 0 {
		t.Fatalf("download created for library item: %+v calls=%d err=%v", result, tasks.calls, err)
	}
}
