package subscription

import (
	"context"
	"testing"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type fakeStashClient struct {
	performers map[string]*stashgraphql.PerformerFragment
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
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				AliasList:    []string{"RION"},
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds: []*stashgraphql.StashIDFragment{
					{Endpoint: "https://javstash.org/graphql", StashID: "js-1"},
				},
			},
		},
	}
	title := "New Release"
	code := "ABCD-123"
	date := "2026-06-01"
	url := "https://javstash.org/scenes/js-scene-1"

	service, err := NewService(
		stashClient,
		&fakeStashboxClient{
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
		nil,
		NewMemoryStore(),
	)
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
}
