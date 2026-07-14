package graphqlapi

import (
	"context"
	"testing"

	performerdomain "github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/subscription"
)

func TestPerformerWorkspaceBuildsListAndSubscriptionsFromOneSnapshot(t *testing.T) {
	performers := &workspacePerformerService{items: []performerdomain.Performer{{ID: "p1", Name: "Alpha", Subscribed: true}, {ID: "p2", Name: "Beta"}}}
	subscriptions := &workspaceSubscriptionService{}
	resolver := &queryResolver{Resolver: &Resolver{Performer: performers, PerformerSubscription: subscriptions}}
	search, page, pageSize := "alp", 1, 24
	snapshot, err := resolver.PerformerWorkspace(context.Background(), &search, &page, &pageSize)
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
