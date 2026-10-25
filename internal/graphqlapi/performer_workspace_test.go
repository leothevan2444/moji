package graphqlapi

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	performerdomain "github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/subscription"
)

func TestPerformerWorkspaceBuildsListAndSubscriptionsFromOneSnapshot(t *testing.T) {
	performers := &workspacePerformerService{items: []performerdomain.Performer{{ID: "p1", Name: "Alpha", Subscribed: true}, {ID: "p2", Name: "Beta"}}}
	subscriptions := &workspaceSubscriptionService{}
	resolver := &queryResolver{Resolver: &Resolver{Performer: performers, PerformerSubscription: subscriptions}}
	search, page, pageSize := "alp", 1, 24
	snapshot, err := resolver.PerformerWorkspace(context.Background(), &search, &page, &pageSize, nil)
	if err != nil {
		t.Fatalf("PerformerWorkspace: %v", err)
	}
	if performers.listCalls != 1 || subscriptions.buildCalls != 1 || subscriptions.receivedCount != 2 {
		t.Fatalf("expected one shared snapshot, performer calls=%d subscription calls=%d received=%d", performers.listCalls, subscriptions.buildCalls, subscriptions.receivedCount)
	}
	if snapshot.Performers.TotalCount != 1 || snapshot.Performers.Items[0].ID != "p1" || len(snapshot.SubscribedPerformers) != 1 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestSubscribePerformersResolverMapsBatchPayload(t *testing.T) {
	subscriptions := &workspaceSubscriptionService{batch: subscription.PerformerBatchPayload{
		BatchID: "performer-batch-1",
		Summary: subscription.PerformerBatchSummary{RequestedCount: 1, SucceededCount: 1},
		Results: []subscription.PerformerBatchResult{{PerformerID: "p1", Status: subscription.PerformerBatchStatusSucceeded, ReasonCode: subscription.PerformerBatchReasonSubscribed, Performer: &performerdomain.Performer{ID: "p1", Name: "Alpha", Subscribed: true}}},
	}}
	resolver := &mutationResolver{Resolver: &Resolver{PerformerSubscription: subscriptions}}
	payload, err := resolver.SubscribePerformers(context.Background(), []string{"p1"})
	if err != nil {
		t.Fatalf("SubscribePerformers: %v", err)
	}
	if payload.BatchID != "performer-batch-1" || payload.Summary.SucceededCount != 1 || payload.Results[0].ReasonCode != subscription.PerformerBatchReasonSubscribed || payload.Results[0].Performer == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

type workspacePerformerService struct {
	items     []performerdomain.Performer
	listCalls int
}

func (s *workspacePerformerService) List(context.Context, string) ([]performerdomain.Performer, error) {
	s.listCalls++
	return append([]performerdomain.Performer(nil), s.items...), nil
}
func (s *workspacePerformerService) QueuePerformerScenes(context.Context, string, []performerdomain.QueueSceneSelection) (performerdomain.QueueScenesResult, error) {
	return performerdomain.QueueScenesResult{}, nil
}
func (s *workspacePerformerService) GetPerformerDetail(context.Context, string) (performerdomain.Detail, error) {
	return performerdomain.Detail{}, nil
}
func (s *workspacePerformerService) ListPerformerScenes(context.Context, string, performerdomain.SceneQuery) (performerdomain.ScenePage, error) {
	return performerdomain.ScenePage{}, nil
}
func (s *workspacePerformerService) RefreshPerformerScenes(context.Context, string, performerdomain.SceneQuery) (performerdomain.ScenePage, error) {
	return performerdomain.ScenePage{}, nil
}

type workspaceSubscriptionService struct {
	batch         subscription.PerformerBatchPayload
	buildCalls    int
	receivedCount int
}

func (s *workspaceSubscriptionService) ListSubscribedPerformers(context.Context) ([]subscription.SubscribedPerformer, error) {
	return nil, nil
}
func (s *workspaceSubscriptionService) BuildSubscribedPerformers(_ context.Context, performers []performerdomain.Performer) ([]subscription.SubscribedPerformer, error) {
	s.buildCalls++
	s.receivedCount = len(performers)
	out := make([]subscription.SubscribedPerformer, 0)
	for _, performer := range performers {
		if performer.Subscribed {
			out = append(out, subscription.SubscribedPerformer{Performer: performer})
		}
	}
	return out, nil
}
func (s *workspaceSubscriptionService) SubscribePerformer(context.Context, string) (subscription.SubscribedPerformer, error) {
	return subscription.SubscribedPerformer{}, nil
}
func (s *workspaceSubscriptionService) UnsubscribePerformer(context.Context, string) error {
	return nil
}
func (s *workspaceSubscriptionService) RefreshSubscribedPerformer(context.Context, string) (subscription.SubscribedPerformer, error) {
	return subscription.SubscribedPerformer{}, nil
}
func (s *workspaceSubscriptionService) RefreshAll(context.Context) ([]subscription.SubscribedPerformer, error) {
	return nil, nil
}
func (s *workspaceSubscriptionService) SubscribePerformers(context.Context, []string) (subscription.PerformerBatchPayload, error) {
	return s.batch, nil
}
func (s *workspaceSubscriptionService) UnsubscribePerformers(context.Context, []string) (subscription.PerformerBatchPayload, error) {
	return s.batch, nil
}
func (s *workspaceSubscriptionService) RefreshSubscribedPerformers(context.Context, []string) (subscription.PerformerBatchPayload, error) {
	return s.batch, nil
}

func TestPerformerListOptionsApplyAcrossPagesAndKeepReviewOnlyActors(t *testing.T) {
	items := []performerdomain.Performer{
		{ID: "a", Name: "Alpha", SceneCount: 2, Subscribed: true},
		{ID: "b", Name: "Beta", SceneCount: 20, Favorite: true},
		{ID: "c", Name: "Charlie", SceneCount: 10, Subscribed: true},
	}
	lastCheck, failed := "2026-10-24T10:00:00Z", "check failed"
	states := []*model.SubscribedPerformer{
		{Performer: &model.StashPerformer{ID: "a"}, LastError: &failed},
		{Performer: &model.StashPerformer{ID: "c"}, LastCheckedAt: &lastCheck},
	}
	for _, test := range []struct {
		filter model.PerformerListFilter
		want   string
	}{
		{model.PerformerListFilterSubscribed, "ac"},
		{model.PerformerListFilterFavorite, "b"},
		{model.PerformerListFilterReview, "b"},
		{model.PerformerListFilterError, "a"},
	} {
		t.Run(string(test.filter), func(t *testing.T) {
			result := filterAndSortPerformers(items, states, map[string]int{"b": 2}, &model.PerformerListOptions{Filter: &test.filter})
			var ids string
			for _, item := range result {
				ids += item.ID
			}
			if ids != test.want {
				t.Fatalf("got %q, want %q", ids, test.want)
			}
		})
	}
	for _, test := range []struct {
		sort model.PerformerListSort
		want string
	}{
		{model.PerformerListSortName, "a"},
		{model.PerformerListSortSceneCount, "b"},
		{model.PerformerListSortLastChecked, "c"},
	} {
		result := filterAndSortPerformers(items, states, nil, &model.PerformerListOptions{Sort: &test.sort})
		page, pageSize := 1, 1
		paged := buildStashPerformerPage(result, "", &page, &pageSize)
		if paged.Items[0].ID != test.want || paged.TotalCount != 3 {
			t.Fatalf("sort %s produced %#v", test.sort, paged)
		}
	}
	if items[0].ID != "a" {
		t.Fatal("options changed the shared input snapshot")
	}
}
