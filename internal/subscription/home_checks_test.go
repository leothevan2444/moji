package subscription

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestHomeChecksIsBoundedAndDoesNotNeedUpstreamServices(t *testing.T) {
	now := time.Now().UTC()
	next := now.Add(time.Hour)
	tracker := &checkTracker{enabled: true, nextRunAt: &next}
	for index := 0; index < 12; index++ {
		status := "RUNNING"
		if index >= 6 {
			status = "FAILED"
		}
		run := CheckRun{ID: fmt.Sprintf("run-%d", index), Trigger: "MANUAL", Scope: "SELECTED", Status: status, StartedAt: now.Add(-time.Duration(index) * time.Minute)}
		if index == 11 {
			run.Trigger = "SCHEDULED"
		}
		for actor := 0; actor < 100; actor++ {
			item := CheckItem{PerformerID: fmt.Sprintf("p-%d-%d", index, actor), Name: "Actor", Status: "RUNNING"}
			if actor < 40 {
				item.Status = "SUCCEEDED"
			} else if actor < 50 {
				item.Status = "FAILED"
			} else if actor < 60 {
				item.Status = "SKIPPED"
			}
			run.Items = append(run.Items, item)
		}
		tracker.runs = append(tracker.runs, run)
	}
	// Home reads only the local registry, even when Stash and metadata services are absent.
	service := &Service{checks: tracker}
	snapshot, err := service.HomeChecks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Enabled || snapshot.ActiveCount != 6 || len(snapshot.ActiveRuns) != 3 || len(snapshot.RecentRuns) != 3 || snapshot.LatestScheduled == nil || snapshot.LatestScheduled.ID != "run-11" {
		t.Fatalf("incorrect bounded snapshot: %+v", snapshot)
	}
	run := snapshot.ActiveRuns[0]
	if run.TotalCount != 100 || run.CompletedCount != 60 || run.SucceededCount != 40 || run.FailedCount != 10 || run.SkippedCount != 10 || len(run.CurrentPerformers) != 3 {
		t.Fatalf("counts must cover every actor while samples stay bounded: %+v", run)
	}
	snapshot.ActiveRuns[0].CurrentPerformers[0].Name = "changed"
	*snapshot.NextRunAt = time.Time{}
	if tracker.runs[0].Items[60].Name != "Actor" || tracker.nextRunAt.IsZero() {
		t.Fatal("Home snapshot aliases mutable check state")
	}
}

func TestCheckActivityOnlyReturnsRequestedActorReservations(t *testing.T) {
	service := &Service{checks: &checkTracker{runs: []CheckRun{
		{ID: "a", Status: "RUNNING", Scope: "SELECTED", Items: []CheckItem{{PerformerID: "p1", Status: "RUNNING"}, {PerformerID: "p2", Status: "SUCCEEDED"}}},
		{ID: "b", Status: "RUNNING", Scope: "SINGLE", Items: []CheckItem{{PerformerID: "p3", Status: "PENDING"}}},
		{ID: "finished", Status: "SUCCEEDED", Items: []CheckItem{{PerformerID: "p4", Status: "SUCCEEDED"}}},
	}}}
	activity, err := service.CheckActivity(context.Background(), []string{"p1", "p1", "p4"})
	if err != nil {
		t.Fatal(err)
	}
	if activity.ActiveCount != 2 || len(activity.Performers) != 1 || activity.Performers[0].RunID != "a" || activity.Performers[0].PerformerID != "p1" {
		t.Fatalf("activity leaked unrelated actors or history: %+v", activity)
	}
	service.checks.runs = []CheckRun{{ID: "all", Status: "RUNNING", Scope: "ALL"}}
	activity, err = service.CheckActivity(context.Background(), []string{"p5"})
	if err != nil || activity.AllRunID != "all" || activity.ActiveCount != 1 {
		t.Fatalf("enumerating all-run must reserve every actor: %+v %v", activity, err)
	}
	ids := make([]string, maxCheckActivityPerformers+1)
	for i := range ids {
		ids[i] = fmt.Sprint(i)
	}
	if _, err := service.CheckActivity(context.Background(), ids); err == nil {
		t.Fatal("unbounded actor query accepted")
	}
}
