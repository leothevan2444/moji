package subscription

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type fakeStashClient struct {
	performers map[string]*stashgraphql.PerformerFragment
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
	performer *stashboxgraphql.PerformerFragment
	scenes    []*stashboxgraphql.SceneFragment
}

func (f *fakeStashboxClient) FindPerformerByID(_ context.Context, id string) (*stashboxgraphql.PerformerFragment, error) {
	if f.performer != nil && f.performer.ID == id {
		return f.performer, nil
	}
	return nil, nil
}

func (f *fakeStashboxClient) SearchPerformer(_ context.Context, _ string) ([]*stashboxgraphql.PerformerFragment, error) {
	if f.performer == nil {
		return nil, nil
	}
	return []*stashboxgraphql.PerformerFragment{f.performer}, nil
}

func (f *fakeStashboxClient) QueryScenes(_ context.Context, _ stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error) {
	return f.scenes, nil
}

// stubFactory is a StashboxClientFactory that always returns the same client
// regardless of the endpoint. Tests that need per-endpoint dispatch should
// use perEndpointFactory instead.
type stubFactory struct {
	client StashboxClient
}

func (s stubFactory) NewClient(_, _ string) StashboxClient { return s.client }

// perEndpointFactory returns a different fakeStashboxClient for each
// endpoint. The client assigned to an unknown endpoint panics so tests catch
// accidental mis-routing.
type perEndpointFactory struct {
	clients map[string]StashboxClient
}

func (f perEndpointFactory) NewClient(endpoint, _ string) StashboxClient {
	client, ok := f.clients[normalizeStashBoxEndpoint(endpoint)]
	if !ok {
		panic("perEndpointFactory: no client registered for " + endpoint)
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
