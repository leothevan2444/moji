package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

var errNoMatchingStashBoxMapping = errors.New("subscription: no stash_id matched any configured stash-box endpoint")

type StashClient interface {
	AllPerformers(ctx context.Context) ([]*stashgraphql.PerformerFragment, error)
	FindPerformerByID(ctx context.Context, id string) (*stashgraphql.PerformerFragment, error)
	FindScenes(ctx context.Context, sceneFilter *stashgraphql.SceneFilterType, filter *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error)
	UpdatePerformerCustomFields(ctx context.Context, id string, partial map[string]any, remove []string) (*stashgraphql.PerformerFragment, error)
	GetStashBoxes(ctx context.Context) ([]stash.StashBoxEndpoint, error)
}

type StashboxClient interface {
	FindPerformerByID(ctx context.Context, id string) (*stashboxgraphql.PerformerFragment, error)
	FindSceneByID(ctx context.Context, id string) (*stashboxgraphql.SceneFragment, error)
	SearchPerformer(ctx context.Context, term string) ([]*stashboxgraphql.PerformerFragment, error)
	SearchScene(ctx context.Context, term string) ([]*stashboxgraphql.SceneFragment, error)
	QueryScenes(ctx context.Context, input stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error)
}

type TaskRuntime interface {
	AddTorrentContext(ctx context.Context, req taskruntime.AddTorrentRequest) (*taskruntime.Task, error)
	DownloadMediaContext(ctx context.Context, req taskruntime.DownloadRequest) (*taskruntime.Task, error)
}

type TaskLister interface {
	ListTasks(ctx context.Context) ([]*taskruntime.Task, error)
}

type TaskCreator interface {
	QueueDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (*taskruntime.Task, error)
	QueueSubscriptionRelease(ctx context.Context, code, title string) (*taskruntime.Task, error)
}

type Service struct {
	stash          StashClient
	stashbox       *stashboxRegistry
	taskCreator    TaskCreator
	taskLister     TaskLister
	store          Store
	customFieldKey string
	now            func() time.Time

	loadMu       sync.RWMutex
	loaded       bool
	loadErrorMsg string

	orderMu       sync.RWMutex
	endpointOrder []string

	policyMu      sync.RWMutex
	releasePolicy ReleasePolicyConfig
}

const (
	releaseQueryPerPage          = 24
	releaseQueryPollMaxPages     = 3
	releaseQueryBackfillMaxPages = 10
)

type releaseFetchMode string

const (
	releaseFetchModePoll     releaseFetchMode = "poll"
	releaseFetchModeBackfill releaseFetchMode = "backfill"
)

type releaseFetchStrategy struct {
	mode     releaseFetchMode
	perPage  int
	maxPages int
}

type releaseFetchStats struct {
	pagesRequested  int
	pagesWithResult int
	stopReason      string
}

func NewService(stash StashClient, stashbox *stashboxRegistry, taskRuntime TaskRuntime, store Store) (*Service, error) {
	if stash == nil {
		return nil, errors.New("subscription: stash client is required")
	}
	if store == nil {
		store = NewMemoryStore()
	}
	if stashbox == nil {
		stashbox = newStashboxRegistry(defaultStashboxClientFactory{})
	}
	creator := newDefaultTaskCreator(taskRuntime, stashbox)
	var taskLister TaskLister
	if candidate, ok := taskRuntime.(TaskLister); ok {
		taskLister = candidate
	}

	return &Service{
		stash:          stash,
		stashbox:       stashbox,
		taskCreator:    creator,
		taskLister:     taskLister,
		store:          store,
		customFieldKey: DefaultCustomFieldKey,
		now:            time.Now,
		releasePolicy:  DefaultReleasePolicyConfig(),
	}, nil
}

func (s *Service) SetTaskLister(lister TaskLister) {
	if s == nil {
		return
	}
	s.taskLister = lister
}

func (s *Service) SetTaskCreator(creator TaskCreator) {
	if s == nil {
		return
	}
	s.taskCreator = creator
}

func (s *Service) ListStashPerformers(ctx context.Context, search string) ([]Performer, error) {
	performers, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}

	needle := normalize(search)
	out := make([]Performer, 0, len(performers))
	for _, performer := range performers {
		item := performerFromStash(performer, s.customFieldKey)
		if needle != "" && !performerMatches(item, needle) {
			continue
		}
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Subscribed != out[j].Subscribed {
			return out[i].Subscribed
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *Service) ListSubscribedPerformers(ctx context.Context) ([]SubscribedPerformer, error) {
	performers, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SubscribedPerformer, 0)
	for _, performer := range performers {
		item := performerFromStash(performer, s.customFieldKey)
		if !item.Subscribed {
			continue
		}
		state, err := s.store.Get(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, buildSubscribedPerformer(item, state))
	}

	sort.Slice(out, func(i, j int) bool {
		left := out[i]
		right := out[j]
		if left.LastCheckedAt == nil || right.LastCheckedAt == nil {
			return left.LastCheckedAt != nil
		}
		return left.LastCheckedAt.After(*right.LastCheckedAt)
	})
	return out, nil
}

func (s *Service) SubscribePerformer(ctx context.Context, performerID string) (SubscribedPerformer, error) {
	performer, err := s.stash.UpdatePerformerCustomFields(ctx, performerID, map[string]any{s.customFieldKey: true}, nil)
	if err != nil {
		logging.Errorf("subscription: subscribe performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}

	item := performerFromStash(performer, s.customFieldKey)
	state, err := s.store.Get(ctx, item.ID)
	if err != nil {
		logging.Errorf("subscription: load state for performer %s after subscribe failed: %v", item.ID, err)
		return SubscribedPerformer{}, err
	}
	logging.Infof("subscription: subscribed performer %s (%s)", item.ID, item.Name)

	return buildSubscribedPerformer(item, state), nil
}

func (s *Service) UnsubscribePerformer(ctx context.Context, performerID string) error {
	if _, err := s.stash.UpdatePerformerCustomFields(ctx, performerID, nil, []string{s.customFieldKey}); err != nil {
		logging.Errorf("subscription: unsubscribe performer %s failed: %v", performerID, err)
		return err
	}
	if err := s.store.Delete(ctx, performerID); err != nil {
		logging.Errorf("subscription: delete state for performer %s failed: %v", performerID, err)
		return err
	}
	logging.Infof("subscription: unsubscribed performer %s", performerID)
	return nil
}

func (s *Service) RefreshSubscribedPerformer(ctx context.Context, performerID string) (SubscribedPerformer, error) {
	logging.Infof("subscription: refresh started for performer %s", performerID)
	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		logging.Errorf("subscription: load performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}
	if performer == nil {
		return SubscribedPerformer{}, fmt.Errorf("subscription: performer %q not found", performerID)
	}

	item := performerFromStash(performer, s.customFieldKey)
	state, err := s.store.Get(ctx, performerID)
	if err != nil {
		logging.Errorf("subscription: load state for performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}
	if state == nil {
		state = &PerformerState{PerformerID: performerID}
	}

	now := s.now().UTC()
	state.LastCheckedAt = &now
	previousLastError := state.LastError
	state.LastError = ""

	releases, err := s.fetchReleases(ctx, performer, state)
	if err != nil {
		state.LastError = err.Error()
		if putErr := s.store.Put(ctx, state); putErr != nil {
			logging.Errorf("subscription: persist error state for performer %s failed: %v", performerID, putErr)
			return SubscribedPerformer{}, putErr
		}
		logging.Errorf("subscription: refresh failed for performer %s (%s): %v", performerID, performer.Name, err)
		if errors.Is(err, errNoMatchingStashBoxMapping) {
			return buildSubscribedPerformer(item, state), nil
		}
		return buildSubscribedPerformer(item, state), err
	}
	logging.Infof("subscription: refresh fetched %d releases for performer %s (%s)", len(releases), performerID, performer.Name)

	processed := make(map[string]RecordedRelease, len(state.ProcessedReleases)+len(state.PendingReleases))
	for _, release := range state.ProcessedReleases {
		processed[release.Key] = release
	}
	for _, release := range state.PendingReleases {
		processed[release.Key] = release
	}

	existingPending := append([]RecordedRelease(nil), state.PendingReleases...)
	pending := make([]RecordedRelease, 0)
	skippedInLibrary := 0
	for _, release := range releases {
		if _, exists := processed[release.Key]; exists {
			continue
		}
		inLibrary, err := s.stashSceneExistsForRelease(ctx, release)
		if err != nil {
			return SubscribedPerformer{}, err
		}
		if inLibrary {
			skippedInLibrary++
			logging.Infof(
				"subscription: skipped in-library release performer=%s release_key=%s stash_box=%s scene_id=%s",
				performerID,
				release.Key,
				release.Source,
				release.SceneID,
			)
			continue
		}
		record := RecordedRelease{
			Key:            release.Key,
			Source:         release.Source,
			Title:          release.Title,
			Code:           release.Code,
			Date:           release.Date,
			URL:            release.URL,
			SeenAt:         now,
			PerformerCount: release.PerformerCount,
			PerformerNames: append([]string(nil), release.PerformerNames...),
			Classification: release.Classification,
			Decision:       release.Decision,
			DecisionReason: release.DecisionReason,
		}
		pending = append(pending, record)
	}
	if len(pending) > 0 {
		logging.Infof("subscription: detected %d new releases for performer %s (%s)", len(pending), performerID, performer.Name)
	}

	if s.taskCreator != nil {
		nextPending := append([]RecordedRelease(nil), existingPending...)
		for i := range pending {
			if pending[i].Decision != ReleaseDecisionDownloaded {
				nextPending = append(nextPending, pending[i])
				continue
			}
			task, err := s.taskCreator.QueueSubscriptionRelease(ctx, pending[i].Code, pending[i].Title)
			if err != nil {
				state.LastError = err.Error()
				logging.Errorf("subscription: auto-download failed for performer %s code %q: %v", performerID, pending[i].Code, err)
				nextPending = append(nextPending, pending[i])
				continue
			}
			if task != nil {
				pending[i].TaskID = task.ID
				logging.Infof("subscription: auto-download created task %s for performer %s code %q", task.ID, performerID, pending[i].Code)
			}
			state.ProcessedReleases = append([]RecordedRelease{pending[i]}, state.ProcessedReleases...)
		}
		state.PendingReleases = nextPending
	} else {
		state.PendingReleases = append(existingPending, pending...)
		if len(pending) > 0 {
			logging.Infof("subscription: queued %d pending releases for performer %s (%s)", len(pending), performerID, performer.Name)
		}
	}
	if state.LastError == "" && len(state.PendingReleases) > 0 && previousLastError != "" {
		state.LastError = previousLastError
	}

	state.ProcessedReleases = trimRecordedReleases(state.ProcessedReleases, 25)
	state.PendingReleases = trimRecordedReleases(state.PendingReleases, 25)
	if err := s.store.Put(ctx, state); err != nil {
		logging.Errorf("subscription: persist state for performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}
	logging.Infof(
		"subscription: refresh completed for performer %s (%s), processed=%d pending=%d skipped_in_library=%d",
		performerID,
		performer.Name,
		len(state.ProcessedReleases),
		len(state.PendingReleases),
		skippedInLibrary,
	)

	return buildSubscribedPerformer(item, state), nil
}

func (s *Service) RefreshAll(ctx context.Context) ([]SubscribedPerformer, error) {
	items, err := s.ListSubscribedPerformers(ctx)
	if err != nil {
		logging.Errorf("subscription: list subscribed performers failed: %v", err)
		return nil, err
	}
	logging.Infof("subscription: refresh all started for %d performers", len(items))

	out := make([]SubscribedPerformer, 0, len(items))
	for _, item := range items {
		refreshed, refreshErr := s.RefreshSubscribedPerformer(ctx, item.Performer.ID)
		if refreshErr != nil {
			out = append(out, refreshed)
			continue
		}
		out = append(out, refreshed)
	}
	logging.Infof("subscription: refresh all completed for %d performers", len(out))
	return out, nil
}

func (s *Service) fetchReleases(ctx context.Context, performer *stashgraphql.PerformerFragment, state *PerformerState) ([]Release, error) {
	if s.stashbox == nil || len(s.stashbox.Endpoints()) == 0 {
		return nil, errors.New("subscription: no stash-box endpoints configured in Stash")
	}

	target, err := s.resolveStashboxPerformer(ctx, performer)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("subscription: no stash-box performer match for %q", performer.Name)
	}

	source := "stash-box:" + target.Endpoint
	keyPrefix := "stashbox:" + endpointKey(target.Endpoint) + ":"
	strategy := selectReleaseFetchStrategy(state)
	policy := s.currentReleasePolicy()
	now := s.now()
	knownReleaseKeys := recordedReleaseKeys(state)
	seenSceneIDs := make(map[string]struct{})
	seenReleaseKeys := make(map[string]struct{})
	releases := make([]Release, 0, releaseQueryPerPage)
	stats := releaseFetchStats{}

	for page := 1; page <= strategy.maxPages; page++ {
		stats.pagesRequested++
		scenes, err := target.Client.QueryScenes(ctx, stashboxgraphql.SceneQueryInput{
			Performers: &stashboxgraphql.MultiIDCriterionInput{
				Value:    []string{target.Performer.ID},
				Modifier: stashboxgraphql.CriterionModifierIncludes,
			},
			Page:      page,
			PerPage:   strategy.perPage,
			Direction: stashboxgraphql.SortDirectionEnumDesc,
			Sort:      stashboxgraphql.SceneSortEnumDate,
		})
		if err != nil {
			return nil, err
		}
		if len(scenes) == 0 {
			stats.stopReason = "empty_page"
			break
		}
		stats.pagesWithResult++

		pageHasUniqueScene := false
		pageReleaseCount := 0
		pageKnownReleaseCount := 0
		pageDownloadCandidateCount := 0
		pageDateBoundaryHitCount := 0

		for _, scene := range scenes {
			if scene == nil {
				continue
			}
			if _, exists := seenSceneIDs[scene.ID]; exists {
				continue
			}
			seenSceneIDs[scene.ID] = struct{}{}
			pageHasUniqueScene = true

			evaluation, matched := evaluateReleasePolicy(policy, now, target.Performer, scene)
			if !matched {
				continue
			}
			code := strings.TrimSpace(stringValue(scene.Code))
			if code == "" {
				return nil, fmt.Errorf("subscription: stash-box scene %q is missing code", scene.ID)
			}
			pageReleaseCount++
			release := Release{
				SceneID:        scene.ID,
				Key:            keyPrefix + scene.ID,
				Source:         source,
				Title:          stringValue(scene.Title),
				Code:           code,
				Date:           stringValue(scene.Date),
				PerformerCount: evaluation.PerformerCount,
				PerformerNames: append([]string(nil), evaluation.PerformerNames...),
				Classification: evaluation.Classification,
				Decision:       evaluation.Decision,
				DecisionReason: evaluation.DecisionReason,
			}
			if len(scene.Urls) > 0 && scene.Urls[0] != nil {
				release.URL = scene.Urls[0].URL
			}
			if _, exists := seenReleaseKeys[release.Key]; !exists {
				seenReleaseKeys[release.Key] = struct{}{}
				releases = append(releases, release)
			}
			if _, exists := knownReleaseKeys[release.Key]; exists {
				pageKnownReleaseCount++
			}
			if candidate, hitBoundary := releaseDateBoundaryCandidate(policy, now, target.Performer, scene, evaluation); candidate {
				pageDownloadCandidateCount++
				if hitBoundary {
					pageDateBoundaryHitCount++
				}
			}
		}

		if !pageHasUniqueScene || pageReleaseCount == 0 {
			stats.stopReason = "empty_page"
			break
		}
		if strategy.mode == releaseFetchModePoll {
			if page > 1 && pageKnownReleaseCount == pageReleaseCount {
				stats.stopReason = "known_release_boundary"
				break
			}
			if pageDownloadCandidateCount > 0 && pageDateBoundaryHitCount == pageDownloadCandidateCount {
				stats.stopReason = "release_date_boundary"
				break
			}
		}
	}
	if stats.stopReason == "" {
		stats.stopReason = "max_pages_reached"
	}
	logging.Infof(
		"subscription: fetched releases performer=%s stash_box=%s mode=%s pages_requested=%d pages_with_results=%d releases=%d stop_reason=%s",
		performer.ID,
		target.Endpoint,
		strategy.mode,
		stats.pagesRequested,
		stats.pagesWithResult,
		len(releases),
		stats.stopReason,
	)
	return releases, nil
}

// resolvedStashbox pairs a Stash-Box client with the performer fragment it
// returned and the endpoint that produced the match.
type resolvedStashbox struct {
	Client    StashboxClient
	Performer *stashboxgraphql.PerformerFragment
	Endpoint  string
	Name      string
}

func (s *Service) resolveStashboxPerformer(ctx context.Context, performer *stashgraphql.PerformerFragment) (*resolvedStashbox, error) {
	if performer == nil {
		return nil, nil
	}

	// Only use explicit stash_ids. We intentionally do not fall back to name
	// search because a false-positive performer match is more dangerous than a
	// visible "no mapping" state.
	stashIDsByEndpoint := make(map[string]*stashgraphql.StashIDFragment, len(performer.StashIds))
	for _, stashID := range performer.StashIds {
		if stashID == nil || strings.TrimSpace(stashID.StashID) == "" {
			continue
		}
		endpoint := normalizeStashBoxEndpoint(stashID.Endpoint)
		if endpoint == "" {
			continue
		}
		if _, exists := stashIDsByEndpoint[endpoint]; exists {
			continue
		}
		stashIDsByEndpoint[endpoint] = stashID
	}

	var firstLookupErr error
	for _, box := range s.orderedEndpoints() {
		stashID, ok := stashIDsByEndpoint[normalizeStashBoxEndpoint(box.Endpoint)]
		if !ok {
			continue
		}
		client, ok := s.stashbox.Get(box.Endpoint)
		if !ok {
			continue
		}
		matched, err := client.FindPerformerByID(ctx, stashID.StashID)
		if err != nil {
			if firstLookupErr == nil {
				firstLookupErr = err
			}
			logging.Warnf("subscription: stash-box lookup failed for endpoint=%s performer=%s: %v", box.Endpoint, stashID.StashID, err)
			continue
		}
		if matched == nil {
			continue
		}
		return &resolvedStashbox{
			Client:    client,
			Performer: matched,
			Endpoint:  normalizeStashBoxEndpoint(box.Endpoint),
			Name:      box.Name,
		}, nil
	}

	if firstLookupErr != nil {
		return nil, firstLookupErr
	}
	return nil, errNoMatchingStashBoxMapping
}

// RefreshStashBoxes asks the Stash server for the current Stash-Box list and
// replaces the registry. Endpoints that the user no longer has selected are
// still cached so subsequent reads don't rebuild clients; this call is the
// single source of truth for which endpoints are available.
func (s *Service) RefreshStashBoxes(ctx context.Context) error {
	if s == nil || s.stash == nil {
		return errors.New("subscription: stash client is not configured")
	}
	boxes, err := s.stash.GetStashBoxes(ctx)
	s.loadMu.Lock()
	if err != nil {
		s.loadErrorMsg = err.Error()
	} else {
		s.loadErrorMsg = ""
		s.loaded = true
	}
	s.loadMu.Unlock()
	if err != nil {
		return err
	}
	s.stashbox.Replace(boxes)
	return nil
}

// SnapshotState returns the Stash-Box endpoints currently cached in the
// registry together with the outcome of the most recent load attempt.
type LoadState struct {
	Loaded   bool
	ErrorMsg string
}

// SnapshotState returns the currently configured Stash-Box endpoints and the
// outcome of the last refresh. Returns nil when the service is not running
// (e.g. in tests that don't exercise the worker).
func (s *Service) SnapshotState() (endpoints []StashBoxEndpoint, state LoadState) {
	if s == nil || s.stashbox == nil {
		return nil, LoadState{}
	}
	s.loadMu.RLock()
	state = LoadState{Loaded: s.loaded, ErrorMsg: s.loadErrorMsg}
	s.loadMu.RUnlock()
	return s.stashbox.Endpoints(), state
}

// SetEndpointOrder records the user's preferred Stash-Box lookup order.
// The order is the priority queue consumed by resolveStashboxPerformer's
// name-search branch. Endpoints present in the registry but missing from
// `order` are queried after the listed ones (in registry order). An empty
// `order` falls back to the registry order entirely.
func (s *Service) SetEndpointOrder(order []string) {
	if s == nil {
		return
	}
	cleaned := make([]string, 0, len(order))
	seen := make(map[string]struct{}, len(order))
	for _, ep := range order {
		ep = strings.TrimSpace(ep)
		if ep == "" {
			continue
		}
		key := normalizeStashBoxEndpoint(ep)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, key)
	}
	s.orderMu.Lock()
	s.endpointOrder = cleaned
	s.orderMu.Unlock()
}

func (s *Service) SetReleasePolicy(policy ReleasePolicyConfig) {
	if s == nil {
		return
	}
	s.policyMu.Lock()
	s.releasePolicy = policy.Effective()
	s.policyMu.Unlock()
}

func (s *Service) currentReleasePolicy() ReleasePolicyConfig {
	if s == nil {
		return DefaultReleasePolicyConfig()
	}
	s.policyMu.RLock()
	policy := s.releasePolicy
	s.policyMu.RUnlock()
	return policy.Effective()
}

// orderedEndpoints merges the user-defined order with the registry order:
// endpoints present in the user list come first (in user order), endpoints
// missing from the user list are appended (in registry order). When the
// user list is empty the registry order is returned unchanged.
func (s *Service) orderedEndpoints() []StashBoxEndpoint {
	if s == nil || s.stashbox == nil {
		return nil
	}
	registry := s.stashbox.Endpoints()
	s.orderMu.RLock()
	userOrder := append([]string(nil), s.endpointOrder...)
	s.orderMu.RUnlock()

	if len(userOrder) == 0 {
		return registry
	}
	byKey := make(map[string]StashBoxEndpoint, len(registry))
	for _, box := range registry {
		byKey[normalizeStashBoxEndpoint(box.Endpoint)] = box
	}
	out := make([]StashBoxEndpoint, 0, len(registry))
	seen := make(map[string]struct{}, len(registry))
	for _, key := range userOrder {
		box, ok := byKey[key]
		if !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		out = append(out, box)
		seen[key] = struct{}{}
	}
	for _, box := range registry {
		key := normalizeStashBoxEndpoint(box.Endpoint)
		if _, dup := seen[key]; dup {
			continue
		}
		out = append(out, box)
		seen[key] = struct{}{}
	}
	return out
}

func endpointKey(endpoint string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(strings.ToLower(endpoint)), "/", "_"), ":", "_")
}

func performerFromStash(performer *stashgraphql.PerformerFragment, customFieldKey string) Performer {
	if performer == nil {
		return Performer{}
	}

	return Performer{
		ID:         performer.ID,
		Name:       performer.Name,
		AliasList:  append([]string(nil), performer.AliasList...),
		Favorite:   performer.Favorite,
		ImagePath:  derefString(performer.ImagePath),
		SceneCount: performer.SceneCount,
		Subscribed: customFieldTruthy(performer.CustomFields, customFieldKey),
	}
}

func buildSubscribedPerformer(performer Performer, state *PerformerState) SubscribedPerformer {
	item := SubscribedPerformer{
		Performer: performer,
	}
	if state == nil {
		return item
	}

	item.LastCheckedAt = state.LastCheckedAt
	item.LastError = state.LastError
	item.PendingReleaseCount = len(state.PendingReleases)
	item.ProcessedReleaseCount = len(state.ProcessedReleases)
	item.RecentReleases = append([]RecordedRelease(nil), state.PendingReleases...)
	item.RecentReleases = append(item.RecentReleases, state.ProcessedReleases...)
	item.RecentReleases = trimRecordedReleases(item.RecentReleases, 10)
	return item
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func performerMatches(performer Performer, needle string) bool {
	if strings.Contains(normalize(performer.Name), needle) {
		return true
	}
	for _, alias := range performer.AliasList {
		if strings.Contains(normalize(alias), needle) {
			return true
		}
	}
	return false
}

func customFieldTruthy(fields map[string]any, key string) bool {
	value, ok := fields[key]
	if !ok {
		return false
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	case float64:
		return typed != 0
	}

	return false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func buildReleaseCode(code, _ string) string {
	return strings.TrimSpace(code)
}

func selectReleaseFetchStrategy(state *PerformerState) releaseFetchStrategy {
	if state == nil || (len(state.ProcessedReleases) == 0 && len(state.PendingReleases) == 0) {
		return releaseFetchStrategy{
			mode:     releaseFetchModeBackfill,
			perPage:  releaseQueryPerPage,
			maxPages: releaseQueryBackfillMaxPages,
		}
	}
	return releaseFetchStrategy{
		mode:     releaseFetchModePoll,
		perPage:  releaseQueryPerPage,
		maxPages: releaseQueryPollMaxPages,
	}
}

func recordedReleaseKeys(state *PerformerState) map[string]struct{} {
	if state == nil {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(state.ProcessedReleases)+len(state.PendingReleases))
	for _, release := range state.ProcessedReleases {
		out[release.Key] = struct{}{}
	}
	for _, release := range state.PendingReleases {
		out[release.Key] = struct{}{}
	}
	return out
}

func (s *Service) stashSceneExistsForRelease(ctx context.Context, release Release) (bool, error) {
	sceneID := strings.TrimSpace(release.SceneID)
	source := strings.TrimSpace(release.Source)
	if sceneID == "" || !strings.HasPrefix(source, "stash-box:") {
		return false, nil
	}
	endpoint := strings.TrimPrefix(source, "stash-box:")
	scenes, err := s.stash.FindScenes(ctx, &stashgraphql.SceneFilterType{
		StashIDEndpoint: &stashgraphql.StashIDCriterionInput{
			Endpoint: stringPointer(endpoint),
			StashID:  stringPointer(sceneID),
			Modifier: stashgraphql.CriterionModifierEquals,
		},
	}, &stashgraphql.FindFilterType{
		Page:    intPointer(1),
		PerPage: intPointer(1),
	})
	if err != nil {
		return false, fmt.Errorf("subscription: check stash library for release %q: %w", release.Key, err)
	}
	return len(scenes) > 0, nil
}

func trimRecordedReleases(items []RecordedRelease, limit int) []RecordedRelease {
	if len(items) <= limit {
		return items
	}
	return append([]RecordedRelease(nil), items[:limit]...)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
