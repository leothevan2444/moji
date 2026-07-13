package metadata

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashboxcache"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

var ErrNoPerformerMapping = errors.New("metadata: no stash_id matched any configured stash-box endpoint")

type EndpointLoader interface {
	GetStashBoxes(context.Context) ([]stash.StashBoxEndpoint, error)
}

type LoadState struct {
	Loaded   bool
	ErrorMsg string
}

type MatchedPerformer struct {
	Client    Client
	Performer *stashboxgraphql.PerformerFragment
	Endpoint  string
	Name      string
}

type Service struct {
	loader    EndpointLoader
	registry  *Registry
	loadMu    sync.RWMutex
	loaded    bool
	loadError string
	orderMu   sync.RWMutex
	order     []string
	cache     *stashboxcache.Service
}

func (s *Service) SetCache(cache *stashboxcache.Service) { s.cache = cache }
func (s *Service) HasCache() bool                        { return s != nil && s.cache != nil }

type CachePolicy = stashboxcache.FreshnessPolicy

const (
	CachePreferred = stashboxcache.CachePreferred
	RequireFresh   = stashboxcache.RequireFresh
	ForceRefresh   = stashboxcache.ForceRefresh
	RefreshHead    = stashboxcache.RefreshHead
)

func NewService(loader EndpointLoader, registry *Registry) *Service {
	if registry == nil {
		registry = NewDefaultRegistry()
	}
	return &Service{loader: loader, registry: registry}
}

func (s *Service) Registry() *Registry                { return s.registry }
func (s *Service) Get(endpoint string) (Client, bool) { return s.registry.Get(endpoint) }
func (s *Service) APIKey(endpoint string) string      { return s.registry.APIKey(endpoint) }

func (s *Service) RefreshStashBoxes(ctx context.Context) error {
	if s == nil || s.loader == nil {
		return errors.New("metadata: stash endpoint loader is not configured")
	}
	boxes, err := s.loader.GetStashBoxes(ctx)
	s.loadMu.Lock()
	if err != nil {
		s.loadError = err.Error()
	} else {
		s.loadError = ""
		s.loaded = true
	}
	s.loadMu.Unlock()
	if err != nil {
		return err
	}
	s.registry.Replace(boxes)
	return nil
}

func (s *Service) SnapshotState() ([]StashBoxEndpoint, LoadState) {
	if s == nil || s.registry == nil {
		return nil, LoadState{}
	}
	s.loadMu.RLock()
	state := LoadState{Loaded: s.loaded, ErrorMsg: s.loadError}
	s.loadMu.RUnlock()
	return s.registry.Endpoints(), state
}

func (s *Service) SetEndpointOrder(order []string) {
	cleaned := make([]string, 0, len(order))
	seen := map[string]struct{}{}
	for _, endpoint := range order {
		key := NormalizeEndpoint(endpoint)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, key)
	}
	s.orderMu.Lock()
	s.order = cleaned
	s.orderMu.Unlock()
}

func (s *Service) Endpoints() []StashBoxEndpoint {
	if s == nil || s.registry == nil {
		return nil
	}
	registered := s.registry.Endpoints()
	s.orderMu.RLock()
	order := append([]string(nil), s.order...)
	s.orderMu.RUnlock()
	if len(order) == 0 {
		return registered
	}
	byKey := make(map[string]StashBoxEndpoint, len(registered))
	for _, box := range registered {
		byKey[NormalizeEndpoint(box.Endpoint)] = box
	}
	out := make([]StashBoxEndpoint, 0, len(registered))
	seen := map[string]struct{}{}
	for _, key := range order {
		if box, ok := byKey[key]; ok {
			out = append(out, box)
			seen[key] = struct{}{}
		}
	}
	for _, box := range registered {
		key := NormalizeEndpoint(box.Endpoint)
		if _, ok := seen[key]; !ok {
			out = append(out, box)
		}
	}
	return out
}

func (s *Service) ResolvePerformer(ctx context.Context, performer *stashgraphql.PerformerFragment) (*MatchedPerformer, error) {
	return s.ResolvePerformerWithPolicy(ctx, performer, CachePreferred)
}

func (s *Service) ResolvePerformerWithPolicy(ctx context.Context, performer *stashgraphql.PerformerFragment, policy CachePolicy) (*MatchedPerformer, error) {
	if performer == nil {
		return nil, nil
	}
	ids := make(map[string]*stashgraphql.StashIDFragment, len(performer.StashIds))
	for _, id := range performer.StashIds {
		if id == nil || strings.TrimSpace(id.StashID) == "" {
			continue
		}
		key := NormalizeEndpoint(id.Endpoint)
		if key == "" {
			continue
		}
		if _, exists := ids[key]; !exists {
			ids[key] = id
		}
	}
	var firstErr error
	for _, box := range s.Endpoints() {
		id, ok := ids[NormalizeEndpoint(box.Endpoint)]
		if !ok {
			continue
		}
		client, ok := s.registry.Get(box.Endpoint)
		if !ok {
			continue
		}
		var matched *stashboxgraphql.PerformerFragment
		var err error
		if s.cache != nil {
			matched, _, err = s.cache.ResolvePerformer(ctx, client, box.Endpoint, id.StashID, policy)
		} else {
			matched, err = client.FindPerformerByID(ctx, id.StashID)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			logging.Warnf("metadata: stash-box lookup failed endpoint=%s performer=%s: %v", box.Endpoint, id.StashID, err)
			continue
		}
		if matched != nil {
			return &MatchedPerformer{Client: client, Performer: matched, Endpoint: NormalizeEndpoint(box.Endpoint), Name: box.Name}, nil
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, ErrNoPerformerMapping
}

func (s *Service) LoadPerformerScenes(ctx context.Context, target *MatchedPerformer, minimum int, policy CachePolicy) (stashboxcache.Result, error) {
	if target == nil || target.Performer == nil {
		return stashboxcache.Result{Complete: true}, nil
	}
	if s.cache == nil {
		result := stashboxcache.Result{}
		for pageNumber := 1; ; pageNumber++ {
			page, err := target.Client.QueryScenesPage(ctx, stashboxgraphql.SceneQueryInput{
				Performers: &stashboxgraphql.MultiIDCriterionInput{Value: []string{target.Performer.ID}, Modifier: stashboxgraphql.CriterionModifierIncludes},
				Page:       pageNumber, PerPage: stashboxcache.PageSize, Direction: stashboxgraphql.SortDirectionEnumDesc, Sort: stashboxgraphql.SceneSortEnumDate,
			})
			if err != nil {
				return stashboxcache.Result{}, err
			}
			result.Scenes = append(result.Scenes, page.Scenes...)
			result.RemoteCount = page.Count
			result.LoadedCount = len(result.Scenes)
			result.Complete = result.LoadedCount >= result.RemoteCount || len(page.Scenes) == 0
			if result.Complete || result.LoadedCount >= minimum {
				return result, nil
			}
		}
	}
	return s.cache.Load(ctx, target.Client, target.Endpoint, target.Performer.ID, minimum, policy)
}
