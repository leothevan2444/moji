package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/discovery"
	"github.com/leothevan2444/moji/internal/metadata"
	performerdomain "github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type fakeStashClient struct {
	performers      map[string]*stashgraphql.PerformerFragment
	scenes          []*stashgraphql.SceneFragment
	boxes           []stash.StashBoxEndpoint
	findScenesCalls []fakeFindScenesCall
}

func performerServiceForTest(t *testing.T, service *Service) *performerdomain.Service {
	t.Helper()
	creator, _ := service.taskCreator.(performerdomain.TaskCreator)
	var lister performerdomain.TaskLister
	if candidate, ok := any(service.taskCreator).(performerdomain.TaskLister); ok {
		lister = candidate
	}
	result, err := performerdomain.NewService(service.stash, service.metadata, creator, lister, nil, nil)
	if err != nil {
		t.Fatalf("New performer service: %v", err)
	}
	return result
}

func discoveryServiceForTest(service *Service) *discovery.Service {
	if flow, ok := service.taskCreator.(*taskflow.Service); ok {
		flow.SetDiscoveredSceneResolver(discovery.NewDiscoveredSceneResolver(service.metadata.Registry()))
	}
	creator, _ := service.taskCreator.(discovery.TaskCreator)
	return discovery.NewService(service.metadata, creator, nil)
}

type fakeFindScenesCall struct {
	SceneFilter *stashgraphql.SceneFilterType
	Filter      *stashgraphql.FindFilterType
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

func (f *fakeStashClient) FindScenes(_ context.Context, sceneFilter *stashgraphql.SceneFilterType, filter *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error) {
	f.findScenesCalls = append(f.findScenesCalls, fakeFindScenesCall{SceneFilter: sceneFilter, Filter: filter})
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
				if stashID.StashID == *sceneFilter.StashIDEndpoint.StashID && metadata.NormalizeEndpoint(stashID.Endpoint) == metadata.NormalizeEndpoint(endpoint) {
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
	scenesByPage map[int][]*stashboxgraphql.SceneFragment
	searchErr    error
	findSceneErr error
	queryInputs  []stashboxgraphql.SceneQueryInput
}

type fakeTaskRuntime struct {
	tasks   []*taskruntime.Task
	err     error
	calls   int
	codes   []string
	sources []taskruntime.TaskSource
}

type fakeTaskCreator struct {
	queueTask        *taskruntime.Task
	queueErr         error
	queueCalls       []fakeQueuedSceneCall
	subscriptionTask *taskruntime.Task
	subscriptionCode string
	subscriptionErr  error
}

type fakeQueuedSceneCall struct {
	SceneID          string
	StashBoxEndpoint string
}

func (f *fakeTaskRuntime) AddTorrentContext(_ context.Context, _ taskruntime.AddTorrentRequest) (*taskruntime.Task, error) {
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

func (f *fakeStashboxClient) QueryScenes(_ context.Context, input stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error) {
	f.queryInputs = append(f.queryInputs, input)
	sourceScenes := f.scenes
	if f.scenesByPage != nil {
		sourceScenes = f.scenesByPage[input.Page]
	}
	out := make([]*stashboxgraphql.SceneFragment, 0, len(sourceScenes))
	for _, scene := range sourceScenes {
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

func (f *fakeStashboxClient) QueryScenesPage(ctx context.Context, input stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error) {
	scenes, err := f.QueryScenes(ctx, input)
	count := len(f.scenes)
	if f.scenesByPage != nil {
		count = 0
		for _, page := range f.scenesByPage {
			count += len(page)
		}
	}
	return stashboxpkg.ScenePage{Scenes: scenes, Count: count}, err
}

func (f *fakeTaskRuntime) DownloadMediaContext(_ context.Context, req taskruntime.DownloadRequest) (*taskruntime.Task, error) {
	f.calls++
	f.codes = append(f.codes, req.Code)
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

func (f *fakeTaskRuntime) ListTasks(_ context.Context) ([]*taskruntime.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]*taskruntime.Task(nil), f.tasks...), nil
}

func (f *fakeTaskCreator) QueueDiscoveredScene(_ context.Context, sceneID string, stashBoxEndpoint string) (*taskruntime.Task, error) {
	f.queueCalls = append(f.queueCalls, fakeQueuedSceneCall{SceneID: sceneID, StashBoxEndpoint: stashBoxEndpoint})
	return f.queueTask, f.queueErr
}

func (f *fakeTaskCreator) QueueSubscriptionRelease(_ context.Context, code, _ string) (*taskruntime.Task, error) {
	if f.subscriptionCode != "" {
		code = f.subscriptionCode
	}
	_ = code
	return f.subscriptionTask, f.subscriptionErr
}

// stubFactory is a StashboxClientFactory that always returns the same client
// regardless of the endpoint. Tests that need per-endpoint dispatch should
// use perEndpointFactory instead.
type stubFactory struct {
	client metadata.Client
}

func (s stubFactory) NewClient(stash.StashBoxEndpoint) metadata.Client { return s.client }

// perEndpointFactory returns a different fakeStashboxClient for each
// endpoint. The client assigned to an unknown endpoint panics so tests catch
// accidental mis-routing.
type perEndpointFactory struct {
	clients map[string]metadata.Client
}

func (f perEndpointFactory) NewClient(box stash.StashBoxEndpoint) metadata.Client {
	client, ok := f.clients[metadata.NormalizeEndpoint(box.Endpoint)]
	if !ok {
		panic("perEndpointFactory: no client registered for " + box.Endpoint)
	}
	return client
}

func TestListStashPerformersMarksCustomFieldSubscribers(t *testing.T) {
	service, err := newServiceForTest(&fakeStashClient{
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

	items, err := performerServiceForTest(t, service).List(context.Background(), "mik")
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

	service, err := newServiceForTest(stashClient, nil, nil, NewMemoryStore())
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
	if performerdomain.IsSubscribed(stashClient.performers["p1"].CustomFields, DefaultCustomFieldKey) {
		t.Fatalf("expected custom field to be removed")
	}
}

func TestRefreshPerformerStoresPendingReleasesWithoutTaskRuntime(t *testing.T) {
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

	registry := metadata.NewRegistry(stubFactory{
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
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
	if item.RecentReleases[0].Code != code {
		t.Fatalf("expected code %q, got %q", code, item.RecentReleases[0].Code)
	}
	if got, want := item.RecentReleases[0].Source, "stash-box:"+endpoint; got != want {
		t.Fatalf("unexpected source %q, want %q", got, want)
	}
}

func TestRefreshPerformerAllowsUnsubscribedPerformer(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:       "p1",
				Name:     "Unsubscribed Performer",
				StashIds: []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}
	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Unsubscribed Performer"},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{
		Name:     "javstash",
		Endpoint: endpoint,
		APIKey:   "ignored",
	}})
	store := NewMemoryStore()
	service, err := newServiceForTest(stashClient, registry, nil, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.Performer.Subscribed {
		t.Fatalf("manual refresh must not subscribe the performer")
	}
	if item.LastCheckedAt == nil {
		t.Fatalf("expected manual refresh to record its check time")
	}
	state, err := store.Get(context.Background(), "p1")
	if err != nil {
		t.Fatalf("load stored performer state: %v", err)
	}
	if state == nil || state.LastCheckedAt == nil {
		t.Fatalf("expected manual refresh state to be persisted")
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
	registry := metadata.NewRegistry(stubFactory{
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
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

func TestRefreshPerformerSkipsReleaseAlreadyInStashLibrary(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	code := "ABCD-123"
	title := "Library Hit"
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
		scenes: []*stashgraphql.SceneFragment{{
			ID:   "stash-scene-1",
			Code: &code,
			StashIds: []*stashgraphql.StashIDFragment{
				{Endpoint: endpoint, StashID: "js-scene-1"},
			},
		}},
	}
	registry := metadata.NewRegistry(stubFactory{
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
	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 0 {
		t.Fatalf("expected in-library release to be skipped, got %d pending", item.PendingReleaseCount)
	}
	if len(stashClient.findScenesCalls) != 1 {
		t.Fatalf("expected 1 stash dedupe lookup, got %d", len(stashClient.findScenesCalls))
	}
	call := stashClient.findScenesCalls[0]
	if call.SceneFilter == nil || call.SceneFilter.StashIDEndpoint == nil || call.SceneFilter.StashIDEndpoint.StashID == nil || *call.SceneFilter.StashIDEndpoint.StashID != "js-scene-1" {
		t.Fatalf("unexpected stash dedupe query: %+v", call.SceneFilter)
	}
}

func TestRefreshPerformerKeepsReleaseWhenNotFoundInStashLibrary(t *testing.T) {
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
	registry := metadata.NewRegistry(stubFactory{
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
	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 1 {
		t.Fatalf("expected release to remain pending, got %d", item.PendingReleaseCount)
	}
	if len(stashClient.findScenesCalls) != 1 {
		t.Fatalf("expected 1 stash dedupe lookup, got %d", len(stashClient.findScenesCalls))
	}
}

func TestRefreshPerformerSkipsStashLookupForKnownRelease(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	code := "ABCD-123"
	title := "Known Release"
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
	registry := metadata.NewRegistry(stubFactory{
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
	store := NewMemoryStore()
	if err := store.Put(context.Background(), &PerformerState{
		PerformerID: "p1",
		PendingReleases: []RecordedRelease{{
			Key:  "stashbox:https___javstash.example.org_graphql:js-scene-1",
			Code: code,
		}},
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}
	service, err := newServiceForTest(stashClient, registry, nil, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 1 {
		t.Fatalf("expected known release to stay pending once, got %d", item.PendingReleaseCount)
	}
	if len(stashClient.findScenesCalls) != 0 {
		t.Fatalf("expected no stash dedupe lookup for known release, got %d", len(stashClient.findScenesCalls))
	}
}

func TestRefreshPerformerMixedLibraryAndNewReleases(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
		scenes: []*stashgraphql.SceneFragment{{
			ID: "stash-scene-1",
			StashIds: []*stashgraphql.StashIDFragment{
				{Endpoint: endpoint, StashID: "js-scene-1"},
			},
		}},
	}
	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenesByPage: map[int][]*stashboxgraphql.SceneFragment{
				1: {
					{ID: "js-scene-1", Code: stringPtr("ABCD-001")},
					{ID: "js-scene-2", Code: stringPtr("ABCD-002")},
				},
				2: {},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 1 {
		t.Fatalf("expected only non-library release to remain, got %d", item.PendingReleaseCount)
	}
	if item.RecentReleases[0].Key != "stashbox:https___javstash.example.org_graphql:js-scene-2" {
		t.Fatalf("unexpected pending release key: %q", item.RecentReleases[0].Key)
	}
	if len(stashClient.findScenesCalls) != 2 {
		t.Fatalf("expected 2 stash dedupe lookups, got %d", len(stashClient.findScenesCalls))
	}
}

func TestRefreshPerformerInLibraryReleaseIsCheckedAgainOnNextPoll(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	code := "ABCD-123"
	stashClient := &fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
		scenes: []*stashgraphql.SceneFragment{{
			ID: "stash-scene-1",
			StashIds: []*stashgraphql.StashIDFragment{
				{Endpoint: endpoint, StashID: "js-scene-1"},
			},
		}},
	}
	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenes: []*stashboxgraphql.SceneFragment{{
				ID:   "js-scene-1",
				Code: &code,
			}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
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
	if first.PendingReleaseCount != 0 || second.PendingReleaseCount != 0 {
		t.Fatalf("expected in-library release to stay skipped, got first=%d second=%d", first.PendingReleaseCount, second.PendingReleaseCount)
	}
	if len(stashClient.findScenesCalls) != 2 {
		t.Fatalf("expected repeated stash dedupe lookup across polls, got %d", len(stashClient.findScenesCalls))
	}
}

func TestRefreshPerformerBackfillsAcrossMultiplePages(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	client := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
		scenesByPage: map[int][]*stashboxgraphql.SceneFragment{
			1: {{ID: "js-scene-1", Code: stringPtr("ABCD-001"), Date: stringPtr("2026-06-01")}},
			2: {{ID: "js-scene-2", Code: stringPtr("ABCD-002"), Date: stringPtr("2026-05-01")}},
			3: {},
		},
	}
	registry := metadata.NewRegistry(stubFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(&fakeStashClient{
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

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 2 {
		t.Fatalf("expected 2 backfilled releases, got %d", item.PendingReleaseCount)
	}
	if len(client.queryInputs) != 3 {
		t.Fatalf("expected 3 page fetches, got %d", len(client.queryInputs))
	}
	if client.queryInputs[0].PerPage != releaseQueryPerPage || client.queryInputs[0].Page != 1 || client.queryInputs[1].Page != 2 {
		t.Fatalf("unexpected paging inputs: %+v", client.queryInputs)
	}
}

func TestRefreshPerformerPollFindsNewReleaseOnLaterPage(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	store := NewMemoryStore()
	if err := store.Put(context.Background(), &PerformerState{
		PerformerID: "p1",
		PendingReleases: []RecordedRelease{{
			Key:  "stashbox:https___javstash.example.org_graphql:js-scene-1",
			Code: "ABCD-001",
		}},
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}
	client := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
		scenesByPage: map[int][]*stashboxgraphql.SceneFragment{
			1: {{ID: "js-scene-1", Code: stringPtr("ABCD-001"), Date: stringPtr("2026-06-01")}},
			2: {{ID: "js-scene-2", Code: stringPtr("ABCD-002"), Date: stringPtr("2026-05-01")}},
			3: {},
		},
	}
	registry := metadata.NewRegistry(stubFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}, registry, nil, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 2 {
		t.Fatalf("expected existing + new pending release, got %d", item.PendingReleaseCount)
	}
	if len(client.queryInputs) != 2 {
		t.Fatalf("expected poll to inspect two 40-item batches, got %d calls", len(client.queryInputs))
	}
	if client.queryInputs[0].PerPage != releaseQueryPerPage || client.queryInputs[1].Page != 2 {
		t.Fatalf("unexpected query inputs: %+v", client.queryInputs)
	}
}

func TestRefreshPerformerPollStopsAtKnownReleaseBoundary(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	store := NewMemoryStore()
	if err := store.Put(context.Background(), &PerformerState{
		PerformerID: "p1",
		PendingReleases: []RecordedRelease{{
			Key:  "stashbox:https___javstash.example.org_graphql:js-scene-2",
			Code: "ABCD-002",
		}},
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}
	client := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
		scenesByPage: map[int][]*stashboxgraphql.SceneFragment{
			1: {{ID: "js-scene-1", Code: stringPtr("ABCD-001"), Date: stringPtr("2026-06-01")}},
			2: {{ID: "js-scene-2", Code: stringPtr("ABCD-002"), Date: stringPtr("2026-05-01")}},
			3: {{ID: "js-scene-3", Code: stringPtr("ABCD-003"), Date: stringPtr("2026-04-01")}},
		},
	}
	registry := metadata.NewRegistry(stubFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}, registry, nil, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	_, err = service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if len(client.queryInputs) != 2 {
		t.Fatalf("expected known-release boundary to stop on second page, got %d calls", len(client.queryInputs))
	}
}

func TestRefreshPerformerPollStopsAtReleaseDateBoundary(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	store := NewMemoryStore()
	if err := store.Put(context.Background(), &PerformerState{
		PerformerID:       "p1",
		ProcessedReleases: []RecordedRelease{{Key: "existing", Code: "EXISTING-001"}},
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}
	client := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
		scenesByPage: map[int][]*stashboxgraphql.SceneFragment{
			1: {{ID: "js-scene-1", Code: stringPtr("ABCD-001"), Date: stringPtr("2020-01-01")}},
			2: {{ID: "js-scene-2", Code: stringPtr("ABCD-002"), Date: stringPtr("2019-01-01")}},
		},
	}
	registry := metadata.NewRegistry(stubFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:           "p1",
				Name:         "Rara Anzai",
				CustomFields: map[string]any{DefaultCustomFieldKey: true},
				StashIds:     []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "js-1"}},
			},
		},
	}, registry, nil, store)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	service.SetReleasePolicy(ReleasePolicyConfig{
		SoloBehavior:     config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:    config.SubscriptionReleaseBehaviorDownload,
		ReleaseDateRange: config.SubscriptionReleaseDateRangeOneYear,
	})

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if len(client.queryInputs) != 1 {
		t.Fatalf("expected release-date boundary to stop after first page, got %d calls", len(client.queryInputs))
	}
	if item.PendingReleaseCount != 1 || item.RecentReleases[0].DecisionReason != "release_date_out_of_range_review" {
		t.Fatalf("unexpected date-boundary release result: %+v", item.RecentReleases)
	}
}

func TestRefreshPerformerDeduplicatesScenesAcrossPages(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	client := &fakeStashboxClient{
		performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
		scenesByPage: map[int][]*stashboxgraphql.SceneFragment{
			1: {{ID: "js-scene-1", Code: stringPtr("ABCD-001"), Date: stringPtr("2026-06-01")}},
			2: {{ID: "js-scene-1", Code: stringPtr("ABCD-001"), Date: stringPtr("2026-06-01")}},
		},
	}
	registry := metadata.NewRegistry(stubFactory{client: client})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "javstash", Endpoint: endpoint, APIKey: "ignored"}})
	service, err := newServiceForTest(&fakeStashClient{
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

	item, err := service.RefreshSubscribedPerformer(context.Background(), "p1")
	if err != nil {
		t.Fatalf("RefreshSubscribedPerformer failed: %v", err)
	}
	if item.PendingReleaseCount != 1 {
		t.Fatalf("expected 1 deduped release, got %d", item.PendingReleaseCount)
	}
	if len(client.queryInputs) != 2 {
		t.Fatalf("expected duplicate-only second page to stop fetch, got %d calls", len(client.queryInputs))
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
	registry := metadata.NewRegistry(stubFactory{
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

	taskRuntime := &fakeTaskRuntime{err: errors.New("temporary add failure")}
	service, err := newServiceForTest(stashClient, registry, taskRuntime, NewMemoryStore())
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
	if taskRuntime.calls != 1 {
		t.Fatalf("expected taskRuntime to be called once for the new release, got %d", taskRuntime.calls)
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

	registry := metadata.NewRegistry(perEndpointFactory{
		clients: map[string]metadata.Client{
			metadata.NormalizeEndpoint(endpointA): clientA,
			metadata.NormalizeEndpoint(endpointB): clientB,
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
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
	registry := metadata.NewDefaultRegistry()
	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	if err := service.metadata.RefreshStashBoxes(context.Background()); err != nil {
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
	registry := metadata.NewDefaultRegistry() // no Replace called
	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
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

	registry := metadata.NewRegistry(stubFactory{client: &fakeStashboxClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
		{Name: "C", Endpoint: endpointC, APIKey: "k"},
		{Name: "D", Endpoint: endpointD, APIKey: "k"},
	})

	service, err := newServiceForTest(&fakeStashClient{}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	service.metadata.SetEndpointOrder([]string{endpointC, endpointA})

	got := service.metadata.Endpoints()
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

	registry := metadata.NewRegistry(stubFactory{client: &fakeStashboxClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
		{Name: "C", Endpoint: endpointC, APIKey: "k"},
	})

	service, err := newServiceForTest(&fakeStashClient{}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	service.metadata.SetEndpointOrder([]string{endpointA, "https://removed.example.org/graphql", endpointB})

	got := service.metadata.Endpoints()
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

	registry := metadata.NewRegistry(stubFactory{client: &fakeStashboxClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "A", Endpoint: endpointA, APIKey: "k"},
		{Name: "B", Endpoint: endpointB, APIKey: "k"},
		{Name: "C", Endpoint: endpointC, APIKey: "k"},
	})

	service, err := newServiceForTest(&fakeStashClient{}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	got := service.metadata.Endpoints()
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

	registry := metadata.NewRegistry(perEndpointFactory{
		clients: map[string]metadata.Client{
			metadata.NormalizeEndpoint(endpointA): clientA,
			metadata.NormalizeEndpoint(endpointB): clientB,
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	// User places B before A — the first stash_id-backed hit must be B.
	service.metadata.SetEndpointOrder([]string{endpointB, endpointA})

	target, err := service.metadata.ResolvePerformer(context.Background(), stashClient.performers["p1"])
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

	registry := metadata.NewRegistry(perEndpointFactory{
		clients: map[string]metadata.Client{
			metadata.NormalizeEndpoint(endpointA): clientA,
			metadata.NormalizeEndpoint(endpointB): clientB,
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	service.metadata.SetEndpointOrder([]string{endpointB, endpointA})

	target, err := service.metadata.ResolvePerformer(context.Background(), stashClient.performers["p1"])
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

	registry := metadata.NewRegistry(perEndpointFactory{
		clients: map[string]metadata.Client{
			metadata.NormalizeEndpoint(endpointA): &fakeStashboxClient{
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	target, err := service.metadata.ResolvePerformer(context.Background(), stashClient.performers["p1"])
	if !errors.Is(err, metadata.ErrNoPerformerMapping) {
		t.Fatalf("expected no-mapping sentinel, got %v", err)
	}
	if target != nil {
		t.Fatalf("expected nil target without stash_id fallback, got %+v", target)
	}
}

func TestRefreshPerformerWithoutMatchingStashIDReturnsStateNotError(t *testing.T) {
	endpointA := "https://stashbox-a.example.org/graphql"

	registry := metadata.NewRegistry(perEndpointFactory{
		clients: map[string]metadata.Client{
			metadata.NormalizeEndpoint(endpointA): &fakeStashboxClient{
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

	service, err := newServiceForTest(stashClient, registry, nil, NewMemoryStore())
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
				Performers: []*stashgraphql.SceneFragment_Performers{{ID: "performer-1", Name: "Actor One"}, {ID: "performer-2", Name: "Actor Two"}},
				Tags:       []*stashgraphql.SceneFragment_Tags{{ID: "tag-1", Name: "Featured"}},
			},
		},
	}

	registry := metadata.NewRegistry(stubFactory{
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

	taskRuntime := &fakeTaskRuntime{tasks: []*taskruntime.Task{{
		ID:               "task-abp-123",
		Code:             "ABP-123",
		Stage:            taskruntime.TaskStageDownloading,
		StageStatus:      taskruntime.TaskStageStatusRunning,
		StageLabel:       "下载",
		StageStatusLabel: "进行中",
		Progress:         0.42,
	}}}
	service, err := newServiceForTest(stashClient, registry, taskRuntime, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	creator, _ := service.taskCreator.(performerdomain.TaskCreator)
	catalog, err := performerdomain.NewService(stashClient, service.metadata, creator, taskRuntime, nil, nil)
	if err != nil {
		t.Fatalf("New performer service: %v", err)
	}
	page, err := catalog.ListPerformerScenes(context.Background(), "p1", performerdomain.SceneQuery{})
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
	if item.PerformerCount != 2 || item.TagCount != 1 {
		t.Fatalf("expected Stash-local counts 2 performers / 1 tag, got %d / %d", item.PerformerCount, item.TagCount)
	}
	if len(item.Performers) != 2 || item.Performers[0].Name != "Actor One" || len(item.Tags) != 1 || item.Tags[0].Name != "Featured" {
		t.Fatalf("expected Stash-local performer and tag summaries, got %+v / %+v", item.Performers, item.Tags)
	}
	if item.MojiTask == nil || item.MojiTask.ID != "task-abp-123" || item.MojiTask.Progress != 0.42 {
		t.Fatalf("expected matching Moji task summary, got %+v", item.MojiTask)
	}
}

func TestSearchPreferredStashBoxScenesFallsBackByConfiguredOrder(t *testing.T) {
	firstEndpoint := "https://first.example/graphql"
	secondEndpoint := "https://second.example/graphql"
	code := "ABCD-123"
	title := "Fallback Hit"

	registry := metadata.NewRegistry(perEndpointFactory{
		clients: map[string]metadata.Client{
			metadata.NormalizeEndpoint(firstEndpoint):  &fakeStashboxClient{scenes: nil},
			metadata.NormalizeEndpoint(secondEndpoint): &fakeStashboxClient{scenes: []*stashboxgraphql.SceneFragment{{ID: "scene-2", Title: &title, Code: &code}}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "First", Endpoint: firstEndpoint, APIKey: "ignored"},
		{Name: "Second", Endpoint: secondEndpoint, APIKey: "ignored"},
	})

	service, err := newServiceForTest(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, registry, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	service.metadata.SetEndpointOrder([]string{firstEndpoint, secondEndpoint})

	page, err := discoveryServiceForTest(service).Search(context.Background(), "ABCD", 24, discovery.SortRelevance)
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
	downloaderTask := &taskruntime.Task{ID: "task-1", Source: taskruntime.TaskSourceSearch}

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			scenes: []*stashboxgraphql.SceneFragment{{ID: sceneID, Code: &code}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	fakeDL := &fakeTaskRuntime{tasks: []*taskruntime.Task{downloaderTask}}
	service, err := newServiceForTest(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, registry, fakeDL, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := discoveryServiceForTest(service).Queue(context.Background(), sceneID, endpoint)
	if err != nil {
		t.Fatalf("QueueDiscoveredScene failed: %v", err)
	}
	if task == nil || task.ID != "task-1" {
		t.Fatalf("unexpected queued task: %+v", task)
	}
	if len(fakeDL.codes) != 1 || fakeDL.codes[0] != code {
		t.Fatalf("unexpected taskRuntime codes: %+v", fakeDL.codes)
	}
	if len(fakeDL.sources) != 1 || fakeDL.sources[0] != taskruntime.TaskSourceSearch {
		t.Fatalf("unexpected taskRuntime sources: %+v", fakeDL.sources)
	}
}

func TestRefreshSubscribedPerformerFailsWhenStashBoxSceneCodeMissing(t *testing.T) {
	endpoint := "https://javstash.example.org/graphql"
	title := "Release Without Code"

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "js-1", Name: "Rara Anzai"},
			scenes: []*stashboxgraphql.SceneFragment{
				{ID: "js-scene-1", Title: &title},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "JavStash", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := newServiceForTest(&fakeStashClient{
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

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			scenes: []*stashboxgraphql.SceneFragment{{ID: sceneID}},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := newServiceForTest(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, registry, &fakeTaskRuntime{}, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := discoveryServiceForTest(service).Queue(context.Background(), sceneID, endpoint)
	if err == nil || err.Error() != `discovery: scene "scene-1" is missing code` {
		t.Fatalf("expected missing code error, got task=%+v err=%v", task, err)
	}
}

func TestQueuePerformerScenesQueuesEligibleScene(t *testing.T) {
	endpoint := "https://box.example/graphql"
	sceneID := "scene-1"
	code := "ABCD-123"
	task := &taskruntime.Task{ID: "task-1", Source: taskruntime.TaskSourceSearch}

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "sb-1", Name: "Rara"},
			scenes: []*stashboxgraphql.SceneFragment{
				{ID: sceneID, Code: &code},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:       "p1",
				Name:     "Rara",
				StashIds: []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "sb-1"}},
			},
		},
	}, registry, &fakeTaskRuntime{}, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	creator := &fakeTaskCreator{queueTask: task}
	service.SetTaskCreator(creator)

	result, err := performerServiceForTest(t, service).QueuePerformerScenes(context.Background(), "p1", []performerdomain.QueueSceneSelection{{
		Key: "stashbox:" + endpointKey(endpoint) + ":" + sceneID,
	}})
	if err != nil {
		t.Fatalf("QueuePerformerScenes failed: %v", err)
	}
	if len(creator.queueCalls) != 1 || creator.queueCalls[0].SceneID != sceneID || creator.queueCalls[0].StashBoxEndpoint != endpoint {
		t.Fatalf("unexpected task creator calls: %+v", creator.queueCalls)
	}
	if result.Summary.RequestedCount != 1 || result.Summary.QueuedCount != 1 || result.Summary.SkippedCount != 0 || result.Summary.FailedCount != 0 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if len(result.QueuedTasks) != 1 || result.QueuedTasks[0].ID != "task-1" {
		t.Fatalf("unexpected queued tasks: %+v", result.QueuedTasks)
	}
	if len(result.Results) != 1 || result.Results[0].Status != performerdomain.QueueSceneStatusQueued || result.Results[0].ReasonCode != "QUEUED" {
		t.Fatalf("unexpected result: %+v", result.Results)
	}
}

func TestQueuePerformerScenesMapsDuplicateCodeToSkipped(t *testing.T) {
	endpoint := "https://box.example/graphql"
	sceneID := "scene-1"
	code := "ABCD-123"

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "sb-1", Name: "Rara"},
			scenes: []*stashboxgraphql.SceneFragment{
				{ID: sceneID, Code: &code},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:       "p1",
				Name:     "Rara",
				StashIds: []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "sb-1"}},
			},
		},
	}, registry, &fakeTaskRuntime{}, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	service.SetTaskCreator(&fakeTaskCreator{queueErr: taskruntime.ErrDuplicateCodeTask})

	result, err := performerServiceForTest(t, service).QueuePerformerScenes(context.Background(), "p1", []performerdomain.QueueSceneSelection{{
		Key: "stashbox:" + endpointKey(endpoint) + ":" + sceneID,
	}})
	if err != nil {
		t.Fatalf("QueuePerformerScenes failed: %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].Status != performerdomain.QueueSceneStatusSkipped || result.Results[0].ReasonCode != "DUPLICATE_CODE_TASK" {
		t.Fatalf("unexpected result: %+v", result.Results)
	}
	if result.Summary.SkippedCount != 1 || result.Summary.QueuedCount != 0 || result.Summary.FailedCount != 0 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
}

func TestQueuePerformerScenesReturnsSceneNotFound(t *testing.T) {
	endpoint := "https://box.example/graphql"

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "sb-1", Name: "Rara"},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:       "p1",
				Name:     "Rara",
				StashIds: []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "sb-1"}},
			},
		},
	}, registry, &fakeTaskRuntime{}, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	service.SetTaskCreator(&fakeTaskCreator{})

	result, err := performerServiceForTest(t, service).QueuePerformerScenes(context.Background(), "p1", []performerdomain.QueueSceneSelection{{
		Key: "missing-key",
	}})
	if err != nil {
		t.Fatalf("QueuePerformerScenes failed: %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].Status != performerdomain.QueueSceneStatusFailed || result.Results[0].ReasonCode != "SCENE_NOT_FOUND" {
		t.Fatalf("unexpected result: %+v", result.Results)
	}
	if result.Summary.FailedCount != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
}

func TestQueuePerformerScenesReturnsMixedSummary(t *testing.T) {
	endpoint := "https://box.example/graphql"
	sceneID := "scene-1"
	code := "ABCD-123"
	inLibraryID := "stash-1"

	registry := metadata.NewRegistry(stubFactory{
		client: &fakeStashboxClient{
			performer: &stashboxgraphql.PerformerFragment{ID: "sb-1", Name: "Rara"},
			scenes: []*stashboxgraphql.SceneFragment{
				{ID: sceneID, Code: &code},
			},
		},
	})
	registry.Replace([]stash.StashBoxEndpoint{{Name: "Preferred", Endpoint: endpoint, APIKey: "ignored"}})

	service, err := newServiceForTest(&fakeStashClient{
		performers: map[string]*stashgraphql.PerformerFragment{
			"p1": {
				ID:       "p1",
				Name:     "Rara",
				StashIds: []*stashgraphql.StashIDFragment{{Endpoint: endpoint, StashID: "sb-1"}},
			},
		},
		scenes: []*stashgraphql.SceneFragment{
			{ID: inLibraryID, Code: &code},
		},
	}, registry, &fakeTaskRuntime{}, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	service.SetTaskCreator(&fakeTaskCreator{queueTask: &taskruntime.Task{ID: "task-1"}})

	result, err := performerServiceForTest(t, service).QueuePerformerScenes(context.Background(), "p1", []performerdomain.QueueSceneSelection{
		{Key: "stashbox:" + endpointKey(endpoint) + ":" + sceneID},
		{Key: "stash:" + inLibraryID},
		{Key: "missing-key"},
	})
	if err != nil {
		t.Fatalf("QueuePerformerScenes failed: %v", err)
	}
	if result.Summary.RequestedCount != 3 || result.Summary.QueuedCount != 1 || result.Summary.SkippedCount != 1 || result.Summary.FailedCount != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected one result per selected key, got %+v", result.Results)
	}
	wantKeys := []string{"stashbox:" + endpointKey(endpoint) + ":" + sceneID, "stash:" + inLibraryID, "missing-key"}
	wantStatuses := []performerdomain.QueueSceneStatus{performerdomain.QueueSceneStatusQueued, performerdomain.QueueSceneStatusSkipped, performerdomain.QueueSceneStatusFailed}
	for index := range wantKeys {
		if result.Results[index].Key != wantKeys[index] || result.Results[index].Status != wantStatuses[index] {
			t.Fatalf("result %d = %+v, want key %q status %s", index, result.Results[index], wantKeys[index], wantStatuses[index])
		}
	}
}
