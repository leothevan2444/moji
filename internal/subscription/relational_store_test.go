package subscription

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

func managedSubscriptionStore(t *testing.T) (*SQLiteStore, *storage.Manager) {
	t.Helper()
	m, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	return NewManagedStore(m.Core), m
}
func TestManagedSubscriptionDeltaAndIndependentDecisions(t *testing.T) {
	s, m := managedSubscriptionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	state := &PerformerState{PerformerID: "actor", LastCheckedAt: &now, PendingReleases: []RecordedRelease{{Key: "release", Title: "movie", SeenAt: now}}}
	if err := s.Put(ctx, state); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER forbid_link_delete BEFORE DELETE ON subscription_performer_releases BEGIN SELECT RAISE(ABORT,'link churn'); END; CREATE TRIGGER forbid_release_update BEFORE UPDATE ON subscription_release_entities BEGIN SELECT RAISE(ABORT,'release churn'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(ctx, state); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Get(ctx, "actor")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, loaded) {
		t.Fatalf("roundtrip: %+v", loaded)
	}
	record := ReleaseReview{RecordedRelease: state.PendingReleases[0], Status: ReviewRejected, ReviewedAt: &now, Performers: []ReviewPerformer{{ID: "actor", Name: "name"}}}
	if err := s.PutReleaseReviews(ctx, []ReleaseReview{record}); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`DROP TRIGGER forbid_link_delete; DROP TRIGGER forbid_release_update;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "actor"); err != nil {
		t.Fatal(err)
	}
	reviews, err := s.ListReleaseReviews(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 || !reflect.DeepEqual(reviews[0], record) {
		t.Fatalf("unsubscribe lost review: %+v", reviews)
	}
	var releases, links, audit int
	if err := m.Core.Reader.Get(&releases, "SELECT count(*) FROM subscription_release_entities"); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.Reader.Get(&links, "SELECT count(*) FROM subscription_review_performers"); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.Reader.Get(&audit, "SELECT count(*) FROM audit_events WHERE event_type='release-review'"); err != nil {
		t.Fatal(err)
	}
	if releases != 1 || links != 1 || audit != 1 {
		t.Fatalf("durable history: releases=%d links=%d audit=%d", releases, links, audit)
	}
}
func TestManagedCheckRunTypedItemsAndMembershipGuard(t *testing.T) {
	s, m := managedSubscriptionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	run := CheckRun{ID: "run", Trigger: "MANUAL", Scope: "SELECTED", Status: "RUNNING", StartedAt: now, Items: []CheckItem{{PerformerID: "a", Name: "A", Status: "RUNNING", StartedAt: &now}, {PerformerID: "b", Name: "B", Status: "PENDING"}}}
	if err := s.SaveCheckRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER forbid_unchanged_item BEFORE UPDATE ON subscription_check_items WHEN OLD.performer_id='b' BEGIN SELECT RAISE(ABORT,'unchanged item churn'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	run.Items[0].Status = "DONE"
	run.Items[0].FinishedAt = &now
	if err := s.SaveCheckRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	runs, err := s.LoadCheckRuns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || !reflect.DeepEqual(runs[0], run) {
		t.Fatalf("check roundtrip: %+v", runs)
	}
	changed := cloneCheckRun(run)
	changed.Status = "DONE"
	changed.Items = changed.Items[:1]
	if err := s.SaveCheckRun(ctx, changed); err == nil {
		t.Fatal("membership change accepted")
	}
	runs, err = s.LoadCheckRuns(ctx)
	if err != nil || !reflect.DeepEqual(runs[0], run) {
		t.Fatalf("failed update was not atomic: %+v err=%v", runs, err)
	}
}
