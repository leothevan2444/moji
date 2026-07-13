package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/taskruntime"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

var errNoMatchingStashBoxMapping = errors.New("subscription: no stash_id matched any configured stash-box endpoint")

type StashClient interface {
	AllPerformers(ctx context.Context) ([]*stashgraphql.PerformerFragment, error)
	FindPerformerByID(ctx context.Context, id string) (*stashgraphql.PerformerFragment, error)
	FindScenes(ctx context.Context, sceneFilter *stashgraphql.SceneFilterType, filter *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error)
	UpdatePerformerCustomFields(ctx context.Context, id string, partial map[string]any, remove []string) (*stashgraphql.PerformerFragment, error)
}

type TaskCreator interface {
	QueueSubscriptionRelease(ctx context.Context, code, title string) (*taskruntime.Task, error)
}

type Service struct {
	stash            StashClient
	metadata         *metadata.Service
	taskCreator      TaskCreator
	imageProxy       *imagecache.Service
	stashImageConfig func() (string, string)
	store            Store
	customFieldKey   string
	now              func() time.Time
	eventPublisher   PerformerSubscriptionEventPublisher

	policyMu      sync.RWMutex
	releasePolicy ReleasePolicyConfig
}

func (s *Service) SetImageProxy(proxy *imagecache.Service, stashConfig func() (string, string)) {
	s.imageProxy = proxy
	s.stashImageConfig = stashConfig
}

func (s *Service) proxyImage(ctx context.Context, kind imagecache.SourceKind, instance, raw, apiKey string) string {
	if s.imageProxy == nil || strings.TrimSpace(raw) == "" {
		return raw
	}
	value, err := s.imageProxy.Register(ctx, imagecache.Descriptor{Kind: kind, InstanceURL: instance, ImageURL: raw, APIKey: apiKey})
	if err != nil {
		logging.Warnf("subscription: register image proxy: %v", err)
		return ""
	}
	return value
}

func (s *Service) proxyStashImage(ctx context.Context, raw string) string {
	if s.stashImageConfig == nil {
		return raw
	}
	base, key := s.stashImageConfig()
	return s.proxyImage(ctx, imagecache.SourceStash, base, raw, key)
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

func NewService(stash StashClient, source *metadata.Service, taskCreator TaskCreator, store Store) (*Service, error) {
	if stash == nil {
		return nil, errors.New("subscription: stash client is required")
	}
	if store == nil {
		store = NewMemoryStore()
	}
	if source == nil {
		return nil, errors.New("subscription: metadata source is required")
	}
	return &Service{
		stash:          stash,
		metadata:       source,
		taskCreator:    taskCreator,
		store:          store,
		customFieldKey: DefaultCustomFieldKey,
		now:            time.Now,
		releasePolicy:  DefaultReleasePolicyConfig(),
	}, nil
}

func (s *Service) SetTaskCreator(creator TaskCreator) {
	if s == nil {
		return
	}
	s.taskCreator = creator
}

func (s *Service) SetEventPublisher(publisher PerformerSubscriptionEventPublisher) {
	if s == nil {
		return
	}
	s.eventPublisher = publisher
}

func (s *Service) publishEvent(eventType PerformerSubscriptionEventType, performerID string, state *SubscribedPerformer) {
	if s == nil || s.eventPublisher == nil {
		return
	}
	s.eventPublisher.Publish(&PerformerSubscriptionEvent{Type: eventType, PerformerID: performerID, State: state})
}

func (s *Service) ListSubscribedPerformers(ctx context.Context) ([]SubscribedPerformer, error) {
	performers, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SubscribedPerformer, 0)
	for _, performer := range performers {
		item := performerFromStash(performer, s.customFieldKey)
		item.ImagePath = s.proxyStashImage(ctx, item.ImagePath)
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
	item.ImagePath = s.proxyStashImage(ctx, item.ImagePath)
	state, err := s.store.Get(ctx, item.ID)
	if err != nil {
		logging.Errorf("subscription: load state for performer %s after subscribe failed: %v", item.ID, err)
		return SubscribedPerformer{}, err
	}
	logging.Infof("subscription: subscribed performer %s (%s)", item.ID, item.Name)

	result := buildSubscribedPerformer(item, state)
	s.publishEvent(PerformerSubscriptionEventCreated, item.ID, &result)
	return result, nil
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
	s.publishEvent(PerformerSubscriptionEventDeleted, performerID, nil)
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
	item.ImagePath = s.proxyStashImage(ctx, item.ImagePath)
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
		result := buildSubscribedPerformer(item, state)
		s.publishEvent(PerformerSubscriptionEventUpdated, performerID, &result)
		logging.Errorf("subscription: refresh failed for performer %s (%s): %v", performerID, performer.Name, err)
		if errors.Is(err, errNoMatchingStashBoxMapping) {
			return result, nil
		}
		return result, err
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
	result := buildSubscribedPerformer(item, state)
	s.publishEvent(PerformerSubscriptionEventUpdated, performerID, &result)
	return result, nil
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
	if s.metadata == nil || len(s.metadata.Endpoints()) == 0 {
		return nil, errors.New("subscription: no stash-box endpoints configured in Stash")
	}

	target, err := s.metadata.ResolvePerformer(ctx, performer)
	if err != nil {
		if errors.Is(err, metadata.ErrNoPerformerMapping) {
			return nil, errNoMatchingStashBoxMapping
		}
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

func endpointKey(endpoint string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(strings.ToLower(endpoint)), "/", "_"), ":", "_")
}

func performerFromStash(raw *stashgraphql.PerformerFragment, customFieldKey string) performer.Performer {
	if raw == nil {
		return performer.Performer{}
	}

	return performer.Performer{
		ID:         raw.ID,
		Name:       raw.Name,
		AliasList:  append([]string(nil), raw.AliasList...),
		Favorite:   raw.Favorite,
		ImagePath:  derefString(raw.ImagePath),
		SceneCount: raw.SceneCount,
		Subscribed: performer.IsSubscribed(raw.CustomFields, customFieldKey),
	}
}

func buildSubscribedPerformer(performer performer.Performer, state *PerformerState) SubscribedPerformer {
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

func stringPointer(value string) *string { return &value }
func intPointer(value int) *int          { return &value }
