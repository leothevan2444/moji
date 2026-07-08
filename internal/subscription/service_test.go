package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type fakeStashClient struct {
	performers map[string]*stashgraphql.PerformerFragment
	scenes     []*stashgraphql.SceneFragment
	boxes      []stash.StashBoxEndpoint
}

func (f *fakeStashClient) AllPerformers(_ context.Context) ([]*stashgraphql.PerformerFragment, error) {
	out := make([]*stashgraphql.PerformerFragment, 0, len(f.performers))
	for _, performer := range f.performers {
		out = append(out, performer)
	}
	return out, nil
}

func (f *fakeStashClient) FindPerformerByID(_ context.Context, id string) (*stashgraphql.PerformerFragment, error) {
	return f.performers[id], nil
}

func (f *fakeStashClient) FindScenes(_ context.Context, sceneFilter *stashgraphql.SceneFilterType, _ *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error) {
	if sceneFilter != nil && sceneFilter.StashIDEndpoint != nil && sceneFilter.StashIDEndpoint.StashID != nil {
		out := make([]*stashgraphql.SceneFragment, 0)
		for _, scene := range f.scenes {
			if scene == nil {
				continue
			}
			for _, stashID := range scene.StashIds {
				if stashID == nil {
					continue
				}
				endpoint := ""
				if sceneFilter.StashIDEndpoint.Endpoint != nil {
					endpoint = *sceneFilter.StashIDEndpoint.Endpoint
				}
				if stashID.StashID == *sceneFilter.StashIDEndpoint.StashID && normalizeStashBoxEndpoint(stashID.Endpoint) == normalizeStashBoxEndpoint(endpoint) {
					out = append(out, scene)
					break
				}
			}
		}
		return out, nil
	}
	return append([]*stashgraphql.SceneFragment(nil), f.scenes...), nil
}

func (f *fakeStashClient) UpdatePerformerCustomFields(_ context.Context, id string, partial map[string]any, remove []string) (*stashgraphql.PerformerFragment, error) {
	performer := f.performers[id]
	if performer.CustomFields == nil {
		performer.CustomFields = map[string]any{}
	}
	for key, value := range partial {
		performer.CustomFields[key] = value
	}
	for _, key := range remove {
		delete(performer.CustomFields, key)
	}
	return performer, nil
}

func (f *fakeStashClient) GetStashBoxes(_ context.Context) ([]stash.StashBoxEndpoint, error) {
	return f.boxes, nil
}

type fakeStashboxClient struct {
	performer    *stashboxgraphql.PerformerFragment
	scenes       []*stashboxgraphql.SceneFragment
	searchErr    error
	findSceneErr error
}

type fakeDownloader struct {
	tasks   []*downloader.Task
	err     error
	calls   int
	queries []string
	sources []downloader.TaskSource
}

func (f *fakeStashboxClient) FindPerformerByID(_ context.Context, id string) (*stashboxgraphql.PerformerFragment, error) {
	if f.performer != nil && f.performer.ID == id {
		return f.performer, nil
	}
	return nil, nil
}

func (f *fakeStashboxClient) FindSceneByID(_ context.Context, id string) (*stashboxgraphql.SceneFragment, error) {
	if f.findSceneErr != nil {
		return nil, f.findSceneErr
	}
	for _, scene := range f.scenes {
		if scene != nil && scene.ID == id {
			return scene, nil
		}
	}
	return nil, nil
}

func (f *fakeStashboxClient) SearchPerformer(_ context.Context, _ string) ([]*stashboxgraphql.PerformerFragment, error) {
	if f.performer == nil {
		return nil, nil
	}
	return []*stashboxgraphql.PerformerFragment{f.performer}, nil
}

func (f *fakeStashboxClient) SearchScene(_ context.Context, _ string) ([]*stashboxgraphql.SceneFragment, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return append([]*stashboxgraphql.SceneFragment(nil), f.scenes...), nil
}

func (f *fakeStashboxClient) QueryScenes(_ context.Context, _ stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error) {
	out := make([]*stashboxgraphql.SceneFragment, 0, len(f.scenes))
	for _, scene := range f.scenes {
		if scene == nil {
			continue
		}
		cloned := *scene
		if len(cloned.Performers) == 0 && f.performer != nil {
			cloned.Performers = []*stashboxgraphql.PerformerAppearanceFragment{
				{Performer: &stashboxgraphql.PerformerFragment{ID: f.performer.ID, Name: f.performer.Name}},
			}
		}
		out = append(out, &cloned)
	}
	return out, nil
}

func (f *fakeDownloader) DownloadMediaContext(_ context.Context, req downloader.DownloadRequest) (*downloader.Task, error) {
	f.calls++
	f.queries = append(f.queries, req.Query)
	f.sources = append(f.sources, req.Source)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.tasks) == 0 {
		return nil, nil
	}
	task := f.tasks[0]
	f.tasks = f.tasks[1:]
	return task, nil
}

// stubFactory is a StashboxClientFactory that always returns the same client
// regardless of the endpoint. Tests that need per-endpoint dispatch should
// use perEndpointFactory instead.
type stubFactory struct {
	client StashboxClient
}

func (s stubFactory) NewClient(stash.StashBoxEndpoint) StashboxClient { return s.client }

// perEndpointFactory returns a different fakeStashboxClient for each
// endpoint. The client assigned to an unknown endpoint panics so tests catch
// accidental mis-routing.
type perEndpointFactory struct {
	clients map[string]StashboxClient
}

func (f perEndpointFactory) NewClient(box stash.StashBoxEndpoint) StashboxClient {
	client, ok := f.clients[normalizeStashBoxEndpoint(box.Endpoint)]
	if !ok {
		panic("perEndpointFactory: no client registered for " + box.Endpoint)
	}
	return client
}

func TestListStashPerformersMarksCustomFieldSubscribers(t *testing.T) {
	service, err := NewService(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Yua Mikami",
				AliasList:    []string{"Mikami"},
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
			},
			"p2": {
				ID:        "p2",
				Name:      "Aoi Sora",
				AliasList: []string{"Sola Aoi"},
			},
		},
	}, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	items, err := service.ListStashPerformers(context.Background(), "mik")
	if err != nil {
		t.Fatalf("ListStashPerformers failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 performer, got %d", len(items))
	}
	if !items[0].Subscribed {
		t.Fatalf("expected performer to be subscribed")
	}
}

func TestSubscribeAndUnsubscribePerformerMutatesCustomFields(t *testing.T) {
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {ID: "p1", Name: "Kana Momonogi"},
		},
	}

	service, err := NewService(stashClient, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.SubscribePerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("SubscribePerformer failed: %v", err)
	}
	if !item.Performer.Subscribed {
		t.Fatalf("expected performer to be subscribed")
	}

	if err := service.UnsubscribePerformer(context.Background(), "p1"); err != nil {
		t.Fatalf("UnsubscribePerformer failed: %v", err)
	}
	if customFieldTruthy(stashClient.performers["p1"].CustomFields, DefaultCustomFieldKey) {
		t.Fatalf("expected custom field to be removed")
	}
}

func TestRefreshPerformerStoresPendingReleasesWithoutDownloader(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				AliasList:    []string{"RION"},
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds: []*stashgraphql.StashIDFragment{
					{Endpoint: endpoint, StashID: "js-1"},
				},
			},
		},
	}
	title := "New Release"
	code := "ABCD-123"
	date := "2026-06-01"
	url := "https://javstash.example.org/scenes/js-scene-1"

	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenes: []*stashboxgraphql.SceneFragment{
				{
					ID:    "js-scene-1",
					Title: &title,
					Code:  &code,
					Date:  &date,
					Urls:  []*stashboxgraphql.URLFragment{{URL: url}},
				},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{
		Name:     "javstash",
		Endpoint: endpoint,
		APIKey:   "ignored",
	}})

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 1 {
		t.Fatalf("expected 1 pending release, got %d", item.PendingReleaseCount)
	}
	if len(item.RecentReleases) != 1 {
		t.Fatalf("expected 1 recent release, got %d", len(item.RecentReleases))
	}
	if item.RecentReleases[0].Query != code {
		t.Fatalf("expected query %q, got %q", code, item.RecentReleases[0].Query)
	}
	if got, want := item.RecentReleases[0].Source, "stash-box:"+endpoint; got != want {
		t.Fatalf("unexpected source %q, want %q", got, want)
	}
}

func TestRefreshPerformerDoesNotDuplicatePendingReleasesAcrossPolls(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	code := "ABCD-123"
	title := "New Release"

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}
	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenes: []*stashboxgraphql.SceneFragment{{
				ID:    "js-scene-1",
				Title: &title,
				Code:  &code,
			}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	first, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}
	second, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("second refresh failed: %v", err)
	}

	if first.PendingReleaseCount != 1 || second.PendingReleaseCount != 1 {
		t.Fatalf("expected stable pending count of 1, got first=%d second=%d", first.PendingReleaseCount, second.PendingReleaseCount)
	}
	if len(second.RecentReleases) != 1 {
		t.Fatalf("expected 1 recent release after repeated refresh, got %d", len(second.RecentReleases))
	}
	if second.RecentReleases[0].Key != "stashbox:https___javstash.example.org_graphql:js-scene-1" {
		t.Fatalf("unexpected release key after repeated refresh: %q", second.RecentReleases[0].Key)
	}
}

func TestRefreshPerformerKeepsFailedAutoDownloadsPending(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	code := "ABCD-123"
	title := "New Release"

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}
	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenes: []*stashboxgraphql.SceneFragment{{
				ID:    "js-scene-1",
				Title: &title,
				Code:  &code,
			}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})

	downloader := &fakeDownloader{err: errors.New("temporary add failure")}
	service, err := NewService(stashClient, registry, downloader, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	first, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("refresh should return performer state even when auto-download fails: %v", err)
	}
	second, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("second refresh should still return performer state: %v", err)
	}

	if first.PendingReleaseCount != 1 || second.PendingReleaseCount != 1 {
		t.Fatalf("expected failed release to remain pending once, got first=%d second=%d", first.PendingReleaseCount, second.PendingReleaseCount)
	}
	if downloader.calls != 1 {
		t.Fatalf("expected downloader to be called once for the new release, got %d", downloader.calls)
	}
	if second.LastError == "" {
		t.Fatalf("expected last error to be preserved after auto-download failure")
	}
}

func TestFetchReleasesUsesMultipleStashBoxes(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"

	clientA := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "a-1", Name: "Hibiki Otsuki"},
		scenes: []*stashboxgraphql.SceneFragment{{
			ID:   "a-scene-1",
			Code: stringPtr("A-001"),
		}},
	}
	clientB := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "b-1", Name: "Tsubasa Amami"},
		scenes: []*stashboxgraphql.SceneFragment{{
			ID:   "b-scene-1",
			Code: stringPtr("B-001"),
		}},
	}

	registry := newStashboxRegistry(perEndpointFactory{
		clients: map[string]StashboxClient{
			normalizeStashBoxEndpoint(endpointA): clientA,
			normalizeStashBoxEndpoint(endpointB): clientB,
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "stashbox-a", Endpoint: endpointA, APIKey: "k-a"},
		{Name: "stashbox-b", Endpoint: endpointB, APIKey: "k-b"},
	})

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"pa": {
				ID:           "pa",
				Name:         "Hibiki Otsuki",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpointA, StashID: "a-1"}},
			},
			"pb": {
				ID:           "pb",
				Name:         "Tsubasa Amami",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpointB, StashID: "b-1"}},
			},
		},
	}

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	a, err := service.RefreshSubscribedPerformer(context.Background(), "pa")
	if err != nil {
		t.Fatalf("refresh pa failed: %v", err)
	}
	b, err := service.RefreshSubscribedPerformer(context.Background(), "pb")
	if err != nil {
		t.Fatalf("refresh pb failed: %v", err)
	}

	if a.PendingReleaseCount != 1 || b.PendingReleaseCount != 1 {
		t.Fatalf("expected 1 release per performer, got a=%d b=%d", a.PendingReleaseCount, b.PendingReleaseCount)
	}
	if a.RecentReleases[0].Source != "stash-box:"+endpointA {
		t.Fatalf("unexpected source for a: %q", a.RecentReleases[0].Source)
	}
	if b.RecentReleases[0].Source != "stash-box:"+endpointB {
		t.Fatalf("unexpected source for b: %q", b.RecentReleases[0].Source)
	}
}

func TestRefreshStashBoxesReplacesRegistry(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"

	stashClient := &fakeStashClient{
		boxes: []stash.StashBoxEndpoint{
			{Name: "stashbox-a", Endpoint: endpointA, APIKey: "k-a"},
			{Name: "stashbox-b", Endpoint: endpointB, APIKey: "k-b"},
		},
	}
	registry := NewDefaultStashboxRegistry()
	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	if err := service.RefreshStashBoxes(context.Background()); err != nil {
		t.Fatalf("RefreshStashBoxes failed: %v", err)
	}

	got := registry.Endpoints()
	if len(got) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(got))
	}
	if got[0].Endpoint != endpointA || got[1].Endpoint != endpointB {
		t.Fatalf("endpoints out of order: %+v", got)
	}
}

func TestRefreshWithEmptyRegistrySurfacesError(t *testing.T) {
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
			},
		},
	}
	registry := NewDefaultStashboxRegistry() // no Replace called
	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err == nil {
		t.Fatalf("expected error when registry is empty")
	}
	if item.LastError == "" {
		t.Fatalf("expected LastError to be populated on the response")
	}
}

func stringPtr(s string) *string { return &s }

func TestOrderedEndpointsMergesUserAndRegistry(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"
	endpointC := "https://stashbox-c.example.org/graphql"
	endpointD := "https://stashbox-d.example.org/graphql"

	registry := newStashboxRegistry(stubFactory{client: &fakeStashboxClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
		{Name: "C", Endpoint: endpointC, APIKey: "k"},
		{Name: "D", Endpoint: endpointD, APIKey: "k"},
	})

	service, err := NewService(&fakeStashClient{}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	service.SetEndpointOrder([]string{endpointC, endpointA})

	got := service.orderedEndpoints()
	want := []string{endpointC, endpointA, endpointB, endpointD}
	if len(got) != len(want) {
		t.Fatalf("expected %d endpoints, got %d: %+v", len(want), len(got), got)
	}
	for i, ep := range want {
		if got[i].Endpoint != ep {
			t.Fatalf("position %d: want %q, got %q", i, ep, got[i].Endpoint)
		}
	}
}

func TestOrderedEndpointsDropsUnknown(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"
	endpointC := "https://stashbox-c.example.org/graphql"

	registry := newStashboxRegistry(stubFactory{client: &fakeStashboxClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
		{Name: "C", Endpoint: endpointC, APIKey: "k"},
	})

	service, err := NewService(&fakeStashClient{}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	service.SetEndpointOrder([]string{endpointA, "https://removed.example.org/graphql", endpointB})

	got := service.orderedEndpoints()
	want := []string{endpointA, endpointB, endpointC}
	if len(got) != len(want) {
		t.Fatalf("expected %d endpoints, got %d: %+v", len(want), len(got), got)
	}
	for i, ep := range want {
		if got[i].Endpoint != ep {
			t.Fatalf("position %d: want %q, got %q", i, ep, got[i].Endpoint)
		}
	}
}

func TestOrderedEndpointsEmptyUserFallsBackToRegistry(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"
	endpointC := "https://stashbox-c.example.org/graphql"

	registry := newStashboxRegistry(stubFactory{client: &fakeStashboxClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
		{Name: "C", Endpoint: endpointC, APIKey: "k"},
	})

	service, err := NewService(&fakeStashClient{}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	got := service.orderedEndpoints()
	if len(got) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(got))
	}
	if got[0].Endpoint != endpointA || got[1].Endpoint != endpointB || got[2].Endpoint != endpointC {
		t.Fatalf("registry order not preserved: %+v", got)
	}
}

func TestResolveStashboxPerformerFollowsUserOrder(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"

	clientA := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "a-1", Name: "Shared Performer"},
	}
	clientB := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "b-1", Name: "Shared Performer"},
	}

	registry := newStashboxRegistry(perEndpointFactory{
		clients: map[string]StashboxClient{
			normalizeStashBoxEndpoint(endpointA): clientA,
			normalizeStashBoxEndpoint(endpointB): clientB,
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
	})

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Shared Performer",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds: []*stashgraphql.StashIDFragment{
					{Endpoint: endpointA, StashID: "a-1"},
					{Endpoint: endpointB, StashID: "b-1"},
				},
			},
		},
	}

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	// User places B before A — the first stash_id-backed hit must be B.
	service.SetEndpointOrder([]string{endpointB, endpointA})

	target, err := service.resolveStashboxPerformer(context.Background(), stashClient.performers["p1"])
	if err != nil {
		t.Fatalf("resolveStashboxPerformer failed: %v", err)
	}
	if target == nil {
		t.Fatalf("expected a target, got nil")
	}
	if target.Endpoint != endpointB {
		t.Fatalf("expected endpoint %q, got %q", endpointB, target.Endpoint)
	}
}

func TestResolveStashboxPerformerPrefersHighestPriorityMatchingStashID(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"
	endpointB := "https://stashbox-b.example.org/graphql"

	clientA := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "a-1", Name: "Shared Performer"},
	}
	clientB := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "b-1", Name: "Shared Performer"},
	}

	registry := newStashboxRegistry(perEndpointFactory{
		clients: map[string]StashboxClient{
			normalizeStashBoxEndpoint(endpointA): clientA,
			normalizeStashBoxEndpoint(endpointB): clientB,
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
	})

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Shared Performer",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds: []*stashgraphql.StashIDFragment{
					{Endpoint: endpointA, StashID: "a-1"},
					{Endpoint: endpointB, StashID: "b-1"},
				},
			},
		},
	}

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	service.SetEndpointOrder([]string{endpointB, endpointA})

	target, err := service.resolveStashboxPerformer(context.Background(), stashClient.performers["p1"])
	if err != nil {
		t.Fatalf("resolveStashboxPerformer failed: %v", err)
	}
	if target == nil {
		t.Fatalf("expected a target, got nil")
	}
	if target.Endpoint != endpointB {
		t.Fatalf("expected endpoint %q, got %q", endpointB, target.Endpoint)
	}
}

func TestResolveStashboxPerformerDoesNotFallBackToNameSearch(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"

	registry := newStashboxRegistry(perEndpointFactory{
		clients: map[string]StashboxClient{
			normalizeStashBoxEndpoint(endpointA): &fakeStashboxClient{
				performer: &stashboxgraphql.PerformerFragment{ID: "a-1", Name: "Name Match Only"},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
	})

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Name Match Only",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
			},
		},
	}

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	target, err := service.resolveStashboxPerformer(context.Background(), stashClient.performers["p1"])
	if !errors.Is(err, errNoMatchingStashBoxMapping) {
		t.Fatalf("expected no-mapping sentinel, got %v", err)
	}
	if target != nil {
		t.Fatalf("expected nil target without stash_id fallback, got %+v", target)
	}
}

func TestRefreshPerformerWithoutMatchingStashIDReturnsStateNotError(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"

	registry := newStashboxRegistry(perEndpointFactory{
		clients: map[string]StashboxClient{
			normalizeStashBoxEndpoint(endpointA): &fakeStashboxClient{
				performer: &stashboxgraphql.PerformerFragment{ID: "a-1", Name: "Name Match Only"},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
	})

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Name Match Only",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
			},
		},
	}

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("expected no hard error for missing stash_id mapping, got %v", err)
	}
	if item.LastError == "" {
		t.Fatalf("expected last error to explain missing stash-box mapping")
	}
	if item.PendingReleaseCount != 0 {
		t.Fatalf("expected no pending releases, got %d", item.PendingReleaseCount)
	}
}

func TestListPerformerScenesDeduplicatesMatchedStashBoxScenes(t *testing.T) {
	endpoint := "https://stashbox.example.org/graphql"
	title := "Matched Title"
	code := "ABP-123"
	date := "2026-06-01"
	studio := "Studio A"
	stashURL := "https://stash.example.org/scenes/scene-1"
	screenshot := "https://stash.example.org/screenshot.jpg"
	boxURL := "https://stashbox.example.org/scenes/box-1"

	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Actor",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds: []*stashgraphql.StashIDFragment{
					{Endpoint: endpoint, StashID: "box-performer-1"},
				},
			},
		},
		scenes: []*stashgraphql.SceneFragment{
			{
				ID:    "scene-1",
				Title: &title,
				Code:  &code,
				Date:  &date,
				Urls:  []string{stashURL},
				Studio: &stashgraphql.StudioNameFragment{
					ID:   "studio-1",
					Name: studio,
				},
				Paths: stashgraphql.SceneFragment_Paths{Screenshot: &screenshot},
				StashIds: []*stashgraphql.StashIDFragment{
					{Endpoint: endpoint, StashID: "box-scene-1"},
				},
			},
		},
	}

	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "box-performer-1", Name: "Actor"},
			scenes: []*stashboxgraphql.SceneFragment{
				{
					ID:    "box-scene-1",
					Title: &title,
					Code:  &code,
					Date:  &date,
					Urls:  []*stashboxgraphql.URLFragment{{URL: boxURL}},
					Studio: &stashboxgraphql.StudioFragment{
						ID:   "studio-1",
						Name: studio,
					},
				},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := NewService(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	page, err := service.ListPerformerScenes(context.Background(), "p1", PerformerSceneQuery{})
	if err != nil {
		t.Fatalf("ListPerformerScenes failed: %v", err)
	}
	if page.StashSceneCount != 1 || page.StashBoxCount != 1 || page.DedupedCount != 1 {
		t.Fatalf("unexpected counts: %+v", page)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 deduped item, got %d", len(page.Items))
	}
	item := page.Items[0]
	if !item.HasStashSource || !item.HasStashBoxSource {
		t.Fatalf("expected merged source flags, got %+v", item)
	}
	if item.MatchedStashSceneID != "scene-1" {
		t.Fatalf("expected matched stash scene id scene-1, got %q", item.MatchedStashSceneID)
	}
	if item.URL != stashURL {
		t.Fatalf("expected stash url precedence, got %q", item.URL)
	}
	if item.ImageURL != screenshot {
		t.Fatalf("expected stash screenshot precedence, got %q", item.ImageURL)
	}
	if len(item.SourceLabels) != 2 || item.SourceLabels[0] != "Stash" || item.SourceLabels[1] != "StashBox" {
		t.Fatalf("unexpected source labels: %+v", item.SourceLabels)
	}
}

func TestSearchPreferredStashBoxScenesFallsBackByConfiguredOrder(t *testing.T) {
	firstEndpoint := "https://first.example/graphql"
	secondEndpoint := "https://second.example/graphql"
	code := "ABCD-123"
	title := "Fallback Hit"

	registry := newStashboxRegistry(perEndpointFactory{
		clients: map[string]StashboxClient{
			normalizeStashBoxEndpoint(firstEndpoint):  &fakeStashboxClient{scenes: nil},
			normalizeStashBoxEndpoint(secondEndpoint): &fakeStashboxClient{scenes: []*stashboxgraphql.SceneFragment{{ID: "scene-2", Title: &title, Code: &code}}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "First", Endpoint: firstEndpoint, APIKey: "ignored"},
		{Name: "Second", Endpoint: secondEndpoint, APIKey: "ignored"},
	})

	service, err := NewService(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	service.endpointOrder = []string{normalizeStashBoxEndpoint(firstEndpoint), normalizeStashBoxEndpoint(secondEndpoint)}

	page, err := service.SearchPreferredStashBoxScenes(context.Background(), "ABCD", 24, DiscoverSortRelevance)
	if err != nil {
		t.Fatalf("SearchPreferredStashBoxScenes failed: %v", err)
	}
	if page.UsedStashBox == nil || page.UsedStashBox.Endpoint != secondEndpoint {
		t.Fatalf("expected second endpoint to be used, got %+v", page.UsedStashBox)
	}
	if page.FallbackCount != 1 {
		t.Fatalf("expected fallback count 1, got %d", page.FallbackCount)
	}
	if len(page.Items) != 1 || page.Items[0].DerivedQuery != code {
		t.Fatalf("unexpected discovery page: %+v", page)
	}
}

func TestQueueDiscoveredSceneUsesSearchTaskSource(t *testing.T) {
	endpoint := "https://box.example/graphql"
	code := "ABCD-123"
	sceneID := "scene-1"
	downloaderTask := &downloader.Task{ID: "task-1", Source: downloader.TaskSourceSearch}

	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			scenes: []*stashboxgraphql.SceneFragment{{ID: sceneID, Code: &code}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	fakeDL := &fakeDownloader{tasks: []*downloader.Task{downloaderTask}}
	service, err := NewService(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, registry, fakeDL, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.QueueDiscoveredScene(context.Background(), sceneID, endpoint)
	if err != nil {
		t.Fatalf("QueueDiscoveredScene failed: %v", err)
	}
	if task == nil || task.ID != "task-1" {
		t.Fatalf("unexpected queued task: %+v", task)
	}
	if len(fakeDL.queries) != 1 || fakeDL.queries[0] != code {
		t.Fatalf("unexpected downloader queries: %+v", fakeDL.queries)
	}
	if len(fakeDL.sources) != 1 || fakeDL.sources[0] != downloader.TaskSourceSearch {
		t.Fatalf("unexpected downloader sources: %+v", fakeDL.sources)
	}
}

func TestRefreshSubscribedPerformerFailsWhenStashBoxSceneCodeMissing(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	title := "Release Without Code"

	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenes: []*stashboxgraphql.SceneFragment{
				{ID: "js-scene-1", Title: &title},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "JavStash", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := NewService(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	_, err = service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err == nil || err.Error() != `subscription: stash-box scene "js-scene-1" is missing code` {
		t.Fatalf("expected missing code error, got %v", err)
	}
}

func TestQueueDiscoveredSceneRejectsMissingCode(t *testing.T) {
	endpoint := "https://box.example/graphql"
	sceneID := "scene-1"

	registry := newStashboxRegistry(stubFactory{
		client: &fakeStashboxClient{
			scenes: []*stashboxgraphql.SceneFragment{{ID: sceneID}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := NewService(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, registry, &fakeDownloader{}, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.QueueDiscoveredScene(context.Background(), sceneID, endpoint)
	if err == nil || err.Error() != `subscription: scene "scene-1" is missing code` {
		t.Fatalf("expected missing code error, got task=%+v err=%v", task, err)
	}
}
