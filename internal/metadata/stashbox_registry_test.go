package metadata

import (
	"context"
	"reflect"
	"testing"

	"github.com/leothevan2444/moji/pkg/stash"
	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestRegistryReplacesByEndpoint(t *testing.T) {
	clientA := &registryTestClient{performer: &stashboxgraphql.PerformerFragment{ID: "a"}}
	clientB := &registryTestClient{performer: &stashboxgraphql.PerformerFragment{ID: "b"}}

	var built []string
	factory := endpointRecordingFactory{
		clients: map[string]Client{
			"https://a.example.org/graphql": clientA,
			"https://b.example.org/graphql": clientB,
		},
		record: &built,
	}

	registry := NewRegistry(factory)

	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-a"},
		{Name: "b", Endpoint: "https://b.example.org/graphql", APIKey: "k-b"},
	})
	if len(built) != 2 {
		t.Fatalf("expected 2 clients built, got %d (%v)", len(built), built)
	}

	// Re-replacing with the same endpoints should NOT rebuild clients
	// (rate limiter state must survive).
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-a"},
		{Name: "b", Endpoint: "https://b.example.org/graphql", APIKey: "k-b"},
	})
	if len(built) != 2 {
		t.Fatalf("expected 2 clients built (no rebuild), got %d (%v)", len(built), built)
	}

	// Dropping an endpoint should remove its client.
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-a"},
	})
	got, ok := registry.Get("https://b.example.org/graphql")
	if ok || got != nil {
		t.Fatalf("expected b to be evicted, got ok=%v client=%v", ok, got)
	}
}

func TestRegistryGetMissingEndpoint(t *testing.T) {
	registry := NewRegistry(registryTestFactory{client: &registryTestClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-a"},
	})

	if got, ok := registry.Get("https://missing.example.org/graphql"); ok || got != nil {
		t.Fatalf("expected missing endpoint to return ok=false, got ok=%v client=%v", ok, got)
	}

	// Case + whitespace insensitive
	got, ok := registry.Get("  HTTPS://A.Example.Org/graphql  ")
	if !ok || got == nil {
		t.Fatalf("expected normalized lookup to hit, got ok=%v", ok)
	}
}

func TestRegistryEndpointsOrder(t *testing.T) {
	registry := NewRegistry(registryTestFactory{client: &registryTestClient{}})
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-a"},
		{Name: "b", Endpoint: "https://b.example.org/graphql", APIKey: ""},
	})

	got := registry.Endpoints()
	want := []StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKeyConfigured: true},
		{Name: "b", Endpoint: "https://b.example.org/graphql", APIKeyConfigured: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected endpoints:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestRegistryNilSafety(t *testing.T) {
	var registry *Registry
	registry.Replace(nil)
	if got := registry.Endpoints(); got != nil {
		t.Fatalf("expected nil endpoints on nil registry, got %+v", got)
	}
	if got, ok := registry.Get("https://x"); ok || got != nil {
		t.Fatalf("expected nil registry Get to miss, got ok=%v client=%v", ok, got)
	}
}

// endpointRecordingFactory returns a prebuilt client and records every
// endpoint it was asked to build, so tests can assert on construction calls.
type endpointRecordingFactory struct {
	clients map[string]Client
	record  *[]string
}

func (f endpointRecordingFactory) NewClient(box stash.StashBoxEndpoint) Client {
	*f.record = append(*f.record, box.Endpoint)
	return f.clients[NormalizeEndpoint(box.Endpoint)]
}

func TestRegistryRebuildsClientWhenConfigChanges(t *testing.T) {
	clientA := &registryTestClient{performer: &stashboxgraphql.PerformerFragment{ID: "a"}}
	clientB := &registryTestClient{performer: &stashboxgraphql.PerformerFragment{ID: "b"}}

	var built []string
	factory := endpointRecordingFactory{
		clients: map[string]Client{
			"https://a.example.org/graphql": clientA,
		},
		record: &built,
	}

	registry := NewRegistry(factory)
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-a", MaxRequestsPerMinute: 60},
	})
	if len(built) != 1 {
		t.Fatalf("expected first build, got %d", len(built))
	}

	factory.clients["https://a.example.org/graphql"] = clientB
	registry.SetFactory(factory)
	registry.Replace([]stash.StashBoxEndpoint{
		{Name: "a", Endpoint: "https://a.example.org/graphql", APIKey: "k-b", MaxRequestsPerMinute: 60},
	})
	if len(built) != 2 {
		t.Fatalf("expected rebuild after api key change, got %d", len(built))
	}
	got, ok := registry.Get("https://a.example.org/graphql")
	if !ok || got != clientB {
		t.Fatalf("expected rebuilt client after config change, got ok=%v client=%v", ok, got)
	}
}

type registryTestFactory struct {
	client Client
}

func (f registryTestFactory) NewClient(stash.StashBoxEndpoint) Client { return f.client }

type registryTestClient struct {
	performer *stashboxgraphql.PerformerFragment
}

func (c *registryTestClient) FindPerformerByID(context.Context, string) (*stashboxgraphql.PerformerFragment, error) {
	return c.performer, nil
}

func (*registryTestClient) FindSceneByID(context.Context, string) (*stashboxgraphql.SceneFragment, error) {
	return nil, nil
}

func (*registryTestClient) SearchPerformer(context.Context, string) ([]*stashboxgraphql.PerformerFragment, error) {
	return nil, nil
}

func (*registryTestClient) SearchScene(context.Context, string) ([]*stashboxgraphql.SceneFragment, error) {
	return nil, nil
}

func (*registryTestClient) QueryScenes(context.Context, stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error) {
	return nil, nil
}

func (*registryTestClient) QueryScenesPage(context.Context, stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error) {
	return stashboxpkg.ScenePage{}, nil
}
