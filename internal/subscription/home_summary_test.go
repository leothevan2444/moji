package subscription

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/performer"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestHomeSummaryUsesCurrentSubscriptionMembership(t *testing.T) {
	ctx := context.Background()
	sqlite, err := NewSQLiteStore(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.db.Close()
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for name, store := range map[string]Store{"sqlite": sqlite, "memory": NewMemoryStore()} {
		t.Run(name, func(t *testing.T) {
			stash := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}
			for i := 0; i < 9; i++ {
				id := fmt.Sprintf("p-%02d", i)
				stash.performers[id] = &stashgraphql.PerformerFragment{ID: id, CustomFields: map[string]any{DefaultCustomFieldKey: i < 4 || i == 8}}
				if i == 8 {
					continue
				}
				state := &PerformerState{PerformerID: fmt.Sprintf("p-%02d", i)}
				if i < 5 {
					state.LastError = "upstream unavailable"
				}
				if i > 0 {
					checked := now.Add(-time.Duration(i) * time.Minute)
					state.LastCheckedAt = &checked
				}
				if err := store.Put(ctx, state); err != nil {
					t.Fatal(err)
				}
			}
			// Unsubscribed check states must not inflate membership or latest issues.
			// A newly subscribed actor without state must still count as unchecked.
			summary, err := (&Service{store: store, stash: stash, customFieldKey: DefaultCustomFieldKey}).HomeSummary(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if summary.Total != 5 || summary.Attention != 4 || summary.Unchecked != 2 || summary.LastCheckedAt == nil || !summary.LastCheckedAt.Equal(now.Add(-time.Minute)) {
				t.Fatalf("incorrect summary: %+v", summary)
			}
			if len(summary.AttentionPerformerIDs) != 3 || summary.AttentionPerformerIDs[0] != "p-00" {
				t.Fatalf("expected bounded issue links: %+v", summary)
			}
		})
	}
}

type homeSummaryStash struct {
	*fakeStashClient
	err error
}

func (s *homeSummaryStash) AllPerformers(context.Context) ([]*stashgraphql.PerformerFragment, error) {
	panic("homepage must not fetch full actor metadata")
}
func (s *homeSummaryStash) PerformerSubscriptionFlags(ctx context.Context) ([]*stashgraphql.PerformerSubscriptionFlags_AllPerformers, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.fakeStashClient.PerformerSubscriptionFlags(ctx)
}

func TestHomeSummaryLatestOutcomeAndMembershipFailure(t *testing.T) {
	ctx := context.Background()
	stash := &homeSummaryStash{fakeStashClient: &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"actor": {ID: "actor", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
	}}}
	store := NewMemoryStore()
	started := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	if err := store.Put(ctx, &PerformerState{PerformerID: "actor", LastCheckedAt: &started, LastError: "previous failure"}); err != nil {
		t.Fatal(err)
	}
	newer := started.Add(time.Hour)
	service := &Service{stash: stash, store: store, customFieldKey: DefaultCustomFieldKey, checks: &checkTracker{runs: []CheckRun{{StartedAt: newer, Items: []CheckItem{{PerformerID: "actor", Status: "SUCCEEDED"}}}}}}
	summary, err := service.HomeSummary(ctx)
	if err != nil || summary.Attention != 0 {
		t.Fatalf("successful recovery must clear old failure: %+v %v", summary, err)
	}
	service.checks.runs[0].Items[0].Status = "INTERRUPTED"
	summary, err = service.HomeSummary(ctx)
	if err != nil || summary.Attention != 1 {
		t.Fatalf("latest interruption must count once: %+v %v", summary, err)
	}
	service.checks.runs[0].Items[0].Status = "RUNNING"
	summary, err = service.HomeSummary(ctx)
	if err != nil || summary.Attention != 1 {
		t.Fatalf("running check must retain previous outcome: %+v %v", summary, err)
	}
	stash.err = errors.New("Stash unavailable")
	if _, err = service.HomeSummary(ctx); !errors.Is(err, stash.err) {
		t.Fatalf("membership failure must not appear as zero: %v", err)
	}
	stash.err = nil
	stash.performers = map[string]*stashgraphql.PerformerFragment{}
	summary, err = service.HomeSummary(ctx)
	if err != nil || summary.Total != 0 || summary.Attention != 0 || summary.Unchecked != 0 {
		t.Fatalf("unsubscribed actor must disappear: %+v %v", summary, err)
	}
}

func TestHomeAndActorWorkspaceShareLatestCheckOutcomes(t *testing.T) {
	ctx := context.Background()
	stash := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"subscribed": {ID: "subscribed", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
		"manual":     {ID: "manual"},
	}}
	service, err := newServiceForTest(stash, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	service.checks.runs = []CheckRun{{StartedAt: started, Error: "SERVICE_RESTARTED", Items: []CheckItem{
		{PerformerID: "subscribed", Status: "INTERRUPTED"}, {PerformerID: "manual", Status: "FAILED"},
	}}}
	actors := []performer.Performer{{ID: "subscribed", Subscribed: true}, {ID: "manual"}}
	for _, status := range []string{"INTERRUPTED", "SUCCEEDED"} {
		service.checks.runs[0].Items[0].Status = status
		summary, err := service.HomeSummary(ctx)
		if err != nil {
			t.Fatal(err)
		}
		workspace, err := service.BuildSubscribedPerformers(ctx, actors)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if status == "INTERRUPTED" {
			want = 1
		}
		if len(workspace) != 1 || summary.Total != 1 || summary.Attention != want || (workspace[0].LastError != "") != (want == 1) {
			t.Fatalf("Home/workspace must agree for %s: home=%+v actors=%+v", status, summary, workspace)
		}
	}
}
