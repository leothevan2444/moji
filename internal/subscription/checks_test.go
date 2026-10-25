package subscription

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type blockedCheckStash struct {
	*fakeStashClient
	entered chan struct{}
	release chan struct{}
}

func (s *blockedCheckStash) FindPerformerByID(ctx context.Context, id string) (*stashgraphql.PerformerFragment, error) {
	select {
	case s.entered <- struct{}{}:
	default:
	}
	select {
	case <-s.release:
		return s.fakeStashClient.FindPerformerByID(ctx, id)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func waitCheck(t *testing.T, s *Service, id string) *CheckRun {
	t.Helper()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		run, err := s.SubscriptionCheck(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if run != nil && run.Status != "RUNNING" {
			return run
		}
		select {
		case <-tick.C:
		case <-deadline:
			t.Fatal("check did not finish")
			return nil
		}
	}
}

func TestBackgroundCheckSurvivesRequestCancellationAndDeduplicates(t *testing.T) {
	stash := &blockedCheckStash{fakeStashClient: &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "One", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
	}}, entered: make(chan struct{}, 1), release: make(chan struct{})}
	service, err := newServiceForTest(stash, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	request, cancel := context.WithCancel(context.Background())
	run, err := service.StartSubscriptionCheck(request, []string{"p1"}, "MANUAL")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-stash.entered:
	case <-time.After(time.Second):
		t.Fatal("background check did not start")
	}
	duplicate, err := service.StartSubscriptionCheck(context.Background(), []string{"p1"}, "MANUAL")
	if err != nil || duplicate.ID != run.ID {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	if _, err := service.RefreshSubscribedPerformer(context.Background(), "p1"); err == nil {
		t.Fatal("legacy synchronous entry must reject overlapping work")
	}
	if _, err := service.StartSubscriptionCheck(context.Background(), nil, "MANUAL"); err == nil {
		t.Fatal("check-all must reject a partial overlap")
	}
	overview, _ := service.HomeChecks(context.Background())
	if len(overview.ActiveRuns) != 1 || overview.ActiveRuns[0].Status != "RUNNING" {
		t.Fatalf("unexpected overview: %+v", overview)
	}
	// Returned snapshots must not alias the live state while GraphQL serializes them.
	if run.Items[0].Status != "PENDING" {
		t.Fatalf("response snapshot was mutated: %+v", run)
	}
	close(stash.release)
	finished := waitCheck(t, service, run.ID)
	if finished.Status != "FAILED" || finished.FinishedAt == nil || finished.Items[0].Status != "FAILED" || finished.Items[0].Error == "" {
		t.Fatalf("failure must be tracked: %+v", finished)
	}
	overview, _ = service.HomeChecks(context.Background())
	if len(overview.ActiveRuns) != 0 {
		t.Fatal("completed check still active")
	}
}

func TestBackgroundCheckInterruptedOnRuntimeShutdown(t *testing.T) {
	stash := &blockedCheckStash{fakeStashClient: &fakeStashClient{}, entered: make(chan struct{}, 1), release: make(chan struct{})}
	service, err := newServiceForTest(stash, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	runtime, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.ConfigureCheckRuntime(runtime)
	run, err := service.StartSubscriptionCheck(context.Background(), []string{"p1"}, "MANUAL")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-stash.entered:
	case <-time.After(time.Second):
		t.Fatal("check did not start")
	}
	cancel()
	if finished := waitCheck(t, service, run.ID); finished.Status != "INTERRUPTED" {
		t.Fatalf("unexpected status: %+v", finished)
	}
}

type failingActorStash struct{ *fakeStashClient }

func (s failingActorStash) FindPerformerByID(ctx context.Context, id string) (*stashgraphql.PerformerFragment, error) {
	if id == "p2" {
		return nil, errors.New("actor lookup unavailable")
	}
	return s.fakeStashClient.FindPerformerByID(ctx, id)
}

func TestCheckAllTracksPartialFailureAndScheduledOrigin(t *testing.T) {
	service := refreshTaskService(t, NewMemoryStore(), &reviewTestTasks{})
	base := service.stash.(*fakeStashClient)
	base.performers["p2"] = &stashgraphql.PerformerFragment{ID: "p2", Name: "Two", CustomFields: map[string]any{DefaultCustomFieldKey: true}}
	service.stash = failingActorStash{base}
	next := time.Now().Add(time.Hour)
	service.SetCheckSchedule(true, &next)
	run, err := service.StartSubscriptionCheck(context.Background(), nil, "SCHEDULED")
	if err != nil {
		t.Fatal(err)
	}
	finished := waitCheck(t, service, run.ID)
	if finished.Status != "PARTIAL" || len(finished.Items) != 2 {
		t.Fatalf("unexpected run: %+v", finished)
	}
	statuses := map[string]string{}
	for _, item := range finished.Items {
		statuses[item.PerformerID] = item.Status
	}
	if statuses["p1"] != "SUCCEEDED" || statuses["p2"] != "FAILED" {
		t.Fatalf("unexpected items: %+v", finished.Items)
	}
	overview, _ := service.HomeChecks(context.Background())
	if !overview.Enabled || overview.NextRunAt == nil || overview.LatestScheduled == nil || overview.LatestScheduled.ID != run.ID {
		t.Fatalf("missing schedule info: %+v", overview)
	}
}

func TestSQLiteCheckHistorySurvivesRestartAndInterruptsActiveRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "moji.db")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, run := range []CheckRun{
		{ID: "active", Trigger: "MANUAL", Scope: "SINGLE", Status: "RUNNING", StartedAt: now, Items: []CheckItem{{PerformerID: "p1", Status: "RUNNING"}}},
		{ID: "done", Trigger: "SCHEDULED", Scope: "ALL", Status: "SUCCEEDED", StartedAt: now.Add(-time.Hour), FinishedAt: &now, Items: []CheckItem{}},
	} {
		if err := store.SaveCheckRun(context.Background(), run); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.db.Close()
	service, err := newServiceForTest(&fakeStashClient{}, nil, nil, reopened)
	if err != nil {
		t.Fatal(err)
	}
	overview, _ := service.HomeChecks(context.Background())
	if len(overview.ActiveRuns) != 0 || len(overview.RecentRuns) != 2 || overview.LatestScheduled == nil {
		t.Fatalf("unexpected restart snapshot: %+v", overview)
	}
	active, _ := service.SubscriptionCheck(context.Background(), "active")
	if active.Status != "INTERRUPTED" || active.Error != "SERVICE_RESTARTED" || active.Items[0].Status != "INTERRUPTED" || active.FinishedAt == nil {
		t.Fatalf("stale running state survived: %+v", active)
	}
	saved, err := reopened.LoadCheckRuns(context.Background())
	if err != nil || saved[0].Status != "INTERRUPTED" {
		t.Fatalf("interruption not persisted: %+v %v", saved, err)
	}
}

func TestCheckAllReservesActorsBeforeEnumeration(t *testing.T) {
	service, err := newServiceForTest(&fakeStashClient{}, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	ctx, run, _, err := service.beginCheck(context.Background(), "ALL", nil, "MANUAL")
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := service.StartSubscriptionCheck(context.Background(), []string{"p1"}, "MANUAL")
	if err != nil || duplicate.ID != run.ID {
		t.Fatalf("all-run reservation missing: %+v %v", duplicate, err)
	}
	service.finishCheck(ctx, nil)
}

func TestCheckStartDoesNotAcceptExplicitEmptySelection(t *testing.T) {
	service, err := newServiceForTest(&fakeStashClient{}, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartSubscriptionCheck(context.Background(), []string{}, "MANUAL"); !errors.Is(err, ErrPerformerBatchEmpty) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckHistoryRetentionKeepsActiveAndLatestScheduledRuns(t *testing.T) {
	sqlite, err := NewSQLiteStore(filepath.Join(t.TempDir(), "moji.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.db.Close()
	for name, store := range map[string]CheckRunStore{"memory": NewMemoryStore(), "sqlite": sqlite} {
		t.Run(name, func(t *testing.T) {
			now := time.Now().UTC()
			for _, run := range []CheckRun{{ID: "scheduled", Trigger: "SCHEDULED", Status: "FAILED", StartedAt: now.Add(-2 * time.Hour)}, {ID: "active", Trigger: "MANUAL", Status: "RUNNING", StartedAt: now.Add(-time.Hour)}} {
				if err := store.SaveCheckRun(context.Background(), run); err != nil {
					t.Fatal(err)
				}
			}
			for index := 0; index < checkHistoryLimit+1; index++ {
				run := CheckRun{ID: fmt.Sprintf("manual-%d", index), Trigger: "MANUAL", Status: "SUCCEEDED", StartedAt: now.Add(time.Duration(index) * time.Second)}
				if err := store.SaveCheckRun(context.Background(), run); err != nil {
					t.Fatal(err)
				}
			}
			runs, err := store.LoadCheckRuns(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			found := map[string]bool{}
			for _, run := range runs {
				found[run.ID] = true
			}
			if len(runs) != checkHistoryLimit+2 || !found["scheduled"] || !found["active"] || found["manual-0"] {
				t.Fatalf("retention lost protected history or retained too much: %d %+v", len(runs), found)
			}
		})
	}
}
