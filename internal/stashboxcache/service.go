package stashboxcache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type Service struct {
	store  *sqliteStore
	config ConfigProvider
	now    func() time.Time
	group  singleflight.Group
	locks  sync.Map
	opsMu  sync.RWMutex

	statusMu    sync.RWMutex
	lastCleanup *time.Time
	lastError   string
}

func New(path string, provider ConfigProvider) (*Service, error) {
	store, err := openSQLiteStore(path)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		provider = func() Config { return Config{} }
	}
	return &Service{store: store, config: provider, now: time.Now}, nil
}

func (s *Service) Close() error {
	if s == nil || s.store == nil || s.store.db == nil {
		return nil
	}
	s.opsMu.Lock()
	defer s.opsMu.Unlock()
	return s.store.db.Close()
}

func normalizeEndpoint(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func (s *Service) lockFor(key PerformerKey) *sync.Mutex {
	value, _ := s.locks.LoadOrStore(key.Endpoint+"\x00"+key.PerformerID, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (s *Service) ResolvePerformer(ctx context.Context, client Client, endpoint, performerID string, policy FreshnessPolicy) (*stashboxgraphql.PerformerFragment, bool, error) {
	s.opsMu.RLock()
	defer s.opsMu.RUnlock()
	key := PerformerKey{Endpoint: normalizeEndpoint(endpoint), PerformerID: strings.TrimSpace(performerID)}
	if client == nil || key.Endpoint == "" || key.PerformerID == "" {
		return nil, false, errors.New("stashboxcache: invalid performer lookup")
	}
	mutex := s.lockFor(key)
	mutex.Lock()
	defer mutex.Unlock()
	now := s.now().UTC()
	cfg := s.config().normalize()
	cached, fetchedAt, lastAccessed, found, err := s.store.getPerformer(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if found && policy != ForceRefresh && now.Sub(fetchedAt) < cfg.TTL {
		if shouldTouch(lastAccessed, now) {
			s.store.touchPerformer(ctx, key, now)
		}
		return cached.toGraphQL(), false, nil
	}
	value, err := client.FindPerformerByID(ctx, key.PerformerID)
	if err != nil {
		if found && policy == CachePreferred {
			return cached.toGraphQL(), true, nil
		}
		return nil, false, err
	}
	if value == nil {
		return nil, false, nil
	}
	if err := s.store.putPerformer(ctx, key, performerFromGraphQL(value), now); err != nil {
		return nil, false, err
	}
	return value, false, nil
}

func (s *Service) Load(ctx context.Context, client Client, endpoint, performerID string, minimum int, policy FreshnessPolicy) (Result, error) {
	s.opsMu.RLock()
	defer s.opsMu.RUnlock()
	key := PerformerKey{Endpoint: normalizeEndpoint(endpoint), PerformerID: strings.TrimSpace(performerID)}
	if client == nil || key.Endpoint == "" || key.PerformerID == "" {
		return Result{}, errors.New("stashboxcache: invalid scene lookup")
	}
	if minimum < PageSize {
		minimum = PageSize
	}
	groupKey := fmt.Sprintf("%s\x00%s\x00%d\x00%s", key.Endpoint, key.PerformerID, minimum, policy)
	value, err, _ := s.group.Do(groupKey, func() (any, error) {
		mutex := s.lockFor(key)
		mutex.Lock()
		defer mutex.Unlock()
		return s.loadLocked(ctx, client, key, minimum, policy)
	})
	if err != nil {
		return Result{}, err
	}
	return value.(Result), nil
}

func (s *Service) loadLocked(ctx context.Context, client Client, key PerformerKey, minimum int, policy FreshnessPolicy) (Result, error) {
	now := s.now().UTC()
	cfg := s.config().normalize()
	current, err := s.store.getSnapshot(ctx, key, true)
	if err != nil {
		return Result{}, err
	}
	previous, err := s.store.getSnapshot(ctx, key, false)
	if err != nil {
		return Result{}, err
	}
	fallback := preferredFallback(current, previous, minimum)
	if current != nil && shouldTouch(current.LastAccessed, now) {
		s.store.touchSnapshot(ctx, current, now)
		current.LastAccessed = now
	}
	if policy == ForceRefresh {
		current = nil
	}
	neededPages := (minimum + PageSize - 1) / PageSize
	if policy == RefreshHead {
		neededPages = 1
	}
	for pageNumber := 1; pageNumber <= neededPages; pageNumber++ {
		if current != nil && current.Complete && current.loadedCount() < (pageNumber-1)*PageSize+1 {
			break
		}
		cachedPage := pageAt(current, pageNumber)
		mustFetch := cachedPage == nil || now.Sub(cachedPage.FetchedAt) >= cfg.TTL
		if pageNumber == 1 && (policy == RefreshHead || policy == ForceRefresh) {
			mustFetch = true
		}
		if !mustFetch {
			continue
		}
		upstream, fetchErr := client.QueryScenesPage(ctx, stashboxgraphql.SceneQueryInput{
			Performers: &stashboxgraphql.MultiIDCriterionInput{Value: []string{key.PerformerID}, Modifier: stashboxgraphql.CriterionModifierIncludes},
			Page:       pageNumber, PerPage: PageSize, Direction: stashboxgraphql.SortDirectionEnumDesc, Sort: stashboxgraphql.SceneSortEnumDate,
		})
		if fetchErr != nil {
			if policy == CachePreferred && fallback != nil {
				if shouldTouch(fallback.LastAccessed, now) {
					s.store.touchSnapshot(ctx, fallback, now)
					fallback.LastAccessed = now
				}
				return resultFromSnapshot(fallback, true), nil
			}
			return Result{}, fetchErr
		}
		ids := sceneIDs(upstream.Scenes)
		changed := cachedPage != nil && (!reflect.DeepEqual(cachedPage.SceneIDs, ids) || current.RemoteCount != upstream.Count)
		if current == nil || (pageNumber == 1 && (changed || policy == ForceRefresh)) {
			generation := int64(1)
			if fallback != nil {
				generation = fallback.Generation + 1
			}
			current = &snapshot{Key: key, Generation: generation, RemoteCount: upstream.Count, UpdatedAt: now, LastAccessed: now, Scenes: map[string]sceneDTO{}}
		} else if changed {
			generation := current.Generation + 1
			next := &snapshot{Key: key, Generation: generation, RemoteCount: upstream.Count, UpdatedAt: now, LastAccessed: now, Scenes: map[string]sceneDTO{}}
			for _, oldPage := range current.Pages {
				if oldPage.Number >= pageNumber {
					break
				}
				next.Pages = append(next.Pages, oldPage)
				for _, id := range oldPage.SceneIDs {
					next.Scenes[id] = current.Scenes[id]
				}
			}
			current = next
		}
		current.RemoteCount = upstream.Count
		current.UpdatedAt = now
		current.LastAccessed = now
		setPage(current, cachedPageFrom(pageNumber, now, upstream.Scenes))
		pageDTOs := make([]sceneDTO, 0, len(upstream.Scenes))
		for _, raw := range upstream.Scenes {
			if raw == nil {
				continue
			}
			dto := sceneFromGraphQL(raw)
			current.Scenes[dto.ID] = dto
			pageDTOs = append(pageDTOs, dto)
		}
		current.Complete = current.loadedCount() >= upstream.Count || len(upstream.Scenes) < PageSize
		if err := s.store.putSnapshot(ctx, current, allSnapshotScenes(current, pageDTOs), true); err != nil {
			return Result{}, err
		}
	}
	if current == nil {
		return Result{}, errors.New("stashboxcache: no scene snapshot available")
	}
	return resultFromSnapshot(current, false), nil
}

func shouldTouch(lastAccessed, now time.Time) bool {
	return lastAccessed.IsZero() || !now.Before(lastAccessed.Add(accessTouchInterval))
}

func pageAt(value *snapshot, number int) *cachedPage {
	if value == nil {
		return nil
	}
	for index := range value.Pages {
		if value.Pages[index].Number == number {
			return &value.Pages[index]
		}
	}
	return nil
}

func setPage(value *snapshot, page cachedPage) {
	for index := range value.Pages {
		if value.Pages[index].Number == page.Number {
			value.Pages[index] = page
			return
		}
	}
	value.Pages = append(value.Pages, page)
}

func preferredFallback(current, previous *snapshot, minimum int) *snapshot {
	if current == nil {
		return previous
	}
	if previous == nil {
		return current
	}
	currentCovers := current.loadedCount() >= minimum || current.Complete
	previousCovers := previous.loadedCount() >= minimum || previous.Complete
	if previousCovers && !currentCovers {
		return previous
	}
	if previousCovers == currentCovers && previous.loadedCount() > current.loadedCount() {
		return previous
	}
	return current
}

func cachedPageFrom(number int, now time.Time, scenes []*stashboxgraphql.SceneFragment) cachedPage {
	return cachedPage{Number: number, FetchedAt: now, SceneIDs: sceneIDs(scenes)}
}

func sceneIDs(scenes []*stashboxgraphql.SceneFragment) []string {
	out := make([]string, 0, len(scenes))
	for _, scene := range scenes {
		if scene != nil && strings.TrimSpace(scene.ID) != "" {
			out = append(out, scene.ID)
		}
	}
	return out
}

func allSnapshotScenes(value *snapshot, extra []sceneDTO) []sceneDTO {
	byID := make(map[string]sceneDTO, len(value.Scenes)+len(extra))
	for id, scene := range value.Scenes {
		byID[id] = scene
	}
	for _, scene := range extra {
		byID[scene.ID] = scene
	}
	out := make([]sceneDTO, 0, len(byID))
	for _, scene := range byID {
		out = append(out, scene)
	}
	return out
}

func resultFromSnapshot(value *snapshot, stale bool) Result {
	return Result{Scenes: value.orderedScenes(), RemoteCount: value.RemoteCount, LoadedCount: value.loadedCount(), Complete: value.Complete, UpdatedAt: value.UpdatedAt, Stale: stale, Generation: value.Generation}
}

func (s *Service) Cleanup(ctx context.Context) error {
	s.opsMu.RLock()
	defer s.opsMu.RUnlock()
	now := s.now().UTC()
	err := s.store.cleanup(ctx, now.Add(-s.config().normalize().StaleRetention))
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if err != nil {
		s.lastError = err.Error()
		return err
	}
	s.lastCleanup = &now
	s.lastError = ""
	return nil
}

func (s *Service) StartCleanup(ctx context.Context) {
	go func() {
		_ = s.Cleanup(ctx)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.Cleanup(ctx)
			}
		}
	}()
}

func (s *Service) Clear(ctx context.Context) (Status, error) {
	s.opsMu.Lock()
	defer s.opsMu.Unlock()
	if err := s.store.clear(ctx); err != nil {
		return Status{}, err
	}
	return s.Status(ctx)
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	status, err := s.store.status(ctx)
	s.statusMu.RLock()
	status.LastCleanupAt = s.lastCleanup
	status.LastError = s.lastError
	s.statusMu.RUnlock()
	return status, err
}

func (s *Service) InvalidateEndpoint(ctx context.Context, endpoint string) error {
	s.opsMu.Lock()
	defer s.opsMu.Unlock()
	return s.store.invalidateEndpoint(ctx, normalizeEndpoint(endpoint))
}
