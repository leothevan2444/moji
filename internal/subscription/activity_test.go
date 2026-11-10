package subscription

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/metadata"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestActivityThresholdsAndIntervals(t *testing.T) {
	for _, tc := range []struct {
		value      float64
		rank, days int
	}{{0, 1, 30}, {.249999, 1, 30}, {.25, 2, 14}, {.999999, 2, 14}, {1, 3, 7}, {2.499999, 3, 7}, {2.5, 4, 3}, {5.999999, 4, 3}, {6, 5, 1}, {100, 5, 1}} {
		rank := activityRank(tc.value)
		if rank != tc.rank || activityInterval(&rank) != time.Duration(tc.days)*24*time.Hour {
			t.Fatalf("value=%v rank=%v interval=%v", tc.value, rank, activityInterval(&rank))
		}
	}
	if activityInterval(nil) != 48*time.Hour {
		t.Fatal("unknown activity must use 48h")
	}
}

func TestActivityDailyMaximumDecayAndSourceDates(t *testing.T) {
	now := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	sample := newActivitySample(now, "UTC")
	scene := func(id, title string, days int) *stashboxgraphql.SceneFragment {
		return &stashboxgraphql.SceneFragment{ID: id, Title: stringPtr(title), Date: stringPtr(now.AddDate(0, 0, -days).Format("2006-01-02"))}
	}
	sample.observe([]*stashboxgraphql.SceneFragment{
		scene("new", "Ordinary release", 0), scene("same-day", "Another ordinary release", 0),
		scene("old", "Ordinary release", 60), scene("comp", "総集編", 120),
		scene("unknown", "", 180), scene("historical", "Ordinary release", 400),
		scene("future", "Ordinary release", -2),
	}, true)
	got := sample.snapshot(now)
	want := 1 + .5 + .2*.25 + .5*.125
	if got.Value == nil || math.Abs(*got.Value-want) > 1e-9 || *got.Rank != 3 {
		t.Fatalf("unexpected rating: %+v want=%v", got, want)
	}
	if len(got.Days) != 4 {
		t.Fatalf("same-day, old and future releases incorrectly included: %+v", got.Days)
	}
	// Date-only arithmetic must agree across a daylight-saving transition.
	a := newActivitySample(time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC), "America/New_York")
	a.observe([]*stashboxgraphql.SceneFragment{{ID: "dst", Title: stringPtr("Release"), Date: stringPtr("2026-09-10")}}, true)
	v := a.snapshot(now)
	if v.Value == nil || math.Abs(*v.Value-.5) > 1e-9 {
		t.Fatalf("calendar-day decay changed across DST: %+v", v)
	}
}

func TestActivityRequiresCoverageUnlessTopTierProven(t *testing.T) {
	now := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	sample := newActivitySample(now, "UTC")
	sample.observe([]*stashboxgraphql.SceneFragment{{ID: "one", Title: stringPtr("Release"), Date: stringPtr("2026-11-09")}}, false)
	if sample.snapshot(now).Rank != nil {
		t.Fatal("partial catalog cannot establish a low rating")
	}
	sample.observe([]*stashboxgraphql.SceneFragment{{ID: "bad", Title: stringPtr("Release")}}, true)
	if sample.snapshot(now).Rank != nil {
		t.Fatal("unknown dates cannot establish a low rating")
	}
	for i := 1; i < 10; i++ {
		sample.observe([]*stashboxgraphql.SceneFragment{{ID: time.Duration(i).String(), Title: stringPtr("Release"), Date: stringPtr(now.AddDate(0, 0, -i).Format("2006-01-02"))}}, false)
	}
	got := sample.snapshot(now)
	if got.Rank == nil || *got.Rank != 5 || !got.LowerBound {
		t.Fatalf("top tier lower bound not recognized: %+v", got)
	}
}

func TestActivityCompilationDoesNotDependOnCastSize(t *testing.T) {
	for _, tc := range []struct {
		title string
		want  float64
	}{{"Ordinary release", 1}, {"Best of", .2}, {"Compilation", .2}, {"", .5}, {"Bestial story", 1}} {
		scene := &stashboxgraphql.SceneFragment{Title: stringPtr(tc.title), Performers: make([]*stashboxgraphql.PerformerAppearanceFragment, 20)}
		if got := activityWeight(scene); got != tc.want {
			t.Fatalf("%q got=%v want=%v", tc.title, got, tc.want)
		}
	}
}

func TestSchedulePreservesRatingOnFailureAndSeparatesTaskFailure(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	s, _ := NewService(&fakeStashClient{}, metadata.NewService(nil, nil), nil, store)
	s.now = func() time.Time { return now }
	rank, value := 4, 3.0
	previous := now.Add(-time.Hour)
	store.Put(ctx, &PerformerState{PerformerID: "p", Activity: ActivityState{Value: &value, Rank: &rank, ObservedAt: &previous, LastSuccessAt: &previous}})
	for i, hours := range []int{1, 2, 4, 8, 16, 24, 24} {
		state, err := s.recordCheckSchedule(ctx, "p", now, errors.New("upstream unavailable"))
		if err != nil || state.Activity.Failures != i+1 || *state.Activity.Value != 3 || !state.Activity.NextCheckAt.Equal(now.Add(time.Duration(hours)*time.Hour)) || !state.Activity.LastSuccessAt.Equal(previous) {
			t.Fatalf("attempt=%d state=%+v error=%v", i, state, err)
		}
	}
	state, _ := store.Get(ctx, "p")
	state.Activity.ObservedAt = &now
	state.Activity.LastSuccessAt = &now
	store.Put(ctx, state)
	state, err := s.recordCheckSchedule(ctx, "p", now, errors.New("task creation failed"))
	if err != nil || state.Activity.Failures != 0 || !state.Activity.NextCheckAt.Equal(now.Add(72*time.Hour)) {
		t.Fatalf("task failure changed catalog cadence: %+v %v", state, err)
	}
}

func TestDurableScheduleSurvivesReopenAndDispatchesOnlyDueActors(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "subscriptions.db")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(7 * 24 * time.Hour)
	past := now.Add(-time.Hour)
	for id, next := range map[string]time.Time{"future": future, "due": past, "unsubscribed": past} {
		next := next
		if err := store.Put(ctx, &PerformerState{PerformerID: id, Activity: ActivityState{NextCheckAt: &next}}); err != nil {
			t.Fatal(err)
		}
	}
	store.db.Close()
	store, err = NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	stash := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}
	for _, id := range []string{"future", "due", "unsubscribed"} {
		stash.performers[id] = &stashgraphql.PerformerFragment{ID: id, Name: id, CustomFields: map[string]any{DefaultCustomFieldKey: id != "unsubscribed"}}
	}
	s, err := NewService(stash, metadata.NewService(nil, nil), nil, store)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return now }
	if err = s.DispatchDueCheck(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitBackground()
	state, _ := store.Get(ctx, "future")
	if !state.Activity.NextCheckAt.Equal(future) {
		t.Fatal("restart moved future deadline")
	}
	history, _ := s.CheckHistory(ctx)
	if len(history) != 1 || history[0].TotalCount != 1 {
		t.Fatalf("expected one due actor: %+v", history)
	}
	run, _ := s.SubscriptionCheck(ctx, history[0].ID)
	if run.Items[0].PerformerID != "due" {
		t.Fatalf("wrong actor checked: %+v", run.Items)
	}
	state, _ = store.Get(ctx, "due")
	if !state.Activity.NextCheckAt.Equal(now.Add(time.Hour)) {
		t.Fatal("upstream failure did not persist retry")
	}
}

func TestActivityDeduplicatesWorkAcrossDifferentReleaseDates(t *testing.T) {
	now := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	sample := newActivitySample(now, "UTC")
	sample.observe([]*stashboxgraphql.SceneFragment{
		{ID: "copy", Code: stringPtr("ABCD-001"), Title: stringPtr("A release"), Date: stringPtr("2026-11-09")},
		{ID: "original", Code: stringPtr("ABCD-001"), Title: stringPtr("A release"), Date: stringPtr("2026-09-10")},
	}, true)
	got := sample.snapshot(now)
	if got.Value == nil || math.Abs(*got.Value-1) > 1e-9 {
		t.Fatalf("duplicate work inflated rating: %+v", got)
	}
}

func TestManagedActivityScheduleRoundTrip(t *testing.T) {
	store, _ := managedSubscriptionStore(t)
	now := time.Now().UTC()
	value, rank := 3.25, 4
	original := &PerformerState{PerformerID: "managed", Activity: ActivityState{Value: &value, Rank: &rank, ObservedAt: &now, LastSuccessAt: &now, NextCheckAt: &now, Failures: 2, Days: []ActivityDay{{Date: "2026-11-09", Weight: .2}}}}
	if err := store.Put(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	all, err := store.ListCheckSchedules(context.Background())
	if err != nil || len(all) != 1 || all[0].PerformerID != "managed" || all[0].NextCheckAt == nil || !all[0].NextCheckAt.Equal(now) {
		t.Fatalf("managed schedule projection lost state: %+v %v", all, err)
	}
	loaded, err := store.Get(context.Background(), "managed")
	if err != nil || loaded.Activity.Value == nil || *loaded.Activity.Value != value || loaded.Activity.Failures != 2 || loaded.Activity.Rank == nil || *loaded.Activity.Rank != 4 || len(loaded.Activity.Days) != 1 {
		t.Fatalf("managed observation lost: %+v %v", loaded, err)
	}
}

func TestActivityUnavailableReasonsSurvivePersistence(t *testing.T) {
	now := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		sample *activitySample
		want   ActivityEvaluationStatus
	}{
		{"coverage", newActivitySample(now, "UTC"), ActivityIncompleteCatalog},
		{"dates", &activitySample{records: map[string]ActivityDay{}, covered: true, invalidDate: true, today: now}, ActivityInvalidDates},
		{"membership", &activitySample{records: map[string]ActivityDay{}, covered: true, invalidMembership: true, today: now}, ActivityInvalidMembership},
	}
	store, _ := managedSubscriptionStore(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &PerformerState{PerformerID: tc.name, Activity: tc.sample.snapshot(now)}
			if state.Activity.EvaluationStatus() != tc.want {
				t.Fatalf("wrong unavailable reason: %+v", state.Activity)
			}
			if err := store.Put(context.Background(), state); err != nil {
				t.Fatal(err)
			}
			loaded, err := store.Get(context.Background(), tc.name)
			if err != nil || loaded.Activity.EvaluationStatus() != tc.want || loaded.Activity.Value != nil || loaded.Activity.Rank != nil {
				t.Fatalf("unavailable reason lost: %+v %v", loaded, err)
			}
		})
	}
	if (ActivityState{}).EvaluationStatus() != ActivityNotChecked {
		t.Fatal("new actor must be unassessed")
	}
	if (ActivityState{Failures: 1}).EvaluationStatus() != ActivityCheckFailed {
		t.Fatal("first read failure must remain visible")
	}
	if (ActivityState{ObservedAt: &now}).EvaluationStatus() != ActivityIncompleteCatalog {
		t.Fatal("older unknown observation must remain unscored")
	}
	value, rank := 6.0, 5
	if (ActivityState{Value: &value, Rank: &rank, EvaluationReason: ActivityInvalidDates, Failures: 1}).EvaluationStatus() != ActivityReady {
		t.Fatal("a confirmed rating must survive a later failure")
	}
}
