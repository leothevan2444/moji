package metadata

import (
	"context"
	"strings"
	"sync"

	"github.com/leothevan2444/moji/pkg/stash"
	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type Client interface {
	FindPerformerByID(context.Context, string) (*stashboxgraphql.PerformerFragment, error)
	FindSceneByID(context.Context, string) (*stashboxgraphql.SceneFragment, error)
	SearchPerformer(context.Context, string) ([]*stashboxgraphql.PerformerFragment, error)
	SearchScene(context.Context, string) ([]*stashboxgraphql.SceneFragment, error)
	QueryScenes(context.Context, stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error)
	QueryScenesPage(context.Context, stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error)
}

// StashBoxEndpoint is the public description of a Stash-Box instance. The API
// key is intentionally hidden from the GraphQL surface; callers that need to
// build a real client can read it via the registry's internal accessors.
type StashBoxEndpoint struct {
	Name             string
	Endpoint         string
	APIKeyConfigured bool
}

// StashboxClientFactory constructs a Stash-Box client bound to a specific
// endpoint. The default implementation wraps `pkg/stashbox.NewClient`.
type ClientFactory interface {
	NewClient(box stash.StashBoxEndpoint) Client
}

// defaultStashboxClientFactory uses the real pkg/stashbox client builder.
type defaultClientFactory struct{}

func (defaultClientFactory) NewClient(box stash.StashBoxEndpoint) Client {
	return stashboxpkg.NewClient(
		box.Endpoint,
		box.APIKey,
		stashboxpkg.MaxRequestsPerMinute(box.MaxRequestsPerMinute),
	)
}

// NewDefaultRegistry returns a registry backed by the real Stash-Box
// client builder.
func NewDefaultRegistry() *Registry {
	return NewRegistry(defaultClientFactory{})
}

// Registry caches Stash-Box clients keyed by endpoint URL so application
// services can dispatch lookups to the correct backend.
type Registry struct {
	mu      sync.RWMutex
	clients map[string]Client
	specs   map[string]stash.StashBoxEndpoint
	factory ClientFactory
	order   []string
	keys    map[string]StashBoxEndpoint
}

func NewRegistry(factory ClientFactory) *Registry {
	return &Registry{
		clients: map[string]Client{},
		specs:   map[string]stash.StashBoxEndpoint{},
		factory: factory,
		keys:    map[string]StashBoxEndpoint{},
	}
}

func (r *Registry) SetFactory(factory ClientFactory) {
	r.mu.Lock()
	r.factory = factory
	r.mu.Unlock()
}

// Replace rebuilds the registry from the given Stash-Box list. Existing
// clients for endpoints that no longer exist are dropped; new endpoints get
// fresh clients built via the factory.
func (r *Registry) Replace(boxes []stash.StashBoxEndpoint) {
	if r == nil {
		return
	}

	r.mu.RLock()
	existingClients := make(map[string]Client, len(r.clients))
	for key, client := range r.clients {
		existingClients[key] = client
	}
	existingSpecs := make(map[string]stash.StashBoxEndpoint, len(r.specs))
	for key, spec := range r.specs {
		existingSpecs[key] = spec
	}
	r.mu.RUnlock()

	next := make(map[string]Client, len(boxes))
	nextSpecs := make(map[string]stash.StashBoxEndpoint, len(boxes))
	nextKeys := make(map[string]StashBoxEndpoint, len(boxes))
	order := make([]string, 0, len(boxes))
	for _, box := range boxes {
		key := NormalizeEndpoint(box.Endpoint)
		if key == "" {
			continue
		}
		if _, exists := next[key]; exists {
			continue
		}
		// Reuse the existing client only when its auth / rate-limit config is
		// unchanged; otherwise rebuild so runtime refreshes take effect.
		if existing, ok := existingClients[key]; ok && sameClientConfig(existingSpecs[key], box) {
			next[key] = existing
		} else {
			next[key] = r.factory.NewClient(box)
		}
		nextSpecs[key] = box
		nextKeys[key] = StashBoxEndpoint{
			Name:             box.Name,
			Endpoint:         box.Endpoint,
			APIKeyConfigured: strings.TrimSpace(box.APIKey) != "",
		}
		order = append(order, key)
	}

	r.mu.Lock()
	r.clients = next
	r.specs = nextSpecs
	r.keys = nextKeys
	r.order = order
	r.mu.Unlock()
}

func sameClientConfig(left, right stash.StashBoxEndpoint) bool {
	return NormalizeEndpoint(left.Endpoint) == NormalizeEndpoint(right.Endpoint) &&
		left.APIKey == right.APIKey &&
		left.MaxRequestsPerMinute == right.MaxRequestsPerMinute
}

// Get returns the client for a normalized endpoint, or false if none is
// configured.
func (r *Registry) Get(endpoint string) (Client, bool) {
	if r == nil {
		return nil, false
	}
	key := NormalizeEndpoint(endpoint)
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, ok := r.clients[key]
	return client, ok
}

// Endpoints returns a snapshot of the currently configured Stash-Box
// endpoints, in the order they were last received from Stash.
func (r *Registry) Endpoints() []StashBoxEndpoint {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]StashBoxEndpoint, 0, len(r.order))
	for _, key := range r.order {
		if ep, ok := r.keys[key]; ok {
			out = append(out, ep)
		}
	}
	return out
}

func (r *Registry) APIKey(endpoint string) string {
	if r == nil {
		return ""
	}
	key := NormalizeEndpoint(endpoint)
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.specs[key].APIKey
}

// normalizeStashBoxEndpoint trims surrounding whitespace and lowercases the
// endpoint so equivalent URLs (different trailing slashes, case in the host,
// etc.) map to the same registry entry.
func NormalizeEndpoint(endpoint string) string {
	return strings.ToLower(strings.TrimSpace(endpoint))
}
