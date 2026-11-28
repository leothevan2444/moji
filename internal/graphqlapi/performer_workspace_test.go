package graphqlapi

import (
	"context"
	"errors"
	"testing"
	"time"

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
	sceneQueries []performerdomain.SceneQuery
	items        []performerdomain.Performer
	listCalls    int
}

func (s *workspacePerformerService) GetPerformerProfile(ctx context.Context, id string) (performerdomain.Detail, error) {
	return s.GetPerformerDetail(ctx, id)
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
func (s *workspacePerformerService) ListPerformerScenes(_ context.Context, _ string, query performerdomain.SceneQuery) (performerdomain.ScenePage, error) {
	s.sceneQueries = append(s.sceneQueries, query)
	return performerdomain.ScenePage{}, nil
}
func (s *workspacePerformerService) RefreshPerformerScenes(_ context.Context, _ string, query performerdomain.SceneQuery) (performerdomain.ScenePage, error) {
	s.sceneQueries = append(s.sceneQueries, query)
	return performerdomain.ScenePage{}, nil
}

type workspaceSubscriptionService struct {
	batch         subscription.PerformerBatchPayload
	buildCalls    int
	receivedCount int
	activityIDs   []string
	activityCalls int
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
		{Performer: &model.StashPerformer{ID: "b"}, LastError: &failed},
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
			result := filterAndSortPerformers(items, states, map[string]int{"b": 2}, &model.PerformerListOptions{Filter: &test.filter}, nil)
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
		direction := model.SortDirectionDesc
		if test.sort == model.PerformerListSortName {
			direction = model.SortDirectionAsc
		}
		result := filterAndSortPerformers(items, states, nil, &model.PerformerListOptions{Sort: &test.sort, Direction: &direction}, nil)
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

func (s *workspaceSubscriptionService) PerformerActivities(_ context.Context, ids []string) (map[string]subscription.ActivityState, error) {
	s.activityIDs = append([]string(nil), ids...)
	s.activityCalls++
	value, rank := 0.0, 1
	out := map[string]subscription.ActivityState{}
	for _, id := range ids {
		out[id] = subscription.ActivityState{Value: &value, Rank: &rank}
	}
	return out, nil
}

func TestPerformerWorkspaceProjectsActivityOnlyForVisibleActorsIncludingUnsubscribed(t *testing.T) {
	performers := &workspacePerformerService{items: []performerdomain.Performer{{ID: "p1", Name: "Alpha", Subscribed: true}, {ID: "p2", Name: "Beta"}, {ID: "p3", Name: "Gamma"}}}
	subscriptions := &workspaceSubscriptionService{}
	resolver := &queryResolver{Resolver: &Resolver{Performer: performers, PerformerSubscription: subscriptions}}
	page, size := 2, 1
	snapshot, err := resolver.PerformerWorkspace(context.Background(), nil, &page, &size, nil)
	if err != nil {
		t.Fatal(err)
	}
	if subscriptions.activityCalls != 1 || len(subscriptions.activityIDs) != 1 || subscriptions.activityIDs[0] != "p2" {
		t.Fatalf("unbounded activity projection: %+v", subscriptions)
	}
	if len(snapshot.Activities) != 1 || snapshot.Activities[0].ID != "p2" || snapshot.Activities[0].Rank == nil || *snapshot.Activities[0].Rank != 1 || snapshot.Performers.Items[0].Subscribed {
		t.Fatalf("unsubscribed rating not preserved: %+v", snapshot)
	}
}

func TestPerformerCastFilterReachesListAndRefresh(t *testing.T) {
	service := &workspacePerformerService{}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.Performer = service
	for _, operation := range []string{`query { stashPerformerScenes(id:"p1",input:{cast:SOLO}) {totalCount} }`, `mutation { refreshStashPerformerScenes(id:"p1",input:{cast:GROUP}) {totalCount} }`} {
		response := executeGraphQL(t, resolver, operation)
		if len(response.Errors) > 0 {
			t.Fatal(response.Errors)
		}
	}
	if len(service.sceneQueries) != 2 || service.sceneQueries[0].Cast != performerdomain.SceneCastFilterSolo || service.sceneQueries[1].Cast != performerdomain.SceneCastFilterGroup {
		t.Fatal("cast filter lost at API boundary", service.sceneQueries)
	}
}

func TestPerformerFilterCountsMatchEachCategoryAndSearchBeforePagination(t *testing.T) {
	items := []performerdomain.Performer{
		{ID: "a", Name: "Alpha", Subscribed: true, Favorite: true},
		{ID: "b", Name: "Beta", AliasList: []string{"ALPHA alias"}, Favorite: true},
		{ID: "c", Name: "Gamma", Subscribed: true},
	}
	failure := "latest check failed"
	states := []*model.SubscribedPerformer{
		{Performer: &model.StashPerformer{ID: "a"}, LastError: &failure},
		{Performer: &model.StashPerformer{ID: "b"}, LastError: &failure},
	}
	reviews := map[string]int{"a": 9, "b": 3}
	for _, search := range []string{"", " ALPHA ", "unmatched"} {
		counts := performerFilterCounts(items, states, reviews, search, true)
		for _, category := range []struct {
			filter model.PerformerListFilter
			count  int
		}{
			{model.PerformerListFilterAll, counts.Total}, {model.PerformerListFilterSubscribed, counts.Subscribed},
			{model.PerformerListFilterFavorite, counts.Favorite}, {model.PerformerListFilterReview, *counts.Review},
			{model.PerformerListFilterError, counts.Error},
		} {
			page, size := 2, 1
			result := buildStashPerformerPage(filterAndSortPerformers(items, states, reviews, &model.PerformerListOptions{Filter: &category.filter}, nil), search, &page, &size)
			if result.TotalCount != category.count {
				t.Fatalf("%q/%s: list=%d badge=%d", search, category.filter, result.TotalCount, category.count)
			}
		}
	}
	counts := performerFilterCounts(items, states, reviews, "alpha", true)
	if counts.Total != 2 || counts.Subscribed != 1 || counts.Favorite != 2 || *counts.Review != 2 || counts.Error != 1 {
		t.Fatalf("overlapping actor categories counted incorrectly: %+v review=%d", counts, *counts.Review)
	}
	if unknown := performerFilterCounts(items, states, nil, "", false); unknown.Review != nil || unknown.Total != 3 {
		t.Fatalf("unknown review count replaced with zero: %+v", unknown)
	}
}

// One failed optional review summary must not turn all actor counts into zero.
type filterReviewService struct {
	reviewAPIService
	unavailable bool
	calls       int
}

func (s *filterReviewService) PendingReviewPerformerCounts(context.Context) (map[string]int, error) {
	s.calls++
	if s.unavailable {
		return nil, errors.New("review storage unavailable")
	}
	return map[string]int{"p1": 12}, nil
}
func TestPerformerWorkspaceCountsAndOptionalReviewFailure(t *testing.T) {
	service := &filterReviewService{}
	resolver := &queryResolver{Resolver: &Resolver{Performer: &workspacePerformerService{items: []performerdomain.Performer{{ID: "p1", Name: "Alpha", Favorite: true}, {ID: "p2", Name: "Beta"}}}, PerformerSubscription: service}}
	search := "Alpha"
	result, err := resolver.PerformerWorkspace(context.Background(), &search, nil, nil, nil)
	if err != nil || result.FilterCounts.Total != 1 || result.FilterCounts.Favorite != 1 || result.FilterCounts.Review == nil || *result.FilterCounts.Review != 1 || service.calls != 1 {
		t.Fatalf("invalid counts or repeated summary: %+v %v", result, err)
	}
	service.unavailable = true
	result, err = resolver.PerformerWorkspace(context.Background(), &search, nil, nil, nil)
	if err != nil || result.FilterCounts.Review != nil || result.FilterCounts.Total != 1 {
		t.Fatalf("optional summary failure invalidated list: %+v %v", result, err)
	}
	review := model.PerformerListFilterReview
	if _, err = resolver.PerformerWorkspace(context.Background(), &search, nil, nil, &model.PerformerListOptions{Filter: &review}); err == nil {
		t.Fatal("failed review query presented an empty review category")
	}
}

func TestPerformerSortingHasIndependentDirectionAndUnknownTimesLast(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	items := []performerdomain.Performer{{ID: "a", Name: "Actor 10", SceneCount: 10, CreatedAt: now.Add(time.Hour), Subscribed: true}, {ID: "b", Name: "actor 2", SceneCount: 2}, {ID: "c", Name: "Actor 1", CreatedAt: now}}
	checks := map[string]time.Time{"a": now, "c": now.Add(time.Hour)}
	for _, test := range []struct {
		field     model.PerformerListSort
		asc, desc string
	}{{model.PerformerListSortName, "cba", "abc"}, {model.PerformerListSortCreated, "cab", "acb"}, {model.PerformerListSortSceneCount, "cba", "abc"}, {model.PerformerListSortLastChecked, "acb", "cab"}} {
		for _, direction := range []model.SortDirection{model.SortDirectionAsc, model.SortDirectionDesc} {
			result := filterAndSortPerformers(items, nil, nil, &model.PerformerListOptions{Sort: &test.field, Direction: &direction}, checks)
			got := ""
			for _, item := range result {
				got += item.ID
			}
			want := test.asc
			if direction == model.SortDirectionDesc {
				want = test.desc
			}
			if got != want {
				t.Fatalf("%s/%s: %s != %s", test.field, direction, got, want)
			}
		}
	}
	defaultOrder := filterAndSortPerformers(items, nil, nil, nil, nil)
	if defaultOrder[0].ID != "c" {
		t.Fatal("default still prioritizes subscriptions")
	}
}

type workspaceCheckTimesService struct {
	workspaceSubscriptionService
	ids []string
}

func (s *workspaceCheckTimesService) PerformerCheckTimes(_ context.Context, ids []string) (map[string]time.Time, error) {
	s.ids = append([]string(nil), ids...)
	return map[string]time.Time{"unsubscribed": time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func TestPerformerWorkspaceSortsRetainedChecksWithoutChangingSubscriptionMembership(t *testing.T) {
	service := &workspaceCheckTimesService{}
	resolver := &queryResolver{Resolver: &Resolver{Performer: &workspacePerformerService{items: []performerdomain.Performer{{ID: "subscribed", Name: "Alpha", Subscribed: true}, {ID: "unsubscribed", Name: "Beta"}}}, PerformerSubscription: service}}
	field, direction := model.PerformerListSortLastChecked, model.SortDirectionDesc
	page, err := resolver.PerformerWorkspace(context.Background(), nil, nil, nil, &model.PerformerListOptions{Sort: &field, Direction: &direction})
	if err != nil {
		t.Fatal(err)
	}
	if len(service.ids) != 2 || page.Performers.Items[0].ID != "unsubscribed" || len(page.SubscribedPerformers) != 1 || page.FilterCounts.Subscribed != 1 {
		t.Fatalf("check sorting changed membership or dropped history: %+v ids=%v", page, service.ids)
	}
}
