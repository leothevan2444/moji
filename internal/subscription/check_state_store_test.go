package subscription

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestManagedCheckUpdatesPreserveReleaseLinksAndIndependentFields(t *testing.T) {
	store, _ := managedSubscriptionStore(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	value, rank := 3.25, 4
	state := &PerformerState{PerformerID: "actor", PendingReleases: []RecordedRelease{{Key: "release", Code: "ABC-001", Title: "keep", SeenAt: now}}, Activity: ActivityState{Value: &value, Rank: &rank, NextCheckAt: &now, Failures: 2}}
	if err := store.Put(ctx, state); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"subscription_release_entities", "subscription_performer_releases"} {
		for _, op := range []string{"INSERT", "UPDATE", "DELETE"} {
			query := "CREATE TRIGGER forbid_" + table + "_" + op + " BEFORE " + op + " ON " + table + " BEGIN SELECT RAISE(ABORT,'release relationships touched'); END"
			if err := store.withWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.ExecContext(ctx, query); return err }); err != nil {
				t.Fatal(err)
			}
		}
	}
	next := now.Add(time.Hour)
	if err := store.SaveCheckOutcome(ctx, &PerformerState{PerformerID: "actor", LastCheckedAt: &now, LastError: "network", Activity: ActivityState{NextCheckAt: &next, Failures: 3}}); err != nil {
		t.Fatal(err)
	}
	value = 4.5
	if err := store.SaveActivity(ctx, "actor", ActivityState{Value: &value, Rank: &rank, ObservedAt: &now}); err != nil {
		t.Fatal(err)
	}
	if err := store.InitializeCheckSchedule(ctx, "actor", now.Add(100*time.Hour)); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(ctx, "actor")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.PendingReleases) != 1 || loaded.PendingReleases[0].Title != "keep" || loaded.Activity.Failures != 3 || !loaded.Activity.NextCheckAt.Equal(next) || loaded.Activity.Value == nil || *loaded.Activity.Value != 4.5 || loaded.LastError != "network" {
		t.Fatalf("state changed unexpectedly: %+v", loaded)
	}
	var raw string
	if err := store.db.Get(&raw, "SELECT activity_json FROM subscription_performer_state WHERE performer_id='actor'"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "next_check_at") || strings.Contains(raw, "failures") {
		t.Fatalf("duplicated hot fields in JSON: %s", raw)
	}
}

func TestManagedDueSchedulesUseIndexedTimeOrderAndKeyset(t *testing.T) {
	store, _ := managedSubscriptionStore(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 0, 0, 0, 500, time.UTC)
	for _, row := range []struct {
		id string
		at time.Time
	}{{"c", now.Add(time.Nanosecond)}, {"b", now}, {"a", now}, {"z", now.Add(-time.Nanosecond)}} {
		if err := store.InitializeCheckSchedule(ctx, row.id, row.at); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.ListDueCheckSchedules(ctx, now, nil, 2)
	if err != nil || len(first) != 2 || first[0].PerformerID != "z" || first[1].PerformerID != "a" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := store.ListDueCheckSchedules(ctx, now, &first[1], 2)
	if err != nil || len(second) != 1 || second[0].PerformerID != "b" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	rows, err := store.db.Query("EXPLAIN QUERY PLAN SELECT performer_id,next_check_at FROM subscription_performer_state WHERE next_check_at IS NOT NULL AND next_check_at<=? ORDER BY next_check_at,performer_id LIMIT 128", formatCheckDeadline(&now))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		found = found || strings.Contains(detail, "idx_subscription_state_next_check")
	}
	if !found {
		t.Fatal("due query did not use deadline index")
	}
}

func TestManagedActivityBatchDoesNotLoadReleaseHistoryOrOtherActors(t *testing.T) {
	store, _ := managedSubscriptionStore(t)
	ctx := context.Background()
	zero, rank := 0.0, 1
	now := time.Now().UTC()
	for _, id := range []string{"visible", "hidden"} {
		if err := store.Put(ctx, &PerformerState{PerformerID: id, Activity: ActivityState{Value: &zero, Rank: &rank}, PendingReleases: []RecordedRelease{{Key: id, SeenAt: now}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.withWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE subscription_release_entities SET performer_names='broken'"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE subscription_performer_state SET activity_json='broken' WHERE performer_id='hidden'")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{store: store}
	result, err := svc.PerformerActivities(ctx, []string{"visible", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result["visible"].Value == nil || *result["visible"].Value != 0 || result["visible"].Rank == nil || *result["visible"].Rank != 1 || result["missing"].Rank != nil {
		t.Fatalf("unexpected page projection: %+v", result)
	}
	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("actor-%d", i)
	}
	if _, err := svc.PerformerActivities(ctx, tooMany); !errors.Is(err, ErrPerformerBatchTooLarge) {
		t.Fatalf("unbounded page accepted: %v", err)
	}
}
