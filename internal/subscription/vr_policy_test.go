package subscription

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestVRReviewAndSkipNeverCreateTasksAcrossChecks(t *testing.T) {
	for _, behavior := range []config.SubscriptionReleaseBehavior{
		config.SubscriptionReleaseBehaviorReview,
		config.SubscriptionReleaseBehaviorBlock,
		config.SubscriptionReleaseBehaviorDownload,
	} {
		t.Run(string(behavior), func(t *testing.T) {
			const endpoint = "https://metadata.example.org/graphql"
			stashClient := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
				"p1": {ID: "p1", Name: "Actor", CustomFields: map[string]any{DefaultCustomFieldKey: true},
					StashIds: []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "box-p1"}}},
			}}
			registry := metadata.NewRegistry(stubFactory{client: &fakeStashboxClient{
				performer: &stashboxgraphql.PerformerFragment{ID: "box-p1", Name: "Actor"},
				scenes:    []*stashboxgraphql.SceneFragment{{ID: "s1", Code: stringPointer("ABC-VR-123"), Title: stringPointer("Movie")}},
			}})
			registry.Replace([]stash.StashBoxEndpoint{{Name: "metadata", Endpoint: endpoint}})
			creator := &fakeTaskCreator{subscriptionTask: &taskruntime.Task{ID: "task-vr"}}
			store := NewMemoryStore()
			service, err := newServiceForTest(stashClient, registry, nil, store)
			if err != nil {
				t.Fatal(err)
			}
			service.SetTaskCreator(creator)
			policy := config.DefaultSubscriptionReleasePolicyConfig()
			policy.VRBehavior = behavior
			service.SetReleasePolicy(policy)
			for check := 0; check < 2; check++ {
				if _, err := service.RefreshSubscribedPerformer(context.Background(), "p1"); err != nil {
					t.Fatal(err)
				}
			}
			state, err := store.Get(context.Background(), "p1")
			if err != nil {
				t.Fatal(err)
			}
			if behavior == config.SubscriptionReleaseBehaviorDownload {
				if len(creator.subscriptionCalls) != 1 || len(state.ProcessedReleases) != 1 || state.ProcessedReleases[0].TaskID != "task-vr" {
					t.Fatalf("download was not queued exactly once: calls=%v state=%+v", creator.subscriptionCalls, state)
				}
				return
			}
			if len(creator.subscriptionCalls) != 0 || len(state.ProcessedReleases) != 0 || len(state.PendingReleases) != 1 || state.PendingReleases[0].TaskID != "" {
				t.Fatalf("review or skip created a task or lost its record: calls=%v state=%+v", creator.subscriptionCalls, state)
			}
			want := ReleaseDecisionQueued
			if behavior == config.SubscriptionReleaseBehaviorBlock {
				want = ReleaseDecisionBlocked
			}
			if state.PendingReleases[0].Decision != want {
				t.Fatalf("record decision = %s, want %s", state.PendingReleases[0].Decision, want)
			}
		})
	}
}
